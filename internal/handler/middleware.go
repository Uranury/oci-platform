package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Uranury/oci-platform/internal/auth"
)

// requireAuth wraps next so it only runs when the request carries a valid
// "Authorization: Bearer <jwt>" header, injecting the user ID into the
// request context for handlers to read via userIDFromContext.
func requireAuth(jwtIssuer *auth.JWTIssuer, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		raw, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || raw == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims, err := jwtIssuer.ParseAccessToken(raw)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

// statusRecorder captures the status code and body size a handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// logRequests logs every request at DEBUG and every 5xx at ERROR. In
// production (LOG_LEVEL=info) only the 5xx lines show; nginx already keeps
// the access log there. Query strings are left out: they could carry user
// input, and the path is enough to find the handler.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		level := slog.LevelDebug
		if rec.status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		slog.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration", time.Since(start).Round(time.Microsecond),
		)
	})
}
