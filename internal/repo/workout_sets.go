package repo

import (
	"context"

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
