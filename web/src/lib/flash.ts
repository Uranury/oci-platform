// A message for whichever screen shows next ("Workout saved · 12 sets"),
// set right before navigating away. Screens take it on mount.
let pending: string | null = null;

export function flash(message: string): void {
  pending = message;
}

export function takeFlash(): string | null {
  const m = pending;
  pending = null;
  return m;
}
