package models_test

import (
	"io/fs"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/migrations"
)

func strs[T ~string](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

var quoted = regexp.MustCompile(`'([^']*)'`)

// latestCheckValues returns the quoted values of the most recent definition
// of the named CHECK constraint across all up-migrations, in file order, so
// a later migration that redefines a list wins.
func latestCheckValues(t *testing.T, constraint string) []string {
	t.Helper()
	names, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)

	var found []string
	for _, name := range names {
		b, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		sql := string(b)
		for {
			i := strings.Index(sql, "CONSTRAINT "+constraint+" CHECK")
			if i < 0 {
				break
			}
			sql = sql[i+len(constraint):]
			// The value list ends at the first ')' or ']' after it starts;
			// enum values never contain either.
			start := strings.IndexAny(sql, "'")
			end := start + strings.IndexAny(sql[start:], ")]")
			found = nil
			for _, m := range quoted.FindAllStringSubmatch(sql[start:end], -1) {
				found = append(found, m[1])
			}
		}
	}
	if found == nil {
		t.Fatalf("no CHECK constraint %s found in migrations", constraint)
	}
	return found
}

func TestEnumsMatchMigrations(t *testing.T) {
	cases := []struct {
		constraint string
		values     []string
	}{
		{"exercises_movement_pattern_check", strs(models.MovementPatterns.Values())},
		{"exercises_primary_muscle_check", strs(models.Muscles.Values())},
		{"exercises_secondary_muscles_check", strs(models.Muscles.Values())},
		{"exercises_mechanic_check", strs(models.Mechanics.Values())},
		{"exercises_equipment_check", strs(models.Equipments.Values())},
		{"exercises_tracking_type_check", strs(models.TrackingTypes.Values())},
	}
	for _, c := range cases {
		t.Run(c.constraint, func(t *testing.T) {
			db := latestCheckValues(t, c.constraint)
			goVals := slices.Clone(c.values)
			slices.Sort(db)
			slices.Sort(goVals)
			if !slices.Equal(db, goVals) {
				t.Errorf("Go enum and migration disagree:\n  go: %v\n  db: %v", goVals, db)
			}
		})
	}
}

func TestOptionsHaveLabels(t *testing.T) {
	o := models.AllExerciseOptions()
	for _, list := range [][]models.Option{o.MovementPatterns, o.Muscles, o.Mechanics, o.Equipment, o.TrackingTypes} {
		if len(list) == 0 {
			t.Fatal("empty option list")
		}
		for _, opt := range list {
			if opt.Value == "" || opt.Label == "" {
				t.Errorf("option %+v missing value or label", opt)
			}
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestSetValidateFor(t *testing.T) {
	cases := []struct {
		name string
		tt   models.TrackingType
		set  models.WorkoutSet
		ok   bool
	}{
		{"weight_reps ok", models.WeightReps, models.WorkoutSet{Reps: ptr(5), WeightKg: ptr(100.0)}, true},
		{"weight_reps zero weight ok", models.WeightReps, models.WorkoutSet{Reps: ptr(5), WeightKg: ptr(0.0)}, true},
		{"weight_reps missing weight", models.WeightReps, models.WorkoutSet{Reps: ptr(5)}, false},
		{"weight_reps missing reps", models.WeightReps, models.WorkoutSet{WeightKg: ptr(100.0)}, false},
		{"weight_reps with duration", models.WeightReps, models.WorkoutSet{Reps: ptr(5), WeightKg: ptr(1.0), DurationSeconds: ptr(30)}, false},
		{"bodyweight no added weight", models.BodyweightReps, models.WorkoutSet{Reps: ptr(10)}, true},
		{"bodyweight added weight", models.BodyweightReps, models.WorkoutSet{Reps: ptr(10), WeightKg: ptr(20.0)}, true},
		{"bodyweight zero reps", models.BodyweightReps, models.WorkoutSet{Reps: ptr(0)}, false},
		{"duration ok", models.Duration, models.WorkoutSet{DurationSeconds: ptr(60)}, true},
		{"duration with reps", models.Duration, models.WorkoutSet{DurationSeconds: ptr(60), Reps: ptr(1)}, false},
		{"duration missing", models.Duration, models.WorkoutSet{WeightKg: ptr(10.0)}, false},
		{"distance with weight and time", models.Distance, models.WorkoutSet{DistanceM: ptr(40.0), WeightKg: ptr(32.0), DurationSeconds: ptr(30)}, true},
		{"distance zero", models.Distance, models.WorkoutSet{DistanceM: ptr(0.0)}, false},
		{"negative weight", models.WeightReps, models.WorkoutSet{Reps: ptr(5), WeightKg: ptr(-1.0)}, false},
		{"weight too big", models.WeightReps, models.WorkoutSet{Reps: ptr(5), WeightKg: ptr(10000.0)}, false},
		{"unknown type", models.TrackingType("nope"), models.WorkoutSet{Reps: ptr(5)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.set.ValidateFor(c.tt)
			if (err == nil) != c.ok {
				t.Errorf("ValidateFor(%s) = %v, want ok=%v", c.tt, err, c.ok)
			}
		})
	}
}

func TestExerciseNormalizeAndValidate(t *testing.T) {
	e := models.Exercise{
		Name:             "  Close   grip bench ",
		MovementPattern:  models.HorizontalPush,
		PrimaryMuscle:    models.Triceps,
		SecondaryMuscles: []models.Muscle{models.Chest, models.Triceps, models.Chest},
		Mechanic:         models.Compound,
		Equipment:        models.Barbell,
		TrackingType:     models.WeightReps,
	}
	e.Normalize()
	if e.Name != "Close grip bench" {
		t.Errorf("name = %q", e.Name)
	}
	if !slices.Equal(e.SecondaryMuscles, []models.Muscle{models.Chest}) {
		t.Errorf("secondary = %v, want [chest] (deduped, primary dropped)", e.SecondaryMuscles)
	}
	if err := e.Validate(); err != nil {
		t.Errorf("valid exercise rejected: %v", err)
	}

	bad := e
	bad.Equipment = "trampoline"
	if err := bad.Validate(); err == nil {
		t.Error("unknown equipment accepted")
	}
	bad = e
	bad.SecondaryMuscles = []models.Muscle{"neck"}
	if err := bad.Validate(); err == nil {
		t.Error("unknown secondary muscle accepted")
	}
	bad = e
	bad.Name = ""
	if err := bad.Validate(); err == nil {
		t.Error("empty name accepted")
	}
}
