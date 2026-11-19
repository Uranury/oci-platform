// Lift Log frontend (pre-redesign): plain JS, bundled as-is by Vite. Hash routing:
//   #/login  #/  #/workouts/<id>  #/templates  #/templates/<id|new>  #/exercises

// ---------- API ----------

// Shared with the Svelte screens: one token store, one refresh in flight.
import { api, logout, store } from "../lib/api";
import { takeFlash } from "../lib/flash";
import { hideScreen, showScreen } from "../main";

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

function fmtWeight(kg) {
  return (Math.round(kg * 100) / 100).toString();
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
  if (!loggedIn && hash !== "#/login") return go("#/login");
  if (loggedIn && hash === "#/login") return go("#/");

  const topbar = document.getElementById("topbar");
  const legacy = legacyView(hash);
  if (!legacy) {
    // Redesigned screens render through Svelte into #screen.
    app.hidden = true;
    topbar.hidden = true;
    const m = hash.match(/^#\/workouts\/([0-9a-f-]{36})$/);
    return showScreen(document.getElementById("screen"),
      hash === "#/login" ? { kind: "login" }
        : hash === "#/exercises" ? { kind: "exercises" }
          : m ? { kind: "workout", id: m[1] } : { kind: "home" });
  }
  hideScreen();
  app.hidden = false;
  topbar.hidden = false;
  const f = takeFlash();
  if (f) toast(f);
  legacy();
}

// The screens still rendered by this file, or null for the Svelte ones.
function legacyView(hash) {
  let m;
  if (hash === "#/templates") return () => (nav("templates"), viewTemplates().catch(fail));
  if (hash === "#/templates/new") return () => (nav("templates"), viewTemplate(null).catch(fail));
  if ((m = hash.match(/^#\/templates\/([0-9a-f-]{36})$/))) return () => (nav("templates"), viewTemplate(m[1]).catch(fail));
  return null;
}

function nav(active) {
  document.querySelectorAll("[data-nav]").forEach((a) => {
    a.classList.toggle("active", a.dataset.nav === active);
  });
}

window.addEventListener("hashchange", route);

document.getElementById("logout").addEventListener("click", logout);

// ---------- helpers for the template views ----------

function fmtDuration(sec) {
  const m = Math.floor(sec / 60), s = sec % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

function select(list, value, { placeholder, required } = {}) {
  return h("select", { required },
    placeholder ? h("option", { value: "" }, placeholder) : null,
    list.map((o) => h("option", { value: o.value, selected: o.value === value }, o.label)));
}

function field(label, control, hint) {
  return h("div", {}, h("label", {}, label), control, hint ? h("p.hint", {}, hint) : null);
}

// "1:30" or plain seconds -> seconds; used by template targets.
function parseDuration(str) {
  const t = str.trim();
  if (!t) return null;
  if (t.includes(":")) {
    const [m, s] = t.split(":").map((x) => parseInt(x || "0", 10));
    return m * 60 + s;
  }
  return parseInt(t, 10); // plain number = seconds
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

route();
