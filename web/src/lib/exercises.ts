import type { Exercise, ExerciseOptions, NewExercise, Option } from "./types";

// The display label for a category value; the value itself if unknown.
export function label(list: Option[], value: string): string {
  return list.find((o) => o.value === value)?.label ?? value;
}

// "Barbell · Chest"
export function exerciseMeta(options: ExerciseOptions, e: Exercise): string {
  return `${label(options.equipment, e.equipment)} · ${label(options.muscles, e.primary_muscle)}`;
}

// The editable fields of an exercise, for the edit form.
export function toNewExercise(e: Exercise): NewExercise {
  return {
    name: e.name,
    tracking_type: e.tracking_type,
    equipment: e.equipment,
    movement_pattern: e.movement_pattern,
    primary_muscle: e.primary_muscle,
    secondary_muscles: [...e.secondary_muscles],
    mechanic: e.mechanic,
  };
}
