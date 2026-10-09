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
	"mcrflow/internal/resolution"
	"mcrflow/internal/schedule"
	"mcrflow/internal/server"
	"mcrflow/internal/storage"
	"mcrflow/internal/tmdb"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server listening port")
	dataDir := flag.String("data-dir", "./data", "Directory for local databases and persistent assets")
	tmdbKey := flag.String("tmdb-key", "", "TMDb API key for movie metadata")
	uiDir := flag.String("ui-dir", "./ui-mockup", "Path to web UI static assets")
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
