package models

import (
	"time"

	"github.com/google/uuid"
)

// Exercise is a per-user catalog entry ("Bench Press", "Squat"), referenced
// by every set logged against it.
type Exercise struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Workout is one training session.
type Workout struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        *string   `json:"name,omitempty" db:"name"`
	Notes       *string   `json:"notes,omitempty" db:"notes"`
	PerformedAt time.Time `json:"performed_at" db:"performed_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WorkoutSet is a single set inside a workout. ExerciseName is not a column
// of workout_sets; repos fill it from a join so clients needn't look it up.
type WorkoutSet struct {
	ID           uuid.UUID `json:"id" db:"id"`
	WorkoutID    uuid.UUID `json:"workout_id" db:"workout_id"`
	ExerciseID   uuid.UUID `json:"exercise_id" db:"exercise_id"`
	ExerciseName string    `json:"exercise_name" db:"-"`
	Reps         int       `json:"reps" db:"reps"`
	WeightKg     float64   `json:"weight_kg" db:"weight_kg"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// WorkoutDetail is a workout together with its sets, in logging order.
type WorkoutDetail struct {
	Workout
	Sets []WorkoutSet `json:"sets"`
}
