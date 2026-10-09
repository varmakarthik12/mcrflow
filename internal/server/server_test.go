package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/adtemplate"
	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/bot"
	"github.com/varmakarthik12/mcrflow/internal/channel"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/resolution"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
	"github.com/varmakarthik12/mcrflow/internal/storage"
	"github.com/varmakarthik12/mcrflow/internal/tmdb"
	"github.com/varmakarthik12/mcrflow/internal/user"
)

func createTestServer() (*Server, string) {
	resStore := resolution.NewStore()
	chStore := channel.NewStore(resStore)
	schedStore := schedule.NewStore()
	storageMgr := storage.NewManager()
	tmdbClient := tmdb.NewClient("test-key")
	adStore := adtemplate.NewStore()
	botStore := bot.NewStore(schedStore, storageMgr)
	authMgr := auth.NewManager("")
	userStore := user.NewStore("")

	srv := NewServer(Config{
		ChannelStore:    chStore,
		ResolutionStore: resStore,
		ScheduleStore:   schedStore,
		StorageManager:  storageMgr,
		TmdbClient:      tmdbClient,
		AdTemplateStore: adStore,
		BotStore:        botStore,
		AuthManager:     authMgr,
		UserStore:       userStore,
	})

	_, authResp, _ := userStore.CreateInitialAdmin(user.SetupRequest{
		Username:    "admin",
		Password:    "AdminSecret123!",
		DisplayName: "Master Admin",
		Email:       "admin@mcrflow.tv",
	})

	return srv, authResp.Token
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := createTestServer()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res["service"] != "MCRFlow Control Plane" {
		t.Errorf("expected MCRFlow Control Plane, got %v", res["service"])
	}
}

func TestAuthSetupAndLoginFlow(t *testing.T) {
	resStore := resolution.NewStore()
	srv := NewServer(Config{
		ResolutionStore: resStore,
		ChannelStore:    channel.NewStore(resStore),
		UserStore:       user.NewStore(""),
	})

	// 1. Setup required initially
	reqStatus := httptest.NewRequest("GET", "/api/v1/auth/setup-status", nil)
	recStatus := httptest.NewRecorder()
	srv.Router().ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for setup status, got %d", recStatus.Code)
	}
	var st models.SetupStatus
	_ = json.NewDecoder(recStatus.Body).Decode(&st)
	if !st.SetupRequired {
		t.Fatalf("expected setup_required to be true initially")
	}

	// 2. Unauthenticated request to protected route returns 401
	reqProt := httptest.NewRequest("GET", "/api/v1/channels", nil)
	recProt := httptest.NewRecorder()
	srv.Router().ServeHTTP(recProt, reqProt)
	if recProt.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recProt.Code)
	}

	// 3. Create initial admin
	setupPayload, _ := json.Marshal(user.SetupRequest{
		Username:    "chief_admin",
		Password:    "AdminPass1234!",
		DisplayName: "Chief Admin",
		Email:       "admin@station.com",
	})
	reqSetup := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(setupPayload))
	recSetup := httptest.NewRecorder()
	srv.Router().ServeHTTP(recSetup, reqSetup)
	if recSetup.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for setup, got %d", recSetup.Code)
	}
	var setupResp models.UserAuthResponse
	_ = json.NewDecoder(recSetup.Body).Decode(&setupResp)
	if setupResp.Token == "" || setupResp.User.Role != models.RoleAdmin {
		t.Fatalf("invalid setup auth response")
	}

	// 4. Login with credentials
	loginPayload, _ := json.Marshal(user.LoginRequest{
		Username: "chief_admin",
		Password: "AdminPass1234!",
	})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginPayload))
	recLogin := httptest.NewRecorder()
	srv.Router().ServeHTTP(recLogin, reqLogin)
	if recLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for login, got %d", recLogin.Code)
	}

	// 5. Test /auth/me with Bearer token
	reqMe := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+setupResp.Token)
	recMe := httptest.NewRecorder()
	srv.Router().ServeHTTP(recMe, reqMe)
	if recMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /auth/me, got %d", recMe.Code)
	}
}

