package handler

import (
	"net/http"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/gin-gonic/gin"
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

func (h *TemplateHandler) List(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list templates"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *TemplateHandler) Get(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	t, err := h.svc.Get(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("template")
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TemplateHandler) Create(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	t, err := h.svc.Create(c.Request.Context(), userID, req.input())
	if err != nil {
		_ = c.Error(err).SetMeta("template")
		return
	}
	c.JSON(http.StatusCreated, t)
}

// Update replaces the whole template, exercises included.
func (h *TemplateHandler) Update(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	t, err := h.svc.Update(c.Request.Context(), userID, uri.ID, req.input())
	if err != nil {
		_ = c.Error(err).SetMeta("template")
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, uri.ID); err != nil {
		_ = c.Error(err).SetMeta("template")
		return
	}
	c.Status(http.StatusNoContent)
}

// Start creates a workout linked to the template and returns it.
func (h *TemplateHandler) Start(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	wo, err := h.svc.Start(c.Request.Context(), userID, uri.ID)
	if err != nil {
		_ = c.Error(err).SetMeta("template")
		return
	}
	c.JSON(http.StatusCreated, wo)
}

// FromWorkout saves a logged workout as a new template. Body: {"name": ...},
// optional (defaults to the workout's name).
func (h *TemplateHandler) FromWorkout(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	var uri idURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	t, err := h.svc.FromWorkout(c.Request.Context(), userID, uri.ID, req.Name)
	if err != nil {
		_ = c.Error(err).SetMeta("workout")
		return
	}
	c.JSON(http.StatusCreated, t)
}
