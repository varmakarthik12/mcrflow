package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/adtemplate"
	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/bot"
	"github.com/varmakarthik12/mcrflow/internal/channel"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/resolution"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
	"github.com/varmakarthik12/mcrflow/internal/server"
	"github.com/varmakarthik12/mcrflow/internal/storage"
	"github.com/varmakarthik12/mcrflow/internal/tmdb"
	"github.com/varmakarthik12/mcrflow/internal/user"
)

func TestEndToEndPlayoutWorkflow(t *testing.T) {
	// 1. Setup temp directory for persistent auth tokens and users
	tempDir, err := os.MkdirTemp("", "mcrflow-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	agentAuthFile := filepath.Join(tempDir, "agent_auth.json")
	controlAuthFile := filepath.Join(tempDir, "control_auth.json")
	usersFile := filepath.Join(tempDir, "users.json")

	// 2. Initialize Edge Agent Token
	agentAuthMgr := auth.NewManager(agentAuthFile)
	agentCreds, isFresh, err := agentAuthMgr.InitializeOrLoadAgentToken("delhi-edge-primary-01")
	if err != nil || !isFresh {
		t.Fatalf("failed to initialize fresh agent token: %v", err)
	}
	t.Logf("[E2E] Edge Agent Token Generated: %s", agentCreds.PairingToken)

	// 3. Initialize Control Plane Server with User Store and Channel Store
	resStore := resolution.NewStore()
	chStore := channel.NewStore(resStore)
	schedStore := schedule.NewStore()
	storageMgr := storage.NewManager()
	tmdbClient := tmdb.NewClient("tmdb-test-key")
	adStore := adtemplate.NewStore()
	botStore := bot.NewStore(schedStore, storageMgr)
	controlAuthMgr := auth.NewManager(controlAuthFile)
	userStore := user.NewStore(usersFile)

	srv := server.NewServer(server.Config{
		ChannelStore:    chStore,
		ResolutionStore: resStore,
		ScheduleStore:   schedStore,
		StorageManager:  storageMgr,
		TmdbClient:      tmdbClient,
		AdTemplateStore: adStore,
		BotStore:        botStore,
		AuthManager:     controlAuthMgr,
		UserStore:       userStore,
	})

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	// Helper for authenticated requests
	authRequest := func(method, urlStr string, token string, body interface{}) (*http.Response, error) {
		var bodyReader *bytes.Reader
		if body != nil {
			data, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(data)
		} else {
			bodyReader = bytes.NewReader([]byte{})
		}
		req, err := http.NewRequest(method, urlStr, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return http.DefaultClient.Do(req)
	}

	// 4. Test Health Endpoint (Always Public)
	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v", err)
	}
	var health map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&health)
	if health["service"] != "MCRFlow Control Plane" {
		t.Errorf("expected MCRFlow Control Plane service name")
	}

	// 5. Verify First-Launch Setup Status
	resp, err = http.Get(ts.URL + "/api/v1/auth/setup-status")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get setup status: %v", err)
	}
	var setupStatus models.SetupStatus
	_ = json.NewDecoder(resp.Body).Decode(&setupStatus)
	if !setupStatus.SetupRequired {
		t.Fatalf("expected setup_required to be true on first launch")
	}

	// Verify protected endpoint returns 401 Unauthorized before login
	resp, err = http.Get(ts.URL + "/api/v1/channels")
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on channels endpoint, got %v", resp.StatusCode)
	}

	// 6. Complete First-Launch Setup: Create First Administrator
	setupPayload := user.SetupRequest{
		Username:    "admin",
		Password:    "SuperMasterSecret123!",
		DisplayName: "Chief Broadcast Engineer",
		Email:       "chief@mcrflow.tv",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/auth/setup", "", setupPayload)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("initial admin setup failed: %v", err)
	}
	var adminAuth models.UserAuthResponse
	_ = json.NewDecoder(resp.Body).Decode(&adminAuth)
	adminToken := adminAuth.Token
	if adminToken == "" || adminAuth.User.Role != models.RoleAdmin {
		t.Fatalf("invalid admin setup response: %+v", adminAuth)
	}
	t.Logf("[E2E] First Admin Created successfully: %s", adminAuth.User.Username)

	// Verify setup_required is now false
	resp, err = http.Get(ts.URL + "/api/v1/auth/setup-status")
	_ = json.NewDecoder(resp.Body).Decode(&setupStatus)
	if setupStatus.SetupRequired {
		t.Fatalf("expected setup_required to be false after initial setup")
	}

	// 7. Test Login with Admin Credentials
	loginPayload := user.LoginRequest{
		Username: "admin",
		Password: "SuperMasterSecret123!",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/auth/login", "", loginPayload)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login failed: %v", err)
	}
	var loginAuth models.UserAuthResponse
	_ = json.NewDecoder(resp.Body).Decode(&loginAuth)
	if loginAuth.Token == "" {
		t.Fatalf("login did not return a session token")
	}

	// 8. Create a 'content_scheduler' User and Verify RBAC Boundaries
	schedulerPayload := user.CreateUserRequest{
		Username:    "scheduler_rahul",
		Password:    "SchedulerPass789!",
		DisplayName: "Rahul Sharma (Scheduler)",
		Email:       "rahul@mcrflow.tv",
		Role:        models.RoleContentScheduler,
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/users", adminToken, schedulerPayload)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create content scheduler user: %v", err)
	}

	// Login as content_scheduler
	schedLoginPayload := user.LoginRequest{
		Username: "scheduler_rahul",
		Password: "SchedulerPass789!",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/auth/login", "", schedLoginPayload)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("content_scheduler login failed: %v", err)
	}
	var schedAuth models.UserAuthResponse
	_ = json.NewDecoder(resp.Body).Decode(&schedAuth)
	schedToken := schedAuth.Token

	// Verify content_scheduler CAN view channels & schedule
	resp, err = authRequest("GET", ts.URL+"/api/v1/channels", schedToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("scheduler should be able to view channels, got status: %v", resp.StatusCode)
	}

	// Verify content_scheduler CANNOT create channels (403 Forbidden)
	badChPayload := models.Channel{Name: "Unauthorized Channel"}
	resp, err = authRequest("POST", ts.URL+"/api/v1/channels", schedToken, badChPayload)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("scheduler should NOT be able to create channels, expected 403 got: %v", resp.StatusCode)
	}

	// Verify content_scheduler CANNOT view or manage users (403 Forbidden)
	resp, err = authRequest("GET", ts.URL+"/api/v1/users", schedToken, nil)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("scheduler should NOT be able to view users, expected 403 got: %v", resp.StatusCode)
	}
	t.Log("[E2E] RBAC enforcement for content_scheduler verified successfully!")

	// 9. Admin Resolution Management (Indian Cable Presets)
	resp, err = authRequest("GET", ts.URL+"/api/v1/resolutions", adminToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to list resolutions: %v", err)
	}
	var presets []*models.ResolutionPreset
	_ = json.NewDecoder(resp.Body).Decode(&presets)

	found1080i := false
	found576i := false
	for _, p := range presets {
		if p.ID == "res-1080i50-pal-hd" {
			found1080i = true
		}
		if p.ID == "res-576i50-sd-4x3" {
			found576i = true
		}
	}
	if !found1080i || !found576i {
		t.Errorf("expected default Indian cable presets (1080i50 and 576i50)")
	}

	// Create custom cable resolution via API
	customResPayload := models.ResolutionPreset{
		Name:             "720p50 Regional News Preset",
		Category:         "Custom Indian News",
		Width:            1280,
		Height:           720,
		FrameRate:        50.0,
		AspectRatio:      "16:9",
		ScanningMode:     "progressive",
		VideoBitrateKbps: 4500,
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/resolutions", adminToken, customResPayload)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create custom resolution: %v", err)
	}
	var createdRes models.ResolutionPreset
	_ = json.NewDecoder(resp.Body).Decode(&createdRes)

	// 10. Pair Edge Agent with Control Plane using Cryptographic Token
	pairPayload := map[string]string{
		"agent_id":   agentCreds.AgentID,
		"ip_address": "100.64.1.15",
		"token":      agentCreds.PairingToken,
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/agents/pair", adminToken, pairPayload)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to pair agent: %v", err)
	}
	var pairResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&pairResp)
	if pairResp["success"] != true {
		t.Errorf("agent pairing was not successful")
	}

	// 11. Create Channel with HlsWebToken & EpgWebToken Security
	newChannel := models.Channel{
		Name:                 "Star Gold Regional HD",
		CallSign:             "SG-REG-HD",
		LogicalChannelNumber: 105,
		PrimaryAgentID:       agentCreds.AgentID,
		FallbackAgentID:      "mumbai-dc2-hotstandby",
		RedundancyMode:       models.RedundancyActivePassiveAutoFailover,
		ResolutionPresetID:   createdRes.ID,
		HlsWebToken:          "hls_secure_tok_12345",
		EpgWebToken:          "epg_secure_tok_67890",
		Destinations: []models.StreamDestination{
			{
				Protocol:    models.ProtocolUDPMulticast,
				Enabled:     true,
				EndpointURL: "udp://239.255.10.5:5000",
			},
		},
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/channels", adminToken, newChannel)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create channel: %v", err)
	}
	var createdChannel models.Channel
	_ = json.NewDecoder(resp.Body).Decode(&createdChannel)

	// Verify that HlsStreamURL was automatically resolved on channel creation
	expectedHlsURL := fmt.Sprintf("http://%s/hls/%s/master.m3u8?token=%s", ts.Listener.Addr().String(), createdChannel.ID, createdChannel.HlsWebToken)
	if !strings.Contains(createdChannel.HlsStreamURL, "/hls/"+createdChannel.ID+"/master.m3u8") {
		t.Errorf("expected resolved HlsStreamURL containing /hls/%s/master.m3u8, got %s", createdChannel.ID, createdChannel.HlsStreamURL)
	}
	t.Logf("[E2E] Created Channel ID: %s, Resolved HLS URL: %s (expected pattern: %s)", createdChannel.ID, createdChannel.HlsStreamURL, expectedHlsURL)

	// 12. Probe Storage Media File (accessible by scheduler)
	resp, err = authRequest("GET", ts.URL+"/api/v1/storage/probe?path=Jawan.2023.1080p.mkv", schedToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to probe storage media: %v", err)
	}
	var probe models.MediaProbeResult
	_ = json.NewDecoder(resp.Body).Decode(&probe)
	if probe.DurationString != "02:49:12" {
		t.Errorf("expected probed duration 02:49:12, got %s", probe.DurationString)
	}

	// 13. Natural Language Scheduling (ChatOps) with Conflict Resolution
	nlpPayload := map[string]string{
		"text":       "Schedule Avengers at 16:30",
		"channel_id": createdChannel.ID,
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/bots/nlp-command", schedToken, nlpPayload)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to process ChatOps command: %v", err)
	}
	var nlpResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&nlpResp)

	// 14. Scheduler commits item with QUEUE_AFTER conflict resolution
	itemPayload := map[string]interface{}{
		"item": models.ScheduleItem{
			ChannelID:       createdChannel.ID,
			MediaFilePath:   "/movies/Avengers.mkv",
			StartTime:       time.Now().UTC().Add(2 * time.Hour),
			DurationSeconds: probe.DurationSeconds,
		},
		"action": models.ConflictQueueAfter,
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/schedule", schedToken, itemPayload)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to commit schedule item: %v", err)
	}

	// 15. HLS Live Streaming: Push 5 segments and verify sliding 10-segment window & token auth
	stream := srv.HlsManager().GetOrCreateStream(createdChannel.ID)
	for i := 0; i < 5; i++ {
		stream.AppendSegment(6.0)
	}

	// Request HLS master playlist WITHOUT required webtoken -> should be 401
	hlsMasterURL := ts.URL + "/hls/" + createdChannel.ID + "/master.m3u8"
	resp, err = http.Get(hlsMasterURL)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without HLS webtoken, got %v", resp.StatusCode)
	}

	// Request HLS master playlist WITH valid webtoken -> should be 200
	hlsAuthURL := hlsMasterURL + "?token=" + createdChannel.HlsWebToken
	resp, err = http.Get(hlsAuthURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to fetch authenticated HLS playlist: %v", err)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	m3u8Content := buf.String()

	// Verify HLS playlist structure: strictly 10 segments and correct sequence
	if !strings.Contains(m3u8Content, "#EXTM3U") {
		t.Errorf("missing #EXTM3U in playlist")
	}
	tsCount := strings.Count(m3u8Content, ".ts")
	if tsCount != 10 {
		t.Errorf("expected strictly 10 active segments in live HLS playlist, got %d", tsCount)
	}

	// Fetch a specific TS segment with token
	activeSegments := stream.GetActiveSegments()
	if len(activeSegments) != 10 {
		t.Fatalf("expected 10 active segments, got %d", len(activeSegments))
	}
	latestSegment := activeSegments[9].Filename

	tsURL := fmt.Sprintf("%s/hls/%s/%s?token=%s", ts.URL, createdChannel.ID, latestSegment, createdChannel.HlsWebToken)
	resp, err = http.Get(tsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to fetch HLS TS segment with token: %v", err)
	}
	segBuf := new(bytes.Buffer)
	_, _ = segBuf.ReadFrom(resp.Body)
	if segBuf.Len() == 0 {
		t.Errorf("expected non-empty TS segment data")
	}

	// 16. Verify Multilingual EPG Export with EpgWebToken Security
	epgBaseURL := ts.URL + "/api/v1/channels/" + createdChannel.ID + "/epg.xml"

	// Without token -> 401 Unauthorized
	resp, err = http.Get(epgBaseURL)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without EPG token, got %v", resp.StatusCode)
	}

	// With token -> 200 OK
	resp, err = http.Get(epgBaseURL + "?token=" + createdChannel.EpgWebToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to export XMLTV EPG with token: %v", err)
	}
	epgBuf := new(bytes.Buffer)
	_, _ = epgBuf.ReadFrom(resp.Body)
	xmlStr := epgBuf.String()

	if !strings.Contains(xmlStr, "<tv generator-info-name=\"MCRFlow Master Control Playout\">") {
		t.Errorf("expected MCRFlow XMLTV header")
	}

	t.Log("[E2E] End-to-End Playout Verification (Setup, Auth, RBAC, HLS 10-seg sliding window, Token security, EPG) Passed Successfully!")
}
