package handler_test

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// exportRows fetches the CSV export and returns its rows, header included.
func exportRows(t *testing.T, c *client, query string) [][]string {
	t.Helper()
	code, body := c.do("GET", "/workouts/export"+query, nil)
	if code != http.StatusOK {
		t.Fatalf("export%s: got %d: %s", query, code, body)
	}
	text, ok := strings.CutPrefix(string(body), "\xEF\xBB\xBF")
	if !ok {
		t.Fatalf("export%s: no UTF-8 byte order mark", query)
	}
	rows, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil {
		t.Fatalf("export%s: %v", query, err)
	}
	return rows
}

func TestExportWorkouts(t *testing.T) {
	t.Parallel()
	h := newServer(t)
	alice := signUp(t, h)
	bob := signUp(t, h)

	var press, curl idOnly
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("=Press"), &press) // formula-like name
	alice.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Curl"), &curl)

	// 23:30 UTC on 1 March is already 2 March in Berlin (UTC+1).
	var late, early, unnamed idOnly
	alice.must(http.StatusCreated, "POST", "/workouts",
		map[string]any{"name": "Late", "notes": "felt, good", "performed_at": "2026-03-01T23:30:00Z"}, &late)
	alice.must(http.StatusCreated, "POST", "/workouts",
		map[string]any{"name": "Early", "performed_at": "2026-02-10T09:00:00Z"}, &early)
	// No name, as for a workout started empty: exported as the app shows it.
	alice.must(http.StatusCreated, "POST", "/workouts", map[string]any{"performed_at": "2026-04-05T07:15:00Z"}, &unnamed)
	for _, s := range []struct{ w, ex string }{{late.ID, curl.ID}, {late.ID, press.ID}, {late.ID, curl.ID}, {early.ID, press.ID}, {unnamed.ID, curl.ID}} {
		alice.must(http.StatusCreated, "POST", "/workouts/"+s.w+"/sets",
			map[string]any{"exercise_id": s.ex, "reps": 5, "weight_kg": 62.5}, &idOnly{})
	}

	// Bob's sets never show up in Alice's export.
	var bw idOnly
	bob.must(http.StatusCreated, "POST", "/workouts", map[string]any{"name": "Bob's", "performed_at": "2026-03-01T10:00:00Z"}, &bw)
	var bex idOnly
	bob.must(http.StatusCreated, "POST", "/exercises", exerciseBody("Bob press"), &bex)
	bob.must(http.StatusCreated, "POST", "/workouts/"+bw.ID+"/sets", map[string]any{"exercise_id": bex.ID, "reps": 1, "weight_kg": 1}, &idOnly{})

	all := exportRows(t, alice, "")
	want := [][]string{
		{"workout_no", "date", "time", "workout", "exercise", "tracking_type", "set", "reps", "weight_kg", "duration_s", "distance_m", "notes"},
		{"1", "2026-02-10", "09:00", "Early", "'=Press", "weight_reps", "1", "5", "62.5", "", "", ""},
		// The workout's own exercise order (Curl was logged first), sets
		// numbered per exercise.
		{"2", "2026-03-01", "23:30", "Late", "Curl", "weight_reps", "1", "5", "62.5", "", "", "felt, good"},
		{"2", "2026-03-01", "23:30", "Late", "Curl", "weight_reps", "2", "5", "62.5", "", "", "felt, good"},
		{"2", "2026-03-01", "23:30", "Late", "'=Press", "weight_reps", "1", "5", "62.5", "", "", "felt, good"},
		{"3", "2026-04-05", "07:15", "Workout", "Curl", "weight_reps", "1", "5", "62.5", "", "", ""},
	}
	if !equalRows(all, want) {
		t.Fatalf("all of it:\n got %q\nwant %q", all, want)
	}

	// Days are the user's own: in Berlin the late workout is on 2 March.
	berlin := exportRows(t, alice, "?from=2026-03-02&to=2026-03-02&tz=Europe/Berlin")
	if len(berlin) != 4 || berlin[1][0] != "1" || berlin[1][1] != "2026-03-02" || berlin[1][2] != "00:30" {
		t.Fatalf("Berlin 2 March: %q", berlin)
	}
	if got := exportRows(t, alice, "?from=2026-03-02&to=2026-03-02"); len(got) != 1 {
		t.Fatalf("UTC 2 March should be empty, got %q", got)
	}
	if got := exportRows(t, alice, "?to=2026-02-28"); len(got) != 2 {
		t.Fatalf("up to February: %q", got)
	}

	code, body := alice.do("GET", "/workouts/export?from=2026-03-01", nil)
	if code != http.StatusOK {
		t.Fatalf("open end: %d %s", code, body)
	}
	req := httptest.NewRequest("GET", "/workouts/export?from=2026-01-01&to=2026-03-31", nil)
	req.Header.Set("Authorization", "Bearer "+alice.token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := rec.Header()
	if r.Get("Content-Type") != "text/csv; charset=utf-8" ||
		r.Get("Content-Disposition") != `attachment; filename="overlift-2026-01-01-to-2026-03-31.csv"` ||
		r.Get("Cache-Control") != "no-store" {
		t.Fatalf("headers: %v", r)
	}

	for _, q := range []string{"?from=2026-13-01", "?to=yesterday", "?tz=Mars/Olympus", "?from=2026-03-02&to=2026-03-01"} {
		if code, body := alice.do("GET", "/workouts/export"+q, nil); code != http.StatusBadRequest {
			t.Errorf("export%s: got %d, want 400: %s", q, code, body)
		}
	}
	if code, _ := (&client{t: t, h: h}).do("GET", "/workouts/export", nil); code != http.StatusUnauthorized {
		t.Errorf("logged out: got %d, want 401", code)
	}
}

func equalRows(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.Join(a[i], "\x00") != strings.Join(b[i], "\x00") {
			return false
		}
	}
	return true
}
