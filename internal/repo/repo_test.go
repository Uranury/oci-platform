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

func TestTemplatesListExerciseNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	alice := testdb.User(t, pool)
	bench := testdb.SystemExercise(t, pool, "bench-press")
	squat := testdb.SystemExercise(t, pool, "back-squat")
	templates := repo.NewTemplate(pool)

	full := &models.Template{UserID: alice, Name: "A full"}
	empty := &models.Template{UserID: alice, Name: "B empty"}
	for _, tp := range []*models.Template{full, empty} {
		if err := templates.Create(ctx, tp); err != nil {
			t.Fatal(err)
		}
	}
	// Plan order, not name order.
	if err := templates.ReplaceExercises(ctx, full.ID, []models.TemplateExercise{
		{ExerciseID: squat.ID, TargetSets: 3}, {ExerciseID: bench.ID, TargetSets: 3},
	}); err != nil {
		t.Fatal(err)
	}

	list, err := templates.List(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, tp := range list {
		if tp.Exercises == nil {
			t.Errorf("%s: exercises is nil, want a JSON array", tp.Name)
		}
		got[tp.Name] = tp.Exercises
	}
	if want := []string{squat.Name, bench.Name}; !slices.Equal(got["A full"], want) {
		t.Errorf("A full exercises = %v, want %v", got["A full"], want)
	}
	if len(got["B empty"]) != 0 {
		t.Errorf("B empty exercises = %v, want none", got["B empty"])
	}
}

func TestWorkoutListSummaries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	alice, bob := testdb.User(t, pool), testdb.User(t, pool)
	bench := testdb.SystemExercise(t, pool, "bench-press")
	squat := testdb.SystemExercise(t, pool, "back-squat")
	now := time.Now()

	// Squat logged first, then bench, then squat again: order is by first
	// logged, not alphabetical, and squat is listed once.
	older := testdb.Workout(t, pool, alice, now.Add(-48*time.Hour))
	testdb.Set(t, pool, older.ID, squat.ID, 100, 5)
	testdb.Set(t, pool, older.ID, bench.ID, 60, 8)
	testdb.Set(t, pool, older.ID, squat.ID, 100, 5)
	empty := testdb.Workout(t, pool, alice, now)
	bobs := testdb.Workout(t, pool, bob, now.Add(-time.Hour))
	testdb.Set(t, pool, bobs.ID, bench.ID, 200, 1)

	got, err := repo.NewWorkout(pool).ListByUser(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d workouts, want 2 (not bob's): %+v", len(got), got)
	}
	if got[0].ID != empty.ID || got[1].ID != older.ID {
		t.Fatalf("order: got %s, %s; want newest first", got[0].ID, got[1].ID)
	}
	if got[0].SetCount != 0 || got[0].Exercises == nil || len(got[0].Exercises) != 0 {
		t.Errorf("empty workout: set_count %d, exercises %#v; want 0 and an empty (non-nil) list",
			got[0].SetCount, got[0].Exercises)
	}
	if got[1].SetCount != 3 {
		t.Errorf("set_count %d, want 3", got[1].SetCount)
	}
	want := []string{squat.Name, bench.Name}
	if len(got[1].Exercises) != 2 || got[1].Exercises[0] != want[0] || got[1].Exercises[1] != want[1] {
		t.Errorf("exercises %v, want %v", got[1].Exercises, want)
	}
}

func TestExerciseHistory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := testdb.New(t)
	alice, bob := testdb.User(t, pool), testdb.User(t, pool)
	bench := testdb.SystemExercise(t, pool, "bench-press").ID
	squat := testdb.SystemExercise(t, pool, "back-squat").ID
	now := time.Now()

	// Alice benches on days 1..4 ago (two sets each, 60+day kg), squats
	// once without benching, and Bob benches yesterday.
	for d := 4; d >= 1; d-- {
		w := testdb.Workout(t, pool, alice, now.Add(-time.Duration(d)*24*time.Hour))
		testdb.Set(t, pool, w.ID, bench, float64(60+d), 5)
		testdb.Set(t, pool, w.ID, squat, 100, 5)
		testdb.Set(t, pool, w.ID, bench, float64(60+d), 4)
	}
	squatOnly := testdb.Workout(t, pool, alice, now)
	testdb.Set(t, pool, squatOnly.ID, squat, 100, 5)
	bobs := testdb.Workout(t, pool, bob, now.Add(-time.Hour))
	testdb.Set(t, pool, bobs.ID, bench, 200, 1)

	got, err := repo.NewWorkoutSet(pool).ListHistory(ctx, alice, bench, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d workouts, want 3 (the limit; not the squat-only one, not Bob's)", len(got))
	}
	for i, p := range got {
		wantKg := float64(60 + i + 1) // newest (1 day ago) first
		if p.ExerciseID != bench || len(p.Sets) != 2 {
			t.Fatalf("workout %d: %+v, want 2 bench sets", i, p)
		}
		if *p.Sets[0].WeightKg != wantKg || *p.Sets[0].Reps != 5 || *p.Sets[1].Reps != 4 {
			t.Errorf("workout %d: sets %v kg x %d, %d; want %v kg x 5 then 4 (logged order)",
				i, *p.Sets[0].WeightKg, *p.Sets[0].Reps, *p.Sets[1].Reps, wantKg)
		}
	}
}
