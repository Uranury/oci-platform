// Navigation shared by the router (main.ts) and the screens.

// A screen with unsaved work registers a guard. While dirty() is true,
// navigating away stays put and calls ask(to); the screen asks the user
// and calls go(to) itself once it's clean or they choose to leave.
export interface LeaveGuard {
  dirty: () => boolean;
  ask: (to: string) => void;
}

let guard: LeaveGuard | null = null;

export function setLeaveGuard(g: LeaveGuard | null): void {
  guard = g;
}

export function leaveGuard(): LeaveGuard | null {
  return guard;
}

// Shows the screen for hash, re-routing if it's already the address.
export function go(hash: string): void {
  if (location.hash === hash) window.dispatchEvent(new HashChangeEvent("hashchange"));
  else location.hash = hash;
}
