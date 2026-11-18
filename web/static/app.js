"use strict";

// Lift Log frontend: plain JS, no build step. Hash routing:
//   #/login  #/  #/workouts/<id>  #/templates  #/templates/<id|new>  #/exercises

// ---------- tokens ----------

const store = {
  get access() { return localStorage.getItem("access_token"); },
  get refresh() { return localStorage.getItem("refresh_token"); },
  set(pair) {
    localStorage.setItem("access_token", pair.access_token);
    localStorage.setItem("refresh_token", pair.refresh_token);
  },
  clear() {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
  },
};

// ---------- API ----------

class ApiError extends Error {
  constructor(status, message) { super(message); this.status = status; }
}

let refreshing = null; // shared promise so parallel 401s refresh once

async function refreshTokens() {
  if (!refreshing) {
    refreshing = (async () => {
      const rt = store.refresh;
      if (!rt) return false;
      const res = await fetch("/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: rt }),
      });
      if (!res.ok) return false;
      store.set(await res.json());
      return true;
    })().finally(() => { refreshing = null; });
  }
  return refreshing;
}

async function api(method, path, body, { auth = true, retry = true } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth && store.access) headers["Authorization"] = "Bearer " + store.access;

  const res = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (res.status === 401 && auth && retry) {
    if (await refreshTokens()) return api(method, path, body, { auth, retry: false });
    store.clear();
    go("#/login");
    throw new ApiError(401, "Session expired, please log in again");
  }
  if (res.status === 204) return null;

  const data = await res.json().catch(() => null);
  if (!res.ok) throw new ApiError(res.status, (data && data.error) || res.statusText);
  return data;
}

// ---------- DOM helpers ----------

// h("div.card", {onclick}, child, ...) — children are strings (inserted as
// text, never HTML) or nodes, so user data can't inject markup.
function h(tag, attrs, ...children) {
  const [name, ...classes] = tag.split(".");
  const el = document.createElement(name);
  if (classes.length) el.className = classes.join(" ");
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k.startsWith("on")) el.addEventListener(k.slice(2), v);
    else if (k === "value") el.value = v;
    else el.setAttribute(k, v === true ? "" : v);
  }
  for (const c of children.flat()) {
    if (c == null || c === false) continue;
    el.append(c instanceof Node ? c : String(c));
  }
  return el;
}

const app = document.getElementById("app");

function render(...nodes) {
  app.replaceChildren(...nodes.filter((n) => n != null && n !== false));
}

let toastTimer;
function toast(msg, isError = false) {
  const t = document.getElementById("toast");
  t.textContent = msg;
  t.className = "toast" + (isError ? " err" : "");
  t.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { t.hidden = true; }, 2500);
}

function fail(err) {
  console.error(err);
  toast(err.message || "Something went wrong", true);
}

function fmtDate(iso) {
  return new Date(iso).toLocaleDateString(undefined, {
    weekday: "short", day: "numeric", month: "short", year: "numeric",
  });
}

function fmtTime(iso) {
  return new Date(iso).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}

// <input type="datetime-local"> wants local "YYYY-MM-DDTHH:MM".
function toLocalInput(iso) {
  const d = new Date(iso);
  const off = d.getTimezoneOffset() * 60000;
  return new Date(d - off).toISOString().slice(0, 16);
}

function fmtWeight(kg) {
  return (Math.round(kg * 100) / 100).toString();
}

// Session length: "45 min", "1 h 05 min".
function fmtSession(startIso, endIso) {
  const min = Math.max(0, Math.round((new Date(endIso) - new Date(startIso)) / 60000));
  if (min < 1) return "<1 min";
  if (min < 60) return `${min} min`;
  return `${Math.floor(min / 60)} h ${String(min % 60).padStart(2, "0")} min`;
}

// "today", "yesterday", "3 days ago", then a date.
function fmtAgo(iso) {
  const day = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate());
  const days = Math.round((day(new Date()) - day(new Date(iso))) / 86400000);
  if (days <= 0) return "today";
  if (days === 1) return "yesterday";
  if (days < 14) return `${days} days ago`;
  return fmtDate(iso);
}

function templateMeta(t) {
  const n = `${t.exercise_count} exercise${t.exercise_count === 1 ? "" : "s"}`;
  return n + " · " + (t.last_used_at ? "last done " + fmtAgo(t.last_used_at) : "never done");
}

async function startTemplate(t, btn) {
  if (btn) btn.disabled = true;
  try {
    const w = await api("POST", `/templates/${t.id}/start`);
    go("#/workouts/" + w.id);
  } catch (e) {
    fail(e);
    if (btn) btn.disabled = false;
  }
}

// ---------- routing ----------

function go(hash) {
  if (location.hash === hash) route();
  else location.hash = hash;
}

