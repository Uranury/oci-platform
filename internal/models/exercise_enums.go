package models

import (
	"fmt"
	"slices"
)

// The exercise categories below are the single definition of every allowed
// value: validation, the GET /exercise-options endpoint (which feeds the
// frontend dropdowns) and the seed data all read from here. Users can only
// pick from these lists, never add to them.
//
// The database enforces the same lists with CHECK constraints (see the
// migrations); TestEnumsMatchMigrations fails if the two ever drift apart,
// so adding a value means adding it here AND in a new migration.

type (
	MovementPattern string
	Muscle          string
	Mechanic        string
	Equipment       string
	TrackingType    string
)

const (
	HorizontalPush MovementPattern = "horizontal_push"
	VerticalPush   MovementPattern = "vertical_push"
	HorizontalPull MovementPattern = "horizontal_pull"
	VerticalPull   MovementPattern = "vertical_pull"
	Squat          MovementPattern = "squat"
	Hinge          MovementPattern = "hinge"
	Lunge          MovementPattern = "lunge"
	Carry          MovementPattern = "carry"
	Core           MovementPattern = "core"
	OtherPattern   MovementPattern = "other"
)

const (
	Chest       Muscle = "chest"
	Lats        Muscle = "lats"
	UpperBack   Muscle = "upper_back"
	Shoulders   Muscle = "shoulders"
	Biceps      Muscle = "biceps"
	Triceps     Muscle = "triceps"
	Forearms    Muscle = "forearms"
	Quads       Muscle = "quads"
	Hamstrings  Muscle = "hamstrings"
	Glutes      Muscle = "glutes"
	Calves      Muscle = "calves"
	Abs         Muscle = "abs"
	OtherMuscle Muscle = "other"
)

const (
	Compound  Mechanic = "compound"
	Isolation Mechanic = "isolation"
)

const (
	Barbell        Equipment = "barbell"
	Dumbbell       Equipment = "dumbbell"
	Machine        Equipment = "machine"
	Cable          Equipment = "cable"
	Bodyweight     Equipment = "bodyweight"
	Kettlebell     Equipment = "kettlebell"
	Band           Equipment = "band"
	OtherEquipment Equipment = "other"
)

const (
	// WeightReps: weight and reps, both required.
	WeightReps TrackingType = "weight_reps"
	// BodyweightReps: reps required, weight is optional added load.
	BodyweightReps TrackingType = "bodyweight_reps"
	// Duration: time held/performed; weight optional (e.g. weighted plank).
	Duration TrackingType = "duration"
	// Distance: distance covered; weight and time optional (e.g. carries).
	Distance TrackingType = "distance"
)

// Option is one selectable value with its display label.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Enum is an ordered, closed set of values with labels.
type Enum[T ~string] struct {
	values []T
	labels map[T]string
}

type entry[T ~string] struct {
	value T
	label string
}

func newEnum[T ~string](entries []entry[T]) Enum[T] {
	e := Enum[T]{labels: map[T]string{}}
	for _, en := range entries {
		e.values = append(e.values, en.value)
		e.labels[en.value] = en.label
	}
	return e
}

func (e Enum[T]) Valid(v T) bool { _, ok := e.labels[v]; return ok }

func (e Enum[T]) Values() []T { return slices.Clone(e.values) }

func (e Enum[T]) Options() []Option {
	out := make([]Option, len(e.values))
	for i, v := range e.values {
		out[i] = Option{Value: string(v), Label: e.labels[v]}
	}
	return out
}

func (e Enum[T]) check(field string, v T) error {
	if !e.Valid(v) {
		return ValidationError(fmt.Sprintf("%s: %q is not one of the allowed values", field, v))
	}
	return nil
}

var (
	MovementPatterns = newEnum([]entry[MovementPattern]{
		{HorizontalPush, "Horizontal push"},
		{VerticalPush, "Vertical push"},
		{HorizontalPull, "Horizontal pull"},
		{VerticalPull, "Vertical pull"},
		{Squat, "Squat"},
		{Hinge, "Hinge"},
		{Lunge, "Lunge"},
		{Carry, "Carry"},
		{Core, "Core"},
		{OtherPattern, "Other"},
	})
	Muscles = newEnum([]entry[Muscle]{
		{Chest, "Chest"},
		{Lats, "Lats"},
		{UpperBack, "Upper back"},
		{Shoulders, "Shoulders"},
		{Biceps, "Biceps"},
		{Triceps, "Triceps"},
		{Forearms, "Forearms"},
		{Quads, "Quads"},
		{Hamstrings, "Hamstrings"},
		{Glutes, "Glutes"},
		{Calves, "Calves"},
		{Abs, "Abs"},
		{OtherMuscle, "Other"},
	})
	Mechanics = newEnum([]entry[Mechanic]{
		{Compound, "Compound"},
		{Isolation, "Isolation"},
	})
	Equipments = newEnum([]entry[Equipment]{
		{Barbell, "Barbell"},
		{Dumbbell, "Dumbbell"},
		{Machine, "Machine"},
		{Cable, "Cable"},
		{Bodyweight, "Bodyweight"},
		{Kettlebell, "Kettlebell"},
		{Band, "Band"},
		{OtherEquipment, "Other"},
	})
	TrackingTypes = newEnum([]entry[TrackingType]{
		{WeightReps, "Weight × reps"},
		{BodyweightReps, "Bodyweight reps (+ optional weight)"},
		{Duration, "Duration"},
		{Distance, "Distance"},
	})
)

// ExerciseOptions is the GET /exercise-options payload.
type ExerciseOptions struct {
	MovementPatterns []Option `json:"movement_patterns"`
	Muscles          []Option `json:"muscles"`
	Mechanics        []Option `json:"mechanics"`
	Equipment        []Option `json:"equipment"`
	TrackingTypes    []Option `json:"tracking_types"`
}

func AllExerciseOptions() ExerciseOptions {
	return ExerciseOptions{
		MovementPatterns: MovementPatterns.Options(),
		Muscles:          Muscles.Options(),
		Mechanics:        Mechanics.Options(),
		Equipment:        Equipments.Options(),
		TrackingTypes:    TrackingTypes.Options(),
	}
}

// ValidationError is a client-input problem; handlers return it as a 400
// with its message.
type ValidationError string

func (e ValidationError) Error() string { return string(e) }
