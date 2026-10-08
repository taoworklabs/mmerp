package app

import (
	"io/fs"
	"net/http"
	"strings"
)

// spa serves built files, and index.html for every other path so client routes work on reload.
func spa(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() && name != "index.html" {
			// File names under assets/ carry a content hash.
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}
		// index.html must never be cached: after an update it points to new asset hashes.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}