function route() {
  const hash = location.hash || "#/";
  const loggedIn = !!store.access;
  document.getElementById("topbar").hidden = !loggedIn || hash === "#/login";

  if (!loggedIn && hash !== "#/login") return go("#/login");
  if (loggedIn && hash === "#/login") return go("#/");

  let m;
  if (hash === "#/login") return viewLogin();
  if (hash === "#/exercises") return nav("exercises"), viewExercises().catch(fail);
  if (hash === "#/templates") return nav("templates"), viewTemplates().catch(fail);
  if (hash === "#/templates/new") return nav("templates"), viewTemplate(null).catch(fail);
  if ((m = hash.match(/^#\/templates\/([0-9a-f-]{36})$/))) return nav("templates"), viewTemplate(m[1]).catch(fail);
  if ((m = hash.match(/^#\/workouts\/([0-9a-f-]{36})$/))) return nav("workouts"), viewWorkout(m[1]).catch(fail);
  nav("workouts");
  viewWorkouts().catch(fail);
}

function nav(active) {
  document.querySelectorAll("[data-nav]").forEach((a) => {
    a.classList.toggle("active", a.dataset.nav === active);
  });
}

window.addEventListener("hashchange", route);

document.getElementById("logout").addEventListener("click", async () => {
  const rt = store.refresh;
  store.clear();
  if (rt) {
    // Best effort: the local tokens are already gone either way.
    fetch("/auth/logout", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    }).catch(() => {});
  }
  go("#/login");
});

// ---------- views: auth ----------

function viewLogin() {
  let mode = "login";
  const err = h("p.error");
  const username = h("input", { name: "username", autocomplete: "username", placeholder: "Username" });
  const usernameField = h("div", {}, h("label", {}, "Username"), username);
  const email = h("input", { type: "email", name: "email", autocomplete: "email", required: true });
  const password = h("input", { type: "password", name: "password", autocomplete: "current-password", required: true, minlength: 8 });
  const submit = h("button.primary.block", { type: "submit" }, "Log in");

  const tabs = ["login", "register"].map((m) =>
    h("button", { type: "button", "aria-pressed": String(m === mode), onclick: () => setMode(m) },
      m === "login" ? "Log in" : "Sign up"));

  function setMode(m) {
    mode = m;
    tabs.forEach((t, i) => t.setAttribute("aria-pressed", String(["login", "register"][i] === m)));
    usernameField.hidden = m === "login";
    username.required = m === "register";
    password.autocomplete = m === "login" ? "current-password" : "new-password";
    submit.textContent = m === "login" ? "Log in" : "Create account";
    err.textContent = "";
  }

  const form = h("form.stack", {
    onsubmit: async (e) => {
      e.preventDefault();
      err.textContent = "";
      submit.disabled = true;
      try {
        if (mode === "register") {
          await api("POST", "/auth/register",
            { username: username.value.trim(), email: email.value.trim(), password: password.value },
            { auth: false });
        }
        const pair = await api("POST", "/auth/login",
          { email: email.value.trim(), password: password.value }, { auth: false });
        store.set(pair);
        go("#/");
      } catch (ex) {
        err.textContent = ex.message;
      } finally {
        submit.disabled = false;
      }
    },
  },
    usernameField,
    h("div", {}, h("label", {}, "Email"), email),
    h("div", {}, h("label", {}, "Password"), password),
    err,
    submit,
  );

  setMode("login");
  render(h("div.auth", {}, h("span.brand", {}, "Lift Log"), h("div.tabs", {}, tabs), form));
}

// ---------- views: workouts list ----------

async function viewWorkouts() {
  const start = h("button.primary.block", {
    onclick: async () => {
      start.disabled = true;
      try {
        const w = await api("POST", "/workouts", {});
        go("#/workouts/" + w.id);
      } catch (e) {
        fail(e);
        start.disabled = false;
      }
    },
  }, "+ Start workout");

  render(h("h1", {}, "Workouts"), start, h("p.muted", {}, "Loading…"));
  const [workouts, templates] = await Promise.all([api("GET", "/workouts"), api("GET", "/templates")]);

  const fromTemplate = templates.length
    ? [h("h2.section", {}, "Start from a template"),
       h("ul.list.card", {}, templates.map((t) => {
         const btn = h("button.small", { onclick: () => startTemplate(t, btn) }, "Start");
         return h("li", {},
           h("div", {}, h("div", {}, t.name), h("div.muted.small", {}, templateMeta(t))),
           btn);
       }))]
    : [];

  const items = workouts.length
    ? workouts.map((w) =>
        h("a.workout-item", { href: "#/workouts/" + w.id },
          h("div.title", {}, w.name || "Workout"),
          h("div.muted.small", {}, fmtDate(w.performed_at) + " · " + fmtTime(w.performed_at) +
            (w.finished_at ? " · " + fmtSession(w.performed_at, w.finished_at) : ""))))
    : [h("p.muted.empty", {}, "No workouts yet. Start your first one.")];

  render(
    h("h1", {}, "Workouts"),
    h("div", { style: "margin-bottom:16px" }, start),
    ...fromTemplate,
    templates.length || workouts.length ? h("h2.section", {}, "History") : null,
    ...items,
  );
}

// ---------- exercise library helpers ----------

// Category lists come from the server (GET /exercise-options), which reads
// the single Go definition; nothing here hard-codes allowed values.
let optionsPromise = null;
function getOptions() {
  if (!optionsPromise) {
    optionsPromise = api("GET", "/exercise-options", undefined, { auth: false })
      .catch((e) => { optionsPromise = null; throw e; });
  }
  return optionsPromise;
}

function labelOf(list, value) {
  const o = list.find((x) => x.value === value);
  return o ? o.label : value;
}

function exerciseMeta(opts, e) {
  return [labelOf(opts.equipment, e.equipment), labelOf(opts.muscles, e.primary_muscle)].join(" · ");
}

function fmtDuration(sec) {
  const m = Math.floor(sec / 60), s = sec % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

// Renders a set from whatever measurements it has, so old sets still read
// right if their exercise's tracking type is changed later.
function fmtSet(s) {
  const parts = [];
  const w = s.weight_kg != null ? fmtWeight(s.weight_kg) + " kg" : null;
  if (s.reps != null) {
    if (s.exercise_tracking_type === "bodyweight_reps") {
      parts.push(w && s.weight_kg > 0 ? `BW + ${w} × ${s.reps}` : `${s.reps} reps`);
    } else {
      parts.push(w ? `${w} × ${s.reps}` : `${s.reps} reps`);
    }
  }
  if (s.distance_m != null) parts.push(`${fmtWeight(s.distance_m)} m`);
  if (s.duration_seconds != null) parts.push(fmtDuration(s.duration_seconds));
  if (s.reps == null && w && s.weight_kg > 0) parts.push(`@ ${w}`);
  return parts.join(" · ");
}

// "2/3 sets · 8–12 reps"; done is omitted outside a workout.
function fmtTarget(t, done) {
  const out = [`${done != null ? done + "/" : ""}${t.target_sets} set${t.target_sets === 1 && done == null ? "" : "s"}`];
  if (t.target_reps_min != null) {
    out.push(t.target_reps_min === t.target_reps_max
      ? `${t.target_reps_min} reps` : `${t.target_reps_min}–${t.target_reps_max} reps`);
  }
  if (t.target_distance_m != null) out.push(`${fmtWeight(t.target_distance_m)} m`);
  if (t.target_duration_seconds != null) out.push(fmtDuration(t.target_duration_seconds));
  return out.join(" · ");
}

// "3 × 10 @ 60 kg" when every set was the same, else each set in order.
function fmtSets(sets) {
  const f = sets.map(fmtSet);
  if (!f.every((x) => x === f[0])) return f.join(", ");
  const s = sets[0];
  if (sets.length === 1) return f[0];
  if (s.exercise_tracking_type === "weight_reps" && s.reps != null && s.weight_kg != null) {
    return `${sets.length} × ${s.reps} @ ${fmtWeight(s.weight_kg)} kg`;
  }
  return `${sets.length} × ${f[0]}`;
}

// "Last time (3 days ago): 3 × 10 @ 60 kg".
function fmtPrevious(p) {
  return `Last time (${fmtAgo(p.performed_at)}): ${fmtSets(p.sets)}`;
}

function select(list, value, { placeholder, required } = {}) {
  return h("select", { required },
    placeholder ? h("option", { value: "" }, placeholder) : null,
    list.map((o) => h("option", { value: o.value, selected: o.value === value }, o.label)));
}

function field(label, control, hint) {
  return h("div", {}, h("label", {}, label), control, hint ? h("p.hint", {}, hint) : null);
}

// Create/edit form for a custom exercise. onSave receives the request body.
function exerciseForm(opts, { initial = {}, title, submitLabel, onSave, onCancel }) {
  const name = h("input", { value: initial.name || "", required: true, maxlength: 100, placeholder: "e.g. Paused squat" });
  const tracking = select(opts.tracking_types, initial.tracking_type || "weight_reps");
  const equipment = select(opts.equipment, initial.equipment, { placeholder: "Choose…", required: true });
  const pattern = select(opts.movement_patterns, initial.movement_pattern, { placeholder: "Choose…", required: true });
  const primary = select(opts.muscles, initial.primary_muscle, { placeholder: "Choose…", required: true });
  const mechanic = select(opts.mechanics, initial.mechanic || "compound");

  const secondary = new Set(initial.secondary_muscles || []);
  const chips = opts.muscles.map((m) => {
    const box = h("input", { type: "checkbox", value: m.value, checked: secondary.has(m.value) });
    box.addEventListener("change", () => { box.checked ? secondary.add(m.value) : secondary.delete(m.value); });
    return { m, box, el: h("label.chip", {}, box, m.label) };
  });
  // The primary muscle can't also be secondary.
  function syncChips() {
    for (const c of chips) {
      const isPrimary = c.m.value === primary.value;
      c.box.disabled = isPrimary;
      if (isPrimary) { c.box.checked = false; secondary.delete(c.m.value); }
      c.el.classList.toggle("disabled", isPrimary);
    }
  }
  primary.addEventListener("change", syncChips);
  syncChips();

  const submit = h("button.primary", { type: "submit" }, submitLabel);
  return h("form.card.stack", {
    onsubmit: async (e) => {
      e.preventDefault();
      submit.disabled = true;
      try {
        await onSave({
          name: name.value.trim(),
          tracking_type: tracking.value,
          equipment: equipment.value,
          movement_pattern: pattern.value,
          primary_muscle: primary.value,
          secondary_muscles: [...secondary],
          mechanic: mechanic.value,
        });
      } catch (ex) {
        fail(ex);
      } finally {
        submit.disabled = false;
      }
    },
  },
    h("h2", {}, title),
    field("Name", name),
    field("How do you log it?", tracking),
    h("div.row", {}, field("Equipment", equipment), field("Mechanic", mechanic)),
    h("div.row", {}, field("Movement pattern", pattern), field("Primary muscle", primary)),
    h("div", {}, h("label", {}, "Secondary muscles"), h("div.chips", {}, chips.map((c) => c.el))),
    h("div.row", {},
      onCancel ? h("button", { type: "button", onclick: onCancel }, "Cancel") : null,
      submit),
  );
}

// Creates a custom exercise, first warning (never blocking) if the name is
// close to a default. Returns the new exercise, or null if the user backed out.
async function createExercise(body) {
  const similar = await api("GET", "/exercises/similar?name=" + encodeURIComponent(body.name));
  if (similar.length) {
    const names = similar.map((e) => `"${e.name}"`).join(", ");
    if (!confirm(`The library already has ${names}. You can use that instead. Create your own "${body.name}" anyway?`)) {
      return null;
    }
  }
  return api("POST", "/exercises", body);
}

// ---------- views: single workout ----------

// The exercise last picked in the set form, kept across renders.
let lastExerciseId = "";

// A value this many times the largest one logged for the exercise (this
// workout or last time) is probably a typo; asked about before saving.
// min is how far above it must also be, so small numbers don't trip it.
const SUSPICIOUS = {
  weight_kg: { factor: 2, min: 20, unit: " kg", fmt: fmtWeight },
  reps: { factor: 3, min: 10, unit: " reps", fmt: String },
  distance_m: { factor: 3, min: 500, unit: " m", fmt: fmtWeight },
  duration_seconds: { factor: 3, min: 120, unit: "", fmt: fmtDuration },
};

// Fields shown per tracking type; mirrors models.setRules on the server
// (which is the one that enforces it).
const SET_FIELDS = {
  weight_reps: [["weight_kg", "Weight (kg)", true], ["reps", "Reps", true]],
  bodyweight_reps: [["reps", "Reps", true], ["weight_kg", "Added weight (kg)", false]],
  duration: [["duration", "Time (min:sec)", true], ["weight_kg", "Added weight (kg)", false]],
  distance: [["distance_m", "Distance (m)", true], ["weight_kg", "Weight (kg)", false]],
};

function parseDuration(str) {
  const t = str.trim();
  if (!t) return null;
  if (t.includes(":")) {
    const [m, s] = t.split(":").map((x) => parseInt(x || "0", 10));
    return m * 60 + s;
  }
  return parseInt(t, 10); // plain number = seconds
}

// editSetId: a logged set being corrected in the set form instead of a
// new one being logged.
async function viewWorkout(id, { creatingExercise = false, editSetId = null } = {}) {
  if (!app.childElementCount) render(h("p.muted", {}, "Loading…"));
  const [w, exercises, opts, previous] = await Promise.all([
    api("GET", "/workouts/" + id),
    api("GET", "/exercises"),
    getOptions(),
    api("GET", `/workouts/${id}/previous`),
  ]);
  const byId = Object.fromEntries(exercises.map((e) => [e.id, e]));
  // "Last time" per exercise: sets from the latest earlier workout with it.
  const prevBy = Object.fromEntries(previous.map((p) => [p.exercise_id, p]));
  // The plan, if started from a template (gone if it was deleted since).
  const template = w.template_id
    ? await api("GET", "/templates/" + w.template_id).catch((e) => (e.status === 404 ? null : Promise.reject(e)))
    : null;
  const plan = template ? template.exercises : [];

  // --- header: name, date (saved on change) ---
  const name = h("input", { value: w.name || "", placeholder: "Workout name (e.g. Push day)" });
  const when = h("input", { type: "datetime-local", value: toLocalInput(w.performed_at) });
  const notes = h("textarea", { placeholder: "Notes" });
  notes.value = w.notes || "";

  // Finish waits for an in-flight save (editing notes then tapping Finish
  // fires "change" right before the click).
  let pendingSave = Promise.resolve();
  function saveMeta() {
    pendingSave = saveMetaNow();
    return pendingSave;
  }
  async function saveMetaNow() {
    try {
      await api("PUT", "/workouts/" + id, {
        name: name.value,
        notes: notes.value,
        performed_at: when.value ? new Date(when.value).toISOString() : undefined,
      });
      toast("Saved");
    } catch (e) { fail(e); }
  }
  [name, when, notes].forEach((el) => el.addEventListener("change", saveMeta));

  // --- sets grouped by exercise, in the order exercises were first logged ---
  const groups = new Map();
  for (const s of w.sets) {
    if (!groups.has(s.exercise_id)) groups.set(s.exercise_id, { name: s.exercise_name, sets: [] });
    groups.get(s.exercise_id).sets.push(s);
  }

  // Planned exercises first, in template order (with progress and the
  // target), then anything logged that wasn't in the plan.
  const planned = new Set(plan.map((te) => te.exercise_id));
  const cards = [
    ...plan.map((te) => ({ exId: te.exercise_id, te, g: groups.get(te.exercise_id) || { name: te.exercise_name, sets: [] } })),
    ...[...groups].filter(([exId]) => !planned.has(exId)).map(([exId, g]) => ({ exId, g })),
  ];
  const doneOf = (exId) => (groups.get(exId) || { sets: [] }).sets.length;
  const editing = editSetId ? w.sets.find((s) => s.id === editSetId) : null;

  const groupCards = cards.map(({ exId, te, g }) =>
    h("div.card", {},
      h("div.card-head", {},
        h("h2", {}, g.name),
        te && !te.exercise_archived && byId[te.exercise_id]
          ? h("button.small", { type: "button", onclick: () => {
            // While the new-exercise form is open there's no set form to fill.
            if (entry.pick) return entry.pick(te.exercise_id);
            lastExerciseId = te.exercise_id;
            viewWorkout(id).catch(fail);
          } }, "Log")
          : null),
      te ? h("p.muted.small.plan-meta", {},
        fmtTarget(te, g.sets.length) + (te.exercise_archived ? " · archived" : "")) : null,
      te && te.notes ? h("p.muted.small.plan-meta", {}, te.notes) : null,
      prevBy[exId] ? h("p.muted.small.plan-meta", {}, fmtPrevious(prevBy[exId])) : null,
      h("ul.sets", {}, g.sets.map((s, i) =>
        h("li", { class: editing && s.id === editing.id ? "editing" : null },
          h("span.n", {}, i + 1),
          h("button.v", {
            type: "button",
            title: "Edit set",
            onclick: () => viewWorkout(id, { editSetId: s.id }).catch(fail),
          }, fmtSet(s)),
          h("button.icon", {
            "aria-label": "Delete set",
            title: "Delete set",
            onclick: async () => {
              try {
                await api("DELETE", `/workouts/${id}/sets/${s.id}`);
                viewWorkout(id).catch(fail);
              } catch (e) { fail(e); }
            },
          }, "×"))))));

  // --- new-exercise form, shown in place of the set form ---
  let entry;
  if (creatingExercise) {
    entry = exerciseForm(opts, {
      title: "New exercise",
      submitLabel: "Create & use",
      onCancel: () => viewWorkout(id).catch(fail),
      onSave: async (body) => {
        const e = await createExercise(body);
        if (!e) return;
        lastExerciseId = e.id;
        viewWorkout(id).catch(fail);
      },
    });
  } else {
    entry = setForm();
  }

  function setForm() {
    const NEW = "__new__";
    const mine = exercises.filter((e) => !e.system);
    const library = exercises.filter((e) => e.system);
    // A set being edited may use an exercise archived since; keep it
    // selectable so only the numbers need fixing.
    let archivedOpt = null;
    if (editing && !byId[editing.exercise_id]) {
      byId[editing.exercise_id] = {
        id: editing.exercise_id, name: editing.exercise_name + " (archived)",
        tracking_type: editing.exercise_tracking_type,
      };
      archivedOpt = h("option", { value: editing.exercise_id }, byId[editing.exercise_id].name);
    }
    const exSelect = h("select", { required: true },
      h("option", { value: "" }, "Choose exercise…"),
      archivedOpt,
      mine.length ? h("optgroup", { label: "My exercises" }, mine.map((e) => h("option", { value: e.id }, e.name))) : null,
      h("optgroup", { label: "Library" }, library.map((e) => h("option", { value: e.id }, e.name))),
      editing ? null : h("option", { value: NEW }, "+ New exercise…"));

    const lastInWorkout = w.sets.length ? w.sets[w.sets.length - 1].exercise_id : null;
    let prefer = byId[lastExerciseId] ? lastExerciseId : lastInWorkout;
    if (plan.length) {
      // Stay on the current planned exercise until its sets are done, then
      // move to the next unfinished one.
      const open = (te) => byId[te.exercise_id] && doneOf(te.exercise_id) < te.target_sets;
      const cur = plan.find((te) => te.exercise_id === prefer);
      if (!cur || !open(cur)) {
        const next = plan.find(open);
        if (next) prefer = next.exercise_id;
      }
    }
    if (editing) prefer = editing.exercise_id;
    if (prefer && byId[prefer]) exSelect.value = prefer;

    const fieldsBox = h("div.row");
    const prevHint = h("p.hint");
    let inputs = {};

    function buildFields() {
      inputs = {};
      const ex = byId[exSelect.value];
      const last = ex && prevBy[ex.id];
      prevHint.textContent = last ? fmtPrevious(last) : "";
      prevHint.hidden = !last;
      if (!ex) { fieldsBox.replaceChildren(); return; }
      const te = plan.find((p) => p.exercise_id === ex.id);
      // Pre-fill from: the set being edited, else this workout's latest set
      // of the exercise, else the first set last time, else the plan target.
      const logged = (groups.get(ex.id) || { sets: [] }).sets;
      const prev = (editing && editing.exercise_id === ex.id ? editing : null)
        || logged[logged.length - 1]
        || (last && last.sets[0])
        || (te ? {
          reps: te.target_reps_min,
          duration_seconds: te.target_duration_seconds, distance_m: te.target_distance_m,
        } : {});
      fieldsBox.replaceChildren(...SET_FIELDS[ex.tracking_type].map(([key, label, required]) => {
        let input;
        if (key === "duration") {
          input = h("input", { inputmode: "numeric", required, placeholder: "1:30", pattern: "\\d+(:[0-5]?\\d)?",
            value: prev.duration_seconds != null ? fmtDuration(prev.duration_seconds) : "" });
        } else {
          const isInt = key === "reps";
          input = h("input", {
            type: "number", inputmode: isInt ? "numeric" : "decimal", required,
            min: isInt ? 1 : 0, step: isInt ? 1 : "0.25",
            placeholder: required ? "" : "optional",
            value: prev[key] != null ? (isInt ? prev[key] : fmtWeight(prev[key])) : "",
          });
        }
        inputs[key] = input;
        return field(label, input);
      }));
    }

    exSelect.addEventListener("change", () => {
      if (exSelect.value === NEW) return viewWorkout(id, { creatingExercise: true }).catch(fail);
      if (!editing) lastExerciseId = exSelect.value;
      buildFields();
    });
    buildFields();

    const addBtn = h("button.primary.block", { type: "submit" }, editing ? "Save set" : "Log set");
    const form = h("form.card.stack", {
      onsubmit: async (e) => {
        e.preventDefault();
        const ex = byId[exSelect.value];
        if (!ex) return toast("Choose an exercise", true);
        const num = (k, parse) => (inputs[k] && inputs[k].value.trim() !== "" ? parse(inputs[k].value) : undefined);
        const body = {
          exercise_id: ex.id,
          reps: num("reps", (v) => parseInt(v, 10)),
          weight_kg: num("weight_kg", parseFloat),
          duration_seconds: num("duration", parseDuration),
          distance_m: num("distance_m", parseFloat),
        };
        const odd = suspicious(ex.id, body);
        if (odd && !confirm(odd + " Save anyway?")) return;
        addBtn.disabled = true;
        try {
          if (editing) {
            await api("PUT", `/workouts/${id}/sets/${editing.id}`, body);
            toast("Set updated");
          } else {
            await api("POST", `/workouts/${id}/sets`, body);
            lastExerciseId = ex.id;
          }
          viewWorkout(id).catch(fail);
        } catch (ex2) {
          fail(ex2);
          addBtn.disabled = false;
        }
      },
    },
      editing ? h("h2", {}, "Edit set") : null,
      field("Exercise", exSelect),
      prevHint,
      fieldsBox,
      editing
        ? h("div.row", {},
            h("button", { type: "button", onclick: () => viewWorkout(id).catch(fail) }, "Cancel"),
            addBtn)
        : addBtn);

    // A warning like "1000 kg? Last time your most was 100 kg." when a value
    // is far above anything logged for the exercise, else null.
    function suspicious(exId, body) {
      const last = prevBy[exId];
      const history = [
        ...(groups.get(exId) || { sets: [] }).sets.filter((s) => !editing || s.id !== editing.id),
        ...(last ? last.sets : []),
      ];
      for (const [key, r] of Object.entries(SUSPICIOUS)) {
        const v = body[key];
        const seen = history.map((s) => s[key]).filter((x) => x != null && x > 0);
        if (v == null || !seen.length) continue;
        const most = Math.max(...seen);
        if (v > most * r.factor && v - most >= r.min) {
          return `${r.fmt(v)}${r.unit}? Your most for this exercise recently was ${r.fmt(most)}${r.unit}.`;
        }
      }
      return null;
    }

    // Called by a plan card's "Log" button.
    form.pick = (exId) => {
      exSelect.value = exId;
      lastExerciseId = exId;
      buildFields();
      form.scrollIntoView({ behavior: "smooth", block: "center" });
      const first = fieldsBox.querySelector("input");
      if (first) first.focus({ preventScroll: true });
    };
    return form;
  }

  // Sets are saved as they're logged, so finishing only leaves the page;
  // an empty workout is offered for discarding instead of kept as clutter.
  const finish = h("button.primary.block", {
    onclick: async () => {
      await pendingSave;
      if (!w.sets.length) {
        if (!confirm("No sets logged. Discard this empty workout?")) return;
        try {
          await api("DELETE", "/workouts/" + id);
          toast("Workout discarded");
          go("#/");
        } catch (e) { fail(e); }
        return;
      }
      try {
        const done = await api("POST", `/workouts/${id}/finish`);
        const n = w.sets.length;
        toast(`Workout saved · ${n} set${n === 1 ? "" : "s"}` +
          (done.finished_at ? " · " + fmtSession(done.performed_at, done.finished_at) : ""));
        go("#/");
      } catch (e) { fail(e); }
    },
  }, w.finished_at ? "Done" : "Finish workout");

  // Only for workouts not started from a template: turning one you
  // improvised into a reusable plan.
  const saveAsTemplate = w.sets.length && !template ? h("button.block", {
    onclick: async () => {
      const n = prompt("Name for the new template", w.name || "");
      if (n === null) return;
      try {
        const t = await api("POST", `/workouts/${id}/save-as-template`, { name: n.trim() });
        toast(`Saved template "${t.name}"`);
        go("#/templates/" + t.id);
      } catch (e) { fail(e); }
    },
  }, "Save as template") : null;

  const del = h("button.danger.block", {
    onclick: async () => {
      if (!confirm("Delete this workout and all its sets?")) return;
      try {
        await api("DELETE", "/workouts/" + id);
        go("#/");
      } catch (e) { fail(e); }
    },
  }, "Delete workout");

  render(
    h("p", {}, h("a.muted.small", { href: "#/" }, "← All workouts")),
    h("div.card.stack", {}, name, when,
      template || w.finished_at ? h("p.muted.small", { style: "margin-bottom:0" },
        template ? ["From template ", h("a", { href: "#/templates/" + template.id }, template.name)] : null,
        template && w.finished_at ? " · " : null,
        w.finished_at ? "Took " + fmtSession(w.performed_at, w.finished_at) : null) : null),
    ...groupCards,
    groupCards.length ? null : h("p.muted.empty", {}, "No sets yet. Log your first one below."),
    entry,
    h("div.card", {}, h("label", {}, "Notes"), notes),
    h("div.stack", { style: "margin-bottom:12px" }, finish, saveAsTemplate),
    del,
  );
  if (editing) {
    entry.scrollIntoView({ block: "center" });
    const first = entry.querySelector("input");
    if (first) first.focus({ preventScroll: true });
  }
}

// ---------- views: templates ----------

async function viewTemplates() {
  if (!app.childElementCount) render(h("h1", {}, "Templates"), h("p.muted", {}, "Loading…"));
  const templates = await api("GET", "/templates");

  const list = templates.length
    ? h("ul.list.card", {}, templates.map((t) => {
        const start = h("button.small", { onclick: () => startTemplate(t, start) }, "Start");
        return h("li", {},
          h("div", {}, h("div", {}, t.name), h("div.muted.small", {}, templateMeta(t))),
          h("div.actions", {},
            start,
            h("button.small", { onclick: () => go("#/templates/" + t.id) }, "Edit")));
      }))
    : h("p.muted.empty", {}, "No templates yet. Build one here, or open a logged workout and tap “Save as template”.");

  render(
    h("h1", {}, "Templates"),
    h("button.primary.block", { style: "margin-bottom:16px", onclick: () => go("#/templates/new") }, "+ New template"),
    list,
  );
}

// Target inputs per tracking type; mirrors models.setRules on the server
// (which enforces it). All targets are optional. No weight: it's meant to
// go up, so it isn't part of the plan.
const TARGET_FIELDS = {
  weight_reps: ["reps"],
  bodyweight_reps: ["reps"],
  duration: ["duration"],
  distance: ["distance_m", "duration"],
};

const TARGET_INPUTS = {
  reps: { label: "Reps", attrs: { inputmode: "numeric", placeholder: "8 or 8-12", pattern: "\\d+(\\s*[\\-–]\\s*\\d+)?" } },
  duration: { label: "Time (min:sec)", attrs: { inputmode: "numeric", placeholder: "1:30", pattern: "\\d+(:[0-5]?\\d)?" } },
  distance_m: { label: "Distance (m)", attrs: { type: "number", inputmode: "decimal", min: 0, step: "any", placeholder: "optional" } },
};

// "8" → [8, 8], "8-12" → [8, 12]; null if malformed.
function parseReps(str) {
  const m = str.trim().match(/^(\d+)(?:\s*[-–]\s*(\d+))?$/);
  return m ? [parseInt(m[1], 10), parseInt(m[2] || m[1], 10)] : null;
}

// Template editor; id null = new. The whole exercise list is sent on save
// (PUT replaces it), so editing is purely local until then.
async function viewTemplate(id) {
  render(h("p.muted", {}, "Loading…"));
  const [t, exercises] = await Promise.all([
    id ? api("GET", "/templates/" + id) : { name: "", notes: "", exercises: [] },
    api("GET", "/exercises"),
  ]);
  const byId = Object.fromEntries(exercises.map((e) => [e.id, e]));
  // Archived exercises aren't in /exercises but may stay in a template.
  const archived = [];
  for (const te of t.exercises) {
    if (!byId[te.exercise_id]) {
      byId[te.exercise_id] = { id: te.exercise_id, name: te.exercise_name + " (archived)", tracking_type: te.exercise_tracking_type };
      archived.push(byId[te.exercise_id]);
    }
  }
  const mine = exercises.filter((e) => !e.system);
  const library = exercises.filter((e) => e.system);

  // Each row keeps raw input strings; they're parsed on save.
  const blank = () => ({ exercise_id: "", sets: "3", reps: "", duration: "", distance_m: "", notes: "" });
  const items = t.exercises.map((te) => ({
    exercise_id: te.exercise_id,
    sets: String(te.target_sets),
    reps: te.target_reps_min == null ? ""
      : te.target_reps_min === te.target_reps_max ? String(te.target_reps_min)
      : `${te.target_reps_min}-${te.target_reps_max}`,
    duration: te.target_duration_seconds == null ? "" : fmtDuration(te.target_duration_seconds),
    distance_m: te.target_distance_m == null ? "" : fmtWeight(te.target_distance_m),
    notes: te.notes || "",
  }));
  if (!items.length) items.push(blank());

  const name = h("input", { value: t.name, required: true, maxlength: 100, placeholder: "e.g. Push day A" });
  const notes = h("textarea", { placeholder: "Notes (optional)" });
  notes.value = t.notes || "";
  const list = h("div");

  function bind(item, key, attrs) {
    const el = h("input", { ...attrs, value: item[key] });
    el.addEventListener("input", () => { item[key] = el.value; });
    return el;
  }

  function row(item, i) {
    const exSelect = h("select", {},
      h("option", { value: "" }, "Choose exercise…"),
      archived.length ? h("optgroup", { label: "Archived" }, archived.map((e) => h("option", { value: e.id }, e.name))) : null,
      mine.length ? h("optgroup", { label: "My exercises" }, mine.map((e) => h("option", { value: e.id }, e.name))) : null,
      h("optgroup", { label: "Library" }, library.map((e) => h("option", { value: e.id }, e.name))));
    exSelect.value = item.exercise_id;
    exSelect.addEventListener("change", () => { item.exercise_id = exSelect.value; renderList(); });

    const move = (d) => {
      [items[i], items[i + d]] = [items[i + d], items[i]];
      renderList();
    };
    const ex = byId[item.exercise_id];
    const targets = ex
      ? TARGET_FIELDS[ex.tracking_type].map((key) => {
          const { label, attrs } = TARGET_INPUTS[key];
          return field(label, bind(item, key, attrs));
        })
      : [];

    return h("div.card.stack", {},
      h("div.row", { style: "align-items:center" },
        exSelect,
        h("div.actions", {},
          h("button.icon", { type: "button", "aria-label": "Move up", title: "Move up", disabled: i === 0, onclick: () => move(-1) }, "↑"),
          h("button.icon", { type: "button", "aria-label": "Move down", title: "Move down", disabled: i === items.length - 1, onclick: () => move(1) }, "↓"),
          h("button.icon", { type: "button", "aria-label": "Remove", title: "Remove", onclick: () => { items.splice(i, 1); renderList(); } }, "×"))),
      ex ? h("div.row.wrap", {},
        field("Sets", bind(item, "sets", { type: "number", inputmode: "numeric", min: 1, max: 20, required: true })),
        ...targets) : null,
      ex ? bind(item, "notes", { maxlength: 200, placeholder: "Notes (e.g. pause at the bottom)" }) : null,
    );
  }

  function renderList() {
    list.replaceChildren(
      ...items.map(row),
      h("button.block", { type: "button", onclick: () => { items.push(blank()); renderList(); } }, "+ Add exercise"));
  }
  renderList();

  // Rows without an exercise are skipped; only the targets the exercise's
  // tracking type uses are sent.
  function exerciseBodies() {
    const out = [];
    items.forEach((it, i) => {
      const ex = byId[it.exercise_id];
      if (!ex) return;
      const keys = TARGET_FIELDS[ex.tracking_type];
      const has = (k) => keys.includes(k) && it[k].trim() !== "";
      const body = { exercise_id: ex.id, target_sets: parseInt(it.sets, 10), notes: it.notes.trim() || undefined };
      if (has("reps")) {
        const r = parseReps(it.reps);
        if (!r) throw new Error(`Exercise ${i + 1}: reps must look like 8 or 8-12`);
        [body.target_reps_min, body.target_reps_max] = r;
      }
      if (has("duration")) body.target_duration_seconds = parseDuration(it.duration);
      if (has("distance_m")) body.target_distance_m = parseFloat(it.distance_m);
      out.push(body);
    });
    return out;
  }

  const saveBtn = h("button.primary.block", { type: "submit" }, id ? "Save" : "Create template");
  const form = h("form.stack", {
    onsubmit: async (e) => {
      e.preventDefault();
      saveBtn.disabled = true;
      try {
        const body = { name: name.value.trim(), notes: notes.value, exercises: exerciseBodies() };
        await (id ? api("PUT", "/templates/" + id, body) : api("POST", "/templates", body));
        toast("Saved");
        go("#/templates");
      } catch (ex) {
        fail(ex);
      } finally {
        saveBtn.disabled = false;
      }
    },
  },
    h("div.card.stack", {}, field("Name", name), field("Notes", notes)),
    h("h2.section", {}, "Exercises"),
    list,
    saveBtn,
  );

  const extra = id ? [
    h("button.block", { style: "margin-top:12px", onclick: () => startTemplate(t) }, "Start workout from this template"),
    h("button.danger.block", {
      style: "margin-top:12px",
      onclick: async () => {
        if (!confirm(`Delete template "${t.name}"?\n\nWorkouts you started from it are kept.`)) return;
        try {
          await api("DELETE", "/templates/" + id);
          toast("Deleted");
          go("#/templates");
        } catch (ex) { fail(ex); }
      },
    }, "Delete template"),
  ] : [];

  render(
    h("p", {}, h("a.muted.small", { href: "#/templates" }, "← All templates")),
    h("h1", {}, id ? "Edit template" : "New template"),
    form,
    ...extra,
  );
}

// ---------- views: exercises ----------

// mode: null (list), "new", or an exercise id being edited.
async function viewExercises(mode = null) {
  if (!app.childElementCount) render(h("h1", {}, "Exercises"), h("p.muted", {}, "Loading…"));
  const [exercises, opts] = await Promise.all([api("GET", "/exercises"), getOptions()]);
  const mine = exercises.filter((e) => !e.system);
  const library = exercises.filter((e) => e.system);
  const back = () => viewExercises().catch(fail);

  if (mode === "new") {
    return render(h("h1", {}, "Exercises"), exerciseForm(opts, {
      title: "New exercise",
      submitLabel: "Create",
      onCancel: back,
      onSave: async (body) => {
        const e = await createExercise(body);
        if (e) { toast(`Created "${e.name}"`); back(); }
      },
    }));
  }
  const editing = mine.find((e) => e.id === mode);
  if (editing) {
    return render(h("h1", {}, "Exercises"), exerciseForm(opts, {
      initial: editing,
      title: `Edit "${editing.name}"`,
      submitLabel: "Save",
      onCancel: back,
      onSave: async (body) => {
        await api("PUT", "/exercises/" + editing.id, body);
        toast("Saved");
        back();
      },
    }));
  }

  const mineList = mine.length
    ? h("ul.list.card", {}, mine.map((e) =>
        h("li", {},
          h("div", {}, h("div", {}, e.name), h("div.muted.small", {}, exerciseMeta(opts, e))),
          h("div.actions", {},
            h("button.small", { onclick: () => viewExercises(e.id).catch(fail) }, "Edit"),
            h("button.icon", {
              "aria-label": "Delete " + e.name,
              title: "Delete",
              onclick: async () => {
                if (!confirm(`Delete "${e.name}"?\n\nIf it has logged sets it's archived instead: hidden from the picker, still shown in your history.`)) return;
                try {
                  const r = await api("DELETE", "/exercises/" + e.id);
                  toast(r.result === "archived" ? `Archived "${e.name}"` : `Deleted "${e.name}"`);
                  back();
                } catch (ex) { fail(ex); }
              },
            }, "×")))))
    : h("p.muted.small", {}, "None yet. Create one, or customize one from the library below.");

  const libraryList = h("ul.list.card", {}, library.map((e) =>
    h("li", {},
      h("div", {}, h("div", {}, e.name), h("div.muted.small", {}, exerciseMeta(opts, e))),
      h("button.small", {
        title: "Make your own editable copy",
        onclick: async () => {
          const n = prompt(`Name for your copy of "${e.name}"`, e.name);
          if (n === null) return;
          try {
            const cp = await api("POST", `/exercises/${e.id}/duplicate`, { name: n.trim() });
            viewExercises(cp.id).catch(fail);
          } catch (ex) { fail(ex); }
        },
      }, "Customize"))));

  render(
    h("h1", {}, "Exercises"),
    h("button.primary.block", { onclick: () => viewExercises("new").catch(fail) }, "+ New exercise"),
    h("h2.section", {}, "My exercises"),
    mineList,
    h("h2.section", {}, "Library"),
    h("p.muted.small", {}, "Defaults everyone shares. Read-only; tap Customize to make your own version."),
    libraryList,
  );
}

route();
