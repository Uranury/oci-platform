package repo_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/testdb"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// Visibility is enforced in the SQL: reads see system rows and the user's
// own, writes only the user's own. Another user's exercise behaves exactly
// like one that doesn't exist.
func TestExerciseVisibility(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	exercises := repo.NewExercise(pool)
	alice, bob := testdb.User(t, pool), testdb.User(t, pool)
	mine := testdb.CustomExercise(t, pool, alice, "Alice press")
	system := testdb.SystemExercise(t, pool, "bench-press")

	if _, err := exercises.GetVisible(ctx, alice, mine.ID); err != nil {
		t.Errorf("owner GetVisible: %v", err)
	}
	if _, err := exercises.GetVisible(ctx, bob, system.ID); err != nil {
		t.Errorf("system exercise GetVisible: %v", err)
	}
	if _, err := exercises.GetVisible(ctx, bob, mine.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("other user GetVisible: got %v, want ErrNotFound", err)
	}

	list, err := exercises.ListVisible(ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.ID == mine.ID {
			t.Error("ListVisible shows another user's exercise")
		}
	}

	stolen := mine
	stolen.UserID = &bob
	stolen.Name = "Hijacked"
	if err := exercises.Update(ctx, &stolen); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Update as other user: got %v, want ErrNotFound", err)
	}
	if err := exercises.Delete(ctx, bob, mine.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Delete as other user: got %v, want ErrNotFound", err)
	}
	if err := exercises.Archive(ctx, bob, mine.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Archive as other user: got %v, want ErrNotFound", err)
	}
	if err := exercises.Delete(ctx, alice, system.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Delete system exercise: got %v, want ErrNotFound", err)
	}

	got, err := exercises.GetVisible(ctx, alice, mine.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != mine.Name || got.ArchivedAt != nil {
		t.Errorf("exercise changed by another user: %+v", got)
	}
}

func TestErrorTranslation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	alice := testdb.User(t, pool)

	t.Run("duplicate", func(t *testing.T) {
		users := repo.NewUser(pool)
		if err := users.Create(ctx, uuid.New(), "dup", "dup@example.test", "x"); err != nil {
			t.Fatal(err)
		}
		err := users.Create(ctx, uuid.New(), "dup2", "dup@example.test", "x")
		if !errors.Is(err, repo.ErrDuplicate) {
			t.Errorf("got %v, want ErrDuplicate", err)
		}
	})

	t.Run("in use", func(t *testing.T) {
		ex := testdb.CustomExercise(t, pool, alice, "Used")
		w := testdb.Workout(t, pool, alice, time.Now())
		testdb.Set(t, pool, w.ID, ex.ID, 50, 5)
		err := repo.NewExercise(pool).Delete(ctx, alice, ex.ID)
		if !errors.Is(err, repo.ErrInUse) {
			t.Errorf("got %v, want ErrInUse", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.NewWorkout(pool).GetByID(ctx, uuid.New())
		if !errors.Is(err, repo.ErrNotFound) {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

// ListPrevious picks, per exercise, the latest workout before the cutoff
// that included it, and never another user's.
func TestListPrevious(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	alice, bob := testdb.User(t, pool), testdb.User(t, pool)
	bench := testdb.SystemExercise(t, pool, "bench-press").ID
	squat := testdb.SystemExercise(t, pool, "back-squat").ID
	now := time.Now()
	day := 24 * time.Hour

	w10 := testdb.Workout(t, pool, alice, now.Add(-10*day))
	testdb.Set(t, pool, w10.ID, bench, 50, 10)
	testdb.Set(t, pool, w10.ID, squat, 80, 5)
	w5 := testdb.Workout(t, pool, alice, now.Add(-5*day))
	testdb.Set(t, pool, w5.ID, bench, 60, 10)
	testdb.Set(t, pool, w5.ID, bench, 60, 8)
	testdb.Workout(t, pool, alice, now.Add(-3*day)) // empty: never "last time"
	bobs := testdb.Workout(t, pool, bob, now.Add(-1*day))
	testdb.Set(t, pool, bobs.ID, bench, 200, 1)
	current := testdb.Workout(t, pool, alice, now)
	testdb.Set(t, pool, current.ID, bench, 62.5, 5)

	sets := repo.NewWorkoutSet(pool)
	type want struct {
		workout uuid.UUID
		weights []float64
	}
	check := func(t *testing.T, before time.Time, exclude uuid.UUID, exp map[uuid.UUID]want) {
		t.Helper()
		got, err := sets.ListPrevious(ctx, alice, exclude, before)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(exp) {
			t.Fatalf("got %d exercises, want %d: %+v", len(got), len(exp), got)
		}
		for _, p := range got {
			e, ok := exp[p.ExerciseID]
			if !ok {
				t.Fatalf("unexpected exercise %s", p.ExerciseID)
			}
			if p.WorkoutID != e.workout {
				t.Errorf("exercise %s: from workout %s, want %s", p.ExerciseID, p.WorkoutID, e.workout)
			}
			var weights []float64
			for _, s := range p.Sets {
				weights = append(weights, *s.WeightKg)
			}
			if len(weights) != len(e.weights) {
				t.Errorf("exercise %s: weights %v, want %v", p.ExerciseID, weights, e.weights)
				continue
			}
			for i := range weights {
				if weights[i] != e.weights[i] {
					t.Errorf("exercise %s: weights %v, want %v (in logged order)", p.ExerciseID, weights, e.weights)
					break
				}
			}
		}
	}

	t.Run("from today", func(t *testing.T) {
		check(t, current.PerformedAt, current.ID, map[uuid.UUID]want{
			bench: {w5.ID, []float64{60, 60}},
			squat: {w10.ID, []float64{80}},
		})
	})
	t.Run("from an older workout", func(t *testing.T) {
		check(t, w5.PerformedAt, w5.ID, map[uuid.UUID]want{
			bench: {w10.ID, []float64{50}},
			squat: {w10.ID, []float64{80}},
		})
	})
	t.Run("nothing earlier", func(t *testing.T) {
		check(t, w10.PerformedAt, w10.ID, map[uuid.UUID]want{})
	})
}

func TestTemplatesListMostDueFirst(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	alice := testdb.User(t, pool)
	templates, workouts := repo.NewTemplate(pool), repo.NewWorkout(pool)
	now := time.Now()

	// lastUsed: days ago; -1 = never done.
	for _, tc := range []struct {
		name     string
		lastUsed int
	}{{"A recent", 1}, {"B never", -1}, {"C old", 30}, {"D never", -1}, {"E middle", 7}} {
		tp := &models.Template{UserID: alice, Name: tc.name}
		if err := templates.Create(ctx, tp); err != nil {
			t.Fatal(err)
		}
		if tc.lastUsed >= 0 {
			// An earlier use too, so it's the latest use that counts.
			for _, d := range []int{tc.lastUsed + 60, tc.lastUsed} {
				w := &models.Workout{UserID: alice, TemplateID: &tp.ID, PerformedAt: now.Add(-time.Duration(d) * 24 * time.Hour)}
				if err := workouts.Create(ctx, w); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	list, err := templates.List(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tp := range list {
		got = append(got, tp.Name)
	}
	want := []string{"B never", "D never", "C old", "E middle", "A recent"}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}
