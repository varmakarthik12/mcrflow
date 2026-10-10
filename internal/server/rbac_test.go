package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
	}{
		{"/api/v1/health", "/api/v1/health", true},
		{"/api/v1/channels/{id}", "/api/v1/channels/ch-01", true},
		{"/api/v1/channels/{id}/playout/start", "/api/v1/channels/ch-01/playout/start", true},
		{"/api/v1/schedules/channel/{channel_id}", "/api/v1/schedules/channel/ch-01", true},
		{"/api/v1/users/{id}", "/api/v1/users", false},
		{"/api/v1/users", "/api/v1/users/usr-01", false},
		{"/api/v1/storage/browse", "/api/v1/storage/browse", true},
	}

	for _, tt := range tests {
		got := matchPath(tt.pattern, tt.path)
		if got != tt.match {
			t.Errorf("matchPath(%q, %q) = %v; want %v", tt.pattern, tt.path, got, tt.match)
		}
	}
}

func TestEnterpriseAuthMiddleware(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	tokenSvc := auth.NewTokenService("test-secret-key-1234567890123456")

	// Create test users: admin, operator, content_scheduler
	adminUser := &models.User{
		ID:           "u-admin",
		Username:     "admin_tester",
		PasswordHash: "hashed",
		Role:         models.RoleAdmin,
		DisplayName:  "Admin Tester",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_ = repo.CreateUser(adminUser, "admin123")

	opUser := &models.User{
		ID:           "u-operator",
		Username:     "op_tester",
		PasswordHash: "hashed",
		Role:         models.RoleOperator,
		DisplayName:  "Operator Tester",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_ = repo.CreateUser(opUser, "op123")

	schedUser := &models.User{
		ID:           "u-scheduler",
		Username:     "sched_tester",
		PasswordHash: "hashed",
		Role:         models.RoleContentScheduler,
		DisplayName:  "Scheduler Tester",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_ = repo.CreateUser(schedUser, "sched123")

	adminToken, _ := tokenSvc.GenerateToken(adminUser.ID, adminUser.Username, adminUser.Role, time.Hour)
	opToken, _ := tokenSvc.GenerateToken(opUser.ID, opUser.Username, opUser.Role, time.Hour)
	schedToken, _ := tokenSvc.GenerateToken(schedUser.ID, schedUser.Username, schedUser.Role, time.Hour)

	mw := EnterpriseAuthMiddleware(tokenSvc, repo)
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	protected := mw(okHandler)

	// Case 1: Public endpoint passes without token
	reqPublic := httptest.NewRequest("GET", "/api/v1/health", nil)
	rrPublic := httptest.NewRecorder()
	protected.ServeHTTP(rrPublic, reqPublic)
	if rrPublic.Code != http.StatusOK {
		t.Errorf("expected 200 for public health, got %d", rrPublic.Code)
	}

	// Case 2: Protected endpoint without token -> 401
	reqNoToken := httptest.NewRequest("GET", "/api/v1/channels", nil)
	rrNoToken := httptest.NewRecorder()
	protected.ServeHTTP(rrNoToken, reqNoToken)
	if rrNoToken.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing token, got %d", rrNoToken.Code)
	}

	// Case 3: Admin accesses user creation -> 200
	reqAdmin := httptest.NewRequest("POST", "/api/v1/users", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	rrAdmin := httptest.NewRecorder()
	protected.ServeHTTP(rrAdmin, reqAdmin)
	if rrAdmin.Code != http.StatusOK {
		t.Errorf("expected 200 for admin user creation, got %d", rrAdmin.Code)
	}

	// Case 4: Operator tries to create user (Admin only) -> 403 Forbidden
	reqOpUser := httptest.NewRequest("POST", "/api/v1/users", nil)
	reqOpUser.Header.Set("Authorization", "Bearer "+opToken)
	rrOpUser := httptest.NewRecorder()
	protected.ServeHTTP(rrOpUser, reqOpUser)
	if rrOpUser.Code != http.StatusForbidden {
		t.Errorf("expected 403 for operator on admin route, got %d", rrOpUser.Code)
	}

	// Case 5: Scheduler tries to start playout (Admin/Operator only) -> 403 Forbidden
	reqSchedPlayout := httptest.NewRequest("POST", "/api/v1/channels/ch-01/playout/start", nil)
	reqSchedPlayout.Header.Set("Authorization", "Bearer "+schedToken)
	rrSchedPlayout := httptest.NewRecorder()
	protected.ServeHTTP(rrSchedPlayout, reqSchedPlayout)
	if rrSchedPlayout.Code != http.StatusForbidden {
		t.Errorf("expected 403 for scheduler on playout start, got %d", rrSchedPlayout.Code)
	}

	// Case 6: Scheduler accesses schedule endpoint -> 200 OK
	reqSchedSchedule := httptest.NewRequest("GET", "/api/v1/schedules", nil)
	reqSchedSchedule.Header.Set("Authorization", "Bearer "+schedToken)
	rrSchedSchedule := httptest.NewRecorder()
	protected.ServeHTTP(rrSchedSchedule, reqSchedSchedule)
	if rrSchedSchedule.Code != http.StatusOK {
		t.Errorf("expected 200 for scheduler on schedules list, got %d", rrSchedSchedule.Code)
	}
}
