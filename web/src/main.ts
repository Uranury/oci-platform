// Svelte screens, mounted by the legacy router (src/legacy/app.js) for the
// routes that have been redesigned. Everything else still renders through
// the legacy code until its screen is redone.
import { mount, unmount, type Component } from "svelte";
import HomeScreen from "./home/HomeScreen.svelte";
import LoginScreen from "./login/LoginScreen.svelte";
import "./styles/catalog.css";
import WorkoutScreen from "./workout/WorkoutScreen.svelte";

export type Screen = { kind: "home" } | { kind: "login" } | { kind: "workout"; id: string };

let current: Record<string, unknown> | null = null;

// The browser chrome follows the screen: newsprint for redesigned screens,
// the legacy dark otherwise.
const LEGACY_THEME = "#111214";
const CATALOG_THEME = "#dedad1";
function setThemeColor(color: string) {
  document.querySelector('meta[name="theme-color"]')?.setAttribute("content", color);
}

export function showScreen(target: HTMLElement, screen: Screen): void {
  hideScreen();
  document.body.classList.add("catalog-body");
  setThemeColor(CATALOG_THEME);
  target.hidden = false;
  window.scrollTo(0, 0);
  const [component, props] =
    screen.kind === "workout"
      ? [WorkoutScreen as Component<{ id: string }>, { id: screen.id }]
      : [screen.kind === "home" ? HomeScreen : LoginScreen, {}];
  current = mount(component as Component<Record<string, unknown>>, { target, props });
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
