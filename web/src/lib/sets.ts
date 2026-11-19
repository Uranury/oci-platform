import { fmtDuration, fmtWeight } from "./format";
import type { Measure, Plan, SetBody, TrackingType, WorkoutSet } from "./types";

export interface Field {
  key: Measure;
  label: string; // stepper caption
  unit: string; // shown beside the figure
  required: boolean;
  step: number;
  min: number; // smallest value a stepper will go to
  integer: boolean;
}

const WEIGHT = (label: string, required: boolean): Field => ({
  key: "weight_kg", label, unit: "kg", required, step: 2.5, min: 0, integer: false,
});
const REPS: Field = { key: "reps", label: "Reps", unit: "reps", required: true, step: 1, min: 1, integer: true };

// Measurements a set takes per tracking type, in display order. Mirrors
// models.setRules on the server, which is what enforces them.
export const FIELDS: Record<TrackingType, Field[]> = {
  weight_reps: [WEIGHT("Weight", true), REPS],
  bodyweight_reps: [REPS, WEIGHT("Added", false)],
  duration: [
    { key: "duration_seconds", label: "Time", unit: "", required: true, step: 5, min: 5, integer: true },
    WEIGHT("Added", false),
  ],
  distance: [
    { key: "distance_m", label: "Distance", unit: "m", required: true, step: 50, min: 0, integer: false },
    WEIGHT("Weight", false),
  ],
};

export type Values = Partial<Record<Measure, number | null>>;

export function fmtValue(key: Measure, v: number | null | undefined): string {
  if (v == null) return "";
  if (key === "duration_seconds") return fmtDuration(v);
  if (key === "weight_kg" || key === "distance_m") return fmtWeight(v);
  return String(v);
}

// Starting values for the next set: the set being edited, else this
// workout's latest set of the exercise, else the matching set last time,
// else the plan's target.
export function prefill(
  type: TrackingType,
  { editing, logged, last, plan }: {
    editing?: WorkoutSet | null;
    logged: WorkoutSet[];
    last?: WorkoutSet[];
    plan?: Plan | null;
  },
): Values {
  const src: Values =
    editing ??
    logged[logged.length - 1] ??
    last?.[Math.min(logged.length, last.length - 1)] ??
    (plan
      ? {
          reps: plan.target_reps_min,
          duration_seconds: plan.target_duration_seconds,
          distance_m: plan.target_distance_m,
        }
      : {});
  const out: Values = {};
  for (const f of FIELDS[type]) out[f.key] = src[f.key] ?? null;
  return out;
}

// The request body, or the first problem with the values.
export function toBody(exerciseId: string, type: TrackingType, v: Values): SetBody | string {
  const body: SetBody = { exercise_id: exerciseId };
  for (const f of FIELDS[type]) {
    const x = v[f.key];
    if (x == null) {
      if (f.required) return `Enter ${f.label.toLowerCase()}`;
      continue;
    }
    if (!Number.isFinite(x) || x < 0 || (f.required && x <= 0)) return `Check ${f.label.toLowerCase()}`;
    if (f.integer && !Number.isInteger(x)) return `${f.label} must be a whole number`;
    body[f.key] = x;
  }
  return body;
}

// How far above everything seen recently a value must be before it's
// probably a typo: factor × the most, and at least min more than it.
const SUSPICIOUS: Record<Measure, { factor: number; min: number; unit: string }> = {
  weight_kg: { factor: 2, min: 20, unit: " kg" },
  reps: { factor: 3, min: 10, unit: " reps" },
  distance_m: { factor: 3, min: 500, unit: " m" },
  duration_seconds: { factor: 3, min: 120, unit: "" },
};

// "1000 kg? Your most here recently was 100 kg." or null.
export function suspicious(body: SetBody, history: WorkoutSet[]): string | null {
  for (const key of Object.keys(SUSPICIOUS) as Measure[]) {
    const r = SUSPICIOUS[key];
    const v = body[key];
    const seen = history.map((s) => s[key]).filter((x): x is number => x != null && x > 0);
    if (v == null || !seen.length) continue;
    const most = Math.max(...seen);
    if (v > most * r.factor && v - most >= r.min) {
      return `${fmtValue(key, v)}${r.unit}? Your most here recently was ${fmtValue(key, most)}${r.unit}.`;
    }
  }
  return null;
}

// The set as it reads on a chip: "80 × 8", "BW × 8", "+10 × 8", "1:30", "400 m".
export function chip(s: Pick<WorkoutSet, Measure | "exercise_tracking_type">): string {
  const w = s.weight_kg != null && s.weight_kg > 0 ? fmtWeight(s.weight_kg) : null;
  switch (s.exercise_tracking_type) {
    case "weight_reps":
      return `${w ?? 0} × ${s.reps ?? "?"}`;
    case "bodyweight_reps":
      return w ? `+${w} × ${s.reps ?? "?"}` : `BW × ${s.reps ?? "?"}`;
    case "duration":
      return s.duration_seconds != null ? fmtDuration(s.duration_seconds) + (w ? ` @${w}` : "") : "?";
    case "distance":
      return s.distance_m != null ? `${fmtWeight(s.distance_m)} m` : "?";
  }
}
