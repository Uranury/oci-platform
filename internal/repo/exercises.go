package repo

import (
	"context"
	"errors"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

// Exercise visibility is enforced in SQL: every read takes the current user
// and only returns system defaults (user_id IS NULL) plus that user's own
// rows; every write takes the owner and only touches rows they own, so a
// system default or another user's exercise can never match.
type Exercise interface {
	Create(ctx context.Context, e *models.Exercise) error
	// GetVisible returns the exercise if it is a system default or owned by
	// userID (archived included), else ErrNotFound.
	GetVisible(ctx context.Context, userID, id uuid.UUID) (*models.Exercise, error)
	// ListVisible returns non-archived system defaults plus userID's own.
	ListVisible(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error)
	// ListSystem returns non-archived system defaults.
	ListSystem(ctx context.Context) ([]models.Exercise, error)
	// Update overwrites the editable fields of an exercise owned by e.UserID.
	Update(ctx context.Context, e *models.Exercise) error
	// Delete hard-deletes an exercise owned by ownerID; ErrInUse if it has sets.
	Delete(ctx context.Context, ownerID, id uuid.UUID) error
	// Archive soft-deletes an exercise owned by ownerID.
	Archive(ctx context.Context, ownerID, id uuid.UUID) error
	// SyncSystem upserts the given defaults by slug and archives system rows
	// whose slug is no longer listed. Never touches custom exercises.
	SyncSystem(ctx context.Context, defaults []models.Exercise) error
}

type exercise struct {
	q *dbgen.Queries
}

func NewExercise(exec dbutil.Executor) Exercise {
	return &exercise{q: dbgen.New(exec)}
}

func toExercise(r dbgen.Exercise) models.Exercise {
	return models.Exercise{
		ID: r.ID, UserID: r.UserID, System: r.UserID == nil, Slug: r.Slug, Name: r.Name,
		MovementPattern: r.MovementPattern, PrimaryMuscle: r.PrimaryMuscle,
		SecondaryMuscles: r.SecondaryMuscles, Mechanic: r.Mechanic, Equipment: r.Equipment,
		TrackingType: r.TrackingType, ArchivedAt: r.ArchivedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toExercises(rows []dbgen.Exercise) []models.Exercise {
	out := make([]models.Exercise, len(rows))
	for i, r := range rows {
		out[i] = toExercise(r)
	}
	return out
}

func (r *exercise) Create(ctx context.Context, m *models.Exercise) error {
	row, err := r.q.CreateExercise(ctx, dbgen.CreateExerciseParams{
		UserID: m.UserID, Slug: m.Slug, Name: m.Name,
		MovementPattern: m.MovementPattern, PrimaryMuscle: m.PrimaryMuscle,
		SecondaryMuscles: m.SecondaryMuscles, Mechanic: m.Mechanic, Equipment: m.Equipment,
		TrackingType: m.TrackingType,
	})
	if err != nil {
		return translate(err)
	}
	*m = toExercise(row)
	return nil
}

func (r *exercise) GetVisible(ctx context.Context, userID, id uuid.UUID) (*models.Exercise, error) {
	row, err := r.q.GetVisibleExercise(ctx, dbgen.GetVisibleExerciseParams{ID: id, ViewerID: userID})
	if err != nil {
		return nil, translate(err)
	}
	e := toExercise(row)
	return &e, nil
}

func (r *exercise) ListVisible(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error) {
	rows, err := r.q.ListVisibleExercises(ctx, userID)
	if err != nil {
		return nil, translate(err)
	}
	return toExercises(rows), nil
}

func (r *exercise) ListSystem(ctx context.Context) ([]models.Exercise, error) {
	rows, err := r.q.ListSystemExercises(ctx)
	if err != nil {
		return nil, translate(err)
	}
	return toExercises(rows), nil
}

func (r *exercise) Update(ctx context.Context, m *models.Exercise) error {
	if m.UserID == nil {
		return ErrNotFound // system defaults are only changed by SyncSystem
	}
	updatedAt, err := r.q.UpdateExercise(ctx, dbgen.UpdateExerciseParams{
		ID: m.ID, OwnerID: *m.UserID, Name: m.Name,
		MovementPattern: m.MovementPattern, PrimaryMuscle: m.PrimaryMuscle,
		SecondaryMuscles: m.SecondaryMuscles, Mechanic: m.Mechanic, Equipment: m.Equipment,
		TrackingType: m.TrackingType,
	})
	if err != nil {
		return translate(err)
	}
	m.UpdatedAt = updatedAt
	return nil
}

func (r *exercise) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	return affected(r.q.DeleteExercise(ctx, dbgen.DeleteExerciseParams{ID: id, OwnerID: ownerID}))
}

func (r *exercise) Archive(ctx context.Context, ownerID, id uuid.UUID) error {
	return affected(r.q.ArchiveExercise(ctx, dbgen.ArchiveExerciseParams{ID: id, OwnerID: ownerID}))
}

func (r *exercise) SyncSystem(ctx context.Context, defaults []models.Exercise) error {
	slugs := make([]string, 0, len(defaults))
	for _, d := range defaults {
		if d.Slug == nil {
			return errors.New("repo: system exercise without slug")
		}
		slugs = append(slugs, *d.Slug)
		err := r.q.UpsertSystemExercise(ctx, dbgen.UpsertSystemExerciseParams{
			Slug: d.Slug, Name: d.Name,
			MovementPattern: d.MovementPattern, PrimaryMuscle: d.PrimaryMuscle,
			SecondaryMuscles: d.SecondaryMuscles, Mechanic: d.Mechanic, Equipment: d.Equipment,
			TrackingType: d.TrackingType,
		})
		if err != nil {
			return translate(err)
		}
	}
	// Removed from the seed list: archive rather than delete, since users
	// may have logged sets against it.
	return translate(r.q.ArchiveSystemExercisesExcept(ctx, slugs))
}
