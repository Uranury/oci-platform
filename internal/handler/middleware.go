package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Uranury/oci-platform/internal/auth"
	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// userIDKey is the gin context key requireAuth stores the user ID under;
// handlers behind requireAuth read it with c.MustGet(userIDKey).(uuid.UUID).
const userIDKey = "userID"

// idURI and setURI bind the UUID path parameters.
type idURI struct {
	ID uuid.UUID `uri:"id,parser=encoding.TextUnmarshaler"`
}

type workoutExerciseURI struct {
	ID         uuid.UUID `uri:"id,parser=encoding.TextUnmarshaler"`
	ExerciseID uuid.UUID `uri:"exerciseID,parser=encoding.TextUnmarshaler"`
}

type setURI struct {
	ID    uuid.UUID `uri:"id,parser=encoding.TextUnmarshaler"`
	SetID uuid.UUID `uri:"setID,parser=encoding.TextUnmarshaler"`
}

// requireAuth only lets the request through when it carries a valid
// "Authorization: Bearer <jwt>" header, storing the user ID under userIDKey.
func requireAuth(jwtIssuer *auth.JWTIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := jwtIssuer.ParseAccessToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(userIDKey, claims.UserID)
		c.Next()
	}
}

// handleErrors turns a service error a handler attached with
// c.Error(err).SetMeta("<resource>") into a response. Missing and not-owned
// resources are both 404, so a client can't probe which IDs exist.
func handleErrors(c *gin.Context) {
	c.Next()
	last := c.Errors.Last()
	if last == nil || c.Writer.Written() {
		return
	}
	what, _ := last.Meta.(string)
	err := last.Err

	status, msg := http.StatusInternalServerError, "could not process "+what
	var verr models.ValidationError
	switch {
	case errors.As(err, &verr):
		status, msg = http.StatusBadRequest, verr.Error()
	case errors.Is(err, repo.ErrNotFound), errors.Is(err, service.ErrForbidden):
		status, msg = http.StatusNotFound, what+" not found"
	case errors.Is(err, service.ErrReadOnly):
		status, msg = http.StatusForbidden, "default exercises are read-only; duplicate it to customize"
	case errors.Is(err, service.ErrExerciseExists):
		status, msg = http.StatusConflict, "you already have an exercise with that name"
	case errors.Is(err, service.ErrTemplateExists):
		status, msg = http.StatusConflict, "you already have a template with that name"
	case errors.Is(err, service.ErrExerciseArchived):
		status, msg = http.StatusConflict, "exercise is archived"
	}
	c.JSON(status, gin.H{"error": msg})
}

// logRequests logs every request at DEBUG and every 5xx at ERROR. In
// production (LOG_LEVEL=info) only the 5xx lines show; nginx already keeps
// the access log there. Query strings are left out: they could carry user
// input, and the path is enough to find the handler. A 5xx line carries the
// error the handler attached with c.Error, which the client never sees.
func logRequests(c *gin.Context) {
	start := time.Now()
	c.Next()
	status := c.Writer.Status()
	attrs := []any{
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", status,
		"bytes", max(c.Writer.Size(), 0),
		"duration", time.Since(start).Round(time.Microsecond),
	}
	level := slog.LevelDebug
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
		if last := c.Errors.Last(); last != nil {
			attrs = append(attrs, "err", last.Err)
		}
	}
	slog.Log(c.Request.Context(), level, "request", attrs...)
}

// recoverPanic turns a handler panic into a 500 whose request log line
// names the panic. Gin still prints the stack trace to stderr first.
func recoverPanic(c *gin.Context, rec any) {
	_ = c.Error(fmt.Errorf("panic: %v", rec))
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
