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
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create workout"})
		return
	}
	c.JSON(http.StatusCreated, wo)
}

func (h *WorkoutHandler) ListWorkouts(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	list, err := h.svc.ListWorkouts(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list workouts"})
		return
	}
	if list == nil {
		list = []models.WorkoutSummary{}
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

// LastSets is "last time" for every exercise the user has logged, for the
// exercise library.
func (h *WorkoutHandler) LastSets(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	last, err := h.svc.LastSets(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load last sets"})
		return
	}
	c.JSON(http.StatusOK, last)
}

// ExerciseHistory is the user's recent workouts with one exercise.
func (h *WorkoutHandler) ExerciseHistory(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	hist, err := h.svc.ExerciseHistory(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("exercise")
		return
	}
	c.JSON(http.StatusOK, hist)
}

// --- the workout's exercise list ---

// AddExercise lists an exercise in the workout before any set is logged.
func (h *WorkoutHandler) AddExercise(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Everything but exercise_id is optional: undoing a removal sends the
	// plan (target_sets set), the sets to re-log and the position.
	var req struct {
		ExerciseID            uuid.UUID    `json:"exercise_id"`
		TargetSets            *int         `json:"target_sets"`
		TargetRepsMin         *int         `json:"target_reps_min"`
		TargetRepsMax         *int         `json:"target_reps_max"`
		TargetDurationSeconds *int         `json:"target_duration_seconds"`
		TargetDistanceM       *float64     `json:"target_distance_m"`
		Notes                 *string      `json:"notes"`
		Position              *int         `json:"position"`
		Sets                  []setRequest `json:"sets"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ExerciseID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exercise_id is required"})
		return
	}
	var plan *models.TemplateExercise
	if req.TargetSets != nil {
		plan = &models.TemplateExercise{
			TargetSets: *req.TargetSets, TargetRepsMin: req.TargetRepsMin, TargetRepsMax: req.TargetRepsMax,
			TargetDurationSeconds: req.TargetDurationSeconds, TargetDistanceM: req.TargetDistanceM, Notes: req.Notes,
		}
	}
	in := service.AddExerciseInput{Plan: plan, Position: req.Position}
	for _, s := range req.Sets {
		in.Sets = append(in.Sets, s.input())
	}
	if err := h.svc.AddExercise(c.Request.Context(), userID, uri.ID, req.ExerciseID, in); err != nil {
		_ = c.Error(err).SetMeta("workout or exercise")
		return
	}
	c.Status(http.StatusNoContent)
}

// ReorderExercises sets the workout's exercise order: {"exercise_ids": [...]}
// listing each of its exercises once.
func (h *WorkoutHandler) ReorderExercises(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		ExerciseIDs []uuid.UUID `json:"exercise_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exercise_ids must be a list of exercise IDs"})
		return
	}
	if err := h.svc.ReorderExercises(c.Request.Context(), userID, uri.ID, req.ExerciseIDs); err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.Status(http.StatusNoContent)
}

// RemoveExercise takes an exercise out of the workout with its sets.
func (h *WorkoutHandler) RemoveExercise(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri workoutExerciseURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	n, err := h.svc.RemoveExercise(c.Request.Context(), userID, uri.ID, uri.ExerciseID)
	if err != nil {
		_ = c.Error(err).SetMeta("exercise in this workout")
		return
	}
	c.JSON(http.StatusOK, gin.H{"sets_deleted": n})
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
