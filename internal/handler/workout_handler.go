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

// writeLookupError maps missing or not-owned resources to 404 either way so
// a client can't probe which IDs exist.
func writeLookupError(w http.ResponseWriter, err error, what string) {
	switch {
	case errors.Is(err, repo.ErrNotFound), errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusNotFound, what+" not found")
	case errors.Is(err, service.ErrExerciseInUse):
		writeError(w, http.StatusConflict, "exercise has logged sets; delete those first")
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

// --- exercises ---

type exerciseRequest struct {
	Name string `json:"name"`
}

func (h *WorkoutHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	var req exerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	e, err := h.svc.CreateExercise(r.Context(), userID, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrExerciseExists) {
			writeError(w, http.StatusConflict, "exercise already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create exercise")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *WorkoutHandler) ListExercises(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	list, err := h.svc.ListExercises(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list exercises")
		return
	}
	if list == nil {
		list = []models.Exercise{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *WorkoutHandler) DeleteExercise(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteExercise(r.Context(), userID, id); err != nil {
		writeLookupError(w, err, "exercise")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// --- sets ---

type setRequest struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Reps       int       `json:"reps"`
	WeightKg   float64   `json:"weight_kg"`
}

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
	if req.Reps <= 0 || req.Reps > 10000 {
		writeError(w, http.StatusBadRequest, "reps must be between 1 and 10000")
		return
	}
	if req.WeightKg < 0 || req.WeightKg >= 10000 { // NUMERIC(6,2)
		writeError(w, http.StatusBadRequest, "weight_kg must be between 0 and 9999.99")
		return
	}
	set, err := h.svc.AddSet(r.Context(), userID, workoutID, req.ExerciseID, req.Reps, req.WeightKg)
	if err != nil {
		writeLookupError(w, err, "workout or exercise")
		return
	}
	writeJSON(w, http.StatusCreated, set)
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
