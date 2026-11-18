package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/google/uuid"
)

type WorkoutHandler struct {
	svc *service.WorkoutService
}

func NewWorkoutHandler(s *service.WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{svc: s}
}

// pathID parses the named path value as a UUID, writing a 400 on failure.
func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

// mustUser reads the user ID requireAuth put in the context.
func mustUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
	}
	return id, ok
}

// writeLookupError maps service errors to responses. Missing and not-owned
// resources are both 404, so a client can't probe which IDs exist.
func writeLookupError(w http.ResponseWriter, err error, what string) {
	var verr models.ValidationError
	switch {
	case errors.As(err, &verr):
		writeError(w, http.StatusBadRequest, verr.Error())
	case errors.Is(err, repo.ErrNotFound), errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusNotFound, what+" not found")
	case errors.Is(err, service.ErrReadOnly):
		writeError(w, http.StatusForbidden, "default exercises are read-only; duplicate it to customize")
	case errors.Is(err, service.ErrExerciseExists):
		writeError(w, http.StatusConflict, "you already have an exercise with that name")
	case errors.Is(err, service.ErrTemplateExists):
		writeError(w, http.StatusConflict, "you already have a template with that name")
	case errors.Is(err, service.ErrExerciseArchived):
		writeError(w, http.StatusConflict, "exercise is archived")
	default:
		writeError(w, http.StatusInternalServerError, "could not process "+what)
	}
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

func (h *WorkoutHandler) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	var req workoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	wo, err := h.svc.CreateWorkout(r.Context(), userID, emptyToNil(req.Name), emptyToNil(req.Notes), req.performedAt())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create workout")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

func (h *WorkoutHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	list, err := h.svc.ListWorkouts(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list workouts")
		return
	}
	if list == nil {
		list = []models.Workout{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *WorkoutHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	wo, err := h.svc.GetWorkout(r.Context(), userID, id)
	if err != nil {
		writeLookupError(w, err, "workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

func (h *WorkoutHandler) UpdateWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req workoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	wo, err := h.svc.UpdateWorkout(r.Context(), userID, id, emptyToNil(req.Name), emptyToNil(req.Notes), req.performedAt())
	if err != nil {
		writeLookupError(w, err, "workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

func (h *WorkoutHandler) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteWorkout(r.Context(), userID, id); err != nil {
		writeLookupError(w, err, "workout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// FinishWorkout records when the workout ended (once; see
// service.WorkoutService.FinishWorkout) and returns it.
func (h *WorkoutHandler) FinishWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	wo, err := h.svc.FinishWorkout(r.Context(), userID, id)
	if err != nil {
		writeLookupError(w, err, "workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
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
func (h *WorkoutHandler) AddSet(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	workoutID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ExerciseID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "exercise_id is required")
		return
	}
	set, err := h.svc.AddSet(r.Context(), userID, workoutID, req.ExerciseID, req.input())
	if err != nil {
		writeLookupError(w, err, "workout or exercise")
		return
	}
	writeJSON(w, http.StatusCreated, set)
}

// UpdateSet overwrites a set's exercise and measurements; same body and
// validation as AddSet.
func (h *WorkoutHandler) UpdateSet(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	workoutID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	setID, ok := pathID(w, r, "setID")
	if !ok {
		return
	}
	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ExerciseID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "exercise_id is required")
		return
	}
	set, err := h.svc.UpdateSet(r.Context(), userID, workoutID, setID, req.ExerciseID, req.input())
	if err != nil {
		writeLookupError(w, err, "set or exercise")
		return
	}
	writeJSON(w, http.StatusOK, set)
}

func (h *WorkoutHandler) DeleteSet(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	workoutID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	setID, ok := pathID(w, r, "setID")
	if !ok {
		return
	}
	if err := h.svc.DeleteSet(r.Context(), userID, workoutID, setID); err != nil {
		writeLookupError(w, err, "set")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
