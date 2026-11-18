package repo

import (
	"context"
	"errors"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkoutSet interface {
	Create(ctx context.Context, s *models.WorkoutSet) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WorkoutSet, error)
	ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutSet, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type workoutSet struct {
	exec dbutil.Executor
}

func NewWorkoutSet(exec dbutil.Executor) WorkoutSet {
	return &workoutSet{exec: exec}
}

// The join resolves archived exercises too, so history keeps its names.
const setSelect = `SELECT s.id, s.workout_id, s.exercise_id, e.name, e.tracking_type,
	s.reps, s.weight_kg, s.duration_seconds, s.distance_m, s.created_at
	FROM workout_sets s JOIN exercises e ON e.id = s.exercise_id`

func scanSet(row pgx.Row, m *models.WorkoutSet) error {
	return row.Scan(&m.ID, &m.WorkoutID, &m.ExerciseID, &m.ExerciseName, &m.ExerciseTrackingType,
		&m.Reps, &m.WeightKg, &m.DurationSeconds, &m.DistanceM, &m.CreatedAt)
}

func (r *workoutSet) Create(ctx context.Context, m *models.WorkoutSet) error {
	err := r.exec.QueryRow(ctx,
		`INSERT INTO workout_sets (workout_id, exercise_id, reps, weight_kg, duration_seconds, distance_m)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at,
		   (SELECT name FROM exercises WHERE id = $2),
		   (SELECT tracking_type FROM exercises WHERE id = $2)`,
		m.WorkoutID, m.ExerciseID, m.Reps, m.WeightKg, m.DurationSeconds, m.DistanceM,
	).Scan(&m.ID, &m.CreatedAt, &m.ExerciseName, &m.ExerciseTrackingType)
	return translatePgError(err)
}

func (r *workoutSet) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkoutSet, error) {
	var m models.WorkoutSet
	err := scanSet(r.exec.QueryRow(ctx, setSelect+` WHERE s.id = $1`, id), &m)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *workoutSet) ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutSet, error) {
	rows, err := r.exec.Query(ctx, setSelect+` WHERE s.workout_id = $1 ORDER BY s.created_at`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.WorkoutSet
	for rows.Next() {
		var m models.WorkoutSet
		if err := scanSet(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *workoutSet) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.exec.Exec(ctx, "DELETE FROM workout_sets WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
