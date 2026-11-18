package repo

import (
	"context"
	"errors"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Workout interface {
	Create(ctx context.Context, w *models.Workout) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Workout, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workout, error)
	Update(ctx context.Context, w *models.Workout) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type workout struct {
	exec dbutil.Executor
}

func NewWorkout(exec dbutil.Executor) Workout {
	return &workout{exec: exec}
}

const workoutColumns = `id, user_id, name, notes, performed_at, created_at, updated_at`

func scanWorkout(row pgx.Row, m *models.Workout) error {
	return row.Scan(&m.ID, &m.UserID, &m.Name, &m.Notes, &m.PerformedAt, &m.CreatedAt, &m.UpdatedAt)
}

func (r *workout) Create(ctx context.Context, m *models.Workout) error {
	return r.exec.QueryRow(ctx,
		`INSERT INTO workouts (user_id, name, notes, performed_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		m.UserID, m.Name, m.Notes, m.PerformedAt,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *workout) GetByID(ctx context.Context, id uuid.UUID) (*models.Workout, error) {
	var m models.Workout
	err := scanWorkout(r.exec.QueryRow(ctx,
		`SELECT `+workoutColumns+` FROM workouts WHERE id = $1`, id), &m)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *workout) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workout, error) {
	rows, err := r.exec.Query(ctx,
		`SELECT `+workoutColumns+` FROM workouts
		 WHERE user_id = $1 ORDER BY performed_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Workout
	for rows.Next() {
		var m models.Workout
		if err := scanWorkout(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *workout) Update(ctx context.Context, m *models.Workout) error {
	err := r.exec.QueryRow(ctx,
		`UPDATE workouts SET name = $1, notes = $2, performed_at = $3, updated_at = now()
		 WHERE id = $4 AND user_id = $5
		 RETURNING updated_at`,
		m.Name, m.Notes, m.PerformedAt, m.ID, m.UserID,
	).Scan(&m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *workout) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.exec.Exec(ctx, "DELETE FROM workouts WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
