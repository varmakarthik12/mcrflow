package database_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

func setupTestDB(t *testing.T) (*database.DB, *database.Repository) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_mcrflow.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	repo := database.NewRepository(db)
	return db, repo
}

func TestSeedData(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	// 1. Verify Indian Broadcast Resolutions (4 presets)
	resolutions, err := repo.ListResolutions()
	if err != nil {
		t.Fatalf("failed to list resolutions: %v", err)
	}
	if len(resolutions) < 4 {
		t.Fatalf("expected at least 4 seeded resolutions, got %d", len(resolutions))
	}

	resMap := make(map[string]models.ResolutionPreset)
	for _, r := range resolutions {
		resMap[r.ID] = r
	}
	expectedPresets := []string{"res-in-1080i50", "res-in-720p50", "res-in-576i50-169", "res-in-4k50"}
	for _, pid := range expectedPresets {
		if _, ok := resMap[pid]; !ok {
			t.Errorf("expected preset %s to be seeded", pid)
		}
	}

	// 2. Verify Ad Templates (3 seeded)
	templates, err := repo.ListAdTemplates()
	if err != nil {
		t.Fatalf("failed to list ad templates: %v", err)
	}
	if len(templates) < 3 {
		t.Fatalf("expected at least 3 seeded ad templates, got %d", len(templates))
	}

	// 3. Verify Storage Mounts (2 seeded)
	mounts, err := repo.ListStorageMounts()
	if err != nil {
		t.Fatalf("failed to list storage mounts: %v", err)
	}
	if len(mounts) < 2 {
		t.Fatalf("expected at least 2 seeded mounts, got %d", len(mounts))
	}

	// 4. Verify Default Channel ch-01
	ch, err := repo.GetChannelByID("ch-01")
	if err != nil {
		t.Fatalf("failed to get default channel ch-01: %v", err)
	}
	if ch.Name != "DD National HD" {
		t.Errorf("unexpected channel name: %s", ch.Name)
	}
	if len(ch.Destinations) < 2 {
		t.Errorf("expected parsed destinations on ch-01, got %d", len(ch.Destinations))
	}

	// 5. Verify Default Admin User
	u, err := repo.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("failed to get default admin user: %v", err)
	}
	if u.Role != models.RoleAdmin {
		t.Errorf("expected user role admin, got %s", u.Role)
	}
}

func TestUserCRUDAndLastAdminProtection(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	// Initial admin exists
	initialAdmin, err := repo.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("failed to get admin: %v", err)
	}

	// Attempting to delete the ONLY admin should fail
	err = repo.DeleteUser(initialAdmin.ID)
	if err != database.ErrCannotDeleteLastAdmin {
		t.Fatalf("expected ErrCannotDeleteLastAdmin, got: %v", err)
	}

	// Create a second admin
	admin2 := &models.User{
		Username: "admin2",
		FullName: "Second Admin",
		Email:    "admin2@example.com",
		Role:     models.RoleAdmin,
	}
	if err := repo.CreateUser(admin2, "password123"); err != nil {
		t.Fatalf("failed to create admin2: %v", err)
	}

	// Now deleting the first admin should succeed
	if err := repo.DeleteUser(initialAdmin.ID); err != nil {
		t.Fatalf("failed to delete first admin when another admin exists: %v", err)
	}

	// Verify update user
	newPass := "newpassword456"
	admin2.FullName = "Updated Admin"
	if err := repo.UpdateUser(admin2, &newPass); err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	updated, err := repo.GetUserByID(admin2.ID)
	if err != nil {
		t.Fatalf("failed to get updated user: %v", err)
	}
	if updated.FullName != "Updated Admin" {
		t.Errorf("expected updated full name, got %s", updated.FullName)
	}

	// Now admin2 is the only admin, deleting should fail again
	if err := repo.DeleteUser(admin2.ID); err != database.ErrCannotDeleteLastAdmin {
		t.Fatalf("expected ErrCannotDeleteLastAdmin for remaining admin, got: %v", err)
	}
}