func TestRbacContentSchedulerPermissions(t *testing.T) {
	srv, adminToken := createTestServer()

	// 1. Admin creates a user with role content_scheduler
	userPayload, _ := json.Marshal(user.CreateUserRequest{
		Username:    "scheduler_user",
		Password:    "SchedPass123!",
		DisplayName: "Schedule Manager",
		Email:       "scheduler@station.com",
		Role:        models.RoleContentScheduler,
	})
	reqCreate := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(userPayload))
	reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
	recCreate := httptest.NewRecorder()
	srv.Router().ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", recCreate.Code)
	}

	// 2. Login as scheduler
	loginPayload, _ := json.Marshal(user.LoginRequest{
		Username: "scheduler_user",
		Password: "SchedPass123!",
	})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginPayload))
	recLogin := httptest.NewRecorder()
	srv.Router().ServeHTTP(recLogin, reqLogin)
	var loginResp models.UserAuthResponse
	_ = json.NewDecoder(recLogin.Body).Decode(&loginResp)
	schedToken := loginResp.Token

	// Permitted: Content scheduler CAN read channels
	reqChannels := httptest.NewRequest("GET", "/api/v1/channels", nil)
	reqChannels.Header.Set("Authorization", "Bearer "+schedToken)
	recChannels := httptest.NewRecorder()
	srv.Router().ServeHTTP(recChannels, reqChannels)
	if recChannels.Code != http.StatusOK {
		t.Fatalf("expected scheduler to read channels, got %d", recChannels.Code)
	}

	// Permitted: Content scheduler CAN list schedule
	reqSched := httptest.NewRequest("GET", "/api/v1/schedule", nil)
	reqSched.Header.Set("Authorization", "Bearer "+schedToken)
	recSched := httptest.NewRecorder()
	srv.Router().ServeHTTP(recSched, reqSched)
	if recSched.Code != http.StatusOK {
		t.Fatalf("expected scheduler to read schedule, got %d", recSched.Code)
	}

	// FORBIDDEN: Content scheduler CANNOT create channels (403)
	chPayload, _ := json.Marshal(models.Channel{
		Name:     "Unauthorized Channel",
		CallSign: "UNAUTH",
	})
	reqChCreate := httptest.NewRequest("POST", "/api/v1/channels", bytes.NewReader(chPayload))
	reqChCreate.Header.Set("Authorization", "Bearer "+schedToken)
	recChCreate := httptest.NewRecorder()
	srv.Router().ServeHTTP(recChCreate, reqChCreate)
	if recChCreate.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for scheduler creating channel, got %d", recChCreate.Code)
	}

	// FORBIDDEN: Content scheduler CANNOT access user management (403)
	reqUsers := httptest.NewRequest("GET", "/api/v1/users", nil)
	reqUsers.Header.Set("Authorization", "Bearer "+schedToken)
	recUsers := httptest.NewRecorder()
	srv.Router().ServeHTTP(recUsers, reqUsers)
	if recUsers.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for scheduler accessing users, got %d", recUsers.Code)
	}
}

func TestResolutionsEndpoints(t *testing.T) {
	srv, adminToken := createTestServer()

	// 1. GET /api/v1/resolutions -> lists presets including Indian Cable
	req := httptest.NewRequest("GET", "/api/v1/resolutions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var presets []*models.ResolutionPreset
	_ = json.NewDecoder(rec.Body).Decode(&presets)
	if len(presets) < 5 {
		t.Errorf("expected at least 5 default resolution presets, got %d", len(presets))
	}

	// 2. POST /api/v1/resolutions -> create new resolution
	newPreset := models.ResolutionPreset{
		Name:         "Custom 1080p60 Egress",
		Width:        1920,
		Height:       1080,
		FrameRate:    60.0,
		AspectRatio:  "16:9",
		ScanningMode: "progressive",
	}
	body, _ := json.Marshal(newPreset)
	req2 := httptest.NewRequest("POST", "/api/v1/resolutions", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rec2.Code)
	}
}

