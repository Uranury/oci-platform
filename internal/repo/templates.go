package repo

import (
	"context"
	"slices"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

// Template ownership is enforced in SQL: every template query takes the
// owner, so another user's template behaves as if it doesn't exist.
// Exercise methods take a template ID only; callers must load the template
// through Get (owner-scoped) first.
type Template interface {
	Create(ctx context.Context, t *models.Template) error
	Get(ctx context.Context, ownerID, id uuid.UUID) (*models.Template, error)
	// List returns the owner's templates, most due first: never done, then
	// by last use, oldest first; ties by name.
	List(ctx context.Context, ownerID uuid.UUID) ([]models.TemplateSummary, error)
	// Update overwrites name and notes of a template owned by t.UserID.
	Update(ctx context.Context, t *models.Template) error
	Delete(ctx context.Context, ownerID, id uuid.UUID) error

	// Exercises returns the template's exercises by position.
	Exercises(ctx context.Context, templateID uuid.UUID) ([]models.TemplateExercise, error)
	// ReplaceExercises deletes the template's exercises and inserts items,
	// positioned by slice index. Run it inside RunInTx.
	ReplaceExercises(ctx context.Context, templateID uuid.UUID, items []models.TemplateExercise) error
}

type template struct {
	q *dbgen.Queries
}

func NewTemplate(exec dbutil.Executor) Template {
	return &template{q: dbgen.New(exec)}
}

func toTemplate(r dbgen.WorkoutTemplate) models.Template {
	return models.Template{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Notes: r.Notes,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (r *template) Create(ctx context.Context, m *models.Template) error {
	row, err := r.q.CreateTemplate(ctx, dbgen.CreateTemplateParams{
		UserID: m.UserID, Name: m.Name, Notes: m.Notes,
	})
	if err != nil {
		return translate(err)
	}
	*m = toTemplate(row)
	return nil
}

func (r *template) Get(ctx context.Context, ownerID, id uuid.UUID) (*models.Template, error) {
	row, err := r.q.GetTemplate(ctx, dbgen.GetTemplateParams{ID: id, OwnerID: ownerID})
	if err != nil {
		return nil, translate(err)
	}
	t := toTemplate(row)
	return &t, nil
}

func (r *template) List(ctx context.Context, ownerID uuid.UUID) ([]models.TemplateSummary, error) {
	rows, err := r.q.ListTemplates(ctx, ownerID)
	if err != nil {
		return nil, translate(err)
	}
	used, err := r.q.ListTemplateLastUsed(ctx, ownerID)
	if err != nil {
		return nil, translate(err)
	}
	byID := make(map[uuid.UUID]*models.TemplateSummary, len(rows))
	out := make([]models.TemplateSummary, len(rows))
	for i, row := range rows {
		out[i] = models.TemplateSummary{Template: toTemplate(row.WorkoutTemplate), ExerciseCount: row.ExerciseCount}
		byID[row.WorkoutTemplate.ID] = &out[i]
	}
	for _, u := range used {
		if s, ok := byID[u.TemplateID]; ok {
			s.LastUsedAt = &u.LastUsedAt
		}
	}
	// Most due first: never done, then the one done longest ago. Stable, so
	// ties keep the query's name order.
	slices.SortStableFunc(out, func(a, b models.TemplateSummary) int {
		switch {
		case a.LastUsedAt == nil && b.LastUsedAt == nil:
			return 0
		case a.LastUsedAt == nil:
			return -1
		case b.LastUsedAt == nil:
			return 1
		}
		return a.LastUsedAt.Compare(*b.LastUsedAt)
	})
	return out, nil
}

func (r *template) Update(ctx context.Context, m *models.Template) error {
	row, err := r.q.UpdateTemplate(ctx, dbgen.UpdateTemplateParams{
		ID: m.ID, OwnerID: m.UserID, Name: m.Name, Notes: m.Notes,
	})
	if err != nil {
		return translate(err)
	}
	*m = toTemplate(row)
	return nil
}

func (r *template) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	return affected(r.q.DeleteTemplate(ctx, dbgen.DeleteTemplateParams{ID: id, OwnerID: ownerID}))
}

func (r *template) Exercises(ctx context.Context, templateID uuid.UUID) ([]models.TemplateExercise, error) {
	rows, err := r.q.ListTemplateExercises(ctx, templateID)
	if err != nil {
		return nil, translate(err)
	}
	out := make([]models.TemplateExercise, len(rows))
	for i, row := range rows {
		te := row.TemplateExercise
		out[i] = models.TemplateExercise{
			ID: te.ID, TemplateID: te.TemplateID, ExerciseID: te.ExerciseID,
			ExerciseName: row.ExerciseName, ExerciseTrackingType: row.ExerciseTrackingType,
			ExerciseArchived: row.ExerciseArchivedAt != nil, Position: te.Position,
			TargetSets: te.TargetSets, TargetRepsMin: te.TargetRepsMin, TargetRepsMax: te.TargetRepsMax,
			TargetDurationSeconds: te.TargetDurationSeconds,
			TargetDistanceM:       te.TargetDistanceM, Notes: te.Notes,
		}
	}
	return out, nil
}

func (r *template) ReplaceExercises(ctx context.Context, templateID uuid.UUID, items []models.TemplateExercise) error {
	if err := r.q.DeleteTemplateExercises(ctx, templateID); err != nil {
		return translate(err)
	}
	for i, it := range items {
		err := r.q.CreateTemplateExercise(ctx, dbgen.CreateTemplateExerciseParams{
			TemplateID: templateID, ExerciseID: it.ExerciseID, Position: i,
			TargetSets: it.TargetSets, TargetRepsMin: it.TargetRepsMin, TargetRepsMax: it.TargetRepsMax,
			TargetDurationSeconds: it.TargetDurationSeconds,
			TargetDistanceM:       it.TargetDistanceM, Notes: it.Notes,
		})
		if err != nil {
			return translate(err)
		}
	}
	return nil
}
