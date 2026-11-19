package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

var ErrTemplateExists = errors.New("service: template with that name already exists")

// TemplateService owns workout templates: reusable, ordered exercise plans
// with optional targets. Templates are private to their owner. A template's
// exercises are always saved as a whole list (replace, not patch), inside
// one transaction with the template itself.
type TemplateService struct {
	tx        dbutil.TxProvider
	templates repo.Template
	exercises repo.Exercise
	workouts  repo.Workout
	sets      repo.WorkoutSet
}

func NewTemplateService(tx dbutil.TxProvider, templates repo.Template, exercises repo.Exercise, workouts repo.Workout, sets repo.WorkoutSet) *TemplateService {
	return &TemplateService{tx: tx, templates: templates, exercises: exercises, workouts: workouts, sets: sets}
}

// TemplateInput is a full template as submitted: Exercises replaces the
// existing list, in order. Only ExerciseID, the targets and Notes of each
// item are read.
type TemplateInput struct {
	Name      string
	Notes     *string
	Exercises []models.TemplateExercise
}

func (s *TemplateService) List(ctx context.Context, userID uuid.UUID) ([]models.TemplateSummary, error) {
	return s.templates.List(ctx, userID)
}

func (s *TemplateService) Get(ctx context.Context, userID, id uuid.UUID) (*models.TemplateDetail, error) {
	return s.detail(ctx, s.templates, userID, id)
}

func (s *TemplateService) detail(ctx context.Context, templates repo.Template, userID, id uuid.UUID) (*models.TemplateDetail, error) {
	t, err := templates.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	items, err := templates.Exercises(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.TemplateDetail{Template: *t, Exercises: items}, nil
}

func (s *TemplateService) Create(ctx context.Context, userID uuid.UUID, in TemplateInput) (*models.TemplateDetail, error) {
	return s.save(ctx, userID, uuid.Nil, in)
}

func (s *TemplateService) Update(ctx context.Context, userID, id uuid.UUID, in TemplateInput) (*models.TemplateDetail, error) {
	return s.save(ctx, userID, id, in)
}

// save creates (id == uuid.Nil) or fully overwrites a template.
func (s *TemplateService) save(ctx context.Context, userID, id uuid.UUID, in TemplateInput) (*models.TemplateDetail, error) {
	name := models.NormalizeTemplateName(in.Name)
	if err := models.ValidateTemplateName(name); err != nil {
		return nil, err
	}
	if len(in.Exercises) > models.MaxTemplateExercises {
		return nil, models.ValidationError(fmt.Sprintf("a template can have at most %d exercises", models.MaxTemplateExercises))
	}

	var out *models.TemplateDetail
	err := s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		templates := repo.NewTemplate(exec)
		exercises := repo.NewExercise(exec)

		t := &models.Template{ID: id, UserID: userID, Name: name, Notes: in.Notes}
		// Exercises already in the template may stay even if archived since;
		// newly added ones must be active.
		existing := map[uuid.UUID]bool{}
		if id == uuid.Nil {
			if err := templates.Create(ctx, t); err != nil {
				return err
			}
		} else {
			if err := templates.Update(ctx, t); err != nil {
				return err
			}
			cur, err := templates.Exercises(ctx, id)
			if err != nil {
				return err
			}
			for _, c := range cur {
				existing[c.ExerciseID] = true
			}
		}

		items := make([]models.TemplateExercise, len(in.Exercises))
		for i, it := range in.Exercises {
			it.Normalize()
			// Visibility check: a template may only reference system
			// defaults and the user's own exercises.
			ex, err := exercises.GetVisible(ctx, userID, it.ExerciseID)
			if errors.Is(err, repo.ErrNotFound) {
				return models.ValidationError(fmt.Sprintf("exercise %d: not found", i+1))
			}
			if err != nil {
				return err
			}
			if ex.ArchivedAt != nil && !existing[ex.ID] {
				return ErrExerciseArchived
			}
			if err := it.ValidateFor(ex.TrackingType); err != nil {
				return models.ValidationError(fmt.Sprintf("exercise %d (%s): %s", i+1, ex.Name, err))
			}
			items[i] = it
		}
		if err := templates.ReplaceExercises(ctx, t.ID, items); err != nil {
			return err
		}
		var err error
		out, err = s.detail(ctx, templates, userID, t.ID)
		return err
	})
	if errors.Is(err, repo.ErrDuplicate) {
		return nil, ErrTemplateExists
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Delete removes a template. Workouts started from it keep their sets and
// lose only the link (ON DELETE SET NULL).
func (s *TemplateService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.templates.Delete(ctx, userID, id)
}

// Start begins a workout from a template, named after it. Only the link is
// stored: no sets are created until the user logs them.
func (s *TemplateService) Start(ctx context.Context, userID, id uuid.UUID) (*models.Workout, error) {
	t, err := s.templates.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	w := &models.Workout{UserID: userID, Name: &t.Name, PerformedAt: time.Now(), TemplateID: &t.ID}
	// The workout and its frozen copy of the plan commit together.
	err = s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		if err := repo.NewWorkout(exec).Create(ctx, w); err != nil {
			return err
		}
		return repo.NewWorkoutExercise(exec).CopyTemplate(ctx, w.ID, t.ID)
	})
	if err != nil {
		return nil, err
	}
	return w, nil
}

// FromWorkout creates a template from a logged workout: its exercises in
// the workout's own order (as rearranged on the workout screen), those with
// sets only, with targets derived from the sets (see
// models.TemplateExerciseFromSets). Archived exercises are left out. An
// empty name falls back to the workout's name.
func (s *TemplateService) FromWorkout(ctx context.Context, userID, workoutID uuid.UUID, name string) (*models.TemplateDetail, error) {
	w, err := s.workouts.GetByID(ctx, workoutID)
	if err != nil {
		return nil, err
	}
	if w.UserID != userID {
		return nil, ErrForbidden
	}
	sets, err := s.sets.ListByWorkout(ctx, workoutID)
	if err != nil {
		return nil, err
	}

	byExercise := map[uuid.UUID][]models.WorkoutSet{}
	for _, set := range sets {
		byExercise[set.ExerciseID] = append(byExercise[set.ExerciseID], set)
	}
	var listed []models.WorkoutExercise
	err = s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		var err error
		listed, err = repo.NewWorkoutExercise(exec).List(ctx, workoutID)
		return err
	})
	if err != nil {
		return nil, err
	}
	var order []uuid.UUID
	for _, we := range listed {
		if len(byExercise[we.ExerciseID]) > 0 {
			order = append(order, we.ExerciseID)
		}
	}

	var items []models.TemplateExercise
	for _, exID := range order {
		ex, err := s.exercises.GetVisible(ctx, userID, exID)
		if err != nil {
			return nil, err
		}
		if ex.ArchivedAt != nil {
			continue
		}
		items = append(items, models.TemplateExerciseFromSets(ex.ID, ex.TrackingType, byExercise[exID]))
	}
	if len(items) == 0 {
		return nil, models.ValidationError("workout has no sets to save as a template")
	}

	if models.NormalizeTemplateName(name) == "" && w.Name != nil {
		name = *w.Name
	}
	return s.Create(ctx, userID, TemplateInput{Name: name, Exercises: items})
}
