package repo

import (
	"context"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

// WorkoutExercise is a workout's own ordered exercise list. Callers check
// that the workout belongs to the user; these methods only scope by workout.
type WorkoutExercise interface {
	// List returns the workout's exercises in display order.
	List(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutExercise, error)
	// Append adds the exercise at the end, with plan's targets and notes when
	// plan isn't nil; a no-op if it's already listed.
	Append(ctx context.Context, workoutID, exerciseID uuid.UUID, plan *models.TemplateExercise) error
	// CopyTemplate freezes the template's plan into the workout.
	CopyTemplate(ctx context.Context, workoutID, templateID uuid.UUID) error
	// SetPosition moves one exercise; ErrNotFound if it isn't listed.
	SetPosition(ctx context.Context, workoutID, exerciseID uuid.UUID, position int) error
	// Remove takes the exercise out of the workout together with its sets,
	// returning how many sets went; ErrNotFound if it was neither listed
	// nor logged.
	Remove(ctx context.Context, workoutID, exerciseID uuid.UUID) (setsDeleted int64, err error)
}

type workoutExercise struct {
	q *dbgen.Queries
}

func NewWorkoutExercise(exec dbutil.Executor) WorkoutExercise {
	return &workoutExercise{q: dbgen.New(exec)}
}

func (r *workoutExercise) List(ctx context.Context, workoutID uuid.UUID) ([]models.WorkoutExercise, error) {
	rows, err := r.q.ListWorkoutExercises(ctx, workoutID)
	if err != nil {
		return nil, translate(err)
	}
	out := make([]models.WorkoutExercise, len(rows))
	for i, row := range rows {
		we := row.WorkoutExercise
		out[i] = models.WorkoutExercise{
			ExerciseID: we.ExerciseID, ExerciseName: row.ExerciseName,
			ExerciseTrackingType: row.ExerciseTrackingType,
			ExerciseArchived:     row.ExerciseArchived, Position: we.Position,
			TargetSets: we.TargetSets, TargetRepsMin: we.TargetRepsMin, TargetRepsMax: we.TargetRepsMax,
			TargetDurationSeconds: we.TargetDurationSeconds, TargetDistanceM: we.TargetDistanceM,
			Notes: we.Notes,
		}
	}
	return out, nil
}

func (r *workoutExercise) Append(ctx context.Context, workoutID, exerciseID uuid.UUID, plan *models.TemplateExercise) error {
	p := dbgen.AppendWorkoutExerciseParams{WorkoutID: workoutID, ExerciseID: exerciseID}
	if plan != nil {
		p.TargetSets, p.TargetRepsMin, p.TargetRepsMax = &plan.TargetSets, plan.TargetRepsMin, plan.TargetRepsMax
		p.TargetDurationSeconds, p.TargetDistanceM, p.Notes = plan.TargetDurationSeconds, plan.TargetDistanceM, plan.Notes
	}
	_, err := r.q.AppendWorkoutExercise(ctx, p)
	return translate(err)
}

func (r *workoutExercise) CopyTemplate(ctx context.Context, workoutID, templateID uuid.UUID) error {
	return translate(r.q.CopyTemplateIntoWorkout(ctx, dbgen.CopyTemplateIntoWorkoutParams{
		WorkoutID: workoutID, TemplateID: templateID,
	}))
}

func (r *workoutExercise) SetPosition(ctx context.Context, workoutID, exerciseID uuid.UUID, position int) error {
	return affected(r.q.SetWorkoutExercisePosition(ctx, dbgen.SetWorkoutExercisePositionParams{
		WorkoutID: workoutID, ExerciseID: exerciseID, Position: position,
	}))
}

func (r *workoutExercise) Remove(ctx context.Context, workoutID, exerciseID uuid.UUID) (int64, error) {
	sets, err := r.q.DeleteWorkoutExerciseSets(ctx, dbgen.DeleteWorkoutExerciseSetsParams{
		WorkoutID: workoutID, ExerciseID: exerciseID,
	})
	if err != nil {
		return 0, translate(err)
	}
	listed, err := r.q.DeleteWorkoutExercise(ctx, dbgen.DeleteWorkoutExerciseParams{
		WorkoutID: workoutID, ExerciseID: exerciseID,
	})
	if err != nil {
		return 0, translate(err)
	}
	if sets == 0 && listed == 0 {
		return 0, ErrNotFound
	}
	return sets, nil
}
