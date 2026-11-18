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

// ---------- views: single workout ----------

// Remembered across renders so the next set is pre-filled after logging one.
const lastSet = { exerciseId: "", reps: "", weight: "" };

async function viewWorkout(id) {
  render(h("p.muted", {}, "Loading…"));
  const [w, exercises] = await Promise.all([
    api("GET", "/workouts/" + id),
    api("GET", "/exercises"),
  ]);

  // --- header: name, date, notes (saved on change) ---
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
  }

  const groupCards = [...groups.values()].map((g) =>
    h("div.card", {},
      h("h2", {}, g.name),
      h("ul.sets", {}, g.sets.map((s, i) =>
        h("li", {},
          h("span.n", {}, i + 1),
          h("span.v", {}, `${fmtWeight(s.weight_kg)} kg × ${s.reps}`),
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

  // --- add-set form ---
  const NEW = "__new__";
  const exSelect = h("select", { required: true },
    h("option", { value: "" }, exercises.length ? "Choose exercise…" : "No exercises yet"),
    exercises.map((e) => h("option", { value: e.id }, e.name)),
    h("option", { value: NEW }, "+ New exercise…"));

  // Default to the exercise last logged in this workout, else the last used.
  const lastInWorkout = w.sets.length ? w.sets[w.sets.length - 1] : null;
  const preferId = lastInWorkout ? lastInWorkout.exercise_id : lastSet.exerciseId;
  if (exercises.some((e) => e.id === preferId)) exSelect.value = preferId;

  exSelect.addEventListener("change", async () => {
    if (exSelect.value !== NEW) return;
    const n = prompt("New exercise name");
    if (!n || !n.trim()) { exSelect.value = ""; return; }
    try {
      const e = await api("POST", "/exercises", { name: n.trim() });
      lastSet.exerciseId = e.id;
      viewWorkout(id).catch(fail);
    } catch (e) { fail(e); exSelect.value = ""; }
  });

  const reps = h("input", { type: "number", inputmode: "numeric", min: 1, max: 10000, required: true, placeholder: "Reps",
    value: lastInWorkout ? lastInWorkout.reps : lastSet.reps });
  const weight = h("input", { type: "number", inputmode: "decimal", min: 0, max: 9999.99, step: "0.25", placeholder: "kg",
    value: lastInWorkout ? fmtWeight(lastInWorkout.weight_kg) : lastSet.weight });
  const addBtn = h("button.primary.block", { type: "submit" }, "Log set");

  const addForm = h("form.card.stack", {
    onsubmit: async (e) => {
      e.preventDefault();
      if (!exSelect.value || exSelect.value === NEW) return toast("Choose an exercise", true);
      addBtn.disabled = true;
      try {
        await api("POST", `/workouts/${id}/sets`, {
          exercise_id: exSelect.value,
          reps: parseInt(reps.value, 10),
          weight_kg: parseFloat(weight.value || "0"),
        });
        Object.assign(lastSet, { exerciseId: exSelect.value, reps: reps.value, weight: weight.value });
        viewWorkout(id).catch(fail);
      } catch (ex) {
        fail(ex);
        addBtn.disabled = false;
      }
    },
  },
    h("div", {}, h("label", {}, "Exercise"), exSelect),
    h("div.row", {},
      h("div", {}, h("label", {}, "Weight (kg)"), weight),
      h("div", {}, h("label", {}, "Reps"), reps)),
    addBtn);

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
    addForm,
    h("div.card", {}, h("label", {}, "Notes"), notes),
    del,
  );
}

// ---------- views: exercises ----------

async function viewExercises() {
  render(h("h1", {}, "Exercises"), h("p.muted", {}, "Loading…"));
  const exercises = await api("GET", "/exercises");

  const input = h("input", { placeholder: "e.g. Bench Press", required: true });
  const form = h("form.row", {
    style: "margin-bottom:16px",
    onsubmit: async (e) => {
      e.preventDefault();
      try {
        await api("POST", "/exercises", { name: input.value.trim() });
        viewExercises().catch(fail);
      } catch (ex) { fail(ex); }
    },
  }, input, h("button.primary.shrink", { type: "submit" }, "Add"));

  const list = exercises.length
    ? h("ul.list.card", {}, exercises.map((e) =>
        h("li", {},
          h("span", {}, e.name),
          h("button.icon", {
            "aria-label": "Delete " + e.name,
            title: "Delete",
            onclick: async () => {
              if (!confirm(`Delete "${e.name}"?`)) return;
              try {
                await api("DELETE", "/exercises/" + e.id);
                viewExercises().catch(fail);
              } catch (ex) { fail(ex); }
            },
          }, "×"))))
    : h("p.muted.empty", {}, "No exercises yet. You can also add them while logging a set.");

  render(h("h1", {}, "Exercises"), form, list);
}

route();
