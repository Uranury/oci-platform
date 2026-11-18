package service

import (
	"context"
	"errors"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo"
	"github.com/google/uuid"
)

var (
	ErrExerciseExists   = errors.New("service: exercise with that name already exists")
	ErrReadOnly         = errors.New("service: system exercises are read-only")
	ErrExerciseArchived = errors.New("service: exercise is archived")
)

// ExerciseService manages the exercise library: read-only system defaults
// plus each user's private custom exercises.
type ExerciseService struct {
	exercises repo.Exercise
}

func NewExerciseService(exercises repo.Exercise) *ExerciseService {
	return &ExerciseService{exercises: exercises}
}

// ExerciseInput is the user-editable part of an exercise.
type ExerciseInput struct {
	Name             string
	MovementPattern  models.MovementPattern
	PrimaryMuscle    models.Muscle
	SecondaryMuscles []models.Muscle
	Mechanic         models.Mechanic
	Equipment        models.Equipment
	TrackingType     models.TrackingType
}

func (in ExerciseInput) apply(e *models.Exercise) {
	e.Name = in.Name
	e.MovementPattern = in.MovementPattern
	e.PrimaryMuscle = in.PrimaryMuscle
	e.SecondaryMuscles = in.SecondaryMuscles
	e.Mechanic = in.Mechanic
	e.Equipment = in.Equipment
	e.TrackingType = in.TrackingType
}

func (s *ExerciseService) List(ctx context.Context, userID uuid.UUID) ([]models.Exercise, error) {
	return s.exercises.ListVisible(ctx, userID)
}

func (s *ExerciseService) Create(ctx context.Context, userID uuid.UUID, in ExerciseInput) (*models.Exercise, error) {
	e := &models.Exercise{UserID: &userID}
	in.apply(e)
	return e, s.create(ctx, e)
}

func (s *ExerciseService) create(ctx context.Context, e *models.Exercise) error {
	e.Normalize()
	if err := e.Validate(); err != nil {
		return err
	}
	if err := s.exercises.Create(ctx, e); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			return ErrExerciseExists
		}
		return err
	}
	return nil
}

// owned returns an exercise the user may modify: ErrNotFound if it isn't
// visible to them, ErrReadOnly if it's a system default.
func (s *ExerciseService) owned(ctx context.Context, userID, id uuid.UUID) (*models.Exercise, error) {
	e, err := s.exercises.GetVisible(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if e.UserID == nil {
		return nil, ErrReadOnly
	}
	return e, nil
}

func (s *ExerciseService) Update(ctx context.Context, userID, id uuid.UUID, in ExerciseInput) (*models.Exercise, error) {
	e, err := s.owned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	in.apply(e)
	e.Normalize()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.exercises.Update(ctx, e); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			return nil, ErrExerciseExists
		}
		return nil, err
	}
	return e, nil
}

// Delete removes a custom exercise. One with logged sets is archived
// instead (hidden from the picker, still resolving in history); archived
// reports which happened.
func (s *ExerciseService) Delete(ctx context.Context, userID, id uuid.UUID) (archived bool, err error) {
	if _, err := s.owned(ctx, userID, id); err != nil {
		return false, err
	}
	err = s.exercises.Delete(ctx, userID, id)
	if errors.Is(err, repo.ErrInUse) {
		return true, s.exercises.Archive(ctx, userID, id)
	}
	return false, err
}

// Duplicate copies a visible exercise (typically a system default) into an
// independent custom exercise owned by the user. name overrides the copy's
// name when non-empty.
func (s *ExerciseService) Duplicate(ctx context.Context, userID, id uuid.UUID, name string) (*models.Exercise, error) {
	src, err := s.exercises.GetVisible(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	cp := &models.Exercise{
		UserID:           &userID,
		Name:             src.Name,
		MovementPattern:  src.MovementPattern,
		PrimaryMuscle:    src.PrimaryMuscle,
		SecondaryMuscles: src.SecondaryMuscles,
		Mechanic:         src.Mechanic,
		Equipment:        src.Equipment,
		TrackingType:     src.TrackingType,
	}
	if name != "" {
		cp.Name = name
	}
	return cp, s.create(ctx, cp)
}

// SimilarDefaults returns system defaults whose names closely match name,
// so the client can warn before creating a near-duplicate. Advisory only.
func (s *ExerciseService) SimilarDefaults(ctx context.Context, name string) ([]models.Exercise, error) {
	defaults, err := s.exercises.ListSystem(ctx)
	if err != nil {
		return nil, err
	}
	out := []models.Exercise{}
	for _, d := range defaults {
		if similarNames(name, d.Name) {
			out = append(out, d)
		}
	}
	return out, nil
}
