package models_test

import (
	"math"
	"testing"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
)

func TestParseDay(t *testing.T) {
	// Late evening UTC: the user's day may already be tomorrow.
	now := time.Date(2026, 11, 28, 23, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		in string
		ok bool
	}{
		{"2026-11-28", true},
		{"2026-11-29", true},
		{"2026-11-30", false},
		{"2020-02-29", true},
		{"1899-12-31", false},
		{"2026-02-30", false},
		{"28.11.2026", false},
		{"", false},
	} {
		d, err := models.ParseDay(tc.in, now)
		if (err == nil) != tc.ok {
			t.Errorf("ParseDay(%q) err = %v, want ok=%v", tc.in, err, tc.ok)
		}
		if err == nil && d.Format(models.DayLayout) != tc.in {
			t.Errorf("ParseDay(%q) = %v", tc.in, d)
		}
	}
}

func TestValidateBodyweight(t *testing.T) {
	for _, tc := range []struct {
		kg float64
		ok bool
	}{
		{80.4, true}, {20, true}, {500, true}, {19.9, false}, {500.1, false}, {0, false}, {-70, false}, {math.NaN(), false},
	} {
		if err := models.ValidateBodyweight(tc.kg); (err == nil) != tc.ok {
			t.Errorf("ValidateBodyweight(%v) err = %v, want ok=%v", tc.kg, err, tc.ok)
		}
	}
}
