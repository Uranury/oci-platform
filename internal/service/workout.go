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

// ErrForbidden is returned when a workout or set exists but doesn't belong
// to the requesting user — surfaced by handlers as 404, not 403, so as not
// to leak whether the ID belongs to someone else.
var ErrForbidden = errors.New("service: resource does not belong to user")

// WorkoutService owns workouts, their exercise lists and their sets. Every
// method is scoped to a user. It reads exercises only to check that an
// exercise is visible to the user and to validate a set against its tracking
// type. Changes spanning a set and the exercise list run in one transaction.
type WorkoutService struct {
	tx        dbutil.TxProvider
	exercises repo.Exercise
	workouts  repo.Workout
	sets      repo.WorkoutSet
	list      repo.WorkoutExercise // outside transactions
}

func NewWorkoutService(tx dbutil.TxProvider, exercises repo.Exercise, workouts repo.Workout, sets repo.WorkoutSet, list repo.WorkoutExercise) *WorkoutService {
	return &WorkoutService{tx: tx, exercises: exercises, workouts: workouts, sets: sets, list: list}
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

func (s *WorkoutService) ListWorkouts(ctx context.Context, userID uuid.UUID) ([]models.WorkoutSummary, error) {
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
	exercises, err := s.workoutExercises(nil).List(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.WorkoutDetail{Workout: *w, Exercises: exercises, Sets: sets}, nil
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

// LastSets is "last time" for every exercise the user has logged: the sets
// from the most recent workout that included it, up to now.
func (s *WorkoutService) LastSets(ctx context.Context, userID uuid.UUID) ([]models.PreviousSession, error) {
	return s.sets.ListPrevious(ctx, userID, uuid.Nil, time.Now())
}

// ExerciseHistoryLength is how many recent workouts ExerciseHistory returns.
const ExerciseHistoryLength = 10

// ExerciseHistory returns the user's most recent workouts with the exercise,
// newest first, each with its sets of it. ErrNotFound when the exercise
// isn't visible to the user (another user's custom exercise).
func (s *WorkoutService) ExerciseHistory(ctx context.Context, userID, exerciseID uuid.UUID) ([]models.PreviousSession, error) {
	if _, err := s.exercises.GetVisible(ctx, userID, exerciseID); err != nil {
		return nil, err
	}
	return s.sets.ListHistory(ctx, userID, exerciseID, ExerciseHistoryLength)
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
	// The set and its exercise's place in the list (appended if new) commit
	// together.
	err := s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		if err := repo.NewWorkoutSet(exec).Create(ctx, set); err != nil {
			return err
		}
		return s.workoutExercises(exec).Append(ctx, workoutID, exerciseID, nil)
	})
	if err != nil {
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
	// Moving a set to another exercise lists that exercise too.
	err = s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		if err := repo.NewWorkoutSet(exec).Update(ctx, set); err != nil {
			return err
		}
		return s.workoutExercises(exec).Append(ctx, workoutID, exerciseID, nil)
	})
	if err != nil {
		return nil, err
	}
	return set, nil
}

// workoutExercises is the exercise-list repo inside the transaction exec,
// or the pool-backed one outside a transaction (nil).
func (s *WorkoutService) workoutExercises(exec dbutil.Executor) repo.WorkoutExercise {
	if exec == nil {
		return s.list
	}
	return repo.NewWorkoutExercise(exec)
}

// MaxRestoredSets bounds how many sets one AddExercise call may re-log.
const MaxRestoredSets = 100

// AddExerciseInput is what goes with an added exercise. All optional: a
// plain add has none of it; undoing a removal brings back the plan, the
// sets (re-logged in order) and the place in the list.
type AddExerciseInput struct {
	Plan     *models.TemplateExercise // targets, checked by the template rules
	Sets     []SetInput               // checked like any logged set
	Position *int                     // 0-based; clamped to the list
}

// AddExercise puts an exercise in the workout's list, at the end unless
// in.Position says where; adding one already listed doesn't move or replan
// it. Everything commits together, so an undone removal comes back whole
// or not at all.
func (s *WorkoutService) AddExercise(ctx context.Context, userID, workoutID, exerciseID uuid.UUID, in AddExerciseInput) error {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return err
	}
	ex, err := s.exercises.GetVisible(ctx, userID, exerciseID)
	if err != nil {
		return err
	}
	if ex.ArchivedAt != nil {
		return ErrExerciseArchived
	}
	if in.Plan != nil {
		if err := in.Plan.ValidateFor(ex.TrackingType); err != nil {
			return err
		}
	}
	if len(in.Sets) > MaxRestoredSets {
		return models.ValidationError(fmt.Sprintf("at most %d sets at once", MaxRestoredSets))
	}
	sets := make([]*models.WorkoutSet, len(in.Sets))
	for i, si := range in.Sets {
		sets[i] = &models.WorkoutSet{WorkoutID: workoutID, ExerciseID: exerciseID}
		if err := s.checkSet(ctx, userID, sets[i], si, uuid.Nil); err != nil {
			return err
		}
	}

	return s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		list := s.workoutExercises(exec)
		if err := list.Append(ctx, workoutID, exerciseID, in.Plan); err != nil {
			return err
		}
		setRepo := repo.NewWorkoutSet(exec)
		for _, set := range sets {
			if err := setRepo.Create(ctx, set); err != nil {
				return err
			}
		}
		if in.Position == nil {
			return nil
		}
		current, err := list.List(ctx, workoutID)
		if err != nil {
			return err
		}
		order := make([]uuid.UUID, 0, len(current))
		for _, we := range current {
			if we.ExerciseID != exerciseID {
				order = append(order, we.ExerciseID)
			}
		}
		at := min(max(*in.Position, 0), len(order))
		order = append(order[:at], append([]uuid.UUID{exerciseID}, order[at:]...)...)
		for i, id := range order {
			if err := list.SetPosition(ctx, workoutID, id, i); err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveExercise takes an exercise out of the workout together with its
// logged sets, returning how many sets went.
func (s *WorkoutService) RemoveExercise(ctx context.Context, userID, workoutID, exerciseID uuid.UUID) (int64, error) {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return 0, err
	}
	var n int64
	err := s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		var err error
		n, err = s.workoutExercises(exec).Remove(ctx, workoutID, exerciseID)
		return err
	})
	return n, err
}

// ReorderExercises sets the workout's exercise order. order must list
// exactly the exercises the workout has, each once.
func (s *WorkoutService) ReorderExercises(ctx context.Context, userID, workoutID uuid.UUID, order []uuid.UUID) error {
	if _, err := s.getWorkout(ctx, userID, workoutID); err != nil {
		return err
	}
	return s.tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		list := s.workoutExercises(exec)
		current, err := list.List(ctx, workoutID)
		if err != nil {
			return err
		}
		listed := make(map[uuid.UUID]bool, len(current))
		for _, we := range current {
			listed[we.ExerciseID] = true
		}
		if len(order) != len(current) {
			return models.ValidationError("order must list each of the workout's exercises once")
		}
		for i, id := range order {
			if !listed[id] {
				return models.ValidationError("order must list each of the workout's exercises once")
			}
			delete(listed, id) // a repeat now fails the check above
			if err := list.SetPosition(ctx, workoutID, id, i); err != nil {
				return err
			}
		}
		return nil
	})
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