func TestChannelCRUD(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	ch := &models.Channel{
		ID:           "ch-news-24",
		Name:         "24/7 News Prime",
		CallSign:     "NP-24",
		ResolutionID: "res-in-1080i50",
		LogoPath:     "/logos/np24.png",
		IsActive:     true,
		Destinations: []models.StreamDestination{
			{Type: "udp", Enabled: true, URL: "udp://239.255.0.2", Port: 5002},
		},
	}
	if err := repo.CreateChannel(ch); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	fetched, err := repo.GetChannelByID("ch-news-24")
	if err != nil {
		t.Fatalf("failed to fetch channel: %v", err)
	}
	if fetched.Name != ch.Name || len(fetched.Destinations) != 1 {
		t.Errorf("mismatched channel properties: %+v", fetched)
	}

	fetched.Name = "24/7 News HD Prime"
	if err := repo.UpdateChannel(fetched); err != nil {
		t.Fatalf("failed to update channel: %v", err)
	}

	if err := repo.DeleteChannel("ch-news-24"); err != nil {
		t.Fatalf("failed to delete channel: %v", err)
	}

	_, err = repo.GetChannelByID("ch-news-24")
	if err != database.ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestScheduleCRUDAndQuery(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	item1 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Morning Broadcast News",
		MediaPath:       "/media/news/morning_01.mp4",
		StartTime:       now,
		DurationSeconds: 1800,
	}
	if err := repo.CreateScheduleItem(item1); err != nil {
		t.Fatalf("failed to create schedule item 1: %v", err)
	}

	item2 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Regional Feature Documentary",
		MediaPath:       "/media/docs/kerala_doc.mp4",
		StartTime:       now.Add(1800 * time.Second),
		DurationSeconds: 3600,
	}
	if err := repo.CreateScheduleItem(item2); err != nil {
		t.Fatalf("failed to create schedule item 2: %v", err)
	}

	// List by channel
	items, err := repo.ListScheduleByChannel("ch-01")
	if err != nil {
		t.Fatalf("failed to list schedule: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 schedule items, got %d", len(items))
	}

	// List between
	windowItems, err := repo.ListScheduleBetween("ch-01", now.Add(900*time.Second), now.Add(2000*time.Second))
	if err != nil {
		t.Fatalf("failed to list between: %v", err)
	}
	if len(windowItems) != 2 {
		t.Fatalf("expected 2 overlapping schedule items in window, got %d", len(windowItems))
	}

	// Update item
	item1.ProgramTitle = "Morning News Express"
	if err := repo.UpdateScheduleItem(item1); err != nil {
		t.Fatalf("failed to update schedule item: %v", err)
	}

	// Delete item
	if err := repo.DeleteScheduleItem(item1.ID); err != nil {
		t.Fatalf("failed to delete schedule item: %v", err)
	}
}

func TestEdgeAgentAndHeartbeat(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	agent := &models.EdgeAgent{
		ID:             "agt-delhi-01",
		Hostname:       "delhi-playout-host",
		IPAddress:      "192.168.1.100",
		Port:           3082,
		PairingToken:   "agt_sec_1234567890abcdef",
		Status:         "offline",
		ActiveChannels: []string{"ch-01"},
	}
	if err := repo.CreateEdgeAgent(agent); err != nil {
		t.Fatalf("failed to create edge agent: %v", err)
	}

	// Heartbeat update
	if err := repo.UpdateEdgeAgentHeartbeat("agt-delhi-01", 14.5, 42.1, `["ch-01"]`); err != nil {
		t.Fatalf("failed to update heartbeat: %v", err)
	}

	fetched, err := repo.GetEdgeAgentByID("agt-delhi-01")
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if fetched.Status != "online" || fetched.CPUPercent != 14.5 {
		t.Errorf("unexpected agent status after heartbeat: %+v", fetched)
	}
}

func TestAdTemplateAndStorageMountAndBotCRUD(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	// Ad Template
	at := &models.AdTemplate{
		Name:         "Custom Dynamic L-Bar",
		TemplateType: "composite",
		IsActive:     true,
	}
	if err := repo.CreateAdTemplate(at); err != nil {
		t.Fatalf("failed to create ad template: %v", err)
	}
	at.Name = "Custom Dynamic L-Bar v2"
	if err := repo.UpdateAdTemplate(at); err != nil {
		t.Fatalf("failed to update ad template: %v", err)
	}
	if err := repo.DeleteAdTemplate(at.ID); err != nil {
		t.Fatalf("failed to delete ad template: %v", err)
	}

	// Storage Mount
	sm := &models.StorageMount{
		Name:      "Archive Mount",
		MountType: "local",
		MountPath: "/media/archive",
		IsActive:  true,
	}
	if err := repo.CreateStorageMount(sm); err != nil {
		t.Fatalf("failed to create mount: %v", err)
	}
	sm.Name = "Archive Mount 2"
	if err := repo.UpdateStorageMount(sm); err != nil {
		t.Fatalf("failed to update mount: %v", err)
	}
	if err := repo.DeleteStorageMount(sm.ID); err != nil {
		t.Fatalf("failed to delete mount: %v", err)
	}

	// Bot
	bot := &models.Bot{
		Name:     "Telegram MCR Bot",
		Platform: "telegram",
		APIKey:   "bot1234:ABCDEF",
		IsActive: true,
	}
	if err := repo.CreateBot(bot); err != nil {
		t.Fatalf("failed to create bot: %v", err)
	}
	bot.Name = "Telegram MCR Bot Primary"
	if err := repo.UpdateBot(bot); err != nil {
		t.Fatalf("failed to update bot: %v", err)
	}
	if err := repo.DeleteBot(bot.ID); err != nil {
		t.Fatalf("failed to delete bot: %v", err)
	}
}
