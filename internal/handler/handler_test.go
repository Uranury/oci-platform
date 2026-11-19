package handler_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Uranury/oci-platform/internal/auth"
	"github.com/Uranury/oci-platform/internal/handler"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/Uranury/oci-platform/internal/testdb"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// client calls the full HTTP stack (router, middleware, services, real DB)
// as one user.
type client struct {
	t     *testing.T
	h     http.Handler
	token string
}

func newServer(t *testing.T) http.Handler {
	t.Helper()
	h, _ := newServerWithPool(t)
	return h
}

func newServerWithPool(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.New(t)
	exercises, workouts, sets := repo.NewExercise(pool), repo.NewWorkout(pool), repo.NewWorkoutSet(pool)
	tx := dbutil.NewTxProvider(pool)
	jwt := auth.NewJWTIssuer("test-secret-test-secret-test-secret", time.Minute)
	return handler.NewRouter(
		handler.NewAuthHandler(service.NewAuthService(repo.NewUser(pool), repo.NewRefreshToken(pool), tx, jwt, time.Hour)),
		handler.NewExerciseHandler(service.NewExerciseService(exercises)),
		handler.NewWorkoutHandler(service.NewWorkoutService(tx, exercises, workouts, sets, repo.NewWorkoutExercise(pool))),
		handler.NewTemplateHandler(service.NewTemplateService(tx, repo.NewTemplate(pool), exercises, workouts, sets)),
		jwt,
		fstest.MapFS{"index.html": {Data: []byte("<html>")}},
	), pool
}

// signUp registers and logs in a new user.
func signUp(t *testing.T, h http.Handler) *client {
	t.Helper()
	c := &client{t: t, h: h}
	tag := uuid.NewString()[:8]
	creds := map[string]string{"username": "u" + tag, "email": tag + "@example.test", "password": "password1"}
	c.must(http.StatusCreated, "POST", "/auth/register", creds)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	c.must(http.StatusOK, "POST", "/auth/login", creds, &tok)
	c.token = tok.AccessToken
	return c
}

// do sends a request; body is JSON-encoded unless nil.
func (c *client) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, path, nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		r.Header.Set("Authorization", "Bearer "+c.token)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}

