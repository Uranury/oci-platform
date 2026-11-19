package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/Uranury/oci-platform/internal/testdb"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) { testdb.Main(m) }

type services struct {
	exercises *service.ExerciseService
	workouts  *service.WorkoutService
	templates *service.TemplateService
}

func newServices(pool *pgxpool.Pool) services {
	exercises, workouts, sets := repo.NewExercise(pool), repo.NewWorkout(pool), repo.NewWorkoutSet(pool)
	return services{
		exercises: service.NewExerciseService(exercises),
		workouts:  service.NewWorkoutService(dbutil.NewTxProvider(pool), exercises, workouts, sets, repo.NewWorkoutExercise(pool)),
		templates: service.NewTemplateService(dbutil.NewTxProvider(pool), repo.NewTemplate(pool), exercises, workouts, sets),
	}
}

// Deleting a custom exercise removes it outright when nothing references it
// and archives it when sets or a template do, so history keeps resolving.
func TestExerciseDeleteArchivesWhenUsed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	svc := newServices(pool)
	alice := testdb.User(t, pool)

	visible := func(id any) bool {
		t.Helper()
		list, err := svc.exercises.List(ctx, alice)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range list {
			if e.ID == id {
				return true
			}
		}
		return false
	}

	t.Run("unused is deleted", func(t *testing.T) {
		ex := testdb.CustomExercise(t, pool, alice, "Unused")
		archived, err := svc.exercises.Delete(ctx, alice, ex.ID)
		if err != nil || archived {
			t.Fatalf("Delete = (archived %v, %v), want (false, nil)", archived, err)
		}
		if _, err := repo.NewExercise(pool).GetVisible(ctx, alice, ex.ID); !errors.Is(err, repo.ErrNotFound) {
			t.Errorf("row still there after delete: %v", err)
		}
	})

	t.Run("with sets is archived", func(t *testing.T) {
		ex := testdb.CustomExercise(t, pool, alice, "Logged")
		w := testdb.Workout(t, pool, alice, time.Now())
		testdb.Set(t, pool, w.ID, ex.ID, 40, 8)

		archived, err := svc.exercises.Delete(ctx, alice, ex.ID)
		if err != nil || !archived {
			t.Fatalf("Delete = (archived %v, %v), want (true, nil)", archived, err)
		}
		if visible(ex.ID) {
			t.Error("archived exercise still listed")
		}
		detail, err := svc.workouts.GetWorkout(ctx, alice, w.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(detail.Sets) != 1 || detail.Sets[0].ExerciseName != "Logged" {
			t.Errorf("history lost its exercise: %+v", detail.Sets)
		}
		// Archived exercises can't take new sets.
		reps, kg := 5, 40.0
		_, err = svc.workouts.AddSet(ctx, alice, w.ID, ex.ID, service.SetInput{Reps: &reps, WeightKg: &kg})
		if !errors.Is(err, service.ErrExerciseArchived) {
			t.Errorf("AddSet on archived exercise: got %v, want ErrExerciseArchived", err)
		}
	})

	t.Run("in a template is archived", func(t *testing.T) {
		ex := testdb.CustomExercise(t, pool, alice, "Planned")
		_, err := svc.templates.Create(ctx, alice, service.TemplateInput{
			Name: "Plan", Exercises: []models.TemplateExercise{{ExerciseID: ex.ID, TargetSets: 3}},
		})
		if err != nil {
			t.Fatal(err)
		}
		archived, err := svc.exercises.Delete(ctx, alice, ex.ID)
		if err != nil || !archived {
			t.Fatalf("Delete = (archived %v, %v), want (true, nil)", archived, err)
		}
	})
}

// A template and its exercise list are saved in one transaction: when any
// item fails, neither the name nor the list changes.
func TestTemplateSaveIsAtomic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	svc := newServices(pool)
	alice, bob := testdb.User(t, pool), testdb.User(t, pool)
	bench := testdb.SystemExercise(t, pool, "bench-press").ID
	squat := testdb.SystemExercise(t, pool, "back-squat").ID
	bobs := testdb.CustomExercise(t, pool, bob, "Bob's secret lift").ID

	orig, err := svc.templates.Create(ctx, alice, service.TemplateInput{
		Name: "Push", Exercises: []models.TemplateExercise{{ExerciseID: bench, TargetSets: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.templates.Create(ctx, alice, service.TemplateInput{Name: "Legs"}); err != nil {
		t.Fatal(err)
	}

	unchanged := func(t *testing.T) {
		t.Helper()
		got, err := svc.templates.Get(ctx, alice, orig.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Push" || len(got.Exercises) != 1 || got.Exercises[0].ExerciseID != bench {
			t.Errorf("template changed by a failed save: name %q, exercises %+v", got.Name, got.Exercises)
		}
	}

	t.Run("invalid item after a valid one", func(t *testing.T) {
		// The name is written and squat checked before Bob's exercise fails.
		_, err := svc.templates.Update(ctx, alice, orig.ID, service.TemplateInput{
			Name:      "Renamed",
			Exercises: []models.TemplateExercise{{ExerciseID: squat, TargetSets: 5}, {ExerciseID: bobs, TargetSets: 1}},
		})
		if _, ok := errors.AsType[models.ValidationError](err); !ok {
			t.Fatalf("got %v, want a ValidationError", err)
		}
		unchanged(t)
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, err := svc.templates.Update(ctx, alice, orig.ID, service.TemplateInput{
			Name: "Legs", Exercises: []models.TemplateExercise{{ExerciseID: squat, TargetSets: 5}},
		})
		if !errors.Is(err, service.ErrTemplateExists) {
			t.Fatalf("got %v, want ErrTemplateExists", err)
		}
		unchanged(t)
	})

	t.Run("failed create leaves nothing", func(t *testing.T) {
		_, err := svc.templates.Create(ctx, alice, service.TemplateInput{
			Name:      "Half",
			Exercises: []models.TemplateExercise{{ExerciseID: bench, TargetSets: 3}, {ExerciseID: bobs, TargetSets: 3}},
		})
		if err == nil {
			t.Fatal("Create with another user's exercise succeeded")
		}
		list, err := svc.templates.List(ctx, alice)
		if err != nil {
			t.Fatal(err)
		}
		for _, tp := range list {
			if tp.Name == "Half" {
				t.Error("half-saved template exists")
			}
		}
	})
}
