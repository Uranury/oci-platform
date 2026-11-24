package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// exportHeader names the CSV columns: one row per set. Units are in the
// names so a spreadsheet needs no legend. workout_no numbers the workouts
// 1, 2, 3… in the file, so two sessions on the same day (or two unnamed
// "Workout"s) stay apart while every row still stands alone: blank
// separator rows would end the table for a spreadsheet's sort and filter.
var exportHeader = []string{
	"workout_no", "date", "time", "workout", "exercise", "tracking_type", "set",
	"reps", "weight_kg", "duration_s", "distance_m", "notes",
}

// ExportWorkouts sends the user's sets as a CSV download:
// GET /workouts/export?from=YYYY-MM-DD&to=YYYY-MM-DD&tz=<IANA zone>, every
// parameter optional (no from/to: all of it; no tz: UTC).
func (h *WorkoutHandler) ExportWorkouts(c *gin.Context) {
	userID := c.MustGet(userIDKey).(uuid.UUID)
	r, err := service.ParseExportRange(c.Query("from"), c.Query("to"), c.Query("tz"))
	if err != nil {
		_ = c.Error(err).SetMeta("export")
		return
	}
	sets, err := h.svc.ExportSets(c.Request.Context(), userID, r)
	if err != nil {
		_ = c.Error(err).SetMeta("export")
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+exportFilename(r)+`"`)
	c.Header("Cache-Control", "no-store") // training data, not for shared caches
	c.Status(http.StatusOK)
	if err := writeExportCSV(c.Writer, sets, r); err != nil {
		_ = c.Error(err) // headers are out; the log gets it
	}
}

func exportFilename(r service.ExportRange) string {
	switch {
	case r.FromDay == "" && r.ToDay == "":
		return "overlift-history.csv"
	case r.FromDay == "":
		return "overlift-to-" + r.ToDay + ".csv"
	case r.ToDay == "":
		return "overlift-from-" + r.FromDay + ".csv"
	}
	return fmt.Sprintf("overlift-%s-to-%s.csv", r.FromDay, r.ToDay)
}

func writeExportCSV(w io.Writer, sets []models.ExportSet, r service.ExportRange) error {
	// A UTF-8 byte order mark, so Excel doesn't read exercise names with
	// accents as Latin-1. Other spreadsheets and CSV readers skip it.
	if _, err := io.WriteString(w, "\xEF\xBB\xBF"); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(exportHeader); err != nil {
		return err
	}
	no := 0
	for i, s := range sets {
		if i == 0 || s.WorkoutID != sets[i-1].WorkoutID {
			no++
		}
		at := s.PerformedAt.In(r.Loc)
		if err := cw.Write([]string{
			strconv.Itoa(no),
			at.Format("2006-01-02"),
			at.Format("15:04"),
			cellText(workoutName(s.WorkoutName)),
			cellText(s.ExerciseName),
			string(s.ExerciseTrackingType),
			strconv.Itoa(s.SetNumber),
			intCell(s.Reps),
			floatCell(s.WeightKg),
			intCell(s.DurationSeconds),
			floatCell(s.DistanceM),
			cellText(deref(s.WorkoutNotes)),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// cellText guards user text against formula injection: a spreadsheet runs
// a cell starting with = + - @ (or a tab/CR before one) as a formula, so
// such a cell gets a leading apostrophe and is shown as text (OWASP's
// advice for CSV export).
func cellText(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// workoutName is the name as the app shows it: an unnamed workout (one not
// started from a template and never renamed) is "Workout", as on the home
// screen, so the column is never blank.
func workoutName(s *string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return "Workout"
	}
	return strings.TrimSpace(*s)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intCell(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

func floatCell(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}
