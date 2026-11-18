package service

import (
	"context"
	"errors"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/google/uuid"
)

// ErrForbidden is returned when a workout or set exists but doesn't belong
// to the requesting user — surfaced by handlers as 404, not 403, so as not
// to leak whether the ID belongs to someone else.
var ErrForbidden = errors.New("service: resource does not belong to user")

// WorkoutService owns workouts and their sets. Every method is scoped to a
// user. It reads exercises only to check that a set's exercise is visible to
// the user and to validate the set against its tracking type.
type WorkoutService struct {
	exercises repo.Exercise
	workouts  repo.Workout
	sets      repo.WorkoutSet
}

func NewWorkoutService(exercises repo.Exercise, workouts repo.Workout, sets repo.WorkoutSet) *WorkoutService {
	return &WorkoutService{exercises: exercises, workouts: workouts, sets: sets}
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

// PreviousSets returns "last time" for every exercise: the sets from the
// most recent workout dated before this one that included it. Relative to
// this workout's date, so reopening an old workout shows what came before
// it, not what came after.
func (s *WorkoutService) PreviousSets(ctx context.Context, userID, id uuid.UUID) ([]models.PreviousSession, error) {
	w, err := s.getWorkout(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return s.sets.ListPrevious(ctx, userID, id, w.PerformedAt)
}

// UpdateWorkout overwrites name and notes; performedAt is kept when zero.
// Moving performedAt moves FinishedAt by the same amount, so a corrected
// date keeps the session's duration.
func (s *WorkoutService) UpdateWorkout(ctx context.Context, userID, id uuid.UUID, name, notes *string, performedAt time.Time) (*models.Workout, error) {
	w, err := s.getWorkout(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	w.Name = name
	w.Notes = notes
	if !performedAt.IsZero() {
		if w.FinishedAt != nil {
			f := w.FinishedAt.Add(performedAt.Sub(w.PerformedAt))
			w.FinishedAt = &f
		}
		w.PerformedAt = performedAt
	}
	if err := s.workouts.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// MaxAutoFinishAge bounds FinishWorkout: a workout that started longer ago
// than this was logged after the fact, so "now" isn't when it ended.
const MaxAutoFinishAge = 12 * time.Hour

// FinishWorkout records now as the end of the workout, once. It's left
// unset (duration unknown) when the workout started more than
// MaxAutoFinishAge ago or is dated in the future, rather than storing a
// duration that's wrong. Calling it again changes nothing, so reopening an
// old workout to fix a typo doesn't stretch its duration.
func (s *WorkoutService) FinishWorkout(ctx context.Context, userID, id uuid.UUID) (*models.Workout, error) {
	w, err := s.getWorkout(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if age := now.Sub(w.PerformedAt); w.FinishedAt != nil || age < 0 || age > MaxAutoFinishAge {
		return w, nil
	}
	w.FinishedAt = &now
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

// SetInput holds a set's measurements; which ones are allowed or required
// depends on the exercise's tracking type.
type SetInput struct {
	Reps            *int
	WeightKg        *float64
	DurationSeconds *int
	DistanceM       *float64
}

func (s *WorkoutService) AddSet(ctx context.Context, userID, workoutID, exerciseID uuid.UUID, in SetInput) (*models.WorkoutSet, error) {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return nil, err
	}
	set := &models.WorkoutSet{WorkoutID: workoutID, ExerciseID: exerciseID}
	if err := s.checkSet(ctx, userID, set, in, uuid.Nil); err != nil {
		return nil, err
	}
	if err := s.sets.Create(ctx, set); err != nil {
		return nil, err
	}
	return set, nil
}

// UpdateSet overwrites a logged set's exercise and measurements (fixing a
// typo or a wrongly picked exercise). It keeps its place in the workout.
func (s *WorkoutService) UpdateSet(ctx context.Context, userID, workoutID, setID, exerciseID uuid.UUID, in SetInput) (*models.WorkoutSet, error) {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return nil, err
	}
	cur, err := s.sets.GetByID(ctx, setID)
	if err != nil {
		return nil, err
	}
	if cur.WorkoutID != workoutID {
		return nil, ErrForbidden
	}
	set := &models.WorkoutSet{ID: setID, WorkoutID: workoutID, ExerciseID: exerciseID}
	// The set's current exercise may have been archived since; editing the
	// numbers must still work.
	if err := s.checkSet(ctx, userID, set, in, cur.ExerciseID); err != nil {
		return nil, err
	}
	if err := s.sets.Update(ctx, set); err != nil {
		return nil, err
	}
	return set, nil
}

// checkSet fills set's measurements from in and validates them against the
// exercise. The exercise must be visible to the user and not archived,
// unless it is allowArchived (the exercise the set already had).
func (s *WorkoutService) checkSet(ctx context.Context, userID uuid.UUID, set *models.WorkoutSet, in SetInput, allowArchived uuid.UUID) error {
	// Never trust the client's exercise ID: it must be a system default or
	// the user's own, or this would log against (and leak the name of)
	// someone else's private exercise.
	ex, err := s.exercises.GetVisible(ctx, userID, set.ExerciseID)
	if err != nil {
		return err
	}
	if ex.ArchivedAt != nil && ex.ID != allowArchived {
		return ErrExerciseArchived
	}
	set.Reps = in.Reps
	set.WeightKg = in.WeightKg
	set.DurationSeconds = in.DurationSeconds
	set.DistanceM = in.DistanceM
	return set.ValidateFor(ex.TrackingType)
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
