package repo

import (
	"context"
	"errors"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	exec dbutil.Executor
}

func NewExercise(exec dbutil.Executor) Exercise {
	return &exercise{exec: exec}
}

const exerciseColumns = `id, user_id, slug, name, movement_pattern, primary_muscle,
	secondary_muscles, mechanic, equipment, tracking_type, archived_at, created_at, updated_at`

func scanExercise(row pgx.Row, m *models.Exercise) error {
	var secondary []string
	err := row.Scan(&m.ID, &m.UserID, &m.Slug, &m.Name, &m.MovementPattern, &m.PrimaryMuscle,
		&secondary, &m.Mechanic, &m.Equipment, &m.TrackingType, &m.ArchivedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return err
	}
	m.SecondaryMuscles = make([]models.Muscle, len(secondary))
	for i, s := range secondary {
		m.SecondaryMuscles[i] = models.Muscle(s)
	}
	m.System = m.UserID == nil
	return nil
}

func secondaryStrings(ms []models.Muscle) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = string(m)
	}
	return out
}

func (r *exercise) Create(ctx context.Context, m *models.Exercise) error {
	err := r.exec.QueryRow(ctx,
		`INSERT INTO exercises (user_id, slug, name, movement_pattern, primary_muscle,
		   secondary_muscles, mechanic, equipment, tracking_type)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		m.UserID, m.Slug, m.Name, m.MovementPattern, m.PrimaryMuscle,
		secondaryStrings(m.SecondaryMuscles), m.Mechanic, m.Equipment, m.TrackingType,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	m.System = m.UserID == nil
	return translatePgError(err)
}

func (r *exercise) GetVisible(ctx context.Context, userID, id uuid.UUID) (*models.Exercise, error) {
	var m models.Exercise
	err := scanExercise(r.exec.QueryRow(ctx,
		`SELECT `+exerciseColumns+` FROM exercises
		 WHERE id = $1 AND (user_id IS NULL OR user_id = $2)`, id, userID), &m)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *exercise) list(ctx context.Context, sql string, args ...any) ([]models.Exercise, error) {
	rows, err := r.exec.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Exercise
	for rows.Next() {
		var m models.Exercise
		if err := scanExercise(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *exercise) ListVisible(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error) {
	return r.list(ctx,
		`SELECT `+exerciseColumns+` FROM exercises
		 WHERE (user_id IS NULL OR user_id = $1) AND archived_at IS NULL
		 ORDER BY lower(name), user_id NULLS LAST`, userID)
}

func (r *exercise) ListSystem(ctx context.Context) ([]models.Exercise, error) {
	return r.list(ctx,
		`SELECT `+exerciseColumns+` FROM exercises
		 WHERE user_id IS NULL AND archived_at IS NULL ORDER BY lower(name)`)
}

func (r *exercise) Update(ctx context.Context, m *models.Exercise) error {
	if m.UserID == nil {
		return ErrNotFound // system defaults are only changed by SyncSystem
	}
	err := r.exec.QueryRow(ctx,
		`UPDATE exercises SET name = $1, movement_pattern = $2, primary_muscle = $3,
		   secondary_muscles = $4, mechanic = $5, equipment = $6, tracking_type = $7,
		   updated_at = now()
		 WHERE id = $8 AND user_id = $9
		 RETURNING updated_at`,
		m.Name, m.MovementPattern, m.PrimaryMuscle, secondaryStrings(m.SecondaryMuscles),
		m.Mechanic, m.Equipment, m.TrackingType, m.ID, *m.UserID,
	).Scan(&m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return translatePgError(err)
}

func (r *exercise) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	tag, err := r.exec.Exec(ctx, "DELETE FROM exercises WHERE id = $1 AND user_id = $2", id, ownerID)
	if err != nil {
		return translatePgError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *exercise) Archive(ctx context.Context, ownerID, id uuid.UUID) error {
	tag, err := r.exec.Exec(ctx,
		`UPDATE exercises SET archived_at = now(), updated_at = now()
		 WHERE id = $1 AND user_id = $2 AND archived_at IS NULL`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *exercise) SyncSystem(ctx context.Context, defaults []models.Exercise) error {
	slugs := make([]string, 0, len(defaults))
	for _, d := range defaults {
		if d.Slug == nil {
			return errors.New("repo: system exercise without slug")
		}
		slugs = append(slugs, *d.Slug)
		_, err := r.exec.Exec(ctx,
			`INSERT INTO exercises (user_id, slug, name, movement_pattern, primary_muscle,
			   secondary_muscles, mechanic, equipment, tracking_type)
			 VALUES (NULL, $1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (slug) DO UPDATE SET
			   name = EXCLUDED.name,
			   movement_pattern = EXCLUDED.movement_pattern,
			   primary_muscle = EXCLUDED.primary_muscle,
			   secondary_muscles = EXCLUDED.secondary_muscles,
			   mechanic = EXCLUDED.mechanic,
			   equipment = EXCLUDED.equipment,
			   tracking_type = EXCLUDED.tracking_type,
			   archived_at = NULL,
			   updated_at = now()
			 WHERE (exercises.name, exercises.movement_pattern, exercises.primary_muscle,
			        exercises.secondary_muscles, exercises.mechanic, exercises.equipment,
			        exercises.tracking_type, exercises.archived_at)
			   IS DISTINCT FROM
			       (EXCLUDED.name, EXCLUDED.movement_pattern, EXCLUDED.primary_muscle,
			        EXCLUDED.secondary_muscles, EXCLUDED.mechanic, EXCLUDED.equipment,
			        EXCLUDED.tracking_type, NULL::timestamptz)`,
			*d.Slug, d.Name, d.MovementPattern, d.PrimaryMuscle, secondaryStrings(d.SecondaryMuscles),
			d.Mechanic, d.Equipment, d.TrackingType)
		if err != nil {
			return err
		}
	}
	// Removed from the seed list: archive rather than delete, since users
	// may have logged sets against it.
	_, err := r.exec.Exec(ctx,
		`UPDATE exercises SET archived_at = now(), updated_at = now()
		 WHERE user_id IS NULL AND archived_at IS NULL AND NOT (slug = ANY($1))`, slugs)
	return err
}
