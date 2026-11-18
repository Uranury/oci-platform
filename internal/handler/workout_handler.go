package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WorkoutHandler struct {
	svc *service.WorkoutService
}

func NewWorkoutHandler(s *service.WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{svc: s}
}

// emptyToNil turns "" (after trimming) into nil so blank form fields clear
// optional columns instead of storing empty strings.
func emptyToNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// --- workouts ---

type workoutRequest struct {
	Name        *string    `json:"name,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	PerformedAt *time.Time `json:"performed_at,omitempty"` // RFC 3339; omitted = now / unchanged
}

func (req workoutRequest) performedAt() time.Time {
	if req.PerformedAt == nil {
		return time.Time{}
	}
	return *req.PerformedAt
}

func (h *WorkoutHandler) CreateWorkout(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var req workoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	wo, err := h.svc.CreateWorkout(c.Request.Context(), userID, emptyToNil(req.Name), emptyToNil(req.Notes), req.performedAt())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create workout"})
		return
	}
	c.JSON(http.StatusCreated, wo)
}

func (h *WorkoutHandler) ListWorkouts(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	list, err := h.svc.ListWorkouts(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list workouts"})
		return
	}
	if list == nil {
		list = []models.Workout{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *WorkoutHandler) GetWorkout(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	wo, err := h.svc.GetWorkout(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.JSON(http.StatusOK, wo)
}

func (h *WorkoutHandler) UpdateWorkout(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req workoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	wo, err := h.svc.UpdateWorkout(c.Request.Context(), userID, uri.ID, emptyToNil(req.Name), emptyToNil(req.Notes), req.performedAt())
	if err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.JSON(http.StatusOK, wo)
}

func (h *WorkoutHandler) DeleteWorkout(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteWorkout(c.Request.Context(), userID, uri.ID); err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.Status(http.StatusNoContent)
}

// FinishWorkout records when the workout ended (once; see
// service.WorkoutService.FinishWorkout) and returns it.
func (h *WorkoutHandler) FinishWorkout(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	wo, err := h.svc.FinishWorkout(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.JSON(http.StatusOK, wo)
}

// PreviousSets is "last time": per exercise, the sets from the most recent
// earlier workout that included it.
func (h *WorkoutHandler) PreviousSets(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	prev, err := h.svc.PreviousSets(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.JSON(http.StatusOK, prev)
}

// --- sets ---

type setRequest struct {
	ExerciseID      uuid.UUID `json:"exercise_id"`
	Reps            *int      `json:"reps"`
	WeightKg        *float64  `json:"weight_kg"`
	DurationSeconds *int      `json:"duration_seconds"`
	DistanceM       *float64  `json:"distance_m"`
}

func (req setRequest) input() service.SetInput {
	return service.SetInput{
		Reps:            req.Reps,
		WeightKg:        req.WeightKg,
		DurationSeconds: req.DurationSeconds,
		DistanceM:       req.DistanceM,
	}
}

// AddSet logs a set. Which measurements are required depends on the
// exercise's tracking_type (see models.WorkoutSet.ValidateFor).
func (h *WorkoutHandler) AddSet(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req setRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.ExerciseID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exercise_id is required"})
		return
	}
	set, err := h.svc.AddSet(c.Request.Context(), userID, uri.ID, req.ExerciseID, req.input())
	if err != nil {
		_ = c.Error(err).SetMeta("workout or exercise")
		return
	}
	c.JSON(http.StatusCreated, set)
}

// UpdateSet overwrites a set's exercise and measurements; same body and
// validation as AddSet.
func (h *WorkoutHandler) UpdateSet(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri setURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req setRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.ExerciseID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exercise_id is required"})
		return
	}
	set, err := h.svc.UpdateSet(c.Request.Context(), userID, uri.ID, uri.SetID, req.ExerciseID, req.input())
	if err != nil {
		_ = c.Error(err).SetMeta("set or exercise")
		return
	}
	c.JSON(http.StatusOK, set)
}

func (h *WorkoutHandler) DeleteSet(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri setURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteSet(c.Request.Context(), userID, uri.ID, uri.SetID); err != nil {
		_ = c.Error(err).SetMeta("set")
		return
	}
	c.Status(http.StatusNoContent)
}
