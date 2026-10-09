package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/server"
)

func setupTestServer(t *testing.T) (*database.DB, *server.Server, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "server_test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	srv, err := server.NewServer(server.Config{
		Port:     3081,
		DataDir:  tempDir,
		MediaDir: filepath.Join(tempDir, "media"),
	}, db)
	if err != nil {
		t.Fatalf("failed to initialize server: %v", err)
	}

	// Login as admin to get token
	loginBody, _ := json.Marshal(models.UserCredentials{
		Username: "admin",
		Password: "admin123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("failed to login: %d, body: %s", w.Code, w.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)

	return db, srv, res.Token
}

func TestServerHealthAndAuth(t *testing.T) {
	db, srv, token := setupTestServer(t)
	defer db.Close()

	// 1. Health check
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health check returned %d", w.Code)
	}

	// 2. Auth me with token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/auth/me returned %d: %s", w.Code, w.Body.String())
	}

	// 3. Auth me without token (should fail)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", w.Code)
	}
}

func TestServerChannelsAndPlayout(t *testing.T) {
	db, srv, token := setupTestServer(t)
	defer db.Close()

	// 1. List channels
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list channels returned %d", w.Code)
	}

	var channels []models.Channel
	_ = json.NewDecoder(w.Body).Decode(&channels)
	if len(channels) == 0 {
		t.Fatal("expected at least 1 channel")
	}

	// 2. Start playout
	startBody, _ := json.Marshal(map[string]string{
		"media_path":    "/media/movie.mp4",
		"program_title": "Evening Movie",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/channels/ch-01/playout/start", bytes.NewReader(startBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("start playout returned %d: %s", w.Code, w.Body.String())
	}

	// 3. Get Playout Status
	req = httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-01/playout/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get playout status returned %d", w.Code)
	}

	// 4. Get FFmpeg Command
	req = httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-01/ffmpeg-cmd", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get ffmpeg cmd returned %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ffmpeg") {
		t.Fatalf("expected ffmpeg command in body: %s", w.Body.String())
	}
}

func TestServerHLSAndEPG(t *testing.T) {
	db, srv, _ := setupTestServer(t)
	defer db.Close()

	// 1. HLS master without token should fail (ch-01 has token live_sec_dd1_tok_2026)
	req := httptest.NewRequest(http.MethodGet, "/hls/ch-01/master.m3u8", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for HLS without token, got %d", w.Code)
	}

	// 2. HLS master with token should succeed
	req = httptest.NewRequest(http.MethodGet, "/hls/ch-01/master.m3u8?token=live_sec_dd1_tok_2026", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for HLS master, got %d", w.Code)
	}

	// 3. HLS sliding playlist with token
	req = httptest.NewRequest(http.MethodGet, "/hls/ch-01/playlist.m3u8?token=live_sec_dd1_tok_2026", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for HLS playlist, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "#EXT-X-MEDIA-SEQUENCE:") {
		t.Fatalf("missing media sequence in playlist: %s", w.Body.String())
	}

	// 4. HLS segment download
	req = httptest.NewRequest(http.MethodGet, "/hls/ch-01/segment_10.ts?token=live_sec_dd1_tok_2026", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for HLS segment, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "video/mp2t" {
		t.Errorf("expected video/mp2t content type, got %s", w.Header().Get("Content-Type"))
	}

	// 5. EPG XML endpoint (with epg token epg_sec_dd1_xml_2026)
	req = httptest.NewRequest(http.MethodGet, "/epg/ch-01.xml?token=epg_sec_dd1_xml_2026", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for EPG XML, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "<!DOCTYPE tv SYSTEM \"xmltv.dtd\">") {
		t.Fatalf("missing XMLTV doctype in epg output")
	}
}

func TestServerStaticWebUI(t *testing.T) {
	db, srv, _ := setupTestServer(t)
	defer db.Close()

	// Request root /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for root web UI, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "MCRFlow") {
		t.Fatalf("expected MCRFlow in web index response")
	}
}

func TestServerScheduleSingularAndWrapped(t *testing.T) {
	db, srv, token := setupTestServer(t)
	defer db.Close()

	// Post wrapped payload to singular /api/v1/schedule
	payload := map[string]interface{}{
		"item": map[string]interface{}{
			"channel_id":        "ch-01",
			"title":             "Prime Time Feature",
			"media_file_path":   "/media/storage/feature.mp4",
			"start_time":        "2026-10-10T20:00:00Z",
			"duration_seconds":  7200,
			"end_time":          "2026-10-10T22:00:00Z",
		},
		"action": "FORCE_OVERWRITE",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for schedule create, got %d: %s", w.Code, w.Body.String())
	}

	var created models.ScheduleItem
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode created item: %v", err)
	}
	if created.ProgramTitle != "Prime Time Feature" || created.MediaPath != "/media/storage/feature.mp4" {
		t.Fatalf("unexpected fields in created item: %+v", created)
	}

	// Fetch via singular channel endpoint /api/v1/schedule/channel/{channel_id}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/channel/ch-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for schedule channel list, got %d", w.Code)
	}
}

func TestServerAgentPairingAndNLPBot(t *testing.T) {
	db, srv, token := setupTestServer(t)
	defer db.Close()

	// 1. Agent Pairing
	pairBody, _ := json.Marshal(map[string]interface{}{
		"hostname":   "edge-node-mumbai",
		"ip_address": "192.168.1.100",
		"port":       9090,
		"token":      "agt_sec_8f43a9b2c011e749a1d2e8b409c2513f",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/pair", bytes.NewReader(pairBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 for agent pair, got %d: %s", w.Code, w.Body.String())
	}

	// 1b. Agent Heartbeat (Unauthenticated by JWT, authenticated by pairing token)
	hbBody, _ := json.Marshal(models.AgentHeartbeatPayload{
		AgentID:        "edge-node-mumbai",
		PairingToken:   "agt_sec_8f43a9b2c011e749a1d2e8b409c2513f",
		CPUPercent:     18.5,
		MemoryPercent:  42.0,
		ActiveChannels: []string{"ch-01"},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/heartbeat", bytes.NewReader(hbBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for agent heartbeat, got %d: %s", w.Code, w.Body.String())
	}

	// 2. NLP Bot Command
	botBody, _ := json.Marshal(map[string]interface{}{
		"command":    "channels",
		"channel_id": "ch-01",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/bot/nlp-command", bytes.NewReader(botBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for nlp command, got %d: %s", w.Code, w.Body.String())
	}
	var botResp models.BotMessageResponse
	if err := json.NewDecoder(w.Body).Decode(&botResp); err != nil {
		t.Fatalf("failed to decode bot response: %v", err)
	}
	if !botResp.Success || !strings.Contains(botResp.Reply, "Active Broadcast Channels") {
		t.Fatalf("unexpected bot response: %+v", botResp)
	}

	// 3. Setup Status
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/setup-status", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for setup status, got %d", w.Code)
	}
	var setupResp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&setupResp)
	if setupResp["needs_setup"] != false || setupResp["setup_required"] != false {
		t.Fatalf("unexpected setup status: %+v", setupResp)
	}
}
