package service

import "testing"

func TestSimilarNames(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Bench press", "Bench press", true},
		{"bench-press", "Bench Press", true},
		{"Benchpress", "Bench press", true},
		{"Bench pres", "Bench press", true}, // typo
		{"Pull-ups", "Pull-up", true},       // plural
		{"Pullup", "Pull-up", true},
		{"Bench", "Bench press", true}, // contained
		{"Squat", "Back squat", true},
		{"Incline bench press", "Bench press", true},
		{"Deadlfit", "Deadlift", true}, // transposition = 2 edits; allowed for 8+ chars
		{"Leg curl", "Dumbbell curl", false},
		{"Row", "Barbell row", false}, // too short to count as contained
		{"Dip", "Deadlift", false},
		{"Hammer curl", "Dumbbell curl", false},
		{"Face pull", "Pull-up", false},
		{"Press", "Bench press", true},
		{"", "Bench press", false},
		{"Glass", "Gla", false}, // "ss" isn't stripped as a plural; short
	}
	for _, c := range cases {
		if got := similarNames(c.a, c.b); got != c.want {
			t.Errorf("similarNames(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
