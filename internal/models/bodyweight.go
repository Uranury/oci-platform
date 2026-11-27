package models

import (
	"fmt"
	"time"
)

// BodyweightEntry is one day's weigh-in. Day is the user's own calendar day
// (YYYY-MM-DD, as the client sends it), so there is no time zone to get
// wrong; logging the same day again replaces the entry.
type BodyweightEntry struct {
	Day       string    `json:"day"`
	WeightKg  float64   `json:"weight_kg"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	// DayLayout is how calendar days travel in the API.
	DayLayout = "2006-01-02"

	MinBodyweightKg = 20.0
	MaxBodyweightKg = 500.0
)

// ParseDay reads a calendar day for the bodyweight log as midnight UTC. A
// day ahead of the server's is allowed by one, since the user's day can be
// ahead of UTC (up to UTC+14); later days are a mistake.
func ParseDay(s string, now time.Time) (time.Time, error) {
	d, err := time.Parse(DayLayout, s)
	if err != nil {
		return time.Time{}, ValidationError("day must be a date (YYYY-MM-DD)")
	}
	if d.Year() < 1900 {
		return time.Time{}, ValidationError("day is too far in the past")
	}
	n := now.UTC()
	if d.After(time.Date(n.Year(), n.Month(), n.Day()+1, 0, 0, 0, 0, time.UTC)) {
		return time.Time{}, ValidationError("day can't be in the future")
	}
	return d, nil
}

// ValidateBodyweight checks a weigh-in in kilograms.
func ValidateBodyweight(kg float64) error {
	if !(kg >= MinBodyweightKg && kg <= MaxBodyweightKg) { // NaN fails too
		return ValidationError(fmt.Sprintf("weight_kg must be between %g and %g", MinBodyweightKg, MaxBodyweightKg))
	}
	return nil
}
