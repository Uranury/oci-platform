package service

import (
	"context"
	"math"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/google/uuid"
)

const (
	// DefaultBodyweightLimit is how many entries a list returns unless asked.
	DefaultBodyweightLimit = 30
	// MaxBodyweightLimit is about three years of daily weigh-ins.
	MaxBodyweightLimit = 1000
)

// BodyweightService is the user's bodyweight log: one weight per calendar
// day, in kilograms. Every method is scoped to the user, in the SQL itself.
type BodyweightService struct {
	repo repo.Bodyweight
	now  func() time.Time
}

func NewBodyweightService(r repo.Bodyweight) *BodyweightService {
	return &BodyweightService{repo: r, now: time.Now}
}

// Log records the weight for day (YYYY-MM-DD), replacing that day's entry.
// The weight is kept to two decimals, as the column stores it.
func (s *BodyweightService) Log(ctx context.Context, userID uuid.UUID, day string, kg float64) (*models.BodyweightEntry, error) {
	d, err := models.ParseDay(day, s.now())
	if err != nil {
		return nil, err
	}
	if err := models.ValidateBodyweight(kg); err != nil {
		return nil, err
	}
	return s.repo.Upsert(ctx, userID, d, math.Round(kg*100)/100)
}

// List returns the latest entries, newest first; limit 0 means the default.
func (s *BodyweightService) List(ctx context.Context, userID uuid.UUID, limit int) ([]models.BodyweightEntry, error) {
	switch {
	case limit == 0:
		limit = DefaultBodyweightLimit
	case limit < 0 || limit > MaxBodyweightLimit:
		return nil, models.ValidationError("limit must be between 1 and 1000")
	}
	return s.repo.List(ctx, userID, limit)
}

// Delete removes the day's entry; repo.ErrNotFound if there is none.
func (s *BodyweightService) Delete(ctx context.Context, userID uuid.UUID, day string) error {
	d, err := time.Parse(models.DayLayout, day)
	if err != nil {
		return models.ValidationError("day must be a date (YYYY-MM-DD)")
	}
	return s.repo.Delete(ctx, userID, d)
}
