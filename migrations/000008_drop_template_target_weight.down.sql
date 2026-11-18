ALTER TABLE template_exercises
    ADD COLUMN target_weight_kg NUMERIC(6, 2) CHECK (target_weight_kg >= 0);
