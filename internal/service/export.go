package service

import (
	"context"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/google/uuid"
)

// ExportRange is the part of the history to export: workouts performed in
// [From, To), with times written in Loc. FromDay and ToDay are the
// inclusive calendar days asked for, "" for an open end.
type ExportRange struct {
	From, To       time.Time
	FromDay, ToDay string
	Loc            *time.Location
}

const dayLayout = "2006-01-02"

// ParseExportRange reads the export's query: from and to are inclusive
// calendar days (YYYY-MM-DD) in the IANA time zone tz, each optional, so a
// day means the user's own day and not UTC's. tz "" means UTC.
func ParseExportRange(from, to, tz string) (ExportRange, error) {
	loc := time.UTC
	if tz != "" {
		l, err := time.LoadLocation(tz)
		if err != nil {
			return ExportRange{}, models.ValidationError("tz must be an IANA time zone such as Europe/Berlin")
		}
		loc = l
	}
	r := ExportRange{FromDay: from, ToDay: to, Loc: loc}
	// Beyond any date the app accepts, so open ends need no special query.
	r.From = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	r.To = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	if from != "" {
		d, err := time.ParseInLocation(dayLayout, from, loc)
		if err != nil {
			return ExportRange{}, models.ValidationError("from must be a date (YYYY-MM-DD)")
		}
		r.From = d
	}
	if to != "" {
		d, err := time.ParseInLocation(dayLayout, to, loc)
		if err != nil {
			return ExportRange{}, models.ValidationError("to must be a date (YYYY-MM-DD)")
		}
		r.To = d.AddDate(0, 0, 1) // the whole last day
	}
	if !r.From.Before(r.To) {
		return ExportRange{}, models.ValidationError("from must not be after to")
	}
	return r, nil
}

// ExportSets is every set the user logged in the range, oldest workout first.
func (s *WorkoutService) ExportSets(ctx context.Context, userID uuid.UUID, r ExportRange) ([]models.ExportSet, error) {
	return s.sets.ListExport(ctx, userID, r.From, r.To)
}
