package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type AgentAuth struct {
	AgentID      string    `json:"agent_id"`
	PairingToken string    `json:"pairing_token"`
	CreatedAt    time.Time `json:"created_at"`
}

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "agent-edge-01"
	}

	defaultAgentID := hostname
	if envID := os.Getenv("MCRFLOW_AGENT_ID"); envID != "" {
		defaultAgentID = envID
	}

	defaultPort := 3082
	if envPort := os.Getenv("MCRFLOW_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	defaultDataDir := "./data"
	if envData := os.Getenv("MCRFLOW_DATA_DIR"); envData != "" {
		defaultDataDir = envData
	}

	defaultControlURL := "http://localhost:3081"
	if envCtrl := os.Getenv("MCRFLOW_CONTROL_URL"); envCtrl != "" {
		defaultControlURL = envCtrl
	}

	agentID := flag.String("agent-id", defaultAgentID, "Unique node identifier for this edge playout daemon")
	port := flag.Int("port", defaultPort, "Agent RPC/API listening port for control plane communication")
	dataDir := flag.String("data-dir", defaultDataDir, "Working directory for agent runtime state")
	authFile := flag.String("auth-file", "", "Path to persistent 256-bit cryptographic pairing token file")
	controlURL := flag.String("control-url", defaultControlURL, "Central Control Plane URL for heartbeats")

	flag.Parse()

	if *showVersion {
		fmt.Printf("mcrflow-agent version %s (commit: %s, built at: %s)\n", version, commit, date)
		return
	}

	if *authFile == "" {
		*authFile = filepath.Join(*dataDir, "agent_auth.json")
	}

	// 1. Initialize or load cryptographic pairing token
	authCreds, isNew, err := loadOrGenerateAuth(*authFile, *agentID)
	if err != nil {
		log.Fatalf("Fatal: failed to load or generate agent pairing token: %v", err)
	}

	if isNew {
		fmt.Printf(`
================================================================================
  MCRFLOW EDGE PLAYOUT AGENT - CRYPTOGRAPHIC PAIRING REQUIRED
  Agent ID:      %s
  Listen Port:   :%d
  Pairing Token: %s

  Copy this token into MCRFlow Settings > Edge Agents to authorize this node.
================================================================================
`, authCreds.AgentID, *port, authCreds.PairingToken)
	} else {
		log.Printf("[Agent] Loaded existing cryptographic pairing credentials for agent %s", authCreds.AgentID)
	}

	// 2. Start Agent Heartbeat Worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runHeartbeatWorker(ctx, *controlURL, authCreds.AgentID, authCreds.PairingToken)

	// 3. Start Agent local HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "online",
			"agent_id": authCreds.AgentID,
			"time":     time.Now().UTC(),
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"agent_id":        authCreds.AgentID,
			"port":            *port,
			"paired":          true,
			"cpu_percent":     15.2,
			"memory_percent":  32.0,
			"active_channels": []string{"ch-01"},
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[Agent] Edge Playout Agent daemon running on :%d...", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Agent HTTP server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("[Agent] Shutting down Edge Playout Agent daemon...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	log.Println("[Agent] Edge Playout Agent stopped.")
}

func loadOrGenerateAuth(authFilePath, agentID string) (*AgentAuth, bool, error) {
	if _, err := os.Stat(authFilePath); err == nil {
		data, err := os.ReadFile(authFilePath)
		if err == nil {
			var auth AgentAuth
			if err := json.Unmarshal(data, &auth); err == nil && auth.PairingToken != "" {
				return &auth, false, nil
			}
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(authFilePath)
	_ = os.MkdirAll(dir, 0755)

	// Generate 32 bytes cryptographically secure random token (256 bits)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, false, fmt.Errorf("failed to generate random token: %w", err)
	}

	pairingToken := "agt_sec_" + hex.EncodeToString(tokenBytes)
	auth := &AgentAuth{
		AgentID:      agentID,
		PairingToken: pairingToken,
		CreatedAt:    time.Now().UTC(),
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return nil, false, err
	}

	if err := os.WriteFile(authFilePath, data, 0600); err != nil {
		return nil, false, fmt.Errorf("failed to write auth credentials file: %w", err)
	}

	return auth, true, nil
}

func runHeartbeatWorker(ctx context.Context, controlURL, agentID, token string) {
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	endpoint := fmt.Sprintf("%s/api/v1/agents/heartbeat", strings.TrimSuffix(controlURL, "/"))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := models.AgentHeartbeatPayload{
				AgentID:        agentID,
				PairingToken:   token,
				CPUPercent:     12.5,
				MemoryPercent:  38.2,
				ActiveChannels: []string{"ch-01"},
			}
			body, _ := json.Marshal(payload)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
		}
	}
}
