-- Weight is meant to go up over time, so a fixed weight target in a
-- template goes stale within a session or two; the starting weight comes
-- from history instead. Rep, time and distance targets stay.
ALTER TABLE template_exercises DROP COLUMN target_weight_kg;
