// A message for whichever screen shows next ("Workout saved · 12 sets"),
// set right before navigating away. Svelte screens take it on mount; the
// legacy router shows it after rendering a legacy view.
let pending: string | null = null;

export function flash(message: string): void {
  pending = message;
}

export function takeFlash(): string | null {
  const m = pending;
  pending = null;
  return m;
}
