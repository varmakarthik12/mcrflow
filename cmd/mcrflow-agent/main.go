package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mcrflow/internal/auth"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	agentIDFlag := flag.String("agent-id", "", "Unique hostname or ID of this edge playout agent")
	authPathFlag := flag.String("auth-file", "", "Path to persistent agent authentication file")
	portFlag := flag.Int("port", 9095, "Edge Agent RPC / HTTP listening port")
	flag.Parse()

	// Default auth file path based on OS
	authPath := *authPathFlag
	if authPath == "" {
		if os.Getenv("MCRFLOW_AUTH_FILE") != "" {
			authPath = os.Getenv("MCRFLOW_AUTH_FILE")
		} else if _, err := os.Stat("/var/lib/mcrflow"); err == nil {
			authPath = "/var/lib/mcrflow/agent_auth.json"
		} else {
			authPath = filepath.Join(".", "data", "agent_auth.json")
		}
	}

	agentID := *agentIDFlag
	if agentID == "" {
		if h, err := os.Hostname(); err == nil && h != "" {
			agentID = h
		} else {
			agentID = "mcrflow-edge-node"
		}
	}

	authMgr := auth.NewManager(authPath)
	creds, isFresh, err := authMgr.InitializeOrLoadAgentToken(agentID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize agent authentication: %v", err)
	}

	fmt.Println()
	fmt.Println("================================================================================")
	if isFresh {
		fmt.Println("  MCRFLOW EDGE PLAYOUT AGENT - CRYPTOGRAPHIC PAIRING REQUIRED")
		fmt.Println("================================================================================")
		fmt.Printf("  Agent ID:         %s\n", creds.AgentID)
		fmt.Printf("  Listen Port:      :%d\n", *portFlag)
		fmt.Printf("  Persistent Token: %s\n", creds.PairingToken)
		fmt.Printf("  Auth Storage:     %s\n", authPath)
		fmt.Println()
		fmt.Println("  👉 Copy and paste the token above into MCRFlow Web Console:")
		fmt.Println("     Settings > Edge Agents > Pair New Agent Node")
	} else {
		fmt.Println("  MCRFLOW EDGE PLAYOUT AGENT - EXISTING PAIRING RESTORED")
		fmt.Println("================================================================================")
		fmt.Printf("  Agent ID:         %s\n", creds.AgentID)
		fmt.Printf("  Status:           Active (Preserved across container / daemon restart)\n")
		fmt.Printf("  Persistent Token: %s\n", creds.PairingToken)
		fmt.Printf("  Auth Storage:     %s\n", authPath)
	}
	fmt.Println("================================================================================")
	fmt.Println()

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ONLINE","agent_id":"%s","version":"%s"}`, creds.AgentID, version)))
	})

	// Protected control endpoint requiring x-agent-token header
	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		tokenHeader := r.Header.Get("X-Agent-Token")
		if !auth.ValidateToken(tokenHeader, creds.PairingToken) {
			http.Error(w, "unauthorized: invalid agent token", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"OK","message":"authenticated command acknowledged"}`))
	})

	// Protected reset token endpoint
	mux.HandleFunc("/reset-token", func(w http.ResponseWriter, r *http.Request) {
		tokenHeader := r.Header.Get("X-Agent-Token")
		if !auth.ValidateToken(tokenHeader, creds.PairingToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		newCreds, err := authMgr.ResetToken()
		if err != nil {
			http.Error(w, "failed to reset token", http.StatusInternalServerError)
			return
		}

		log.Printf("[AUTH] Token reset via authorized command. New token: %s", newCreds.PairingToken)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"success":true,"new_token":"%s"}`, newCreds.PairingToken)))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *portFlag),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[AGENT] Headless Playout Agent listening on :%d", *portFlag)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Agent server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[SHUTDOWN] Stopping MCRFlow Edge Playout Agent...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	log.Println("[SHUTDOWN] Edge Agent stopped.")
}
