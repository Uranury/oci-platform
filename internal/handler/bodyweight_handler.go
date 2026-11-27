package handler

import (
	"net/http"
	"strconv"

	"github.com/Uranury/oci-platform/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BodyweightHandler struct {
	svc *service.BodyweightService
}

func NewBodyweightHandler(s *service.BodyweightService) *BodyweightHandler {
	return &BodyweightHandler{svc: s}
}

// List is the latest weigh-ins, newest day first: ?limit=N (default 30).
func (h *BodyweightHandler) List(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	limit := 0
	if q := c.Query("limit"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a number"})
			return
		}
		limit = n
	}
	list, err := h.svc.List(c.Request.Context(), userID, limit)
	if err != nil {
		_ = c.Error(err).SetMeta("bodyweight")
		return
	}
	c.JSON(http.StatusOK, list)
}

type bodyweightRequest struct {
	WeightKg *float64 `json:"weight_kg"`
}

// Put logs the weight for the day in the path (YYYY-MM-DD, the user's own
// day), replacing that day's entry if there is one.
func (h *BodyweightHandler) Put(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var req bodyweightRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.WeightKg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weight_kg is required"})
		return
	}
	b, err := h.svc.Log(c.Request.Context(), userID, c.Param("day"), *req.WeightKg)
	if err != nil {
		_ = c.Error(err).SetMeta("bodyweight")
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *BodyweightHandler) Delete(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	if err := h.svc.Delete(c.Request.Context(), userID, c.Param("day")); err != nil {
		_ = c.Error(err).SetMeta("bodyweight entry")
		return
	}
	c.Status(http.StatusNoContent)
}
