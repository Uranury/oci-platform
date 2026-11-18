package repo

import (
	"context"
	"errors"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Exercise interface {
	Create(ctx context.Context, e *models.Exercise) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Exercise, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type exercise struct {
	exec dbutil.Executor
}

func NewExercise(exec dbutil.Executor) Exercise {
	return &exercise{exec: exec}
}

func (r *exercise) Create(ctx context.Context, m *models.Exercise) error {
	err := r.exec.QueryRow(ctx,
		`INSERT INTO exercises (user_id, name) VALUES ($1, $2)
		 RETURNING id, created_at`,
		m.UserID, m.Name,
	).Scan(&m.ID, &m.CreatedAt)
	return translatePgError(err)
}

func (r *exercise) GetByID(ctx context.Context, id uuid.UUID) (*models.Exercise, error) {
	var m models.Exercise
	err := r.exec.QueryRow(ctx,
		`SELECT id, user_id, name, created_at FROM exercises WHERE id = $1`, id,
	).Scan(&m.ID, &m.UserID, &m.Name, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *exercise) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error) {
	rows, err := r.exec.Query(ctx,
		`SELECT id, user_id, name, created_at FROM exercises
		 WHERE user_id = $1 ORDER BY lower(name)`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Exercise
	for rows.Next() {
		var m models.Exercise
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *exercise) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.exec.Exec(ctx, "DELETE FROM exercises WHERE id = $1", id)
	if err != nil {
		return translatePgError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
