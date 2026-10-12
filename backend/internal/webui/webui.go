package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// dist is replaced by the production Vite build before the final Go build.
//
//go:embed dist
var embedded embed.FS

type spaHandler struct {
	root fs.FS
	file http.Handler
}

func Handler() http.Handler {
	root, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return &spaHandler{root: root, file: http.FileServer(http.FS(root))}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clean := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if clean == "." || clean == "" || clean == "index.html" {
		h.serveIndex(w)
		return
	}

	if info, err := fs.Stat(h.root, clean); err == nil && !info.IsDir() {
		// Immutable Vite fingerprinted assets can be cached; HTML should not.
		if strings.HasPrefix(clean, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		if typ := mime.TypeByExtension(path.Ext(clean)); typ != "" {
			w.Header().Set("Content-Type", typ)
		}
		clone := r.Clone(r.Context())
		clone.URL.Path = "/" + clean
		h.file.ServeHTTP(w, clone)
		return
	}

	// React SPA fallback for client-side views/routes.
	h.serveIndex(w)
}

func (h *spaHandler) serveIndex(w http.ResponseWriter) {
	data, err := fs.ReadFile(h.root, "index.html")
	if err != nil {
		http.Error(w, "frontend not available", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}
