package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded filesystem.
func DistFS() embed.FS {
	return distFS
}

// Handler returns an http.Handler serving the embedded React application with SPA routing fallback.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		f, err := sub.Open(path)
		if err != nil {
			// SPA fallback: rewrite to root index.html
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	})
}

// GetIndexHTML returns the raw bytes of the production index.html.
func GetIndexHTML() ([]byte, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(sub, "index.html")
}
