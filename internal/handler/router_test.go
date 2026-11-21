package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Uranury/oci-platform/internal/auth"
	"github.com/Uranury/oci-platform/internal/handler"
)

// TestFrontendFiles needs no database: none of these requests reach a handler.
func TestFrontendFiles(t *testing.T) {
	t.Parallel()
	web := fstest.MapFS{
		"index.html":           {Data: []byte("<html>")},
		"favicon.svg":          {Data: []byte("<svg/>")},
		"sw.js":                {Data: []byte("self")},
		"manifest.webmanifest": {Data: []byte("{}")},
		"assets/index-abc.js":  {Data: []byte("1")},
	}
	h := handler.NewRouter(nil, nil, nil, nil, auth.NewJWTIssuer("test-secret-test-secret-test-secret", time.Minute), web)

	tests := []struct {
		method, path string
		status       int
		cache        string
		ctype        string // checked when set
	}{
		{"GET", "/", http.StatusOK, "no-cache", ""},
		{"GET", "/favicon.svg", http.StatusOK, "no-cache", ""},
		// The service worker must revalidate, or a deploy would never reach it.
		{"GET", "/sw.js", http.StatusOK, "no-cache", "text/javascript; charset=utf-8"},
		{"GET", "/manifest.webmanifest", http.StatusOK, "no-cache", "application/manifest+json"},
		{"GET", "/assets/index-abc.js", http.StatusOK, "public, max-age=31536000, immutable", ""},
		{"HEAD", "/assets/index-abc.js", http.StatusOK, "public, max-age=31536000, immutable", ""},
		{"GET", "/missing.js", http.StatusNotFound, "", ""}, // http.FileServer drops it on errors
		{"POST", "/nope", http.StatusNotFound, "", ""},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
		if rec.Code != tt.status {
			t.Errorf("%s %s: status %d, want %d", tt.method, tt.path, rec.Code, tt.status)
		}
		if got := rec.Header().Get("Cache-Control"); got != tt.cache {
			t.Errorf("%s %s: Cache-Control %q, want %q", tt.method, tt.path, got, tt.cache)
		}
		if got := rec.Header().Get("Content-Type"); tt.ctype != "" && got != tt.ctype {
			t.Errorf("%s %s: Content-Type %q, want %q", tt.method, tt.path, got, tt.ctype)
		}
	}
}
