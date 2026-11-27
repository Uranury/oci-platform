// Shapes of the Go API's JSON (internal/models). Pointer fields in Go are
// `T | null` here; fields tagged omitempty may also be missing.

export type TrackingType = "weight_reps" | "bodyweight_reps" | "duration" | "distance";

export interface WorkoutSet {
  id: string;
  workout_id: string;
  exercise_id: string;
  exercise_name: string;
  exercise_tracking_type: TrackingType;
  reps: number | null;
  weight_kg: number | null;
  duration_seconds: number | null;
  distance_m: number | null;
  created_at: string;
}

// The targets for one exercise: from a template, or frozen into a workout.
export interface Plan {
  target_sets: number;
  target_reps_min: number | null;
  target_reps_max: number | null;
  target_duration_seconds: number | null;
  target_distance_m: number | null;
  notes?: string | null;
}

// One entry in a workout's own exercise list, in display order. Targets are
// null for an exercise added during the workout.
export interface WorkoutExercise extends Omit<Plan, "target_sets"> {
  exercise_id: string;
  exercise_name: string;
  exercise_tracking_type: TrackingType;
  exercise_archived: boolean;
  position: number;
  target_sets: number | null;
}

export interface Workout {
  id: string;
  name?: string | null;
  notes?: string | null;
  performed_at: string;
  template_id?: string | null;
  finished_at?: string | null;
  exercises: WorkoutExercise[];
  sets: WorkoutSet[];
}

// GET /workouts: a workout without its sets, plus a summary of them.
export interface WorkoutSummary extends Omit<Workout, "sets" | "exercises"> {
  set_count: number;
  exercises: string[]; // names, in the order first logged
}

// GET /templates: most due first (never done, then longest since last use).
export interface TemplateSummary {
  id: string;
  name: string;
  notes?: string | null;
  exercise_count: number;
  exercises: string[]; // names, in plan order
  last_used_at?: string | null;
}

export interface Exercise {
  id: string;
  system: boolean;
  name: string;
  movement_pattern: string;
  primary_muscle: string;
  secondary_muscles: string[];
  mechanic: string;
  equipment: string;
  tracking_type: TrackingType;
  archived_at?: string | null;
}

export interface TemplateExercise {
  id: string;
  exercise_id: string;
  exercise_name: string;
  exercise_tracking_type: TrackingType;
  exercise_archived: boolean;
  position: number;
  target_sets: number;
  target_reps_min: number | null;
  target_reps_max: number | null;
  target_duration_seconds: number | null;
  target_distance_m: number | null;
  notes?: string | null;
}

export interface Template {
  id: string;
  name: string;
  notes?: string | null;
  exercises: TemplateExercise[];
}

// GET /workouts/{id}/previous: per exercise, the sets from the most recent
// earlier workout that had it.
export interface Previous {
  exercise_id: string;
  workout_id: string;
  performed_at: string;
  sets: WorkoutSet[];
}

export interface Option {
  value: string;
  label: string;
}

export interface ExerciseOptions {
  movement_patterns: Option[];
  muscles: Option[];
  mechanics: Option[];
  equipment: Option[];
  tracking_types: Option[];
}

export type Measure = "reps" | "weight_kg" | "duration_seconds" | "distance_m";

export interface SetBody {
  exercise_id: string;
  reps?: number;
  weight_kg?: number;
  duration_seconds?: number;
  distance_m?: number;
}

export interface NewExercise {
  name: string;
  tracking_type: string;
  equipment: string;
  movement_pattern: string;
  primary_muscle: string;
  secondary_muscles: string[];
  mechanic: string;
}

// One day's weigh-in; day is the user's calendar day (YYYY-MM-DD).
export interface BodyweightEntry {
  day: string;
  weight_kg: number;
  created_at: string;
  updated_at: string;
}
