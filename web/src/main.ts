// Lift Log frontend: hash routing onto the Svelte screens.
//   #/login  #/  #/workouts/<id>  #/exercises
//   #/templates  #/templates/new  #/templates/<id>  #/templates/<id>/edit
import { mount, unmount, type Component } from "svelte";
import ExercisesScreen from "./exercises/ExercisesScreen.svelte";
import HomeScreen from "./home/HomeScreen.svelte";
import { store } from "./lib/api";
import { go, leaveGuard, setLeaveGuard } from "./lib/nav";
import LoginScreen from "./login/LoginScreen.svelte";
import "./styles/catalog.css";
import TemplateEditor from "./templates/TemplateEditor.svelte";
import TemplateScreen from "./templates/TemplateScreen.svelte";
import TemplatesScreen from "./templates/TemplatesScreen.svelte";
import WorkoutScreen from "./workout/WorkoutScreen.svelte";

const UUID = "[0-9a-f-]{36}";

// The screen for a hash, or null if the hash names none.
function screenFor(hash: string): [Component<any>, Record<string, unknown>] | null {
  let m: RegExpMatchArray | null;
  if (hash === "#/") return [HomeScreen, {}];
  if (hash === "#/login") return [LoginScreen, {}];
  if (hash === "#/exercises") return [ExercisesScreen, {}];
  if (hash === "#/templates") return [TemplatesScreen, {}];
  if (hash === "#/templates/new") return [TemplateEditor, { id: null }];
  if ((m = hash.match(new RegExp(`^#/templates/(${UUID})$`)))) return [TemplateScreen, { id: m[1] }];
  if ((m = hash.match(new RegExp(`^#/templates/(${UUID})/edit$`)))) return [TemplateEditor, { id: m[1] }];
  if ((m = hash.match(new RegExp(`^#/workouts/(${UUID})$`)))) return [WorkoutScreen, { id: m[1] }];
  return null;
}

const target = document.getElementById("screen")!;
let current: Record<string, unknown> | null = null;
let shown = "";

function route(): void {
  const hash = location.hash || "#/";
  if (hash === shown) return;
  const guard = leaveGuard();
  if (guard?.dirty()) {
    // Stay: put the address back without another hashchange.
    history.replaceState(null, "", shown);
    guard.ask(hash);
    return;
  }
  setLeaveGuard(null);

  const loggedIn = !!store.access;
  if (!loggedIn && hash !== "#/login") return go("#/login");
  if (loggedIn && hash === "#/login") return go("#/");
  const screen = screenFor(hash);
  if (!screen) return go("#/");

  if (current) unmount(current);
  shown = hash;
  window.scrollTo(0, 0);
  const [component, props] = screen;
  current = mount(component, { target, props });
}

// Leaving with unsaved changes: the browser asks on reload or close.
window.addEventListener("beforeunload", (e) => {
  if (leaveGuard()?.dirty()) e.preventDefault();
});
window.addEventListener("hashchange", route);
route();
