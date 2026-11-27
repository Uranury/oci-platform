import { fmtAgo, fmtWeight, fromYmd } from "./format";
import type { Field } from "./sets";
import type { BodyweightEntry } from "./types";

// The stepper for a weigh-in: tenths of a kilo, within what the API accepts
// (models.MinBodyweightKg..MaxBodyweightKg).
export const BW_FIELD: Field = {
  key: "weight_kg", label: "Weight", unit: "kg", required: true, step: 0.1, min: 20, integer: false,
};
export const BW_MAX = 500;

// "−0.4 kg since Thu", "+1.2 kg since 12 Nov", "same as Thu"; entries newest
// first. Null with fewer than two entries.
export function fmtChange(entries: BodyweightEntry[], now = new Date()): string | null {
  const [cur, prev] = entries;
  if (!cur || !prev) return null;
  const d = Math.round((cur.weight_kg - prev.weight_kg) * 10) / 10;
  const since = fmtAgo(fromYmd(prev.day).toISOString(), now);
  if (d === 0) return `same as ${since}`;
  return `${d > 0 ? "+" : "−"}${fmtWeight(Math.abs(d))} kg since ${since}`;
}

// "95 kg? Last was 80.6 kg." when a weigh-in is far from the latest other
// day's (or, with none, from what this day already holds): probably a typo.
export function suspiciousWeight(kg: number, day: string, entries: BodyweightEntry[]): string | null {
  const before = entries.find((e) => e.day !== day) ?? entries.find((e) => e.day === day);
  if (!before) return null;
  const diff = Math.abs(kg - before.weight_kg);
  if (diff < 3 || diff < before.weight_kg * 0.08) return null;
  return `${fmtWeight(kg)} kg? Last was ${fmtWeight(before.weight_kg)} kg.`;
}
