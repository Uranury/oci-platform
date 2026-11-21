// Template targets as the editor handles them: raw input strings per
// exercise, turned into the API body on save. All targets but the set count
// are optional; there's no weight target (it's meant to go up). Which
// targets an exercise takes mirrors models.setRules on the server, which is
// what enforces them.
import { fmtWeight } from "./format";
import type { TemplateExercise, TrackingType } from "./types";

export type TargetKey = "reps" | "duration" | "distance";

export const TARGETS: Record<TrackingType, TargetKey[]> = {
  weight_reps: ["reps"],
  bodyweight_reps: ["reps"],
  duration: ["duration"],
  distance: ["distance", "duration"],
};

// Each target is typed as one or two plain numbers, each in its own box, so a
// phone shows its number pad (which has no "-" or ":") and either number can
// be edited on its own. `sep` is printed between the two boxes.
export interface TargetPart {
  label: string; // the box's accessible name
  placeholder: string;
  maxlength: number;
}
export const TARGET_INPUT: Record<
  TargetKey,
  { label: string; unit: string; sep: string; inputmode: "numeric" | "decimal"; parts: TargetPart[] }
> = {
  reps: {
    label: "Reps",
    unit: "",
    sep: "–",
    inputmode: "numeric",
    parts: [
      { label: "Reps from", placeholder: "8", maxlength: 3 },
      { label: "Reps to (optional)", placeholder: "12", maxlength: 3 },
    ],
  },
  duration: {
    label: "Time",
    unit: "",
    sep: ":",
    inputmode: "numeric",
    parts: [
      { label: "Minutes", placeholder: "1", maxlength: 3 },
      { label: "Seconds", placeholder: "30", maxlength: 2 },
    ],
  },
  distance: {
    label: "Distance",
    unit: "m",
    sep: "",
    inputmode: "decimal",
    parts: [{ label: "Distance in metres", placeholder: "400", maxlength: 7 }],
  },
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
  // Raw box contents, one string per TARGET_INPUT part.
  reps: [string, string]; // from, to
  duration: [string, string]; // minutes, seconds
  distance: [string];
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
    reps: ["", ""],
    duration: ["", ""],
    distance: [""],
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
    reps: lo == null ? ["", ""] : [String(lo), hi == null || hi === lo ? "" : String(hi)],
    duration: durationParts(te.target_duration_seconds),
    distance: [te.target_distance_m == null ? "" : fmtWeight(te.target_distance_m)],
    notes: te.notes ?? "",
  };
}

function durationParts(sec: number | null): [string, string] {
  if (sec == null) return ["", ""];
  return [String(Math.floor(sec / 60)), String(sec % 60).padStart(2, "0")];
}

const whole = (t: string) => (/^\d+$/.test(t) ? parseInt(t, 10) : NaN);

// Reps from the two boxes: "8" + "" → [8, 8], "8" + "12" → [8, 12], both
// empty → null (no target). A lone "to" counts as a single figure.
export function parseReps([from, to]: [string, string]): [number, number] | null | "bad" {
  const a = from.trim(), b = to.trim();
  if (!a && !b) return null;
  const lo = whole(a || b);
  const hi = whole(b || a);
  return lo >= 1 && hi >= lo ? [lo, hi] : "bad";
}

// Seconds from the minutes and seconds boxes; null if both are empty. Seconds
// alone may run past 59 ("90"); with minutes they're the clock's 0–59.
export function parseDurationParts([min, sec]: [string, string]): number | null | "bad" {
  const m = min.trim(), s = sec.trim();
  if (!m && !s) return null;
  const mm = m ? whole(m) : 0;
  const ss = s ? whole(s) : 0;
  if (!Number.isFinite(mm) || !Number.isFinite(ss) || (m && ss > 59)) return "bad";
  const total = mm * 60 + ss;
  return total > 0 ? total : "bad";
}

function parseDistance([v]: [string]): number | null | "bad" {
  const t = v.trim();
  if (!t) return null;
  const n = Number(t.replace(",", "."));
  return Number.isFinite(n) && n > 0 ? n : "bad";
}

// What's wrong with one target's boxes, or null if they're fine (empty is fine).
export function targetError(it: DraftItem, key: TargetKey): string | null {
  if (key === "reps") {
    const r = parseReps(it.reps);
    if (r !== "bad") return null;
    const lo = whole(it.reps[0].trim()), hi = whole(it.reps[1].trim());
    return lo >= 1 && hi >= 1 && hi < lo ? "Reps go low to high, like 8 to 12" : "Reps are whole numbers from 1";
  }
  if (key === "duration") return parseDurationParts(it.duration) === "bad" ? "Time is minutes and seconds (0–59), like 1:30" : null;
  return parseDistance(it.distance) === "bad" ? "Distance is a number of metres" : null;
}

export function itemErrors(it: DraftItem): Partial<Record<TargetKey | "sets", string>> {
  const out: Partial<Record<TargetKey | "sets", string>> = {};
  if (it.sets == null || !Number.isInteger(it.sets) || it.sets < 1 || it.sets > MAX_SETS) {
    out.sets = `Sets go from 1 to ${MAX_SETS}`;
  }
  for (const k of TARGETS[it.type]) {
    const e = targetError(it, k);
    if (e) out[k] = e;
  }
  return out;
}

// The PUT/POST body for one item; call only when itemErrors is empty.
export function itemBody(it: DraftItem) {
  const keys = TARGETS[it.type];
  const ok = <T,>(v: T | "bad"): T | null => (v === "bad" ? null : v);
  const reps = keys.includes("reps") ? ok(parseReps(it.reps)) : null;
  return {
    exercise_id: it.exercise_id,
    target_sets: it.sets!,
    target_reps_min: reps ? reps[0] : null,
    target_reps_max: reps ? reps[1] : null,
    target_duration_seconds: keys.includes("duration") ? ok(parseDurationParts(it.duration)) : null,
    target_distance_m: keys.includes("distance") ? ok(parseDistance(it.distance)) : null,
    notes: it.notes.trim() || null,
  };
}
