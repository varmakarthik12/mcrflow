package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/server"
)

func TestEndToEndPlayoutWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcrflow-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "mcrflow_e2e.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	srv, err := server.NewServer(server.Config{
		Port:     3081,
		DataDir:  tempDir,
		MediaDir: filepath.Join(tempDir, "media"),
		TMDBKey:  "",
	}, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

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

	// 1. Test Health Endpoint
	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v", err)
	}

	// 2. Protected endpoint returns 401 before login
	resp, err = http.Get(ts.URL + "/api/v1/channels")
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized before login, got %v", resp.StatusCode)
	}

	// 3. Initial Setup of Root Admin
	setupPayload := map[string]string{
		"username":  "admin",
		"password":  "admin123",
		"full_name": "Master Control Administrator",
		"email":     "admin@mcrflow.tv",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/auth/setup", "", setupPayload)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("initial root admin setup failed: %v, code: %d", err, resp.StatusCode)
	}

	// Verify repeated setup is strictly rejected with 403 Forbidden
	respRepeat, err := authRequest("POST", ts.URL+"/api/v1/auth/setup", "", setupPayload)
	if err != nil || respRepeat.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for repeated setup, got %v", respRepeat.StatusCode)
	}

	// Login with Root Admin Credentials
	loginPayload := models.UserCredentials{
		Username: "admin",
		Password: "admin123",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/auth/login", "", loginPayload)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login failed: %v", err)
	}
	var loginRes struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&loginRes)
	adminToken := loginRes.Token
	if adminToken == "" || loginRes.User.Role != models.RoleAdmin {
		t.Fatalf("invalid admin login response: %+v", loginRes)
	}

	// 4. Create Content Scheduler user
	schedulerPayload := struct {
		models.User
		Password string `json:"password"`
	}{
		User: models.User{
			Username: "scheduler_rahul",
			FullName: "Rahul Sharma",
			Email:    "rahul@mcrflow.tv",
			Role:     models.RoleContentScheduler,
		},
		Password: "SchedulerPass789!",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/users", adminToken, schedulerPayload)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create content scheduler: %v", err)
	}

	// Login as content scheduler
	resp, err = authRequest("POST", ts.URL+"/api/v1/auth/login", "", models.UserCredentials{
		Username: "scheduler_rahul",
		Password: "SchedulerPass789!",
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("scheduler login failed: %v", err)
	}
	var schedRes struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&schedRes)
	schedToken := schedRes.Token

	// 5. Verify RBAC boundaries
	// Scheduler CAN view channels
	resp, err = authRequest("GET", ts.URL+"/api/v1/channels", schedToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("scheduler should view channels, got %d", resp.StatusCode)
	}

	// Scheduler CANNOT create channels (403 Forbidden)
	resp, err = authRequest("POST", ts.URL+"/api/v1/channels", schedToken, models.Channel{Name: "Forbidden Ch", CallSign: "FC"})
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for scheduler creating channel, got %d", resp.StatusCode)
	}

	// Scheduler CANNOT view users (403 Forbidden)
	resp, err = authRequest("GET", ts.URL+"/api/v1/users", schedToken, nil)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for scheduler viewing users, got %d", resp.StatusCode)
	}

	// 6. Verify Seeded Indian Resolutions
	resp, err = authRequest("GET", ts.URL+"/api/v1/resolutions", adminToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to list resolutions: %v", err)
	}
	var presets []models.ResolutionPreset
	_ = json.NewDecoder(resp.Body).Decode(&presets)
	if len(presets) < 4 {
		t.Fatalf("expected at least 4 presets, got %d", len(presets))
	}

	// 7. Create New Channel with Token Protection
	newCh := models.Channel{
		ID:           "ch-stargold",
		Name:         "Star Gold Regional HD",
		CallSign:     "SG-REG-HD",
		ResolutionID: "res-in-1080i50",
		LogoPath:     "/logos/stargold.png",
		HlsWebToken:  "hls_secure_tok_12345",
		EpgWebToken:  "epg_secure_tok_67890",
		Destinations: []models.StreamDestination{
			{Type: "udp", Enabled: true, URL: "udp://239.255.10.5", Port: 5000},
		},
		IsActive: true,
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/channels", adminToken, newCh)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create channel: %v", err)
	}

	// 8. Probe Storage Media File (accessible by scheduler)
	resp, err = authRequest("GET", ts.URL+"/api/v1/storage/probe?path=Jawan.2023.1080p.mkv", schedToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to probe storage media: %v", err)
	}
	var probe models.MediaProbeResult
	_ = json.NewDecoder(resp.Body).Decode(&probe)
	if probe.VideoCodec == "" || len(probe.AudioTracks) == 0 {
		t.Errorf("expected valid probed media properties: %+v", probe)
	}

	// 9. ChatOps NLP Scheduling
	nlpReq := models.BotMessageRequest{
		Message: "schedule RRR at 18:00 on ch-stargold",
	}
	resp, err = authRequest("POST", ts.URL+"/api/v1/bots/bot-01/command", adminToken, nlpReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("ChatOps command failed: %v", err)
	}
	var botResp models.BotMessageResponse
	_ = json.NewDecoder(resp.Body).Decode(&botResp)
	if !botResp.Success {
		t.Fatalf("bot response indicates failure: %+v", botResp)
	}

	// 10. Playout Start & FFmpeg Command Generation
	resp, err = authRequest("POST", ts.URL+"/api/v1/channels/ch-stargold/playout/start", adminToken, map[string]string{
		"media_path":    "/media/movies/rrr.mp4",
		"program_title": "RRR Feature Presentation",
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to start playout: %v", err)
	}

	resp, err = authRequest("GET", ts.URL+"/api/v1/channels/ch-stargold/ffmpeg-cmd", adminToken, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get ffmpeg command: %v", err)
	}
	var cmdRes map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&cmdRes)
	if !strings.Contains(cmdRes["command"], "ffmpeg") || !strings.Contains(cmdRes["command"], "-filter_complex") {
		t.Fatalf("unexpected ffmpeg command: %s", cmdRes["command"])
	}

	// 11. HLS Live Streaming: Token enforcement & Sliding Window
	// Request without token -> 401 Unauthorized
	resp, err = http.Get(ts.URL + "/hls/ch-stargold/master.m3u8")
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for HLS without token, got %v", resp.StatusCode)
	}

	// Request with token -> 200 OK
	resp, err = http.Get(ts.URL + "/hls/ch-stargold/playlist.m3u8?token=" + newCh.HlsWebToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for HLS with token, got %v", resp.StatusCode)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	m3u8Content := buf.String()

	// Verify 10 active segments in sliding window
	tsCount := strings.Count(m3u8Content, ".ts")
	if tsCount != 10 {
		t.Fatalf("expected 10 active segments in HLS playlist, got %d", tsCount)
	}

	// Fetch TS segment with token
	resp, err = http.Get(ts.URL + "/hls/ch-stargold/segment_100.ts?token=" + newCh.HlsWebToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for TS segment, got %v", resp.StatusCode)
	}
	segBytes := new(bytes.Buffer)
	_, _ = segBytes.ReadFrom(resp.Body)
	if segBytes.Len() == 0 || segBytes.Bytes()[0] != 0x47 {
		t.Fatalf("invalid TS segment data")
	}

	// 12. XMLTV EPG Export with Token
	// Without token -> 401 Unauthorized
	resp, err = http.Get(ts.URL + "/epg/ch-stargold.xml")
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for EPG without token, got %v", resp.StatusCode)
	}

	// With token -> 200 OK XML
	resp, err = http.Get(ts.URL + "/epg/ch-stargold.xml?token=" + newCh.EpgWebToken)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for EPG with token, got %v", resp.StatusCode)
	}
	epgBuf := new(bytes.Buffer)
	_, _ = epgBuf.ReadFrom(resp.Body)
	if !strings.Contains(epgBuf.String(), "<!DOCTYPE tv SYSTEM \"xmltv.dtd\">") {
		t.Fatalf("missing XMLTV doctype: %s", epgBuf.String())
	}

	t.Log("[E2E] Complete MCRFlow Playout Workflow Verified Successfully!")
}
