"use strict";

// Lift Log frontend: plain JS, no build step. Hash routing:
//   #/login  #/  #/workouts/<id>  #/exercises

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
  const workouts = await api("GET", "/workouts");

  const items = workouts.length
    ? workouts.map((w) =>
        h("a.workout-item", { href: "#/workouts/" + w.id },
          h("div.title", {}, w.name || "Workout"),
          h("div.muted.small", {}, fmtDate(w.performed_at) + " · " + fmtTime(w.performed_at))))
    : [h("p.muted.empty", {}, "No workouts yet. Start your first one.")];

  render(h("h1", {}, "Workouts"), h("div", { style: "margin-bottom:16px" }, start), ...items);
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

// Remembered across renders (per exercise) so the next set is pre-filled.
const lastByExercise = {};
let lastExerciseId = "";

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

async function viewWorkout(id, { creatingExercise = false } = {}) {
  if (!app.childElementCount) render(h("p.muted", {}, "Loading…"));
  const [w, exercises, opts] = await Promise.all([
    api("GET", "/workouts/" + id),
    api("GET", "/exercises"),
    getOptions(),
  ]);
  const byId = Object.fromEntries(exercises.map((e) => [e.id, e]));

  // --- header: name, date (saved on change) ---
  const name = h("input", { value: w.name || "", placeholder: "Workout name (e.g. Push day)" });
  const when = h("input", { type: "datetime-local", value: toLocalInput(w.performed_at) });
  const notes = h("textarea", { placeholder: "Notes" });
  notes.value = w.notes || "";

  async function saveMeta() {
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
    lastByExercise[s.exercise_id] = s;
  }

  const groupCards = [...groups.values()].map((g) =>
    h("div.card", {},
      h("h2", {}, g.name),
      h("ul.sets", {}, g.sets.map((s, i) =>
        h("li", {},
          h("span.n", {}, i + 1),
          h("span.v", {}, fmtSet(s)),
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
    const exSelect = h("select", { required: true },
      h("option", { value: "" }, "Choose exercise…"),
      mine.length ? h("optgroup", { label: "My exercises" }, mine.map((e) => h("option", { value: e.id }, e.name))) : null,
      h("optgroup", { label: "Library" }, library.map((e) => h("option", { value: e.id }, e.name))),
      h("option", { value: NEW }, "+ New exercise…"));

    const lastInWorkout = w.sets.length ? w.sets[w.sets.length - 1].exercise_id : null;
    const prefer = byId[lastExerciseId] ? lastExerciseId : lastInWorkout;
    if (prefer && byId[prefer]) exSelect.value = prefer;

    const fieldsBox = h("div.row");
    let inputs = {};

    function buildFields() {
      inputs = {};
      const ex = byId[exSelect.value];
      if (!ex) { fieldsBox.replaceChildren(); return; }
      const prev = lastByExercise[ex.id] || {};
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
      lastExerciseId = exSelect.value;
      buildFields();
    });
    buildFields();

    const addBtn = h("button.primary.block", { type: "submit" }, "Log set");
    return h("form.card.stack", {
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
        addBtn.disabled = true;
        try {
          await api("POST", `/workouts/${id}/sets`, body);
          lastExerciseId = ex.id;
          viewWorkout(id).catch(fail);
        } catch (ex2) {
          fail(ex2);
          addBtn.disabled = false;
        }
      },
    },
      field("Exercise", exSelect),
      fieldsBox,
      addBtn);
  }

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
    h("div.card.stack", {}, name, when),
    ...groupCards,
    w.sets.length ? null : h("p.muted.empty", {}, "No sets yet. Log your first one below."),
    entry,
    h("div.card", {}, h("label", {}, "Notes"), notes),
    del,
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
