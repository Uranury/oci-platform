package models

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func ip(v int) *int         { return &v }
func fp(v float64) *float64 { return &v }
func sp(v string) *string   { return &v }

func TestTemplateExerciseValidateFor(t *testing.T) {
	tests := []struct {
		name    string
		tt      TrackingType
		te      TemplateExercise
		wantErr string
	}{
		{"sets only", WeightReps, TemplateExercise{TargetSets: 3}, ""},
		{"full weight_reps", WeightReps, TemplateExercise{TargetSets: 3, TargetRepsMin: ip(8), TargetRepsMax: ip(12), TargetWeightKg: fp(60)}, ""},
		{"zero sets", WeightReps, TemplateExercise{TargetSets: 0}, "target_sets"},
		{"too many sets", WeightReps, TemplateExercise{TargetSets: MaxTemplateSets + 1}, "target_sets"},
		{"reversed range", WeightReps, TemplateExercise{TargetSets: 3, TargetRepsMin: ip(12), TargetRepsMax: ip(8)}, "target_reps_min"},
		{"reps on duration", Duration, TemplateExercise{TargetSets: 3, TargetRepsMin: ip(8), TargetRepsMax: ip(8)}, "target_reps is not used"},
		{"duration on weight_reps", WeightReps, TemplateExercise{TargetSets: 3, TargetDurationSeconds: ip(60)}, "target_duration_seconds is not used"},
		{"distance with duration", Distance, TemplateExercise{TargetSets: 1, TargetDistanceM: fp(5000), TargetDurationSeconds: ip(1500)}, ""},
		{"negative weight", BodyweightReps, TemplateExercise{TargetSets: 3, TargetWeightKg: fp(-1)}, "target_weight_kg must be"},
		{"long note", WeightReps, TemplateExercise{TargetSets: 3, Notes: sp(strings.Repeat("x", MaxTemplateExerciseNote+1))}, "notes"},
		{"unknown type", TrackingType("nope"), TemplateExercise{TargetSets: 3}, "unknown tracking type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.te.Normalize()
			err := tc.te.ValidateFor(tc.tt)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("got %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestTemplateExerciseNormalize(t *testing.T) {
	te := TemplateExercise{TargetRepsMax: ip(10), Notes: sp("  ")}
	te.Normalize()
	if te.TargetRepsMin == nil || *te.TargetRepsMin != 10 {
		t.Fatalf("one-sided range not filled: %v", te.TargetRepsMin)
	}
	if te.Notes != nil {
		t.Fatalf("blank notes not cleared: %q", *te.Notes)
	}
}

func TestTemplateExerciseFromSets(t *testing.T) {
	id := uuid.New()
	sets := []WorkoutSet{
		{Reps: ip(10), WeightKg: fp(60)},
		{Reps: ip(8), WeightKg: fp(65)},
		{Reps: ip(6), WeightKg: fp(62.5)},
	}
	te := TemplateExerciseFromSets(id, WeightReps, sets)
	if te.ExerciseID != id || te.TargetSets != 3 || *te.TargetRepsMin != 6 || *te.TargetRepsMax != 10 || *te.TargetWeightKg != 65 {
		t.Fatalf("got %+v", te)
	}
	if err := te.ValidateFor(WeightReps); err != nil {
		t.Fatalf("derived template invalid: %v", err)
	}

	// Sets logged before the exercise switched to duration: reps must be
	// dropped so the result is still valid for the current type.
	te = TemplateExerciseFromSets(id, Duration, sets)
	if te.TargetRepsMin != nil || te.TargetRepsMax != nil {
		t.Fatalf("reps kept for duration exercise: %+v", te)
	}
	if err := te.ValidateFor(Duration); err != nil {
		t.Fatalf("derived template invalid: %v", err)
	}

	many := make([]WorkoutSet, MaxTemplateSets+5)
	for i := range many {
		many[i].Reps = ip(5)
	}
	if te = TemplateExerciseFromSets(id, BodyweightReps, many); te.TargetSets != MaxTemplateSets {
		t.Fatalf("target_sets not capped: %d", te.TargetSets)
	}
}
