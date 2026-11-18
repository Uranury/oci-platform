package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/google/uuid"
)

var (
	// ErrForbidden is returned when a workout, set or exercise exists but
	// doesn't belong to the requesting user — surfaced by handlers as 404,
	// not 403, so as not to leak whether the ID belongs to someone else.
	ErrForbidden = errors.New("service: resource does not belong to user")

	ErrExerciseExists = errors.New("service: exercise with that name already exists")
	ErrExerciseInUse  = errors.New("service: exercise has logged sets")
)

// WorkoutService owns exercises, workouts and their sets. Every method is
// scoped to a user: anything belonging to someone else is reported as
// ErrForbidden, which handlers turn into 404.
type WorkoutService struct {
	exercises repo.Exercise
	workouts  repo.Workout
	sets      repo.WorkoutSet
}

func NewWorkoutService(exercises repo.Exercise, workouts repo.Workout, sets repo.WorkoutSet) *WorkoutService {
	return &WorkoutService{exercises: exercises, workouts: workouts, sets: sets}
}

// --- exercises ---

func (s *WorkoutService) CreateExercise(ctx context.Context, userID uuid.UUID, name string) (*models.Exercise, error) {
	e := &models.Exercise{UserID: userID, Name: strings.TrimSpace(name)}
	if err := s.exercises.Create(ctx, e); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			return nil, ErrExerciseExists
		}
		return nil, err
	}
	return e, nil
}

func (s *WorkoutService) ListExercises(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error) {
	return s.exercises.ListByUser(ctx, userID)
}

func (s *WorkoutService) getExercise(ctx context.Context, userID, id uuid.UUID) (*models.Exercise, error) {
	e, err := s.exercises.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.UserID != userID {
		return nil, ErrForbidden
	}
	return e, nil
}

func (s *WorkoutService) DeleteExercise(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := s.getExercise(ctx, userID, id); err != nil {
		return err
	}
	if err := s.exercises.Delete(ctx, id); err != nil {
		if errors.Is(err, repo.ErrInUse) {
			return ErrExerciseInUse
		}
		return err
	}
	return nil
}

// --- workouts ---

// CreateWorkout starts a workout; a zero performedAt means "now".
func (s *WorkoutService) CreateWorkout(ctx context.Context, userID uuid.UUID, name, notes *string, performedAt time.Time) (*models.Workout, error) {
	if performedAt.IsZero() {
		performedAt = time.Now()
	}
	w := &models.Workout{UserID: userID, Name: name, Notes: notes, PerformedAt: performedAt}
	if err := s.workouts.Create(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WorkoutService) ListWorkouts(ctx context.Context, userID uuid.UUID) ([]models.Workout, error) {
	return s.workouts.ListByUser(ctx, userID)
}

func (s *WorkoutService) getWorkout(ctx context.Context, userID, id uuid.UUID) (*models.Workout, error) {
	w, err := s.workouts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.UserID != userID {
		return nil, ErrForbidden
	}
	return w, nil
}

func (s *WorkoutService) GetWorkout(ctx context.Context, userID, id uuid.UUID) (*models.WorkoutDetail, error) {
	w, err := s.getWorkout(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	sets, err := s.sets.ListByWorkout(ctx, id)
	if err != nil {
		return nil, err
	}
	if sets == nil {
		sets = []models.WorkoutSet{}
	}
	return &models.WorkoutDetail{Workout: *w, Sets: sets}, nil
}

// UpdateWorkout overwrites name and notes; performedAt is kept when zero.
func (s *WorkoutService) UpdateWorkout(ctx context.Context, userID, id uuid.UUID, name, notes *string, performedAt time.Time) (*models.Workout, error) {
	w, err := s.getWorkout(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	w.Name = name
	w.Notes = notes
	if !performedAt.IsZero() {
		w.PerformedAt = performedAt
	}
	if err := s.workouts.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WorkoutService) DeleteWorkout(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := s.getWorkout(ctx, userID, id); err != nil {
		return err
	}
	return s.workouts.Delete(ctx, id) // sets go with it (ON DELETE CASCADE)
}

// --- sets ---

func (s *WorkoutService) AddSet(ctx context.Context, userID, workoutID, exerciseID uuid.UUID, reps int, weightKg float64) (*models.WorkoutSet, error) {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return nil, err
	}
	// Without this check a user could log sets against someone else's
	// exercise ID and read its name back.
	if _, err := s.getExercise(ctx, userID, exerciseID); err != nil {
		return nil, err
	}
	set := &models.WorkoutSet{WorkoutID: workoutID, ExerciseID: exerciseID, Reps: reps, WeightKg: weightKg}
	if err := s.sets.Create(ctx, set); err != nil {
		return nil, err
	}
	return set, nil
}

func (s *WorkoutService) DeleteSet(ctx context.Context, userID, workoutID, setID uuid.UUID) error {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return err
	}
	set, err := s.sets.GetByID(ctx, setID)
	if err != nil {
		return err
	}
	if set.WorkoutID != workoutID {
		return ErrForbidden
	}
	return s.sets.Delete(ctx, setID)
}
