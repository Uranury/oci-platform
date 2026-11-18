package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/service"
)

type ExerciseHandler struct {
	svc *service.ExerciseService
}

func NewExerciseHandler(s *service.ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{svc: s}
}

// Options serves the fixed category lists (the single definition in
// models/exercise_enums.go) for the frontend's dropdowns.
func (h *ExerciseHandler) Options(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, models.AllExerciseOptions())
}

type exerciseRequest struct {
	Name             string                 `json:"name"`
	MovementPattern  models.MovementPattern `json:"movement_pattern"`
	PrimaryMuscle    models.Muscle          `json:"primary_muscle"`
	SecondaryMuscles []models.Muscle        `json:"secondary_muscles"`
	Mechanic         models.Mechanic        `json:"mechanic"`
	Equipment        models.Equipment       `json:"equipment"`
	TrackingType     models.TrackingType    `json:"tracking_type"`
}

func (req exerciseRequest) input() service.ExerciseInput {
	return service.ExerciseInput{
		Name:             req.Name,
		MovementPattern:  req.MovementPattern,
		PrimaryMuscle:    req.PrimaryMuscle,
		SecondaryMuscles: req.SecondaryMuscles,
		Mechanic:         req.Mechanic,
		Equipment:        req.Equipment,
		TrackingType:     req.TrackingType,
	}
}

// List returns non-archived system defaults plus the user's own exercises.
func (h *ExerciseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	list, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list exercises")
		return
	}
	if list == nil {
		list = []models.Exercise{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ExerciseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	var req exerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e, err := h.svc.Create(r.Context(), userID, req.input())
	if err != nil {
		writeLookupError(w, err, "exercise")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *ExerciseHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req exerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e, err := h.svc.Update(r.Context(), userID, id, req.input())
	if err != nil {
		writeLookupError(w, err, "exercise")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

type deleteExerciseResponse struct {
	// Result is "deleted", or "archived" when the exercise had logged sets.
	Result string `json:"result"`
}

func (h *ExerciseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	archived, err := h.svc.Delete(r.Context(), userID, id)
	if err != nil {
		writeLookupError(w, err, "exercise")
		return
	}
	res := deleteExerciseResponse{Result: "deleted"}
	if archived {
		res.Result = "archived"
	}
	writeJSON(w, http.StatusOK, res)
}

type duplicateRequest struct {
	Name string `json:"name"`
}

// Duplicate copies a visible exercise (usually a default) into a custom one
// owned by the user. The body is optional; {"name"} renames the copy.
func (h *ExerciseHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	userID, ok := mustUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req duplicateRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	e, err := h.svc.Duplicate(r.Context(), userID, id, strings.TrimSpace(req.Name))
	if err != nil {
		writeLookupError(w, err, "exercise")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// Similar lists defaults whose names closely match ?name=, for a
// "did you mean…" warning before creating a custom exercise. Never blocks.
func (h *ExerciseHandler) Similar(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustUser(w, r); !ok {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeJSON(w, http.StatusOK, []models.Exercise{})
		return
	}
	list, err := h.svc.SimilarDefaults(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check names")
		return
	}
	writeJSON(w, http.StatusOK, list)
}
