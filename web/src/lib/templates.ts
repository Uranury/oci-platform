// Template targets as the editor handles them: raw input strings per
// exercise, turned into the API body on save. All targets but the set count
// are optional; there's no weight target (it's meant to go up). Which
// targets an exercise takes mirrors models.setRules on the server, which is
// what enforces them.
import { fmtDuration, fmtWeight, parseDuration } from "./format";
import type { TemplateExercise, TrackingType } from "./types";

export type TargetKey = "reps" | "duration" | "distance";

export const TARGETS: Record<TrackingType, TargetKey[]> = {
  weight_reps: ["reps"],
  bodyweight_reps: ["reps"],
  duration: ["duration"],
  distance: ["distance", "duration"],
};

export const TARGET_INPUT: Record<TargetKey, { label: string; unit: string; placeholder: string; inputmode: "numeric" | "decimal" }> = {
  reps: { label: "Reps", unit: "", placeholder: "8–12", inputmode: "numeric" },
  duration: { label: "Time", unit: "", placeholder: "1:30", inputmode: "numeric" },
  distance: { label: "Distance", unit: "m", placeholder: "400", inputmode: "decimal" },
};

export const MAX_SETS = 20; // models.MaxTemplateSets
export const MAX_NOTE = 200; // models.MaxTemplateExerciseNote
export const MAX_NAME = 100; // models.MaxTemplateNameLen

export interface DraftItem {
  key: number; // stable across reorders, for keyed lists and focus
  exercise_id: string;
  name: string;
  type: TrackingType;
  archived: boolean;
  sets: number | null; // null while the stepper's figure is cleared
  reps: string;
  duration: string;
  distance: string;
  notes: string;
}

let nextKey = 0;

export function draftItem(
  e: { id: string; name: string; tracking_type: TrackingType; archived?: boolean },
): DraftItem {
  return {
    key: nextKey++,
    exercise_id: e.id,
    name: e.name,
    type: e.tracking_type,
    archived: e.archived ?? false,
    sets: 3,
    reps: "",
    duration: "",
    distance: "",
    notes: "",
  };
}

export function toDraft(te: TemplateExercise): DraftItem {
  const lo = te.target_reps_min;
  const hi = te.target_reps_max;
  return {
    key: nextKey++,
    exercise_id: te.exercise_id,
    name: te.exercise_name,
    type: te.exercise_tracking_type,
    archived: te.exercise_archived,
    sets: te.target_sets,
    reps: lo == null ? "" : lo === hi || hi == null ? String(lo) : `${lo}–${hi}`,
    duration: te.target_duration_seconds == null ? "" : fmtDuration(te.target_duration_seconds),
    distance: te.target_distance_m == null ? "" : fmtWeight(te.target_distance_m),
    notes: te.notes ?? "",
  };
}

// "8" → [8, 8], "8-12" / "8–12" → [8, 12]; null if malformed.
export function parseReps(s: string): [number, number] | null {
  const m = s.trim().match(/^(\d+)(?:\s*[-–—]\s*(\d+))?$/);
  if (!m) return null;
  const lo = parseInt(m[1]!, 10);
  const hi = parseInt(m[2] ?? m[1]!, 10);
  return lo >= 1 && hi >= lo ? [lo, hi] : null;
}

// What's wrong with one target's text, or null if it's fine (empty is fine).
export function targetError(key: TargetKey, text: string): string | null {
  const t = text.trim();
  if (!t) return null;
  if (key === "reps") return parseReps(t) ? null : "Reps look like 8 or 8–12";
  if (key === "duration") {
    const s = parseDuration(t);
    return s != null && Number.isFinite(s) && s > 0 ? null : "Time looks like 1:30 or 90";
  }
  const n = Number(t.replace(",", "."));
  return Number.isFinite(n) && n > 0 ? null : "Distance is a number of metres";
}

export function itemErrors(it: DraftItem): Partial<Record<TargetKey | "sets", string>> {
  const out: Partial<Record<TargetKey | "sets", string>> = {};
  if (it.sets == null || !Number.isInteger(it.sets) || it.sets < 1 || it.sets > MAX_SETS) {
    out.sets = `Sets go from 1 to ${MAX_SETS}`;
  }
  for (const k of TARGETS[it.type]) {
    const e = targetError(k, it[k]);
    if (e) out[k] = e;
  }
  return out;
}

// The PUT/POST body for one item; call only when itemErrors is empty.
export function itemBody(it: DraftItem) {
  const keys = TARGETS[it.type];
  const has = (k: TargetKey) => keys.includes(k) && it[k].trim() !== "";
  const reps = has("reps") ? parseReps(it.reps) : null;
  return {
    exercise_id: it.exercise_id,
    target_sets: it.sets!,
    target_reps_min: reps ? reps[0] : null,
    target_reps_max: reps ? reps[1] : null,
    target_duration_seconds: has("duration") ? parseDuration(it.duration) : null,
    target_distance_m: has("distance") ? Number(it.distance.replace(",", ".")) : null,
    notes: it.notes.trim() || null,
  };
}
