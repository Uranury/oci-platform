package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/google/uuid"
)

// fakeWorkouts is an in-memory repo.Workout for the finish/duration rules,
// which don't touch sets or exercises.
type fakeWorkouts map[uuid.UUID]models.Workout

func (f fakeWorkouts) Create(_ context.Context, w *models.Workout) error {
	w.ID = uuid.New()
	f[w.ID] = *w
	return nil
}

func (f fakeWorkouts) GetByID(_ context.Context, id uuid.UUID) (*models.Workout, error) {
	w, ok := f[id]
	if !ok {
		return nil, repo.ErrNotFound
	}
	return &w, nil
}

func (f fakeWorkouts) ListByUser(context.Context, uuid.UUID) ([]models.WorkoutSummary, error) {
	return nil, nil
}

func (f fakeWorkouts) Update(_ context.Context, w *models.Workout) error {
	f[w.ID] = *w
	return nil
}

func (f fakeWorkouts) Delete(_ context.Context, id uuid.UUID) error {
	delete(f, id)
	return nil
}

func newWorkout(t *testing.T, store fakeWorkouts, userID uuid.UUID, performedAt time.Time, finishedAt *time.Time) uuid.UUID {
	t.Helper()
	w := &models.Workout{UserID: userID, PerformedAt: performedAt, FinishedAt: finishedAt}
	if err := store.Create(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	return w.ID
}

func TestFinishWorkout(t *testing.T) {
	ctx := context.Background()
	user := uuid.New()
	now := time.Now()
	earlier := now.Add(-30 * time.Minute)

	tests := []struct {
		name        string
		performedAt time.Time
		finishedAt  *time.Time
		wantSet     bool      // finished_at should be set to about now
		wantKept    time.Time // or kept at this value (zero = expect nil)
	}{
		{"started 45 min ago", now.Add(-45 * time.Minute), nil, true, time.Time{}},
		{"logged after the fact", now.Add(-26 * time.Hour), nil, false, time.Time{}},
		{"dated in the future", now.Add(time.Hour), nil, false, time.Time{}},
		{"already finished", now.Add(-time.Hour), &earlier, false, earlier},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := fakeWorkouts{}
			svc := NewWorkoutService(nil, store, nil)
			id := newWorkout(t, store, user, tc.performedAt, tc.finishedAt)

			w, err := svc.FinishWorkout(ctx, user, id)
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.wantSet:
				if w.FinishedAt == nil || time.Since(*w.FinishedAt) > time.Minute {
					t.Fatalf("finished_at = %v, want about now", w.FinishedAt)
				}
			case !tc.wantKept.IsZero():
				if w.FinishedAt == nil || !w.FinishedAt.Equal(tc.wantKept) {
					t.Fatalf("finished_at = %v, want unchanged %v", w.FinishedAt, tc.wantKept)
				}
			default:
				if w.FinishedAt != nil {
					t.Fatalf("finished_at = %v, want nil (unknown duration)", w.FinishedAt)
				}
			}
			if stored := store[id]; (stored.FinishedAt == nil) != (w.FinishedAt == nil) {
				t.Fatalf("returned and stored finished_at differ: %v vs %v", w.FinishedAt, stored.FinishedAt)
			}
		})
	}
}

func TestFinishWorkoutOtherUser(t *testing.T) {
	store := fakeWorkouts{}
	svc := NewWorkoutService(nil, store, nil)
	id := newWorkout(t, store, uuid.New(), time.Now().Add(-time.Hour), nil)

	if _, err := svc.FinishWorkout(context.Background(), uuid.New(), id); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
	if store[id].FinishedAt != nil {
		t.Fatal("another user's workout was finished")
	}
}

func TestUpdateWorkoutKeepsDuration(t *testing.T) {
	ctx := context.Background()
	user := uuid.New()
	start := time.Date(2026, 11, 18, 18, 0, 0, 0, time.UTC)
	end := start.Add(52 * time.Minute)
	store := fakeWorkouts{}
	svc := NewWorkoutService(nil, store, nil)
	id := newWorkout(t, store, user, start, &end)

	// Moving the date a day back moves the end with it.
	newStart := start.Add(-24 * time.Hour)
	w, err := svc.UpdateWorkout(ctx, user, id, nil, nil, newStart)
	if err != nil {
		t.Fatal(err)
	}
	if !w.PerformedAt.Equal(newStart) || w.FinishedAt == nil || w.FinishedAt.Sub(w.PerformedAt) != 52*time.Minute {
		t.Fatalf("got %v → %v, want %v with 52 min duration", w.PerformedAt, w.FinishedAt, newStart)
	}

	// No new date: both times stay.
	w, err = svc.UpdateWorkout(ctx, user, id, nil, nil, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if !w.PerformedAt.Equal(newStart) || w.FinishedAt.Sub(w.PerformedAt) != 52*time.Minute {
		t.Fatalf("times changed without a new date: %v → %v", w.PerformedAt, w.FinishedAt)
	}
}
