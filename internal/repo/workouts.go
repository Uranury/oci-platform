package repo

import (
	"context"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

type Workout interface {
	Create(ctx context.Context, w *models.Workout) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Workout, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workout, error)
	Update(ctx context.Context, w *models.Workout) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type workout struct {
	q *dbgen.Queries
}

func NewWorkout(exec dbutil.Executor) Workout {
	return &workout{q: dbgen.New(exec)}
}

func toWorkout(r dbgen.Workout) models.Workout {
	return models.Workout{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Notes: r.Notes, PerformedAt: r.PerformedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (r *workout) Create(ctx context.Context, m *models.Workout) error {
	row, err := r.q.CreateWorkout(ctx, dbgen.CreateWorkoutParams{
		UserID: m.UserID, Name: m.Name, Notes: m.Notes, PerformedAt: m.PerformedAt,
	})
	if err != nil {
		return translate(err)
	}
	*m = toWorkout(row)
	return nil
}

func (r *workout) GetByID(ctx context.Context, id uuid.UUID) (*models.Workout, error) {
	row, err := r.q.GetWorkout(ctx, id)
	if err != nil {
		return nil, translate(err)
	}
	w := toWorkout(row)
	return &w, nil
}

func (r *workout) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workout, error) {
	rows, err := r.q.ListWorkoutsByUser(ctx, userID)
	if err != nil {
		return nil, translate(err)
	}
	out := make([]models.Workout, len(rows))
	for i, row := range rows {
		out[i] = toWorkout(row)
	}
	return out, nil
}

func (r *workout) Update(ctx context.Context, m *models.Workout) error {
	updatedAt, err := r.q.UpdateWorkout(ctx, dbgen.UpdateWorkoutParams{
		ID: m.ID, UserID: m.UserID, Name: m.Name, Notes: m.Notes, PerformedAt: m.PerformedAt,
	})
	if err != nil {
		return translate(err)
	}
	m.UpdatedAt = updatedAt
	return nil
}

func (r *workout) Delete(ctx context.Context, id uuid.UUID) error {
	return affected(r.q.DeleteWorkout(ctx, id))
}
