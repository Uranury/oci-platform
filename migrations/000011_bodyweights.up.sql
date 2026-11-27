-- Bodyweight log: at most one entry per user per day (logging again that
-- day replaces it), the day being the user's own calendar day as the
-- client sends it. Always kilograms, like weight_kg on sets.
CREATE TABLE bodyweights (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    day        DATE NOT NULL,
    weight_kg  NUMERIC(5, 2) NOT NULL CHECK (weight_kg > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, day)
);
