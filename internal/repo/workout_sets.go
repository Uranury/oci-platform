package repo

import (
	"context"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

type WorkoutSet interface {
	Create(ctx context.Context, s *models.WorkoutSet) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WorkoutSet, error)
	ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutSet, error)
	// Update overwrites the exercise and measurements of set s.ID within
	// workout s.WorkoutID; ErrNotFound if the set isn't in that workout.
	Update(ctx context.Context, s *models.WorkoutSet) error
	Delete(ctx context.Context, id uuid.UUID) error
	// ListPrevious returns, per exercise, the user's sets from the most
	// recent workout dated before `before` that included it, skipping
	// excludeWorkoutID. Sessions are ordered by exercise ID.
	ListPrevious(ctx context.Context, userID, excludeWorkoutID uuid.UUID, before time.Time) ([]models.PreviousSession, error)
	// ListHistory returns the user's most recent workouts (newest first, at
	// most maxWorkouts) that include the exercise, each with its sets of it.
	ListHistory(ctx context.Context, userID, exerciseID uuid.UUID, maxWorkouts int) ([]models.PreviousSession, error)
	// ListExport is every set the user logged in workouts performed in
	// [from, to), oldest workout first, for the CSV export.
	ListExport(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]models.ExportSet, error)
}

type workoutSet struct {
	q *dbgen.Queries
}

func NewWorkoutSet(exec dbutil.Executor) WorkoutSet {
	return &workoutSet{q: dbgen.New(exec)}
}

func toWorkoutSet(s dbgen.WorkoutSet, exerciseName string, tt models.TrackingType) models.WorkoutSet {
	return models.WorkoutSet{
		ID: s.ID, WorkoutID: s.WorkoutID, ExerciseID: s.ExerciseID,
		ExerciseName: exerciseName, ExerciseTrackingType: tt,
		Reps: s.Reps, WeightKg: s.WeightKg, DurationSeconds: s.DurationSeconds, DistanceM: s.DistanceM,
		CreatedAt: s.CreatedAt,
	}
}

func (r *workoutSet) Create(ctx context.Context, m *models.WorkoutSet) error {
	row, err := r.q.CreateWorkoutSet(ctx, dbgen.CreateWorkoutSetParams{
		WorkoutID: m.WorkoutID, ExerciseID: m.ExerciseID,
		Reps: m.Reps, WeightKg: m.WeightKg, DurationSeconds: m.DurationSeconds, DistanceM: m.DistanceM,
	})
	if err != nil {
		return translate(err)
	}
	*m = toWorkoutSet(dbgen.WorkoutSet{
		ID: row.ID, WorkoutID: row.WorkoutID, ExerciseID: row.ExerciseID,
		Reps: row.Reps, WeightKg: row.WeightKg, DurationSeconds: row.DurationSeconds, DistanceM: row.DistanceM,
		CreatedAt: row.CreatedAt,
	}, row.ExerciseName, row.ExerciseTrackingType)
	return nil
}

func (r *workoutSet) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkoutSet, error) {
	row, err := r.q.GetWorkoutSet(ctx, id)
	if err != nil {
		return nil, translate(err)
	}
	s := toWorkoutSet(row.WorkoutSet, row.ExerciseName, row.ExerciseTrackingType)
	return &s, nil
}

func (r *workoutSet) ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutSet, error) {
	rows, err := r.q.ListWorkoutSets(ctx, workoutID)
	if err != nil {
		return nil, translate(err)
	}
	out := make([]models.WorkoutSet, len(rows))
	for i, row := range rows {
		out[i] = toWorkoutSet(row.WorkoutSet, row.ExerciseName, row.ExerciseTrackingType)
	}
	return out, nil
}

func (r *workoutSet) Update(ctx context.Context, m *models.WorkoutSet) error {
	row, err := r.q.UpdateWorkoutSet(ctx, dbgen.UpdateWorkoutSetParams{
		ID: m.ID, WorkoutID: m.WorkoutID, ExerciseID: m.ExerciseID,
		Reps: m.Reps, WeightKg: m.WeightKg, DurationSeconds: m.DurationSeconds, DistanceM: m.DistanceM,
	})
	if err != nil {
		return translate(err)
	}
	*m = toWorkoutSet(dbgen.WorkoutSet{
		ID: row.ID, WorkoutID: row.WorkoutID, ExerciseID: row.ExerciseID,
		Reps: row.Reps, WeightKg: row.WeightKg, DurationSeconds: row.DurationSeconds, DistanceM: row.DistanceM,
		CreatedAt: row.CreatedAt,
	}, row.ExerciseName, row.ExerciseTrackingType)
	return nil
}

func (r *workoutSet) Delete(ctx context.Context, id uuid.UUID) error {
	return affected(r.q.DeleteWorkoutSet(ctx, id))
}

func (r *workoutSet) ListHistory(ctx context.Context, userID, exerciseID uuid.UUID, maxWorkouts int) ([]models.PreviousSession, error) {
	rows, err := r.q.ListExerciseHistory(ctx, dbgen.ListExerciseHistoryParams{
		UserID: userID, ExerciseID: exerciseID, MaxWorkouts: int32(maxWorkouts), //nolint:gosec // small constant
	})
	if err != nil {
		return nil, translate(err)
	}
	out := []models.PreviousSession{}
	for _, row := range rows {
		set := toWorkoutSet(row.WorkoutSet, row.ExerciseName, row.ExerciseTrackingType)
		if n := len(out); n == 0 || out[n-1].WorkoutID != set.WorkoutID {
			out = append(out, models.PreviousSession{
				ExerciseID: set.ExerciseID, WorkoutID: set.WorkoutID, PerformedAt: row.WorkoutPerformedAt,
			})
		}
		last := &out[len(out)-1]
		last.Sets = append(last.Sets, set)
	}
	return out, nil
}

func (r *workoutSet) ListPrevious(ctx context.Context, userID, excludeWorkoutID uuid.UUID, before time.Time) ([]models.PreviousSession, error) {
	rows, err := r.q.ListPreviousSets(ctx, dbgen.ListPreviousSetsParams{
		UserID: userID, ExcludeWorkoutID: excludeWorkoutID, Before: before,
	})
	if err != nil {
		return nil, translate(err)
	}
	out := []models.PreviousSession{}
	for _, row := range rows {
		set := toWorkoutSet(row.WorkoutSet, row.ExerciseName, row.ExerciseTrackingType)
		if n := len(out); n == 0 || out[n-1].ExerciseID != set.ExerciseID {
			out = append(out, models.PreviousSession{
				ExerciseID: set.ExerciseID, WorkoutID: set.WorkoutID, PerformedAt: row.WorkoutPerformedAt,
			})
		}
		last := &out[len(out)-1]
		last.Sets = append(last.Sets, set)
	}
	return out, nil
}

func (r *workoutSet) ListExport(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]models.ExportSet, error) {
	rows, err := r.q.ListExportSets(ctx, dbgen.ListExportSetsParams{UserID: userID, FromAt: from, ToAt: to})
	if err != nil {
		return nil, translate(err)
	}
	out := make([]models.ExportSet, len(rows))
	for i, row := range rows {
		out[i] = models.ExportSet(row)
	}
	return out, nil
}
