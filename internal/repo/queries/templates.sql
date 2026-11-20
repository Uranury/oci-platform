-- Templates are private: every query is scoped to the owner, so another
-- user's template never matches.

-- name: CreateTemplate :one
INSERT INTO workout_templates (user_id, name, notes)
VALUES (@user_id, @name, @notes)
RETURNING *;

-- name: GetTemplate :one
SELECT * FROM workout_templates WHERE id = @id AND user_id = @owner_id;

-- Name order here; repo.Template.List re-sorts by last use (the last-used
-- times come from ListTemplateLastUsed).
-- name: ListTemplates :many
SELECT sqlc.embed(t),
    (SELECT count(*) FROM template_exercises te WHERE te.template_id = t.id)::int AS exercise_count,
    coalesce((
        SELECT array_agg(e.name ORDER BY te.position)
        FROM template_exercises te JOIN exercises e ON e.id = te.exercise_id
        WHERE te.template_id = t.id
    ), '{}')::text[] AS exercise_names
FROM workout_templates t
WHERE t.user_id = @owner_id
ORDER BY lower(t.name);

-- Separate from ListTemplates: sqlc can't tell that a per-template max() is
-- nullable, so templates never used are simply absent here.
-- name: ListTemplateLastUsed :many
SELECT template_id::uuid AS template_id, max(performed_at)::timestamptz AS last_used_at
FROM workouts
WHERE user_id = @owner_id AND template_id IS NOT NULL
GROUP BY template_id;

-- name: UpdateTemplate :one
UPDATE workout_templates SET name = @name, notes = @notes, updated_at = now()
WHERE id = @id AND user_id = @owner_id
RETURNING *;

-- name: DeleteTemplate :execrows
DELETE FROM workout_templates WHERE id = @id AND user_id = @owner_id;

-- Template items carry no owner; callers load the template (owner-scoped)
-- first. The join resolves archived exercises too, flagged for the client.
-- name: ListTemplateExercises :many
SELECT sqlc.embed(te), e.name AS exercise_name, e.tracking_type AS exercise_tracking_type,
    e.archived_at AS exercise_archived_at
FROM template_exercises te JOIN exercises e ON e.id = te.exercise_id
WHERE te.template_id = @template_id
ORDER BY te.position;

-- name: DeleteTemplateExercises :exec
DELETE FROM template_exercises WHERE template_id = @template_id;

-- name: CreateTemplateExercise :exec
INSERT INTO template_exercises (template_id, exercise_id, position, target_sets,
    target_reps_min, target_reps_max, target_duration_seconds,
    target_distance_m, notes)
VALUES (@template_id, @exercise_id, @position, @target_sets,
    @target_reps_min, @target_reps_max, @target_duration_seconds,
    @target_distance_m, @notes);
