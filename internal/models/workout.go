package models

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Exercise is either a system default (UserID nil, Slug set, read-only) or a
// private custom exercise owned by one user. Both are used identically by
// workouts, sets and history.
type Exercise struct {
	ID     uuid.UUID  `json:"id" db:"id"`
	UserID *uuid.UUID `json:"user_id" db:"user_id"` // nil = system default
	// System mirrors UserID == nil for clients; not a column.
	System bool    `json:"system" db:"-"`
	Slug   *string `json:"slug,omitempty" db:"slug"`
	Name   string  `json:"name" db:"name"`

	MovementPattern  MovementPattern `json:"movement_pattern" db:"movement_pattern"`
	PrimaryMuscle    Muscle          `json:"primary_muscle" db:"primary_muscle"`
	SecondaryMuscles []Muscle        `json:"secondary_muscles" db:"secondary_muscles"`
	Mechanic         Mechanic        `json:"mechanic" db:"mechanic"`
	Equipment        Equipment       `json:"equipment" db:"equipment"`
	TrackingType     TrackingType    `json:"tracking_type" db:"tracking_type"`

	ArchivedAt *time.Time `json:"archived_at,omitempty" db:"archived_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

const MaxExerciseNameLen = 100

// Normalize trims the name and cleans up secondary muscles (drops
// duplicates and the primary muscle) before Validate.
func (e *Exercise) Normalize() {
	e.Name = strings.Join(strings.Fields(e.Name), " ")
	var sec []Muscle
	for _, m := range e.SecondaryMuscles {
		if m != e.PrimaryMuscle && !slices.Contains(sec, m) {
			sec = append(sec, m)
		}
	}
	if sec == nil {
		sec = []Muscle{}
	}
	e.SecondaryMuscles = sec
}

// Validate checks the name and every category against the enums.
func (e *Exercise) Validate() error {
	if e.Name == "" {
		return ValidationError("name is required")
	}
	if len([]rune(e.Name)) > MaxExerciseNameLen {
		return ValidationError(fmt.Sprintf("name must be at most %d characters", MaxExerciseNameLen))
	}
	if err := MovementPatterns.check("movement_pattern", e.MovementPattern); err != nil {
		return err
	}
	if err := Muscles.check("primary_muscle", e.PrimaryMuscle); err != nil {
		return err
	}
	for _, m := range e.SecondaryMuscles {
		if err := Muscles.check("secondary_muscles", m); err != nil {
			return err
		}
	}
	if err := Mechanics.check("mechanic", e.Mechanic); err != nil {
		return err
	}
	if err := Equipments.check("equipment", e.Equipment); err != nil {
		return err
	}
	return TrackingTypes.check("tracking_type", e.TrackingType)
}

// Workout is one training session.
type Workout struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        *string   `json:"name,omitempty" db:"name"`
	Notes       *string   `json:"notes,omitempty" db:"notes"`
	PerformedAt time.Time `json:"performed_at" db:"performed_at"`
	// TemplateID is the template the workout was started from, if any (nil
	// also once that template is deleted).
	TemplateID *uuid.UUID `json:"template_id,omitempty" db:"template_id"`
	// FinishedAt is when the workout was finished; nil while in progress
	// or when unknown (logged after the fact). Never before PerformedAt.
	FinishedAt *time.Time `json:"finished_at,omitempty" db:"finished_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WorkoutSummary is a workout as listed in the history: the workout plus
// how many sets it has and which exercises, in the order first logged.
// Embedding keeps the JSON flat (the Workout fields plus these two).
type WorkoutSummary struct {
	Workout
	SetCount  int      `json:"set_count"`
	Exercises []string `json:"exercises"`
}

// WorkoutSet is a single set inside a workout. Which measurements are set
// depends on the exercise's TrackingType at the time of logging (see
// ValidateFor); the rest are nil. ExerciseName and ExerciseTrackingType
// come from a join, not from workout_sets.
type WorkoutSet struct {
	ID                   uuid.UUID    `json:"id" db:"id"`
	WorkoutID            uuid.UUID    `json:"workout_id" db:"workout_id"`
	ExerciseID           uuid.UUID    `json:"exercise_id" db:"exercise_id"`
	ExerciseName         string       `json:"exercise_name" db:"-"`
	ExerciseTrackingType TrackingType `json:"exercise_tracking_type" db:"-"`

	Reps            *int     `json:"reps" db:"reps"`
	WeightKg        *float64 `json:"weight_kg" db:"weight_kg"`
	DurationSeconds *int     `json:"duration_seconds" db:"duration_seconds"`
	DistanceM       *float64 `json:"distance_m" db:"distance_m"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Limits match the column types (NUMERIC(6,2), NUMERIC(8,2)).
const (
	MaxReps            = 10000
	MaxWeightKg        = 9999.99
	MaxDurationSeconds = 24 * 60 * 60
	MaxDistanceM       = 999999.99
)

type fieldRule int

const (
	notAllowed fieldRule = iota
	optional
	required
)

// setRules says, per tracking type, which measurements a set takes.
var setRules = map[TrackingType]struct{ reps, weight, duration, distance fieldRule }{
	WeightReps:     {reps: required, weight: required},
	BodyweightReps: {reps: required, weight: optional},
	Duration:       {duration: required, weight: optional},
	Distance:       {distance: required, weight: optional, duration: optional},
}

// ValidateFor checks a set's measurements against the exercise's tracking type.
func (s *WorkoutSet) ValidateFor(tt TrackingType) error {
	rules, ok := setRules[tt]
	if !ok {
		return ValidationError(fmt.Sprintf("unknown tracking type %q", tt))
	}
	checks := []struct {
		name    string
		rule    fieldRule
		present bool
		inRange bool
		rangeMs string
	}{
		{"reps", rules.reps, s.Reps != nil, s.Reps == nil || (*s.Reps >= 1 && *s.Reps <= MaxReps), fmt.Sprintf("between 1 and %d", MaxReps)},
		{"weight_kg", rules.weight, s.WeightKg != nil, s.WeightKg == nil || (*s.WeightKg >= 0 && *s.WeightKg <= MaxWeightKg), fmt.Sprintf("between 0 and %.2f", MaxWeightKg)},
		{"duration_seconds", rules.duration, s.DurationSeconds != nil, s.DurationSeconds == nil || (*s.DurationSeconds >= 1 && *s.DurationSeconds <= MaxDurationSeconds), fmt.Sprintf("between 1 and %d", MaxDurationSeconds)},
		{"distance_m", rules.distance, s.DistanceM != nil, s.DistanceM == nil || (*s.DistanceM > 0 && *s.DistanceM <= MaxDistanceM), fmt.Sprintf("greater than 0 and at most %.2f", MaxDistanceM)},
	}
	for _, c := range checks {
		switch {
		case c.rule == required && !c.present:
			return ValidationError(fmt.Sprintf("%s is required for %s exercises", c.name, tt))
		case c.rule == notAllowed && c.present:
			return ValidationError(fmt.Sprintf("%s is not used for %s exercises", c.name, tt))
		case !c.inRange:
			return ValidationError(fmt.Sprintf("%s must be %s", c.name, c.rangeMs))
		}
	}
	return nil
}

// WorkoutDetail is a workout together with its sets, in logging order.
type WorkoutDetail struct {
	Workout
	Sets []WorkoutSet `json:"sets"`
}

// PreviousSession is "last time" for one exercise: its sets from the most
// recent earlier workout that included it.
type PreviousSession struct {
	ExerciseID  uuid.UUID    `json:"exercise_id"`
	WorkoutID   uuid.UUID    `json:"workout_id"`
	PerformedAt time.Time    `json:"performed_at"`
	Sets        []WorkoutSet `json:"sets"`
}
