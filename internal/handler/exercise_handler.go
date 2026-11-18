package handler

import (
	"net/http"
	"strings"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExerciseHandler struct {
	svc *service.ExerciseService
}

func NewExerciseHandler(s *service.ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{svc: s}
}

// Options serves the fixed category lists (the single definition in
// models/exercise_enums.go) for the frontend's dropdowns.
func (h *ExerciseHandler) Options(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, models.AllExerciseOptions())
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
func (h *ExerciseHandler) List(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list exercises"})
		return
	}
	if list == nil {
		list = []models.Exercise{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *ExerciseHandler) Create(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var req exerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	e, err := h.svc.Create(c.Request.Context(), userID, req.input())
	if err != nil {
		_ = c.Error(err).SetMeta("exercise")
		return
	}
	c.JSON(http.StatusCreated, e)
}

func (h *ExerciseHandler) Update(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req exerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	e, err := h.svc.Update(c.Request.Context(), userID, uri.ID, req.input())
	if err != nil {
		_ = c.Error(err).SetMeta("exercise")
		return
	}
	c.JSON(http.StatusOK, e)
}

type deleteExerciseResponse struct {
	// Result is "deleted", or "archived" when the exercise had logged sets.
	Result string `json:"result"`
}

func (h *ExerciseHandler) Delete(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	archived, err := h.svc.Delete(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("exercise")
		return
	}
	res := deleteExerciseResponse{Result: "deleted"}
	if archived {
		res.Result = "archived"
	}
	c.JSON(http.StatusOK, res)
}

type duplicateRequest struct {
	Name string `json:"name"`
}

// Duplicate copies a visible exercise (usually a default) into a custom one
// owned by the user. The body is optional; {"name"} renames the copy.
func (h *ExerciseHandler) Duplicate(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req duplicateRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
	}
	e, err := h.svc.Duplicate(c.Request.Context(), userID, uri.ID, strings.TrimSpace(req.Name))
	if err != nil {
		_ = c.Error(err).SetMeta("exercise")
		return
	}
	c.JSON(http.StatusCreated, e)
}

// Similar lists defaults whose names closely match ?name=, for a
// "did you mean…" warning before creating a custom exercise. Never blocks.
func (h *ExerciseHandler) Similar(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusOK, []models.Exercise{})
		return
	}
	list, err := h.svc.SimilarDefaults(c.Request.Context(), name)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not check names"})
		return
	}
	c.JSON(http.StatusOK, list)
}
