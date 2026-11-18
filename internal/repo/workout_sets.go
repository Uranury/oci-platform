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

func (r *workoutSet) Create(ctx context.Context, m *models.WorkoutSet) error {
	err := r.exec.QueryRow(ctx,
		`INSERT INTO workout_sets (workout_id, exercise_id, reps, weight_kg)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at,
		   (SELECT name FROM exercises WHERE id = $2)`,
		m.WorkoutID, m.ExerciseID, m.Reps, m.WeightKg,
	).Scan(&m.ID, &m.CreatedAt, &m.ExerciseName)
	return translatePgError(err)
}

func (r *workoutSet) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkoutSet, error) {
	var m models.WorkoutSet
	err := r.exec.QueryRow(ctx,
		`SELECT s.id, s.workout_id, s.exercise_id, e.name, s.reps, s.weight_kg, s.created_at
		 FROM workout_sets s JOIN exercises e ON e.id = s.exercise_id
		 WHERE s.id = $1`, id,
	).Scan(&m.ID, &m.WorkoutID, &m.ExerciseID, &m.ExerciseName, &m.Reps, &m.WeightKg, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *workoutSet) ListByWorkout(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutSet, error) {
	rows, err := r.exec.Query(ctx,
		`SELECT s.id, s.workout_id, s.exercise_id, e.name, s.reps, s.weight_kg, s.created_at
		 FROM workout_sets s JOIN exercises e ON e.id = s.exercise_id
		 WHERE s.workout_id = $1 ORDER BY s.created_at`, workoutID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.WorkoutSet
	for rows.Next() {
		var m models.WorkoutSet
		if err := rows.Scan(&m.ID, &m.WorkoutID, &m.ExerciseID, &m.ExerciseName, &m.Reps, &m.WeightKg, &m.CreatedAt); err != nil {
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
