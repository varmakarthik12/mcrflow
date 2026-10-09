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
	"strconv"
	"syscall"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	defaultPort := 3081
	if envPort := os.Getenv("MCRFLOW_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	defaultDataDir := "./data"
	if envData := os.Getenv("MCRFLOW_DATA_DIR"); envData != "" {
		defaultDataDir = envData
	}

	defaultMediaDir := "/media/storage"
	if envMedia := os.Getenv("MCRFLOW_MEDIA_DIR"); envMedia != "" {
		defaultMediaDir = envMedia
	}

	defaultTMDBKey := os.Getenv("MCRFLOW_TMDB_KEY")

	port := flag.Int("port", defaultPort, "HTTP listening port for Web UI, REST API, and native HLS stream")
	dataDir := flag.String("data-dir", defaultDataDir, "Directory for SQLite databases, auth tokens, and HLS segments")
	mediaDir := flag.String("media-dir", defaultMediaDir, "Default media library path registered in storage browser")
	tmdbKey := flag.String("tmdb-key", defaultTMDBKey, "Optional TMDb API key for movie/series metadata lookup")

	flag.Parse()

	if *showVersion {
		fmt.Printf("mcrflow-control version %s (commit: %s, built at: %s)\n", version, commit, date)
		return
	}

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Fatal: failed to create data directory '%s': %v", *dataDir, err)
	}

	dbPath := filepath.Join(*dataDir, "mcrflow.db")
	log.Printf("[MCRFlow] Initializing SQLite database at %s...", dbPath)
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("Fatal: failed to open SQLite database: %v", err)
	}
	defer db.Close()

	srv, err := server.NewServer(server.Config{
		Port:     *port,
		DataDir:  *dataDir,
		MediaDir: *mediaDir,
		TMDBKey:  *tmdbKey,
	}, db)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize control plane server: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown channel
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("================================================================================")
		log.Printf("  MCRFLOW MASTER CONTROL PLANE INITIALIZED")
		log.Printf("  Web Management Console: http://localhost:%d", *port)
		log.Printf("  REST API Endpoint:      http://localhost:%d/api/v1", *port)
		log.Printf("  Live HLS Ingress:       http://localhost:%d/hls/{channel_id}/master.m3u8", *port)
		log.Printf("  Database Storage:       %s", dbPath)
		log.Printf("================================================================================")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fatal: control plane server failed: %v", err)
		}
	}()

	<-stopChan
	log.Println("[MCRFlow] Gracefully shutting down control plane server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("[MCRFlow] Control plane server stopped.")
}