func TestChannelsEndpoints(t *testing.T) {
	srv, adminToken := createTestServer()

	req := httptest.NewRequest("GET", "/api/v1/channels", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var channels []*models.Channel
	_ = json.NewDecoder(rec.Body).Decode(&channels)
	if len(channels) < 2 {
		t.Errorf("expected at least 2 channels, got %d", len(channels))
	}

	// Verify resolved HLS streaming URL is populated
	if !strings.Contains(channels[0].HlsStreamURL, "/hls/") {
		t.Errorf("expected resolved HlsStreamURL, got %s", channels[0].HlsStreamURL)
	}
}

func TestHlsAndEpgWebTokenSecurity(t *testing.T) {
	srv, adminToken := createTestServer()

	// Create channel with both HLS and EPG web tokens
	ch := models.Channel{
		ID:          "ch-secure-01",
		Name:        "Secure Star HD",
		CallSign:    "STARSEC",
		HlsWebToken: "hls_token_xyz99",
		EpgWebToken: "epg_token_abc88",
	}
	body, _ := json.Marshal(ch)
	reqCreate := httptest.NewRequest("POST", "/api/v1/channels", bytes.NewReader(body))
	reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
	recCreate := httptest.NewRecorder()
	srv.Router().ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", recCreate.Code)
	}

	// 1. EPG URL without token -> 401 Unauthorized
	reqEpgNo := httptest.NewRequest("GET", "/api/v1/channels/ch-secure-01/epg.xml", nil)
	recEpgNo := httptest.NewRecorder()
	srv.Router().ServeHTTP(recEpgNo, reqEpgNo)
	if recEpgNo.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for EPG without token, got %d", recEpgNo.Code)
	}

	// 2. EPG URL with correct token -> 200 OK
	reqEpgOk := httptest.NewRequest("GET", "/api/v1/channels/ch-secure-01/epg.xml?token=epg_token_abc88", nil)
	recEpgOk := httptest.NewRecorder()
	srv.Router().ServeHTTP(recEpgOk, reqEpgOk)
	if recEpgOk.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for EPG with valid token, got %d", recEpgOk.Code)
	}

	// 3. HLS master.m3u8 without token -> 401 Unauthorized
	reqHlsNo := httptest.NewRequest("GET", "/hls/ch-secure-01/master.m3u8", nil)
	recHlsNo := httptest.NewRecorder()
	srv.Router().ServeHTTP(recHlsNo, reqHlsNo)
	if recHlsNo.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for HLS without token, got %d", recHlsNo.Code)
	}

	// 4. HLS master.m3u8 with valid token -> 200 OK
	reqHlsOk := httptest.NewRequest("GET", "/hls/ch-secure-01/master.m3u8?token=hls_token_xyz99", nil)
	recHlsOk := httptest.NewRecorder()
	srv.Router().ServeHTTP(recHlsOk, reqHlsOk)
	if recHlsOk.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for HLS with valid token, got %d", recHlsOk.Code)
	}
	if !strings.Contains(recHlsOk.Body.String(), "#EXTM3U") {
		t.Fatalf("expected #EXTM3U in master.m3u8 response")
	}
}

func TestNlpCommandEndpoint(t *testing.T) {
	srv, adminToken := createTestServer()

	payload := map[string]string{
		"text":       "Schedule Jawan at 11 AM",
		"channel_id": "ch-01",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/bots/nlp-command", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res["parsed_movie_title"] != "Jawan" {
		t.Errorf("expected parsed movie title Jawan, got %v", res["parsed_movie_title"])
	}
}

func TestAgentPairingEndpoint(t *testing.T) {
	srv, adminToken := createTestServer()

	token, _ := auth.GenerateSecureToken()
	payload := map[string]string{
		"agent_id":   "delhi-edge-node-03",
		"ip_address": "100.64.3.50",
		"token":      token,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/agents/pair", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res["success"] != true {
		t.Errorf("expected success true on valid pairing token")
	}

	// Test invalid token prefix
	badPayload := map[string]string{
		"agent_id":   "bad-node",
		"ip_address": "10.0.0.1",
		"token":      "invalid_plain_text",
	}
	badBody, _ := json.Marshal(badPayload)
	reqBad := httptest.NewRequest("POST", "/api/v1/agents/pair", bytes.NewReader(badBody))
	reqBad.Header.Set("Authorization", "Bearer "+adminToken)
	recBad := httptest.NewRecorder()
	srv.Router().ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid token format, got %d", recBad.Code)
	}
}

func TestExportXmltvEndpoint(t *testing.T) {
	srv, _ := createTestServer()

	// Public EPG URL works without bearer token
	req := httptest.NewRequest("GET", "/api/v1/channels/ch-01/epg.xml", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/xml") {
		t.Errorf("expected application/xml content-type, got %s", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<tv generator-info-name=\"MCRFlow Master Control Playout\">") {
		t.Errorf("expected MCRFlow XMLTV header in output")
	}
}
