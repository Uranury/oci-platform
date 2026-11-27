package handler_test

import (
	"net/http"
	"testing"
	"time"
)

type entry struct {
	Day      string  `json:"day"`
	WeightKg float64 `json:"weight_kg"`
}

func TestBodyweight(t *testing.T) {
	t.Parallel()
	h := newServer(t)
	alice, bob := signUp(t, h), signUp(t, h)

	var list []entry
	alice.must(http.StatusOK, "GET", "/bodyweight", nil, &list)
	if list == nil || len(list) != 0 {
		t.Fatalf("new user: got %v, want []", list)
	}

	today := time.Now().UTC().Format("2006-01-02")
	var e entry
	alice.must(http.StatusOK, "PUT", "/bodyweight/2026-01-10", map[string]any{"weight_kg": 82.456}, &e)
	if e.Day != "2026-01-10" || e.WeightKg != 82.46 {
		t.Fatalf("logged %+v", e)
	}
	alice.must(http.StatusOK, "PUT", "/bodyweight/"+today, map[string]any{"weight_kg": 81})
	// The same day again replaces it.
	alice.must(http.StatusOK, "PUT", "/bodyweight/"+today, map[string]any{"weight_kg": 80.6})

	alice.must(http.StatusOK, "GET", "/bodyweight", nil, &list)
	if len(list) != 2 || list[0] != (entry{today, 80.6}) || list[1] != (entry{"2026-01-10", 82.46}) {
		t.Fatalf("list = %+v", list)
	}
	alice.must(http.StatusOK, "GET", "/bodyweight?limit=1", nil, &list)
	if len(list) != 1 || list[0].Day != today {
		t.Fatalf("limit=1: %+v", list)
	}

	for _, tc := range []struct {
		path string
		body any
	}{
		{"/bodyweight/2026-13-01", map[string]any{"weight_kg": 80}},
		{"/bodyweight/tomorrowish", map[string]any{"weight_kg": 80}},
		{"/bodyweight/2999-01-01", map[string]any{"weight_kg": 80}},
		{"/bodyweight/" + today, map[string]any{"weight_kg": 5}},
		{"/bodyweight/" + today, map[string]any{"weight_kg": 900}},
		{"/bodyweight/" + today, map[string]any{}},
		{"/bodyweight/" + today, map[string]any{"weight_kg": "80"}},
	} {
		if code, resp := alice.do("PUT", tc.path, tc.body); code != http.StatusBadRequest {
			t.Errorf("PUT %s %v: got %d %s, want 400", tc.path, tc.body, code, resp)
		}
	}
	for _, q := range []string{"limit=0x", "limit=-1", "limit=5000"} {
		if code, _ := alice.do("GET", "/bodyweight?"+q, nil); code != http.StatusBadRequest {
			t.Errorf("GET ?%s: got %d, want 400", q, code)
		}
	}

	// Another user sees none of it and can't delete it.
	bob.must(http.StatusOK, "GET", "/bodyweight", nil, &list)
	if len(list) != 0 {
		t.Fatalf("bob sees %+v", list)
	}
	bob.must(http.StatusNotFound, "DELETE", "/bodyweight/"+today, nil)

	alice.must(http.StatusNoContent, "DELETE", "/bodyweight/"+today, nil)
	alice.must(http.StatusNotFound, "DELETE", "/bodyweight/"+today, nil)
	alice.must(http.StatusOK, "GET", "/bodyweight", nil, &list)
	if len(list) != 1 || list[0].Day != "2026-01-10" {
		t.Fatalf("after delete: %+v", list)
	}
}
