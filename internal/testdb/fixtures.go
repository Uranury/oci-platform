package testdb

import (
	"context"
	"testing"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemExercise returns the seeded default with this slug.
func SystemExercise(t testing.TB, pool *pgxpool.Pool, slug string) models.Exercise {
	t.Helper()
	list, err := repo.NewExercise(pool).ListSystem(context.Background())
	if err != nil {
		t.Fatalf("testdb: list system exercises: %v", err)
	}
	for _, e := range list {
		if e.Slug != nil && *e.Slug == slug {
			return e
		}
	}
	t.Fatalf("testdb: no system exercise %q", slug)
	return models.Exercise{}
}

// CustomExercise creates a weight_reps exercise owned by ownerID.
func CustomExercise(t testing.TB, pool *pgxpool.Pool, ownerID uuid.UUID, name string) models.Exercise {
	t.Helper()
	e := models.Exercise{
		UserID: &ownerID, Name: name,
		MovementPattern: models.HorizontalPush, PrimaryMuscle: models.Chest, SecondaryMuscles: []models.Muscle{},
		Mechanic: models.Compound, Equipment: models.Barbell, TrackingType: models.WeightReps,
	}
	if err := repo.NewExercise(pool).Create(context.Background(), &e); err != nil {
		t.Fatalf("testdb: create exercise: %v", err)
	}
	return e
}

// Workout creates a workout owned by ownerID, performed at performedAt.
func Workout(t testing.TB, pool *pgxpool.Pool, ownerID uuid.UUID, performedAt time.Time) models.Workout {
	t.Helper()
	w := models.Workout{UserID: ownerID, PerformedAt: performedAt}
	if err := repo.NewWorkout(pool).Create(context.Background(), &w); err != nil {
		t.Fatalf("testdb: create workout: %v", err)
	}
	return w
}

// Set logs a weight × reps set without going through validation.
func Set(t testing.TB, pool *pgxpool.Pool, workoutID, exerciseID uuid.UUID, kg float64, reps int) models.WorkoutSet {
	t.Helper()
	s := models.WorkoutSet{WorkoutID: workoutID, ExerciseID: exerciseID, WeightKg: &kg, Reps: &reps}
	if err := repo.NewWorkoutSet(pool).Create(context.Background(), &s); err != nil {
		t.Fatalf("testdb: create set: %v", err)
	}
	return s
}
