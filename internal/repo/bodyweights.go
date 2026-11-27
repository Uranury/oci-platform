package repo

import (
	"context"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

// Bodyweight is the per-user bodyweight log, keyed by calendar day.
type Bodyweight interface {
	// Upsert logs the day's weight, replacing an entry for the same day.
	Upsert(ctx context.Context, userID uuid.UUID, day time.Time, kg float64) (*models.BodyweightEntry, error)
	// List returns the user's latest entries, newest day first.
	List(ctx context.Context, userID uuid.UUID, limit int) ([]models.BodyweightEntry, error)
	Delete(ctx context.Context, userID uuid.UUID, day time.Time) error
}

type bodyweight struct {
	q *dbgen.Queries
}

func NewBodyweight(exec dbutil.Executor) Bodyweight {
	return &bodyweight{q: dbgen.New(exec)}
}

func toBodyweight(r dbgen.Bodyweight) models.BodyweightEntry {
	return models.BodyweightEntry{
		Day: r.Day.Format(models.DayLayout), WeightKg: r.WeightKg, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (r *bodyweight) Upsert(ctx context.Context, userID uuid.UUID, day time.Time, kg float64) (*models.BodyweightEntry, error) {
	row, err := r.q.UpsertBodyweight(ctx, dbgen.UpsertBodyweightParams{UserID: userID, Day: day, WeightKg: kg})
	if err != nil {
		return nil, translate(err)
	}
	b := toBodyweight(row)
	return &b, nil
}

func (r *bodyweight) List(ctx context.Context, userID uuid.UUID, limit int) ([]models.BodyweightEntry, error) {
	rows, err := r.q.ListBodyweights(ctx, dbgen.ListBodyweightsParams{
		UserID: userID, MaxRows: int32(limit), //nolint:gosec // capped by the service
	})
	if err != nil {
		return nil, translate(err)
	}
	out := make([]models.BodyweightEntry, len(rows))
	for i, row := range rows {
		out[i] = toBodyweight(row)
	}
	return out, nil
}

func (r *bodyweight) Delete(ctx context.Context, userID uuid.UUID, day time.Time) error {
	return affected(r.q.DeleteBodyweight(ctx, dbgen.DeleteBodyweightParams{UserID: userID, Day: day}))
}
