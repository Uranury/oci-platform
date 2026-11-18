// Package seed holds the system default exercise library and syncs it into
// the database at startup.
//
// To fix or add a default, edit Exercises and redeploy: rows are matched by
// slug, so a changed name or category updates the existing row (and every
// set already logged against it), a new slug adds a row, and a removed slug
// archives its row (history keeps resolving). Custom exercises are never
// touched. Never reuse or change a slug: it's the row's identity.
package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/pkg/dbutil"
)

type muscles = []models.Muscle

func ex(slug, name string, p models.MovementPattern, primary models.Muscle, secondary muscles,
	mech models.Mechanic, eq models.Equipment, tt models.TrackingType) models.Exercise {
	if secondary == nil {
		secondary = muscles{}
	}
	return models.Exercise{
		Slug: &slug, Name: name, MovementPattern: p, PrimaryMuscle: primary,
		SecondaryMuscles: secondary, Mechanic: mech, Equipment: eq, TrackingType: tt,
	}
}

// Exercises is the deliberately small library of common lifts every user
// sees. Keep it short; users add anything else as custom exercises.
var Exercises = []models.Exercise{
	// Push
	ex("bench-press", "Bench press", models.HorizontalPush, models.Chest, muscles{models.Triceps, models.Shoulders}, models.Compound, models.Barbell, models.WeightReps),
	ex("incline-dumbbell-press", "Incline dumbbell press", models.HorizontalPush, models.Chest, muscles{models.Shoulders, models.Triceps}, models.Compound, models.Dumbbell, models.WeightReps),
	ex("push-up", "Push-up", models.HorizontalPush, models.Chest, muscles{models.Triceps, models.Shoulders}, models.Compound, models.Bodyweight, models.BodyweightReps),
	ex("overhead-press", "Overhead press", models.VerticalPush, models.Shoulders, muscles{models.Triceps}, models.Compound, models.Barbell, models.WeightReps),
	ex("dip", "Dip", models.VerticalPush, models.Triceps, muscles{models.Chest, models.Shoulders}, models.Compound, models.Bodyweight, models.BodyweightReps),

	// Pull
	ex("pull-up", "Pull-up", models.VerticalPull, models.Lats, muscles{models.Biceps}, models.Compound, models.Bodyweight, models.BodyweightReps),
	ex("lat-pulldown", "Lat pulldown", models.VerticalPull, models.Lats, muscles{models.Biceps}, models.Compound, models.Cable, models.WeightReps),
	ex("barbell-row", "Barbell row", models.HorizontalPull, models.UpperBack, muscles{models.Lats, models.Biceps}, models.Compound, models.Barbell, models.WeightReps),
	ex("seated-cable-row", "Seated cable row", models.HorizontalPull, models.UpperBack, muscles{models.Lats, models.Biceps}, models.Compound, models.Cable, models.WeightReps),

	// Legs
	ex("back-squat", "Back squat", models.Squat, models.Quads, muscles{models.Glutes}, models.Compound, models.Barbell, models.WeightReps),
	ex("leg-press", "Leg press", models.Squat, models.Quads, muscles{models.Glutes}, models.Compound, models.Machine, models.WeightReps),
	ex("deadlift", "Deadlift", models.Hinge, models.Hamstrings, muscles{models.Glutes, models.UpperBack, models.Forearms}, models.Compound, models.Barbell, models.WeightReps),
	ex("romanian-deadlift", "Romanian deadlift", models.Hinge, models.Hamstrings, muscles{models.Glutes}, models.Compound, models.Barbell, models.WeightReps),
	ex("hip-thrust", "Hip thrust", models.Hinge, models.Glutes, muscles{models.Hamstrings}, models.Compound, models.Barbell, models.WeightReps),
	ex("dumbbell-lunge", "Dumbbell lunge", models.Lunge, models.Quads, muscles{models.Glutes}, models.Compound, models.Dumbbell, models.WeightReps),
	ex("standing-calf-raise", "Standing calf raise", models.OtherPattern, models.Calves, nil, models.Isolation, models.Machine, models.WeightReps),

	// Arms & shoulders
	ex("dumbbell-curl", "Dumbbell curl", models.OtherPattern, models.Biceps, muscles{models.Forearms}, models.Isolation, models.Dumbbell, models.WeightReps),
	ex("triceps-pushdown", "Triceps pushdown", models.OtherPattern, models.Triceps, nil, models.Isolation, models.Cable, models.WeightReps),
	ex("lateral-raise", "Lateral raise", models.OtherPattern, models.Shoulders, nil, models.Isolation, models.Dumbbell, models.WeightReps),

	// Core & carries
	ex("plank", "Plank", models.Core, models.Abs, nil, models.Isolation, models.Bodyweight, models.Duration),
	ex("hanging-leg-raise", "Hanging leg raise", models.Core, models.Abs, nil, models.Isolation, models.Bodyweight, models.BodyweightReps),
	ex("farmers-carry", "Farmer's carry", models.Carry, models.Forearms, muscles{models.UpperBack, models.Abs}, models.Compound, models.Dumbbell, models.Distance),
}

// Validate checks every default against the enums and for duplicate slugs
// or names; run by tests and again before every sync.
func Validate(list []models.Exercise) error {
	slugs, names := map[string]bool{}, map[string]bool{}
	for i := range list {
		e := list[i] // copy: Normalize mutates
		if e.Slug == nil || *e.Slug == "" {
			return fmt.Errorf("seed: %q has no slug", e.Name)
		}
		e.Normalize()
		if err := e.Validate(); err != nil {
			return fmt.Errorf("seed: %s: %w", *e.Slug, err)
		}
		key := strings.ToLower(e.Name)
		if slugs[*e.Slug] || names[key] {
			return fmt.Errorf("seed: duplicate slug or name: %s / %s", *e.Slug, e.Name)
		}
		slugs[*e.Slug], names[key] = true, true
	}
	return nil
}

// Sync validates the library and upserts it in one transaction.
func Sync(ctx context.Context, tx dbutil.TxProvider) error {
	if err := Validate(Exercises); err != nil {
		return err
	}
	return tx.RunInTx(ctx, func(exec dbutil.Executor) error {
		return repo.NewExercise(exec).SyncSystem(ctx, Exercises)
	})
}
