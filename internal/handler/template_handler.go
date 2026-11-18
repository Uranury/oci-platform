package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/google/uuid"
)

type TemplateHandler struct {
	svc *service.TemplateService
}

func NewTemplateHandler(s *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: s}
}

type templateExerciseRequest struct {
	ExerciseID            uuid.UUID `json:"exercise_id"`
	TargetSets            int       `json:"target_sets"`
	TargetRepsMin         *int      `json:"target_reps_min"`
	TargetRepsMax         *int      `json:"target_reps_max"`
	TargetDurationSeconds *int      `json:"target_duration_seconds"`
	TargetDistanceM       *float64  `json:"target_distance_m"`
	Notes                 *string   `json:"notes"`
}

// templateRequest is a whole template: exercises replaces the current list,
// in order.
type templateRequest struct {
	Name      string                    `json:"name"`
	Notes     *string                   `json:"notes"`
	Exercises []templateExerciseRequest `json:"exercises"`
}

func (req templateRequest) input() service.TemplateInput {
	items := make([]models.TemplateExercise, len(req.Exercises))
	for i, e := range req.Exercises {
		items[i] = models.TemplateExercise{
			ExerciseID: e.ExerciseID, TargetSets: e.TargetSets,
			TargetRepsMin: e.TargetRepsMin, TargetRepsMax: e.TargetRepsMax,
			TargetDurationSeconds: e.TargetDurationSeconds,
			TargetDistanceM:       e.TargetDistanceM, Notes: e.Notes,
		}
	}
	return service.TemplateInput{Name: req.Name, Notes: emptyToNil(req.Notes), Exercises: items}
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	list, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list templates")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	t, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		writeLookupError(w, err, "template")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	var req templateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.Create(r.Context(), userID, req.input())
	if err != nil {
		writeLookupError(w, err, "template")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// Update replaces the whole template, exercises included.
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req templateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.Update(r.Context(), userID, id, req.input())
	if err != nil {
		writeLookupError(w, err, "template")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		writeLookupError(w, err, "template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Start creates a workout linked to the template and returns it.
func (h *TemplateHandler) Start(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	wo, err := h.svc.Start(r.Context(), userID, id)
	if err != nil {
		writeLookupError(w, err, "template")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

// FromWorkout saves a logged workout as a new template. Body: {"name": ...},
// optional (defaults to the workout's name).
func (h *TemplateHandler) FromWorkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	workoutID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.FromWorkout(r.Context(), userID, workoutID, req.Name)
	if err != nil {
		writeLookupError(w, err, "workout")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}
