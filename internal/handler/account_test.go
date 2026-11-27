package handler_test

import (
	"context"
	"net/http"
	"testing"
)

// Deleting an account takes everything the user owns with it, needs the
// password, and leaves other users alone.
func TestDeleteAccount(t *testing.T) {
	t.Parallel()
	h, pool := newServerWithPool(t)
	alice, bob := signUp(t, h), signUp(t, h)

	var bench idOnly
	{
		var list []struct {
			ID   string  `json:"id"`
			Slug *string `json:"slug"`
		}
		alice.must(http.StatusOK, "GET", "/exercises", nil, &list)
		for _, e := range list {
			if e.Slug != nil && *e.Slug == "bench-press" {
				bench.ID = e.ID
			}
		}
		if bench.ID == "" {
			t.Fatal("no barbell-bench-press default")
		}
	}

	// Alice: a custom exercise used in a workout and a template, a set on a
	// default exercise too, and a weigh-in. Bob: a workout on the default.
	var ex, w, bw idOnly
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Alice press"), &ex)
	alice.must(http.StatusCreated, "POST", "/workouts", map[string]any{}, &w)
	for _, id := range []string{ex.ID, bench.ID} {
		alice.must(http.StatusCreated, "POST", "/workouts/"+w.ID+"/sets", map[string]any{"exercise_id": id, "reps": 5, "weight_kg": 100})
	}
	alice.must(http.StatusCreated, "POST", "/templates",
		map[string]any{"name": "Alice plan", "exercises": []map[string]any{{"exercise_id": ex.ID, "target_sets": 3}}})
	alice.must(http.StatusOK, "PUT", "/bodyweight/2026-01-10", map[string]any{"weight_kg": 80})
	bob.must(http.StatusCreated, "POST", "/workouts", map[string]any{}, &bw)
	bob.must(http.StatusCreated, "POST", "/workouts/"+bw.ID+"/sets", map[string]any{"exercise_id": bench.ID, "reps": 5, "weight_kg": 60})

	alice.must(http.StatusBadRequest, "POST", "/auth/delete-account", map[string]any{})
	alice.must(http.StatusForbidden, "POST", "/auth/delete-account", map[string]any{"password": "wrong-password"})
	alice.must(http.StatusNoContent, "POST", "/auth/delete-account", map[string]any{"password": "password1"})

	count := func(q string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(context.Background(), q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for q, want := range map[string]int{
		"SELECT count(*) FROM users":                                             1,
		"SELECT count(*) FROM workouts":                                          1,
		"SELECT count(*) FROM workout_sets":                                      1,
		"SELECT count(*) FROM workout_exercises":                                 1,
		"SELECT count(*) FROM workout_templates":                                 0,
		"SELECT count(*) FROM exercises WHERE user_id IS NOT NULL":               0,
		"SELECT count(*) FROM bodyweights":                                       0,
		"SELECT count(*) FROM refresh_tokens r JOIN users u ON u.id = r.user_id": count("SELECT count(*) FROM refresh_tokens"),
	} {
		if got := count(q); got != want {
			t.Errorf("%s = %d, want %d", q, got, want)
		}
	}

	// Gone for good: the password no longer logs in, bob's data is intact.
	alice.must(http.StatusNotFound, "POST", "/auth/delete-account", map[string]any{"password": "password1"})
	bob.must(http.StatusOK, "GET", "/workouts/"+bw.ID, nil)
}
