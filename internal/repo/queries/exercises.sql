-- Visibility is part of every query: reads match system defaults
-- (user_id IS NULL) or the caller's own rows; writes match the owner only.

-- name: CreateExercise :one
INSERT INTO exercises (user_id, slug, name, movement_pattern, primary_muscle,
    secondary_muscles, mechanic, equipment, tracking_type)
VALUES (@user_id, @slug, @name, @movement_pattern, @primary_muscle,
    @secondary_muscles, @mechanic, @equipment, @tracking_type)
RETURNING *;

-- name: GetVisibleExercise :one
SELECT * FROM exercises
WHERE id = @id AND (user_id IS NULL OR user_id = @viewer_id::uuid);

-- name: ListVisibleExercises :many
SELECT * FROM exercises
WHERE (user_id IS NULL OR user_id = @viewer_id::uuid) AND archived_at IS NULL
ORDER BY lower(name), user_id NULLS LAST;

-- name: ListSystemExercises :many
SELECT * FROM exercises
WHERE user_id IS NULL AND archived_at IS NULL
ORDER BY lower(name);

-- name: UpdateExercise :one
UPDATE exercises SET name = @name, movement_pattern = @movement_pattern,
    primary_muscle = @primary_muscle, secondary_muscles = @secondary_muscles,
    mechanic = @mechanic, equipment = @equipment, tracking_type = @tracking_type,
    updated_at = now()
WHERE id = @id AND user_id = @owner_id::uuid
RETURNING updated_at;

-- name: DeleteExercise :execrows
DELETE FROM exercises WHERE id = @id AND user_id = @owner_id::uuid;

-- name: ArchiveExercise :execrows
UPDATE exercises SET archived_at = now(), updated_at = now()
WHERE id = @id AND user_id = @owner_id::uuid AND archived_at IS NULL;

-- Upsert one default by slug; a no-op (no write, updated_at kept) when the
-- row already matches.
-- name: UpsertSystemExercise :exec
INSERT INTO exercises (user_id, slug, name, movement_pattern, primary_muscle,
    secondary_muscles, mechanic, equipment, tracking_type)
VALUES (NULL, @slug, @name, @movement_pattern, @primary_muscle,
    @secondary_muscles, @mechanic, @equipment, @tracking_type)
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
       EXCLUDED.tracking_type, NULL::timestamptz);

-- Archive defaults no longer in the seed list (they may have logged sets).
-- name: ArchiveSystemExercisesExcept :exec
UPDATE exercises SET archived_at = now(), updated_at = now()
WHERE user_id IS NULL AND archived_at IS NULL AND NOT (slug = ANY(@keep_slugs::text[]));
