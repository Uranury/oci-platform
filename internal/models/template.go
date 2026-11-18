package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Template is a named, reusable workout plan owned by one user.
type Template struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateSummary is a template as listed: without its exercises, but with
// how many it has and when a workout was last started from it.
type TemplateSummary struct {
	Template
	ExerciseCount int        `json:"exercise_count"`
	LastUsedAt    *time.Time `json:"last_used_at"`
}

// TemplateDetail is a template with its exercises in order.
type TemplateDetail struct {
	Template
	Exercises []TemplateExercise `json:"exercises"`
}

// TemplateExercise is one planned exercise in a template. All targets are
// optional; which ones may be set depends on the exercise's tracking type
// (see ValidateFor). ExerciseName, ExerciseTrackingType and
// ExerciseArchived come from a join.
type TemplateExercise struct {
	ID                   uuid.UUID    `json:"id"`
	TemplateID           uuid.UUID    `json:"template_id"`
	ExerciseID           uuid.UUID    `json:"exercise_id"`
	ExerciseName         string       `json:"exercise_name"`
	ExerciseTrackingType TrackingType `json:"exercise_tracking_type"`
	ExerciseArchived     bool         `json:"exercise_archived"`
	Position             int          `json:"position"`

	TargetSets            int      `json:"target_sets"`
	TargetRepsMin         *int     `json:"target_reps_min"`
	TargetRepsMax         *int     `json:"target_reps_max"`
	TargetWeightKg        *float64 `json:"target_weight_kg"`
	TargetDurationSeconds *int     `json:"target_duration_seconds"`
	TargetDistanceM       *float64 `json:"target_distance_m"`
	Notes                 *string  `json:"notes,omitempty"`
}

const (
	MaxTemplateNameLen      = 100
	MaxTemplateExercises    = 50
	MaxTemplateSets         = 20
	MaxTemplateExerciseNote = 200
)

// NormalizeTemplateName collapses whitespace in a template name.
func NormalizeTemplateName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

// ValidateTemplateName checks a normalized template name.
func ValidateTemplateName(name string) error {
	if name == "" {
		return ValidationError("name is required")
	}
	if len([]rune(name)) > MaxTemplateNameLen {
		return ValidationError(fmt.Sprintf("name must be at most %d characters", MaxTemplateNameLen))
	}
	return nil
}

// Normalize fills a one-sided rep range (8 means 8–8) and trims notes.
func (t *TemplateExercise) Normalize() {
	switch {
	case t.TargetRepsMin == nil && t.TargetRepsMax != nil:
		t.TargetRepsMin = t.TargetRepsMax
	case t.TargetRepsMax == nil && t.TargetRepsMin != nil:
		t.TargetRepsMax = t.TargetRepsMin
	}
	if t.Notes != nil {
		n := strings.TrimSpace(*t.Notes)
		if n == "" {
			t.Notes = nil
		} else {
			t.Notes = &n
		}
	}
}

// ValidateFor checks the targets against the exercise's tracking type,
// using the same per-type rules as logged sets (setRules): a target may be
// set wherever a set may record that measurement. Targets are never
// required.
func (t *TemplateExercise) ValidateFor(tt TrackingType) error {
	rules, ok := setRules[tt]
	if !ok {
		return ValidationError(fmt.Sprintf("unknown tracking type %q", tt))
	}
	if t.TargetSets < 1 || t.TargetSets > MaxTemplateSets {
		return ValidationError(fmt.Sprintf("target_sets must be between 1 and %d", MaxTemplateSets))
	}
	repsOK := func(p *int) bool { return p == nil || (*p >= 1 && *p <= MaxReps) }
	checks := []struct {
		name    string
		rule    fieldRule
		present bool
		inRange bool
		rangeMs string
	}{
		{"target_reps", rules.reps, t.TargetRepsMin != nil || t.TargetRepsMax != nil, repsOK(t.TargetRepsMin) && repsOK(t.TargetRepsMax), fmt.Sprintf("between 1 and %d", MaxReps)},
		{"target_weight_kg", rules.weight, t.TargetWeightKg != nil, t.TargetWeightKg == nil || (*t.TargetWeightKg >= 0 && *t.TargetWeightKg <= MaxWeightKg), fmt.Sprintf("between 0 and %.2f", MaxWeightKg)},
		{"target_duration_seconds", rules.duration, t.TargetDurationSeconds != nil, t.TargetDurationSeconds == nil || (*t.TargetDurationSeconds >= 1 && *t.TargetDurationSeconds <= MaxDurationSeconds), fmt.Sprintf("between 1 and %d", MaxDurationSeconds)},
		{"target_distance_m", rules.distance, t.TargetDistanceM != nil, t.TargetDistanceM == nil || (*t.TargetDistanceM > 0 && *t.TargetDistanceM <= MaxDistanceM), fmt.Sprintf("greater than 0 and at most %.2f", MaxDistanceM)},
	}
	for _, c := range checks {
		switch {
		case c.rule == notAllowed && c.present:
			return ValidationError(fmt.Sprintf("%s is not used for %s exercises", c.name, tt))
		case !c.inRange:
			return ValidationError(fmt.Sprintf("%s must be %s", c.name, c.rangeMs))
		}
	}
	if t.TargetRepsMin != nil && t.TargetRepsMax != nil && *t.TargetRepsMin > *t.TargetRepsMax {
		return ValidationError("target_reps_min must not be greater than target_reps_max")
	}
	if t.Notes != nil && len([]rune(*t.Notes)) > MaxTemplateExerciseNote {
		return ValidationError(fmt.Sprintf("notes must be at most %d characters", MaxTemplateExerciseNote))
	}
	return nil
}

// TemplateExerciseFromSets plans an exercise from the sets logged for it in
// one workout: as many target sets as were logged, the rep range seen and
// the heaviest weight / longest duration / longest distance. Only targets
// the exercise's current tracking type allows are set, so the result
// passes ValidateFor even if old sets were logged under another type.
func TemplateExerciseFromSets(exerciseID uuid.UUID, tt TrackingType, sets []WorkoutSet) TemplateExercise {
	te := TemplateExercise{ExerciseID: exerciseID, TargetSets: min(len(sets), MaxTemplateSets)}
	rules := setRules[tt]
	maxF := func(cur, v *float64) *float64 {
		if v != nil && (cur == nil || *v > *cur) {
			return v
		}
		return cur
	}
	maxI := func(cur, v *int) *int {
		if v != nil && (cur == nil || *v > *cur) {
			return v
		}
		return cur
	}
	for _, s := range sets {
		if rules.reps != notAllowed && s.Reps != nil {
			if te.TargetRepsMin == nil || *s.Reps < *te.TargetRepsMin {
				te.TargetRepsMin = s.Reps
			}
			te.TargetRepsMax = maxI(te.TargetRepsMax, s.Reps)
		}
		if rules.weight != notAllowed {
			te.TargetWeightKg = maxF(te.TargetWeightKg, s.WeightKg)
		}
		if rules.duration != notAllowed {
			te.TargetDurationSeconds = maxI(te.TargetDurationSeconds, s.DurationSeconds)
		}
		if rules.distance != notAllowed {
			te.TargetDistanceM = maxF(te.TargetDistanceM, s.DistanceM)
		}
	}
	return te
}
