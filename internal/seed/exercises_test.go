package seed

import (
	"slices"
	"testing"

	"github.com/Uranury/oci-platform/internal/models"
)

func TestSeedIsValid(t *testing.T) {
	if err := Validate(Exercises); err != nil {
		t.Fatal(err)
	}
}

func TestSeedStaysSmall(t *testing.T) {
	if len(Exercises) > 30 {
		t.Errorf("%d defaults; the library is meant to stay small", len(Exercises))
	}
}

func find(t *testing.T, slug string) models.Exercise {
	t.Helper()
	for _, e := range Exercises {
		if *e.Slug == slug {
			return e
		}
	}
	t.Fatalf("seed is missing %s", slug)
	return models.Exercise{}
}

// The three defaults the spec pins down exactly.
func TestRequiredDefaults(t *testing.T) {
	cases := []struct {
		slug      string
		pattern   models.MovementPattern
		primary   models.Muscle
		secondary []models.Muscle
		mech      models.Mechanic
		equip     models.Equipment
		tracking  models.TrackingType
	}{
		{"pull-up", models.VerticalPull, models.Lats, []models.Muscle{models.Biceps}, models.Compound, models.Bodyweight, models.BodyweightReps},
		{"back-squat", models.Squat, models.Quads, []models.Muscle{models.Glutes}, models.Compound, models.Barbell, models.WeightReps},
		{"bench-press", models.HorizontalPush, models.Chest, []models.Muscle{models.Triceps, models.Shoulders}, models.Compound, models.Barbell, models.WeightReps},
	}
	for _, c := range cases {
		e := find(t, c.slug)
		sec := slices.Clone(e.SecondaryMuscles)
		want := slices.Clone(c.secondary)
		slices.Sort(sec)
		slices.Sort(want)
		if e.MovementPattern != c.pattern || e.PrimaryMuscle != c.primary || !slices.Equal(sec, want) ||
			e.Mechanic != c.mech || e.Equipment != c.equip || e.TrackingType != c.tracking {
			t.Errorf("%s = %+v, doesn't match the spec", c.slug, e)
		}
	}
}
