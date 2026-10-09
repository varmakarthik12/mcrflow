package web

import (
	"embed"
	"net/http"
)

//go:embed index.html
var files embed.FS

// AssetFS returns the embedded filesystem serving the MCRFlow web management console.
func AssetFS() http.FileSystem {
	return http.FS(files)
}
