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

	"mcrflow/internal/adtemplate"
	"mcrflow/internal/auth"
	"mcrflow/internal/bot"
	"mcrflow/internal/channel"
	"mcrflow/internal/hls"
	"mcrflow/internal/resolution"
	"mcrflow/internal/schedule"
	"mcrflow/internal/server"
	"mcrflow/internal/storage"
	"mcrflow/internal/tmdb"
	"mcrflow/internal/user"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		var n int
		if _, err := fmt.Sscanf(val, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func main() {
	defaultPort := getEnvInt("MCRFLOW_PORT", getEnvInt("PORT", 8080))
	defaultDataDir := getEnv("MCRFLOW_DATA_DIR", getEnv("MCRFLOW_STORAGE_PATH", "./data"))
	defaultTmdbKey := getEnv("MCRFLOW_TMDB_KEY", "")
	defaultUiDir := getEnv("MCRFLOW_UI_DIR", "./ui-mockup")

	port := flag.Int("port", defaultPort, "HTTP server listening port (env: MCRFLOW_PORT)")
	dataDir := flag.String("data-dir", defaultDataDir, "Directory for persistent databases and state (env: MCRFLOW_DATA_DIR)")
	tmdbKey := flag.String("tmdb-key", defaultTmdbKey, "TMDb API key for movie metadata (env: MCRFLOW_TMDB_KEY)")
	uiDir := flag.String("ui-dir", defaultUiDir, "Path to web UI static assets (env: MCRFLOW_UI_DIR)")
	flag.Parse()

	log.Printf("================================================================================")
	log.Printf("  MCRFlow Control Plane (Master Control Playout) - Version: %s (%s)", version, commit)
	log.Printf("================================================================================")

	_ = os.MkdirAll(*dataDir, 0700)
	authPath := filepath.Join(*dataDir, "agent_auth.json")

	// Initialize stores & services
	resStore := resolution.NewStore()
	chStore := channel.NewStore(resStore)
	schedStore := schedule.NewStore()
	storageMgr := storage.NewManager()
	tmdbClient := tmdb.NewClient(*tmdbKey)
	adStore := adtemplate.NewStore()
	botStore := bot.NewStore(schedStore, storageMgr)
	authMgr := auth.NewManager(authPath)
	userStore := user.NewStore(filepath.Join(*dataDir, "users.json"))
	hlsMgr := hls.NewManager(filepath.Join(*dataDir, "hls"), func(channelID string) string {
		if ch, err := chStore.GetChannel(channelID); err == nil && ch != nil {
			return ch.HlsWebToken
		}
		return ""
	})

	// Ensure local agent pairing token exists if running in all-in-one mode
	creds, isFresh, err := authMgr.InitializeOrLoadAgentToken("embedded-local-agent")
	if err == nil {
		if isFresh {
			log.Printf("[AUTH] Fresh embedded pairing token initialized: %s", creds.PairingToken)
		} else {
			log.Printf("[AUTH] Restored existing pairing token: %s", creds.PairingToken)
		}
	}

	var staticFs http.FileSystem
	if _, err := os.Stat(*uiDir); err == nil {
		staticFs = http.Dir(*uiDir)
		log.Printf("[UI] Serving UI assets from %s", *uiDir)
	}

	srv := server.NewServer(server.Config{
		ChannelStore:    chStore,
		ResolutionStore: resStore,
		ScheduleStore:   schedStore,
		StorageManager:  storageMgr,
		TmdbClient:      tmdbClient,
		AdTemplateStore: adStore,
		BotStore:        botStore,
		AuthManager:     authMgr,
		UserStore:       userStore,
		HlsManager:      hlsMgr,
		StaticFS:        staticFs,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[HTTP] Control Plane listening at http://0.0.0.0:%d", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[SHUTDOWN] Gracefully shutting down MCRFlow Control Plane...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Server forced to shutdown: %v", err)
	}
	log.Println("[SHUTDOWN] MCRFlow Control Plane exited cleanly.")
}
