package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
	port := getEnv("PORT", "3080")
	backendURL := getEnv("BACKEND_URL", "http://localhost:3081")

	target, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("Invalid backend URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	webDir := "./web"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = "."
	}

	fs := http.FileServer(http.Dir(webDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Reverse proxy API and HLS streams to the Control Plane
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/hls/") {
			log.Printf("[PROXY] %s %s -> %s%s", r.Method, r.URL.Path, backendURL, r.URL.Path)
			r.Host = target.Host
			proxy.ServeHTTP(w, r)
			return
		}

		// Static assets with SPA fallback
		cleanPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		filePath := filepath.Join(webDir, cleanPath)
		stat, err := os.Stat(filePath)
		if err != nil || stat.IsDir() {
			if cleanPath != "" && !strings.HasSuffix(cleanPath, ".html") && !strings.Contains(filepath.Base(cleanPath), ".") {
				// Fallback to index.html for client-side routing
				r.URL.Path = "/"
			}
		}

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
