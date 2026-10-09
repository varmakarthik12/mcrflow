package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mcrflow/internal/adtemplate"
	"mcrflow/internal/auth"
	"mcrflow/internal/bot"
	"mcrflow/internal/channel"
	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
	"mcrflow/internal/schedule"
	"mcrflow/internal/storage"
	"mcrflow/internal/tmdb"
)

func createTestServer() *Server {
	resStore := resolution.NewStore()
	chStore := channel.NewStore(resStore)
	schedStore := schedule.NewStore()
	storageMgr := storage.NewManager()
	tmdbClient := tmdb.NewClient("test-key")
	adStore := adtemplate.NewStore()
	botStore := bot.NewStore(schedStore, storageMgr)
	authMgr := auth.NewManager("")

	return NewServer(Config{
		ChannelStore:    chStore,
		ResolutionStore: resStore,
		ScheduleStore:   schedStore,
		StorageManager:  storageMgr,
		TmdbClient:      tmdbClient,
		AdTemplateStore: adStore,
		BotStore:        botStore,
		AuthManager:     authMgr,
	})
}

func TestHealthEndpoint(t *testing.T) {
	srv := createTestServer()
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

func TestResolutionsEndpoints(t *testing.T) {
	srv := createTestServer()

	// 1. GET /api/v1/resolutions -> lists presets including Indian Cable
	req := httptest.NewRequest("GET", "/api/v1/resolutions", nil)
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
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rec2.Code)
	}
}

func TestChannelsEndpoints(t *testing.T) {
	srv := createTestServer()

	req := httptest.NewRequest("GET", "/api/v1/channels", nil)
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
}

func TestNlpCommandEndpoint(t *testing.T) {
	srv := createTestServer()

	payload := map[string]string{
		"text":       "Schedule Jawan at 11 AM",
		"channel_id": "ch-01",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/bots/nlp-command", bytes.NewReader(body))
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
	srv := createTestServer()

	token, _ := auth.GenerateSecureToken()
	payload := map[string]string{
		"agent_id":   "delhi-edge-node-03",
		"ip_address": "100.64.3.50",
		"token":      token,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/agents/pair", bytes.NewReader(body))
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
	recBad := httptest.NewRecorder()
	srv.Router().ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid token format, got %d", recBad.Code)
	}
}

func TestExportXmltvEndpoint(t *testing.T) {
	srv := createTestServer()

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
