<script lang="ts" module>
  export interface Host {
    // Shown by the screen navigated to (this one is gone by then).
    toast: (msg: string) => void;
  }
</script>

<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "../lib/api";
  import { code, fmtClock, fmtDay, fmtSession, toLocalInput } from "../lib/format";
  import { chip } from "../lib/sets";
  import type {
    Exercise,
    ExerciseOptions,
    Previous,
    SetBody,
    Template,
    TemplateExercise,
    TrackingType,
    Workout,
    WorkoutSet,
  } from "../lib/types";
  import ExerciseCard from "./ExerciseCard.svelte";
  import ExercisePicker from "./ExercisePicker.svelte";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import TabBar from "../ui/TabBar.svelte";
  import Toast, { type ToastMsg } from "../ui/Toast.svelte";

  let { id, host }: { id: string; host: Host } = $props();

  let workout = $state<Workout | null>(null);
  let exercises = $state<Exercise[]>([]);
  let options = $state<ExerciseOptions | null>(null);
  let previous = $state<Previous[]>([]);
  let template = $state<Template | null>(null);
  let loadError = $state<string | null>(null);

  let activeId = $state<string | null>(null);
  let added = $state<string[]>([]); // picked, nothing logged yet
  let freshSetId = $state<string | null>(null);
  let toast = $state<ToastMsg | null>(null);
  let now = $state(new Date());

  let picking = $state(false);
  let confirming = $state<null | "discard" | "delete">(null);
  let naming = $state(false);
  let templateName = $state("");
  let busy = $state(false);

  let toastSeq = 0;
  function say(text: string, extra: Partial<ToastMsg> = {}) {
    toast = { id: ++toastSeq, text, ...extra };
  }
  function fail(e: unknown) {
    console.error(e);
    say(e instanceof Error ? e.message : "Something went wrong", { error: true });
  }

  // ---- data ----

  async function load() {
    try {
      const [w, ex, opts, prev] = await Promise.all([
        api<Workout>("GET", `/workouts/${id}`),
        api<Exercise[]>("GET", "/exercises"),
        api<ExerciseOptions>("GET", "/exercise-options", undefined, { auth: false }),
        api<Previous[]>("GET", `/workouts/${id}/previous`),
      ]);
      // The plan, if started from a template that still exists.
      template = w.template_id
        ? await api<Template>("GET", `/templates/${w.template_id}`).catch((e) =>
            e instanceof ApiError && e.status === 404 ? null : Promise.reject(e),
          )
        : null;
      workout = w;
      exercises = ex;
      options = opts;
      previous = prev;
      activeId = pickActive(null);
    } catch (e) {
      loadError = e instanceof Error ? e.message : "Couldn't load this workout";
    }
  }

  async function reloadWorkout() {
    workout = await api<Workout>("GET", `/workouts/${id}`);
  }

  onMount(() => {
    load();
    const t = setInterval(() => (now = new Date()), 30_000);
    return () => clearInterval(t);
  });

  // ---- derived ----

  const byId = $derived(new Map(exercises.map((e) => [e.id, e])));
  const prevBy = $derived(new Map(previous.map((p) => [p.exercise_id, p])));
  const plan = $derived<TemplateExercise[]>(template?.exercises ?? []);

  interface Card {
    exId: string;
    name: string;
    type: TrackingType;
    plan: TemplateExercise | null;
    sets: WorkoutSet[];
    unavailable: boolean;
  }

  // Planned exercises in template order, then anything logged off-plan in
  // the order first logged, then exercises picked but not logged yet.
  const cards = $derived.by<Card[]>(() => {
    const groups = new Map<string, WorkoutSet[]>();
    for (const s of workout?.sets ?? []) {
      if (!groups.has(s.exercise_id)) groups.set(s.exercise_id, []);
      groups.get(s.exercise_id)!.push(s);
    }
    const out: Card[] = [];
    const seen = new Set<string>();
    const push = (exId: string, name: string, type: TrackingType, te: TemplateExercise | null) => {
      if (seen.has(exId)) return;
      seen.add(exId);
      out.push({
        exId,
        name,
        type,
        plan: te,
        sets: groups.get(exId) ?? [],
        unavailable: !byId.has(exId) || !!te?.exercise_archived,
      });
    };
    for (const te of plan) push(te.exercise_id, te.exercise_name, te.exercise_tracking_type, te);
    for (const [exId, sets] of groups) push(exId, sets[0]!.exercise_name, sets[0]!.exercise_tracking_type, null);
    for (const exId of added) {
      const e = byId.get(exId);
      if (e) push(e.id, e.name, e.tracking_type, null);
    }
    return out;
  });

  const open = (c: Card) => !c.unavailable && (!c.plan || c.sets.length < c.plan.target_sets);

  // Stay on the current exercise until its planned sets are done, then the
  // next unfinished one; without a plan, wherever the last set was logged.
  function pickActive(current: string | null): string | null {
    const sets = workout?.sets ?? [];
    const lastLogged = sets.length ? sets[sets.length - 1]!.exercise_id : null;
    const cur = cards.find((c) => c.exId === (current ?? lastLogged));
    if (cur && open(cur)) return cur.exId;
    const next = cards.find((c) => c.plan && open(c));
    return next?.exId ?? cur?.exId ?? cards.find((c) => !c.unavailable)?.exId ?? null;
  }

  const activeIndex = $derived(cards.findIndex((c) => c.exId === activeId));
  // The route runs red as far as the furthest exercise touched.
  const routeEnd = $derived(
    Math.max(activeIndex, ...cards.map((c, i) => (c.sets.length ? i : -1))),
  );

  const doneSets = $derived(workout?.sets.length ?? 0);
  const plannedSets = $derived(
    cards.reduce((n, c) => n + Math.max(c.plan?.target_sets ?? 0, c.sets.length), 0),
  );

  const title = $derived(workout?.name?.trim() || template?.name || "Workout");

  // ---- actions ----

  async function logSet(card: Card, body: SetBody) {
    const label = `${code(cards.findIndex((c) => c.exId === card.exId))}${card.sets.length + 1}`;
    try {
      const s = await api<WorkoutSet>("POST", `/workouts/${id}/sets`, body);
      await reloadWorkout();
      added = added.filter((x) => x !== card.exId);
      freshSetId = s.id;
      say(`Logged ${label} · ${chip(s)}`, {
        action: {
          label: "Undo",
          run: () => {
            activeId = card.exId; // back to where the set came from, to re-log it
            deleteSet(s.id, true);
          },
        },
      });
      // Planned sets for this exercise done: move along the route.
      const updated = cards.find((c) => c.exId === card.exId);
      if (updated && updated.plan && !open(updated)) activeId = pickActive(null);
    } catch (e) {
      fail(e);
    }
  }

  async function updateSet(setId: string, body: SetBody) {
    try {
      await api("PUT", `/workouts/${id}/sets/${setId}`, body);
      await reloadWorkout();
      say("Set updated");
    } catch (e) {
      fail(e);
    }
  }

  async function deleteSet(setId: string, undo = false) {
    try {
      await api("DELETE", `/workouts/${id}/sets/${setId}`);
      await reloadWorkout();
      say(undo ? "Set removed" : "Set deleted");
    } catch (e) {
      fail(e);
    }
  }

  function pick(e: Exercise) {
    picking = false;
    if (!exercises.some((x) => x.id === e.id)) exercises = [...exercises, e];
    if (!cards.some((c) => c.exId === e.id)) added = [...added, e.id];
    activeId = e.id;
  }

  let saving = Promise.resolve();
  function saveDetails(patch: { name?: string; notes?: string; performed_at?: string }) {
    saving = (async () => {
      try {
        const w = await api<Workout>("PUT", `/workouts/${id}`, patch);
        workout = { ...workout!, ...w, sets: workout!.sets };
        say("Saved");
      } catch (e) {
        fail(e);
      }
    })();
  }

  // Sets are saved as they're logged, so finishing only stamps the end time;
  // an empty workout is offered for discarding instead.
  async function finish() {
    await saving;
    if (!workout) return;
    if (!workout.sets.length) {
      confirming = "discard";
      return;
    }
    busy = true;
    try {
      const done = await api<Workout>("POST", `/workouts/${id}/finish`);
      const n = workout.sets.length;
      host.toast(
        `Workout saved · ${n} set${n === 1 ? "" : "s"}` +
          (done.finished_at ? " · " + fmtSession(done.performed_at, done.finished_at) : ""),
      );
      location.hash = "#/";
    } catch (e) {
      fail(e);
    } finally {
      busy = false;
    }
  }

  async function removeWorkout() {
    busy = true;
    try {
      await api("DELETE", `/workouts/${id}`);
      host.toast(confirming === "discard" ? "Workout discarded" : "Workout deleted");
      confirming = null;
      location.hash = "#/";
    } catch (e) {
      fail(e);
    } finally {
      busy = false;
    }
  }

  async function saveAsTemplate() {
    busy = true;
    try {
      const t = await api<Template>("POST", `/workouts/${id}/save-as-template`, { name: templateName.trim() });
      naming = false;
      host.toast(`Saved template “${t.name}”`);
      location.hash = `#/templates/${t.id}`;
    } catch (e) {
      fail(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="screen" class:flush={!cards.length}>
  <header class="top">
    <a class="back" href="#/">
      <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12.5 4.5L7 10l5.5 5.5" /></svg>
      Workouts
    </a>
    {#if workout}
      <Button size="sm" disabled={busy} onclick={finish}>{workout.finished_at ? "Done" : "Finish"}</Button>
    {/if}
  </header>

  {#if loadError}
    <p class="state" role="alert">{loadError}</p>
    <Button onclick={() => ((loadError = null), load())}>Try again</Button>
  {:else if !workout || !options}
    <p class="state" aria-busy="true">Loading workout…</p>
  {:else}
    <h1>{title}</h1>
    <p class="meta">
      <span>
        {fmtDay(workout.performed_at)} · {fmtClock(workout.performed_at)} ·
        {workout.finished_at
          ? "took " + fmtSession(workout.performed_at, workout.finished_at)
          : fmtSession(workout.performed_at, now)}
      </span>
      <b>{doneSets}{plannedSets > doneSets ? ` / ${plannedSets}` : ""} {plannedSets === 1 || (!plannedSets && doneSets === 1) ? "set" : "sets"}</b>
    </p>

    {#if cards.length}
      <ol class="cards">
        {#each cards as c, i (c.exId)}
          <ExerciseCard
            exerciseId={c.exId}
            code={code(i)}
            name={c.name}
            type={c.type}
            plan={c.plan}
            sets={c.sets}
            last={prevBy.get(c.exId)}
            unavailable={c.unavailable}
            active={c.exId === activeId}
            stop={c.exId === activeId ? "active" : c.sets.length ? "done" : "todo"}
            lineIn={i === 0 ? null : i <= routeEnd ? "red" : "plan"}
            lineOut={i === cards.length - 1 ? null : i < routeEnd ? "red" : "plan"}
            {freshSetId}
            onactivate={() => (activeId = c.exId)}
            onlog={(body) => logSet(c, body)}
            onupdate={updateSet}
            ondelete={(sid) => deleteSet(sid)}
          />
        {/each}
      </ol>
    {:else}
      <p class="state">Nothing logged yet. Add the first exercise.</p>
    {/if}

    <div class="add">
      <Button variant={cards.length ? "ghost" : "ink"} wide onclick={() => (picking = true)}>
        <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M10 4v12" /></svg>
        Add exercise
      </Button>
    </div>

    <details class="details">
      <summary>Workout details</summary>
      <div class="details-body">
      <Field label="Name">
        <input
          value={workout.name ?? ""}
          placeholder="e.g. Push day"
          onchange={(e) => saveDetails({ name: e.currentTarget.value })}
        />
      </Field>
      <Field label="Started">
        <input
          type="datetime-local"
          value={toLocalInput(workout.performed_at)}
          onchange={(e) =>
            e.currentTarget.value && saveDetails({ performed_at: new Date(e.currentTarget.value).toISOString() })}
        />
      </Field>
      <Field label="Notes">
        <textarea rows="3" value={workout.notes ?? ""} onchange={(e) => saveDetails({ notes: e.currentTarget.value })}
        ></textarea>
      </Field>
      {#if template}
        <p class="from">From template <a href="#/templates/{template.id}">{template.name}</a></p>
      {/if}
      {#if workout.sets.length && !template}
        <Button
          wide
          onclick={() => {
            templateName = workout?.name ?? "";
            naming = true;
          }}>Save as template</Button
        >
      {/if}
      <div class="apart">
        <Button variant="danger" onclick={() => (confirming = "delete")}>Delete workout</Button>
      </div>
      </div>
    </details>
  {/if}
</div>

<TabBar active="workouts" />
<Toast bind:toast />

{#if options}
  <Sheet bind:open={picking} title="Add exercise">
    <ExercisePicker
      {exercises}
      {options}
      taken={new Set(cards.map((c) => c.exId))}
      onpick={pick}
      onerror={fail}
    />
  </Sheet>
{/if}

<Sheet
  bind:open={() => confirming !== null, (v) => !v && (confirming = null)}
  title={confirming === "discard" ? "Discard workout?" : "Delete workout?"}
>
  <p class="ask">
    {confirming === "discard"
      ? "No sets were logged, so there's nothing to save."
      : `This deletes “${title}” and all ${doneSets} of its sets.`}
  </p>
  <div class="two">
    <Button onclick={() => (confirming = null)}>Keep it</Button>
    <Button variant="ink" disabled={busy} onclick={removeWorkout}>
      {confirming === "discard" ? "Discard" : "Delete"}
    </Button>
  </div>
</Sheet>

<Sheet bind:open={naming} title="Save as template">
  <form
    onsubmit={(e) => {
      e.preventDefault();
      saveAsTemplate();
    }}
  >
    <Field label="Template name">
      <input bind:value={templateName} required maxlength="100" />
    </Field>
    <div class="two">
      <Button onclick={() => (naming = false)}>Cancel</Button>
      <Button type="submit" variant="ink" disabled={busy || !templateName.trim()}>Save</Button>
    </div>
  </form>
</Sheet>

<style>
  .screen {
    max-width: 34rem;
    margin: 0 auto;
    padding: 0 12px calc(90px + env(safe-area-inset-bottom));
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: max(8px, env(safe-area-inset-top));
    min-height: 56px;
  }
  .back {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    min-height: 44px;
    padding-right: 8px;
    margin-left: -6px;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    text-decoration: none;
  }
  .back svg {
    width: 20px;
    height: 20px;
  }
  .back path,
  .add path {
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  h1 {
    margin: 8px 0 8px;
    font: 400 clamp(52px, 15vw, 72px) / 0.92 var(--gothic);
    text-transform: uppercase;
    letter-spacing: 0.005em;
    overflow-wrap: anywhere;
    text-wrap: balance;
  }
  .meta {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 12px;
    margin: 0 0 16px;
    padding-top: 8px;
    border-top: 2px solid var(--red);
    color: var(--ink-2);
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .meta b {
    flex: none;
    color: var(--ink);
  }
  .cards {
    margin: 0;
    padding: 0;
  }
  .state {
    margin: 32px 0 16px;
    font-weight: 600;
  }
  .flush .add,
  .flush .details {
    width: 100%;
    margin-left: 0;
  }
  .add {
    width: calc(100% - 26px);
    margin: 4px 0 0 26px;
  }
  .details {
    margin: 28px 0 0 26px;
    border-top: 1.5px solid var(--ink);
  }
  /* <details> can't lay out its own contents as a grid, so a wrapper does. */
  .details-body {
    display: grid;
    gap: 16px;
    padding-bottom: 8px;
  }
  summary {
    min-height: 48px;
    display: flex;
    align-items: center;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    cursor: pointer;
  }
  form {
    display: grid;
    gap: 14px;
  }
  /* Delete stays well away from everything else. */
  .apart {
    display: flex;
    justify-content: flex-end;
    margin-top: 22px;
  }
  .from {
    margin: 0;
    font-weight: 600;
  }
  .ask {
    margin: 0 0 16px;
    font-weight: 600;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
</style>
