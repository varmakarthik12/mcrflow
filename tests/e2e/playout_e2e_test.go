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
	"time"

	"mcrflow/internal/adtemplate"
	"mcrflow/internal/auth"
	"mcrflow/internal/bot"
	"mcrflow/internal/channel"
	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
	"mcrflow/internal/schedule"
	"mcrflow/internal/server"
	"mcrflow/internal/storage"
	"mcrflow/internal/tmdb"
)

func TestEndToEndPlayoutWorkflow(t *testing.T) {
	// 1. Setup temp storage for persistent auth tokens
	tempDir, err := os.MkdirTemp("", "mcrflow-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	agentAuthFile := filepath.Join(tempDir, "agent_auth.json")
	controlAuthFile := filepath.Join(tempDir, "control_auth.json")

	// 2. Initialize Edge Agent Token
	agentAuthMgr := auth.NewManager(agentAuthFile)
	agentCreds, isFresh, err := agentAuthMgr.InitializeOrLoadAgentToken("delhi-edge-primary-01")
	if err != nil || !isFresh {
		t.Fatalf("failed to initialize fresh agent token: %v", err)
	}
	t.Logf("[E2E] Edge Agent Token Generated: %s", agentCreds.PairingToken)

	// 3. Initialize Control Plane Server
	resStore := resolution.NewStore()
	chStore := channel.NewStore(resStore)
	schedStore := schedule.NewStore()
	storageMgr := storage.NewManager()
	tmdbClient := tmdb.NewClient("tmdb-test-key")
	adStore := adtemplate.NewStore()
	botStore := bot.NewStore(schedStore, storageMgr)
	controlAuthMgr := auth.NewManager(controlAuthFile)

	srv := server.NewServer(server.Config{
		ChannelStore:    chStore,
		ResolutionStore: resStore,
		ScheduleStore:   schedStore,
		StorageManager:  storageMgr,
		TmdbClient:      tmdbClient,
		AdTemplateStore: adStore,
		BotStore:        botStore,
		AuthManager:     controlAuthMgr,
	})

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	// 4. Test Health Endpoint
	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v", err)
	}
	var health map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&health)
	if health["service"] != "MCRFlow Control Plane" {
		t.Errorf("expected MCRFlow Control Plane service name")
	}

	// 5. Test Resolution Management (Indian Cable Presets)
	resp, err = http.Get(ts.URL + "/api/v1/resolutions")
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
	body, _ := json.Marshal(customResPayload)
	resp, err = http.Post(ts.URL+"/api/v1/resolutions", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create custom resolution: %v", err)
	}
	var createdRes models.ResolutionPreset
	_ = json.NewDecoder(resp.Body).Decode(&createdRes)

	// 6. Pair Edge Agent with Control Plane using Cryptographic Token
	pairPayload := map[string]string{
		"agent_id":   agentCreds.AgentID,
		"ip_address": "100.64.1.15",
		"token":      agentCreds.PairingToken,
	}
	pairBody, _ := json.Marshal(pairPayload)
	resp, err = http.Post(ts.URL+"/api/v1/agents/pair", "application/json", bytes.NewReader(pairBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to pair agent: %v", err)
	}
	var pairResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&pairResp)
	if pairResp["success"] != true {
		t.Errorf("agent pairing was not successful")
	}

	// 7. Create Channel bound to the new resolution preset & paired agent
	newChannel := models.Channel{
		Name:                 "Star Gold Regional",
		CallSign:             "SG-REG",
		LogicalChannelNumber: 105,
		PrimaryAgentID:       agentCreds.AgentID,
		FallbackAgentID:      "mumbai-dc2-hotstandby",
		RedundancyMode:       models.RedundancyActivePassiveAutoFailover,
		ResolutionPresetID:   createdRes.ID,
		Destinations: []models.StreamDestination{
			{
				Protocol:    models.ProtocolUDPMulticast,
				Enabled:     true,
				EndpointURL: "udp://239.255.10.5:5000",
			},
		},
	}
	chBody, _ := json.Marshal(newChannel)
	resp, err = http.Post(ts.URL+"/api/v1/channels", "application/json", bytes.NewReader(chBody))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create channel: %v", err)
	}
	var createdChannel models.Channel
	_ = json.NewDecoder(resp.Body).Decode(&createdChannel)

	// 8. Probe Storage Media File
	resp, err = http.Get(ts.URL + "/api/v1/storage/probe?path=Jawan.2023.1080p.mkv")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to probe storage media: %v", err)
	}
	var probe models.MediaProbeResult
	_ = json.NewDecoder(resp.Body).Decode(&probe)
	if probe.DurationString != "02:49:12" {
		t.Errorf("expected probed duration 02:49:12, got %s", probe.DurationString)
	}

	// 9. Natural Language Scheduling (ChatOps) with Conflict Resolution
	nlpPayload := map[string]string{
		"text":       "Schedule Avengers at 16:30",
		"channel_id": "ch-01",
	}
	nlpBody, _ := json.Marshal(nlpPayload)
	resp, err = http.Post(ts.URL+"/api/v1/bots/nlp-command", "application/json", bytes.NewReader(nlpBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to process ChatOps command: %v", err)
	}
	var nlpResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&nlpResp)

	if nlpResp["conflict_detected"] != true {
		t.Errorf("expected conflict detected at 16:30 with on-air movie")
	}

	// 10. Resolve Conflict via QUEUE_AFTER
	itemPayload := map[string]interface{}{
		"item": models.ScheduleItem{
			ChannelID:       createdChannel.ID,
			MediaFilePath:   "/movies/Avengers.mkv",
			StartTime:       time.Now().UTC().Add(2 * time.Hour),
			DurationSeconds: probe.DurationSeconds,
		},
		"action": models.ConflictQueueAfter,
	}
	itemBody, _ := json.Marshal(itemPayload)
	resp, err = http.Post(ts.URL+"/api/v1/schedule", "application/json", bytes.NewReader(itemBody))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to commit schedule item: %v", err)
	}

	// 11. Verify Multilingual EPG Export
	resp, err = http.Get(ts.URL + "/api/v1/channels/ch-01/epg.xml")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to export XMLTV EPG: %v", err)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	xmlStr := buf.String()

	if !strings.Contains(xmlStr, "<tv generator-info-name=\"MCRFlow Master Control Playout\">") {
		t.Errorf("expected MCRFlow XMLTV header")
	}

	t.Log("[E2E] End-to-End Playout Verification Passed Successfully!")
}
