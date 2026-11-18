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
		handler.NewWorkoutHandler(service.NewWorkoutService(exercises, workouts, sets)),
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

		{"GET", w, nil},
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
