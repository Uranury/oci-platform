// Svelte screens, mounted by the legacy router (src/legacy/app.js) for the
// routes that have been redesigned. Everything else still renders through
// the legacy code until its screen is redone.
import { mount, unmount } from "svelte";
import "./styles/catalog.css";
import WorkoutScreen, { type Host } from "./workout/WorkoutScreen.svelte";

let current: Record<string, unknown> | null = null;

// The browser chrome follows the screen: newsprint for redesigned screens,
// the legacy dark otherwise.
const LEGACY_THEME = "#111214";
const CATALOG_THEME = "#dedad1";
function setThemeColor(color: string) {
  document.querySelector('meta[name="theme-color"]')?.setAttribute("content", color);
}

export function showWorkout(target: HTMLElement, id: string, host: Host): void {
  hideScreen();
  document.body.classList.add("catalog-body");
  setThemeColor(CATALOG_THEME);
  target.hidden = false;
  current = mount(WorkoutScreen, { target, props: { id, host } });
}

export function hideScreen(): void {
  if (current) {
    unmount(current);
    current = null;
  }
  document.body.classList.remove("catalog-body");
  setThemeColor(LEGACY_THEME);
  const screen = document.getElementById("screen");
  if (screen) screen.hidden = true;
}