// must sends a request, fails unless it returns want, and decodes the
// response into out (if given).
func (c *client) must(want int, method, path string, body any, out ...any) {
	c.t.Helper()
	code, resp := c.do(method, path, body)
	if code != want {
		c.t.Fatalf("%s %s: got %d, want %d: %s", method, path, code, want, resp)
	}
	if len(out) > 0 {
		if err := json.Unmarshal(resp, out[0]); err != nil {
			c.t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
}

type idOnly struct {
	ID string `json:"id"`
}

func exerciseBody(name string) map[string]any {
	return map[string]any{
		"name": name, "movement_pattern": "horizontal_push", "primary_muscle": "chest",
		"secondary_muscles": []string{}, "mechanic": "compound", "equipment": "barbell", "tracking_type": "weight_reps",
	}
}

// Every route that takes an ID answers 404 when the ID belongs to another
// user, exactly as for an ID that doesn't exist, and changes nothing.
func TestOtherUsersGet404(t *testing.T) {
	t.Parallel()
	h := newServer(t)
	alice, bob := signUp(t, h), signUp(t, h)

	var ex, workout, set, tmpl idOnly
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Alice press"), &ex)
	alice.must(http.StatusCreated, "POST", "/workouts", map[string]any{"name": "Alice day"}, &workout)
	alice.must(http.StatusCreated, "POST", "/workouts/"+workout.ID+"/sets",
		map[string]any{"exercise_id": ex.ID, "reps": 5, "weight_kg": 100}, &set)
	alice.must(http.StatusCreated, "POST", "/templates",
		map[string]any{"name": "Alice plan", "exercises": []map[string]any{{"exercise_id": ex.ID, "target_sets": 3}}}, &tmpl)

	var bobWorkout idOnly
	bob.must(http.StatusCreated, "POST", "/workouts", map[string]any{}, &bobWorkout)
	var bench idOnly
	{
		var list []struct {
			ID   string  `json:"id"`
			Slug *string `json:"slug"`
		}
		bob.must(http.StatusOK, "GET", "/exercises", nil, &list)
		for _, e := range list {
			if e.Slug != nil && *e.Slug == "bench-press" {
				bench.ID = e.ID
			}
			if e.ID == ex.ID {
				t.Error("Bob's exercise list includes Alice's custom exercise")
			}
		}
	}
	bobSet := map[string]any{"exercise_id": bench.ID, "reps": 1, "weight_kg": 1}

	w, s, e, tp := "/workouts/"+workout.ID, "/sets/"+set.ID, "/exercises/"+ex.ID, "/templates/"+tmpl.ID
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{"PUT", e, exerciseBody("Hijacked")},
		{"DELETE", e, nil},
		{"POST", e + "/duplicate", map[string]any{}},
		{"GET", e + "/history", nil},

		{"GET", w, nil},
		{"POST", w + "/exercises", map[string]any{"exercise_id": bench.ID}},
		{"PUT", w + "/exercises", map[string]any{"exercise_ids": []string{ex.ID}}},
		{"DELETE", w + "/exercises/" + ex.ID, nil},
		{"PUT", w, map[string]any{"name": "Hijacked"}},
		{"DELETE", w, nil},
		{"POST", w + "/finish", nil},
		{"GET", w + "/previous", nil},
		{"POST", w + "/save-as-template", map[string]any{"name": "Stolen"}},
		{"POST", w + "/sets", bobSet},
		{"PUT", w + s, bobSet},
		{"DELETE", w + s, nil},
		// Alice's set under Bob's own workout: sets are scoped to their workout.
		{"PUT", "/workouts/" + bobWorkout.ID + s, bobSet},
		{"DELETE", "/workouts/" + bobWorkout.ID + s, nil},
		// Alice's exercise in Bob's own workout.
		{"POST", "/workouts/" + bobWorkout.ID + "/sets", map[string]any{"exercise_id": ex.ID, "reps": 1, "weight_kg": 1}},

		{"GET", tp, nil},
		{"PUT", tp, map[string]any{"name": "Hijacked", "exercises": []any{}}},
		{"DELETE", tp, nil},
		{"POST", tp + "/start", nil},
	} {
		if code, body := bob.do(tc.method, tc.path, tc.body); code != http.StatusNotFound {
			t.Errorf("%s %s as another user: got %d, want 404: %s", tc.method, tc.path, code, body)
		}
	}

	// Alice's exercise can't be put in Bob's template either (400: the
	// template is Bob's own, the item is what's invalid).
	code, body := bob.do("POST", "/templates", map[string]any{
		"name": "Bob plan", "exercises": []map[string]any{{"exercise_id": ex.ID, "target_sets": 3}},
	})
	if code != http.StatusBadRequest || !strings.Contains(string(body), "not found") {
		t.Errorf("template with another user's exercise: got %d %s, want 400 not found", code, body)
	}

	// Lists don't leak either.
	for _, path := range []string{"/workouts", "/templates"} {
		_, body := bob.do("GET", path, nil)
		if strings.Contains(string(body), workout.ID) || strings.Contains(string(body), tmpl.ID) {
			t.Errorf("GET %s as another user includes Alice's data: %s", path, body)
		}
	}

	// And nothing of Alice's changed.
	var got struct {
		Name string `json:"name"`
		Sets []struct {
			ID       string  `json:"id"`
			Reps     int     `json:"reps"`
			WeightKg float64 `json:"weight_kg"`
		} `json:"sets"`
		FinishedAt *string `json:"finished_at"`
	}
	alice.must(http.StatusOK, "GET", w, nil, &got)
	if got.Name != "Alice day" || got.FinishedAt != nil || len(got.Sets) != 1 ||
		got.Sets[0].ID != set.ID || got.Sets[0].Reps != 5 || got.Sets[0].WeightKg != 100 {
		t.Errorf("Alice's workout changed: %+v", got)
	}
	var gotTmpl struct {
		Name      string `json:"name"`
		Exercises []any  `json:"exercises"`
	}
	alice.must(http.StatusOK, "GET", tp, nil, &gotTmpl)
	if gotTmpl.Name != "Alice plan" || len(gotTmpl.Exercises) != 1 {
		t.Errorf("Alice's template changed: %+v", gotTmpl)
	}
	_, list := alice.do("GET", "/exercises", nil)
	if !strings.Contains(string(list), `"Alice press"`) {
		t.Error("Alice's exercise was renamed or removed")
	}
}

func TestAuthRequired(t *testing.T) {
	t.Parallel()
	h := newServer(t)
	anon := &client{t: t, h: h}
	bogus := &client{t: t, h: h, token: "not-a-jwt"}
	for _, path := range []string{"/workouts", "/exercises", "/templates"} {
		if code, _ := anon.do("GET", path, nil); code != http.StatusUnauthorized {
			t.Errorf("GET %s without a token: got %d, want 401", path, code)
		}
		if code, _ := bogus.do("GET", path, nil); code != http.StatusUnauthorized {
			t.Errorf("GET %s with a bad token: got %d, want 401", path, code)
		}
	}
	// Public routes stay public.
	if code, _ := anon.do("GET", "/exercise-options", nil); code != http.StatusOK {
		t.Errorf("GET /exercise-options: got %d, want 200", code)
	}
}

// A 5xx response hides the cause from the client but logs it. Not parallel:
// it swaps the global logger.
func TestServerErrorsAreLogged(t *testing.T) {
	h, pool := newServerWithPool(t)
	alice := signUp(t, h)
	var w idOnly
	alice.must(http.StatusCreated, "POST", "/workouts", map[string]any{}, &w)

	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	pool.Close() // every query now fails

	for _, path := range []string{
		"/workouts",         // handler answers the 500 itself
		"/workouts/" + w.ID, // 500 from handleErrors' fallback
	} {
		logs.Reset()
		code, body := alice.do("GET", path, nil)
		if code != http.StatusInternalServerError {
			t.Fatalf("GET %s: got %d, want 500", path, code)
		}
		if strings.Contains(string(body), "closed") {
			t.Errorf("GET %s: response leaks the cause: %s", path, body)
		}
		line := logs.String()
		if !strings.Contains(line, "level=ERROR") || !strings.Contains(line, "err=") || !strings.Contains(line, "closed pool") {
			t.Errorf("GET %s: log line lacks the error: %q", path, line)
		}
	}
}

// Last time per exercise across workouts, and one exercise's history: both
// only ever the caller's own sets.
func TestExerciseLastAndHistory(t *testing.T) {
	t.Parallel()
	h := newServer(t)
	alice, bob := signUp(t, h), signUp(t, h)

	var ex idOnly
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Alice press"), &ex)
	logWorkout := func(c *client, kg float64) {
		var w idOnly
		c.must(http.StatusCreated, "POST", "/workouts", map[string]any{}, &w)
		var set idOnly
		c.must(http.StatusCreated, "POST", "/workouts/"+w.ID+"/sets",
			map[string]any{"exercise_id": ex.ID, "reps": 5, "weight_kg": kg}, &set)
	}
	logWorkout(alice, 60)
	logWorkout(alice, 62.5)

	type session struct {
		ExerciseID string `json:"exercise_id"`
		Sets       []struct {
			WeightKg float64 `json:"weight_kg"`
		} `json:"sets"`
	}
	var last []session
	alice.must(http.StatusOK, "GET", "/exercises/last", nil, &last)
	if len(last) != 1 || last[0].ExerciseID != ex.ID || len(last[0].Sets) != 1 || last[0].Sets[0].WeightKg != 62.5 {
		t.Errorf("last: got %+v, want the 62.5 kg set from the newest workout", last)
	}

	var hist []session
	alice.must(http.StatusOK, "GET", "/exercises/"+ex.ID+"/history", nil, &hist)
	if len(hist) != 2 || hist[0].Sets[0].WeightKg != 62.5 || hist[1].Sets[0].WeightKg != 60 {
		t.Errorf("history: got %+v, want 62.5 then 60 (newest first)", hist)
	}

	var bobLast []session
	bob.must(http.StatusOK, "GET", "/exercises/last", nil, &bobLast)
	if len(bobLast) != 0 {
		t.Errorf("Bob's last sets include Alice's: %+v", bobLast)
	}
}

// The workout's own exercise list: frozen from the template at start,
// added to before any set, appended to by logging, reordered, removed.
func TestWorkoutExerciseList(t *testing.T) {
	t.Parallel()
	h := newServer(t)
	alice := signUp(t, h)

	var a, b, c idOnly
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Alpha press"), &a)
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Bravo press"), &b)
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Charlie press"), &c)

	var tmpl idOnly
	alice.must(http.StatusCreated, "POST", "/templates", map[string]any{"name": "Plan", "exercises": []map[string]any{
		{"exercise_id": a.ID, "target_sets": 3, "target_reps_min": 8, "target_reps_max": 8},
		{"exercise_id": b.ID, "target_sets": 2},
	}}, &tmpl)
	var w idOnly
	alice.must(http.StatusCreated, "POST", "/templates/"+tmpl.ID+"/start", nil, &w)

	type entry struct {
		ExerciseID    string `json:"exercise_id"`
		TargetSets    *int   `json:"target_sets"`
		TargetRepsMin *int   `json:"target_reps_min"`
	}
	type detail struct {
		Exercises []entry `json:"exercises"`
		Sets      []struct {
			ExerciseID string `json:"exercise_id"`
		} `json:"sets"`
	}
	get := func() detail {
		var d detail
		alice.must(http.StatusOK, "GET", "/workouts/"+w.ID, nil, &d)
		return d
	}
	ids := func(d detail) []string {
		var out []string
		for _, e := range d.Exercises {
			out = append(out, e.ExerciseID)
		}
		return out
	}
	same := func(got, want []string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}

	// Started from the template: its plan, frozen. Editing the template
	// afterwards doesn't touch this workout.
	d := get()
	if !same(ids(d), []string{a.ID, b.ID}) || d.Exercises[0].TargetSets == nil || *d.Exercises[0].TargetSets != 3 ||
		d.Exercises[0].TargetRepsMin == nil || *d.Exercises[0].TargetRepsMin != 8 {
		t.Fatalf("after start: %+v, want the plan A(3x8), B", d.Exercises)
	}
	var tpl idOnly
	alice.must(http.StatusOK, "PUT", "/templates/"+tmpl.ID, map[string]any{"name": "Plan", "exercises": []map[string]any{
		{"exercise_id": c.ID, "target_sets": 5},
	}}, &tpl)
	if d := get(); !same(ids(d), []string{a.ID, b.ID}) {
		t.Errorf("template edit changed the workout: %v", ids(d))
	}

	// Added before any set; logging a set for an unlisted exercise appends it.
	alice.must(http.StatusNoContent, "POST", "/workouts/"+w.ID+"/exercises", map[string]any{"exercise_id": c.ID})
	alice.must(http.StatusNoContent, "POST", "/workouts/"+w.ID+"/exercises", map[string]any{"exercise_id": c.ID}) // idempotent
	var set idOnly
	alice.must(http.StatusCreated, "POST", "/workouts/"+w.ID+"/sets", map[string]any{"exercise_id": b.ID, "reps": 5, "weight_kg": 50}, &set)
	alice.must(http.StatusCreated, "POST", "/workouts/"+w.ID+"/sets", map[string]any{"exercise_id": c.ID, "reps": 5, "weight_kg": 60}, &set)
	if d := get(); !same(ids(d), []string{a.ID, b.ID, c.ID}) || d.Exercises[2].TargetSets != nil {
		t.Errorf("after add + logging: %+v, want A, B, C (C unplanned)", d.Exercises)
	}

	// Reorder: must be exactly the listed exercises.
	alice.must(http.StatusNoContent, "PUT", "/workouts/"+w.ID+"/exercises", map[string]any{"exercise_ids": []string{c.ID, a.ID, b.ID}})
	if d := get(); !same(ids(d), []string{c.ID, a.ID, b.ID}) {
		t.Errorf("after reorder: %v, want C, A, B", ids(d))
	}
	for _, bad := range [][]string{{c.ID, a.ID}, {c.ID, a.ID, a.ID}, {c.ID, a.ID, b.ID, b.ID}} {
		if code, _ := alice.do("PUT", "/workouts/"+w.ID+"/exercises", map[string]any{"exercise_ids": bad}); code != http.StatusBadRequest {
			t.Errorf("reorder %v: status %d, want 400", bad, code)
		}
	}

	// Save as template follows the workout's order (exercises with sets only).
	var saved struct {
		Exercises []struct {
			ExerciseID string `json:"exercise_id"`
		} `json:"exercises"`
	}
	alice.must(http.StatusCreated, "POST", "/workouts/"+w.ID+"/save-as-template", map[string]any{"name": "From workout"}, &saved)
	if len(saved.Exercises) != 2 || saved.Exercises[0].ExerciseID != c.ID || saved.Exercises[1].ExerciseID != b.ID {
		t.Errorf("saved template order %+v, want C then B", saved.Exercises)
	}

	// Remove takes the exercise and its sets; removing it again is 404.
	var removed struct {
		SetsDeleted int `json:"sets_deleted"`
	}
	alice.must(http.StatusOK, "DELETE", "/workouts/"+w.ID+"/exercises/"+c.ID, nil, &removed)
	if removed.SetsDeleted != 1 {
		t.Errorf("sets_deleted %d, want 1", removed.SetsDeleted)
	}
	d = get()
	if !same(ids(d), []string{a.ID, b.ID}) || len(d.Sets) != 1 || d.Sets[0].ExerciseID != b.ID {
		t.Errorf("after remove: exercises %v, sets %+v; want A, B and only B's set", ids(d), d.Sets)
	}
	if code, _ := alice.do("DELETE", "/workouts/"+w.ID+"/exercises/"+c.ID, nil); code != http.StatusNotFound {
		t.Errorf("removing again: status %d, want 404", code)
	}

	// Undoing the removal of a planned exercise brings its plan back;
	// targets are checked by the template rules.
	alice.must(http.StatusOK, "DELETE", "/workouts/"+w.ID+"/exercises/"+a.ID, nil, &removed)
	if code, _ := alice.do("POST", "/workouts/"+w.ID+"/exercises",
		map[string]any{"exercise_id": a.ID, "target_sets": 0}); code != http.StatusBadRequest {
		t.Errorf("target_sets 0: status %d, want 400", code)
	}
	alice.must(http.StatusNoContent, "POST", "/workouts/"+w.ID+"/exercises",
		map[string]any{"exercise_id": a.ID, "target_sets": 3, "target_reps_min": 8, "target_reps_max": 8})
	d = get()
	last := d.Exercises[len(d.Exercises)-1]
	if last.ExerciseID != a.ID || last.TargetSets == nil || *last.TargetSets != 3 || last.TargetRepsMin == nil || *last.TargetRepsMin != 8 {
		t.Errorf("restored A: %+v, want 3x8 at the end", last)
	}

	// A full undo in one call: plan, sets and position, all or nothing.
	alice.must(http.StatusOK, "DELETE", "/workouts/"+w.ID+"/exercises/"+b.ID, nil, &removed)
	before := get()
	bad := map[string]any{"exercise_id": b.ID, "target_sets": 2, "position": 0,
		"sets": []map[string]any{{"reps": 5, "weight_kg": 50}, {"weight_kg": 50}}} // second set lacks reps
	if code, _ := alice.do("POST", "/workouts/"+w.ID+"/exercises", bad); code != http.StatusBadRequest {
		t.Errorf("restore with an invalid set: status %d, want 400", code)
	}
	if after := get(); !same(ids(after), ids(before)) || len(after.Sets) != len(before.Sets) {
		t.Errorf("a failed restore changed the workout: %v / %d sets, was %v / %d", ids(after), len(after.Sets), ids(before), len(before.Sets))
	}
	alice.must(http.StatusNoContent, "POST", "/workouts/"+w.ID+"/exercises", map[string]any{
		"exercise_id": b.ID, "target_sets": 2, "position": 0,
		"sets": []map[string]any{{"reps": 5, "weight_kg": 50}, {"reps": 4, "weight_kg": 50}},
	})
	d = get()
	var bSets int
	for _, st := range d.Sets {
		if st.ExerciseID == b.ID {
			bSets++
		}
	}
	if d.Exercises[0].ExerciseID != b.ID || d.Exercises[0].TargetSets == nil || *d.Exercises[0].TargetSets != 2 || bSets != 2 {
		t.Errorf("full restore: first %+v with %d sets; want B first, 2 planned, 2 sets", d.Exercises[0], bSets)
	}
}
