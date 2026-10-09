package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	portFlag := flag.String("port", getEnv("PORT", "3080"), "Port for Web UI dev server")
	backendFlag := flag.String("backend", getEnv("BACKEND_URL", "http://localhost:3081"), "Backend Control Plane URL to proxy /api/ and /hls/")
	webDirFlag := flag.String("web-dir", getEnv("WEB_DIR", "./web"), "Directory containing static web assets")
	flag.Parse()

	port := *portFlag
	backendURL := *backendFlag
	webDir := *webDirFlag

	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = "."
	}
	if fi, err := os.Stat(filepath.Join(webDir, "dist")); err == nil && fi.IsDir() {
		webDir = filepath.Join(webDir, "dist")
	}

	target, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("Invalid backend URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[PROXY ERROR] %s %s -> %v", r.Method, r.URL.Path, err)
		http.Error(w, fmt.Sprintf("Backend service unavailable (%s): %v", backendURL, err), http.StatusBadGateway)
	}

	fs := http.FileServer(http.Dir(webDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Reverse proxy API, HLS, and EPG streams to the Control Plane
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/hls/") || strings.HasPrefix(r.URL.Path, "/epg/") || r.URL.Path == "/api" || r.URL.Path == "/hls" || r.URL.Path == "/epg" {
			log.Printf("[PROXY] %s %s -> %s%s", r.Method, r.URL.Path, backendURL, r.URL.Path)
			r.Host = target.Host
			proxy.ServeHTTP(w, r)
			return
		}

		// Check if exact static file exists
		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		filePath := filepath.Join(webDir, filepath.FromSlash(cleanPath))
		stat, err := os.Stat(filePath)
		if err == nil && !stat.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		// Root path or directory: let FileServer serve index.html
		if cleanPath == "" || cleanPath == "." {
			fs.ServeHTTP(w, r)
			return
		}

		// Client-side SPA routes fallback to index.html
		if !strings.Contains(path.Base(cleanPath), ".") {
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}

		// Otherwise return 404 via standard file server
		fs.ServeHTTP(w, r)
	})

	log.Printf("================================================================================")
	log.Printf("  MCRFlow Web UI Dev Server")
	log.Printf("  Listening at:    http://localhost:%s", port)
	log.Printf("  Proxying to:     %s (/api/* and /hls/*)", backendURL)
	log.Printf("  Serving assets:  %s", webDir)
	log.Printf("================================================================================")

	addr := fmt.Sprintf(":%s", port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("UI Dev Server failed: %v", err)
	}
}
