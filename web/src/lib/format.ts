import type { TemplateExercise, WorkoutSet } from "./types";

// The product is English-only; dates follow suit whatever the device locale.
const LOCALE = "en-GB";

export function fmtWeight(kg: number): string {
  return (Math.round(kg * 100) / 100).toString();
}

export function fmtDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

// "1:30" or "90" (seconds) -> 90; "" -> null; garbage -> NaN.
export function parseDuration(str: string): number | null {
  const t = str.trim();
  if (!t) return null;
  if (t.includes(":")) {
    const [m = "0", s = "0"] = t.split(":");
    if (!/^\d*$/.test(m) || !/^\d*$/.test(s)) return NaN;
    return parseInt(m || "0", 10) * 60 + parseInt(s || "0", 10);
  }
  return /^\d+$/.test(t) ? parseInt(t, 10) : NaN;
}

// A set from whatever measurements it has: "80 × 8", "BW + 10 × 8",
// "8 reps", "400 m · 1:45", "1:30 @ 20".
export function fmtSet(s: WorkoutSet): string {
  const parts: string[] = [];
  const w = s.weight_kg != null ? fmtWeight(s.weight_kg) : null;
  if (s.reps != null) {
    if (s.exercise_tracking_type === "bodyweight_reps") {
      parts.push(w && s.weight_kg! > 0 ? `BW + ${w} × ${s.reps}` : `BW × ${s.reps}`);
    } else {
      parts.push(w ? `${w} × ${s.reps}` : `${s.reps} reps`);
    }
  }
  if (s.distance_m != null) parts.push(`${fmtWeight(s.distance_m)} m`);
  if (s.duration_seconds != null) parts.push(fmtDuration(s.duration_seconds));
  if (s.reps == null && w && s.weight_kg! > 0) parts.push(`@ ${w} kg`);
  return parts.join(" · ");
}

// "3 × 8 @ 80 kg" when every set was the same, else each set in order.
export function fmtSets(sets: WorkoutSet[]): string {
  const f = sets.map(fmtSet);
  const first = sets[0];
  if (!first) return "";
  if (!f.every((x) => x === f[0])) return f.join(", ");
  if (sets.length === 1) return f[0]!;
  if (first.exercise_tracking_type === "weight_reps" && first.reps != null && first.weight_kg != null) {
    return `${sets.length} × ${first.reps} @ ${fmtWeight(first.weight_kg)} kg`;
  }
  if (first.exercise_tracking_type === "bodyweight_reps" && first.reps != null) {
    const added = first.weight_kg ? ` + ${fmtWeight(first.weight_kg)} kg` : "";
    return `${sets.length} × ${first.reps} @ BW${added}`;
  }
  return `${sets.length} × ${f[0]}`;
}

// The plan for one exercise: "3 × 8", "3 × 8–12", "3 × 400 m", "3 × 1:30".
export function fmtTarget(t: TemplateExercise): string {
  const each: string[] = [];
  if (t.target_reps_min != null) {
    each.push(
      t.target_reps_min === t.target_reps_max || t.target_reps_max == null
        ? `${t.target_reps_min}`
        : `${t.target_reps_min}–${t.target_reps_max}`,
    );
  }
  if (t.target_distance_m != null) each.push(`${fmtWeight(t.target_distance_m)} m`);
  if (t.target_duration_seconds != null) each.push(fmtDuration(t.target_duration_seconds));
  return each.length ? `${t.target_sets} × ${each.join(" · ")}` : `${t.target_sets} sets`;
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

// "today", "yesterday", "Thu" within the week, then "12 Nov".
export function fmtAgo(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86400000);
  if (days <= 0) return "today";
  if (days === 1) return "yesterday";
  if (days < 7) return d.toLocaleDateString(LOCALE, { weekday: "short" });
  return d.toLocaleDateString(LOCALE, { day: "numeric", month: "short" });
}

// "today", "yesterday", "6 days ago", then "12 Nov".
export function fmtDaysAgo(iso: string, now = new Date()): string {
  const days = Math.round((startOfDay(now) - startOfDay(new Date(iso))) / 86400000);
  if (days <= 0) return "today";
  if (days === 1) return "yesterday";
  if (days < 14) return `${days} days ago`;
  return new Date(iso).toLocaleDateString(LOCALE, { day: "numeric", month: "short" });
}

// "Bench press, Overhead press, Dip +2": the first few names, then a count.
export function fmtNames(names: string[], shown = 3): string {
  if (names.length <= shown) return names.join(", ");
  return `${names.slice(0, shown).join(", ")} +${names.length - shown}`;
}

// The lead-in for a "last time" line: "Today", "Yesterday", "Last Thu",
// "Last 12 Nov".
export function fmtLast(iso: string, now = new Date()): string {
  const when = fmtAgo(iso, now);
  if (when === "today" || when === "yesterday") return when[0]!.toUpperCase() + when.slice(1);
  return `Last ${when}`;
}

export function fmtDay(iso: string): string {
  return new Date(iso).toLocaleDateString(LOCALE, { weekday: "short", day: "numeric", month: "short" });
}

export function fmtClock(iso: string): string {
  return new Date(iso).toLocaleTimeString(LOCALE, { hour: "2-digit", minute: "2-digit" });
}

// Session length: "45 min", "1 h 05 min".
export function fmtSession(startIso: string, end: string | Date): string {
  const min = Math.max(0, Math.round((new Date(end).getTime() - new Date(startIso).getTime()) / 60000));
  if (min < 1) return "<1 min";
  if (min < 60) return `${min} min`;
  return `${Math.floor(min / 60)} h ${String(min % 60).padStart(2, "0")} min`;
}

// <input type="datetime-local"> wants local "YYYY-MM-DDTHH:MM".
export function toLocalInput(iso: string): string {
  const d = new Date(iso);
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}

// Exercise codes in programme notation: A, B, … Z, AA, AB, …
export function code(index: number): string {
  let s = "";
  let n = index;
  do {
    s = String.fromCharCode(65 + (n % 26)) + s;
    n = Math.floor(n / 26) - 1;
  } while (n >= 0);
  return s;
}
