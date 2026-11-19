<script lang="ts">
  import { onMount, tick } from "svelte";
  import { api, ApiError } from "../lib/api";
  import { flash } from "../lib/flash";
  import { code, fmtClock, fmtDay, fmtSession, toLocalInput } from "../lib/format";
  import { chip } from "../lib/sets";
  import type {
    Exercise,
    ExerciseOptions,
    Plan,
    Previous,
    SetBody,
    Template,
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

  let { id }: { id: string } = $props();

  let workout = $state<Workout | null>(null);
  let exercises = $state<Exercise[]>([]);
  let options = $state<ExerciseOptions | null>(null);
  let previous = $state<Previous[]>([]);
  let template = $state<Template | null>(null);
  let loadError = $state<string | null>(null);

  let activeId = $state<string | null>(null);
  let freshSetId = $state<string | null>(null);
  let toast = $state<ToastMsg | null>(null);
  let now = $state(new Date());

  let picking = $state(false);
  let removing = $state<Card | null>(null); // asked before deleting its sets
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
  interface Card {
    exId: string;
    name: string;
    type: TrackingType;
    plan: Plan | null;
    sets: WorkoutSet[];
    unavailable: boolean;
  }

  // The workout's own exercise list, in its saved order, each with its sets.
  const cards = $derived.by<Card[]>(() => {
    const groups = new Map<string, WorkoutSet[]>();
    for (const s of workout?.sets ?? []) {
      if (!groups.has(s.exercise_id)) groups.set(s.exercise_id, []);
      groups.get(s.exercise_id)!.push(s);
    }
    return (workout?.exercises ?? []).map((we) => ({
      exId: we.exercise_id,
      name: we.exercise_name,
      type: we.exercise_tracking_type,
      plan: we.target_sets != null ? { ...we, target_sets: we.target_sets } : null,
      sets: groups.get(we.exercise_id) ?? [],
      unavailable: we.exercise_archived || !byId.has(we.exercise_id),
    }));
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

  // Each exercise's mark on the route; a segment is red only between two
  // exercises that are done or under way, so an untouched exercise moved up
  // the list doesn't look walked past.
  const stops = $derived(
    cards.map((c) => (c.exId === activeId ? "active" : c.sets.length ? "done" : "todo") as "active" | "done" | "todo"),
  );
  const segment = (a: number, b: number) => (stops[a] !== "todo" && stops[b] !== "todo" ? "red" : "plan");

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

  async function pick(e: Exercise) {
    picking = false;
    if (!exercises.some((x) => x.id === e.id)) exercises = [...exercises, e];
    try {
      await api("POST", `/workouts/${id}/exercises`, { exercise_id: e.id });
      await reloadWorkout();
      activeId = e.id;
    } catch (err) {
      fail(err);
    }
  }

  // ---- removing an exercise ----

  function askRemove(card: Card) {
    if (card.sets.length) removing = card;
    else removeExercise(card);
  }

  async function removeExercise(card: Card) {
    removing = null;
    const index = cards.findIndex((c) => c.exId === card.exId);
    const sets = card.sets;
    try {
      await api("DELETE", `/workouts/${id}/exercises/${card.exId}`);
      await reloadWorkout();
      if (activeId === card.exId) activeId = pickActive(null);
      say(sets.length ? `Removed ${card.name} and ${sets.length} ${sets.length === 1 ? "set" : "sets"}` : `Removed ${card.name}`, {
        action: { label: "Undo", run: () => restoreExercise(card.exId, index, sets, card.plan) },
      });
    } catch (e) {
      fail(e);
    }
  }

  // Puts a removed exercise back where it was, with its plan and its sets
  // re-logged, in one request: it comes back whole or not at all.
  async function restoreExercise(exId: string, index: number, sets: WorkoutSet[], plan: Plan | null) {
    try {
      await api("POST", `/workouts/${id}/exercises`, {
        exercise_id: exId,
        ...(plan ?? {}),
        position: index,
        sets: sets.map((s) => ({
          reps: s.reps,
          weight_kg: s.weight_kg,
          duration_seconds: s.duration_seconds,
          distance_m: s.distance_m,
        })),
      });
      await reloadWorkout();
      activeId = pickActive(exId);
      say("Restored");
    } catch (e) {
      fail(e);
    }
  }


  // ---- reordering ----

  let listEl = $state<HTMLOListElement>();
  let announce = $state(""); // for screen readers

  // Saves a new order; the list shows it at once and rolls back on failure.
  async function saveOrder(order: string[]) {
    if (!workout) return;
    const before = workout.exercises;
    const byEx = new Map(before.map((we) => [we.exercise_id, we]));
    workout = { ...workout, exercises: order.map((x, i) => ({ ...byEx.get(x)!, position: i })) };
    try {
      await api("PUT", `/workouts/${id}/exercises`, { exercise_ids: order });
    } catch (e) {
      workout = { ...workout, exercises: before };
      fail(e);
    }
  }

  function moved(order: string[], from: number, to: number): string[] {
    const out = [...order];
    const [x] = out.splice(from, 1);
    out.splice(to, 0, x!);
    return out;
  }

  // Arrow keys on a focused handle move its exercise one place.
  async function onHandleKey(e: KeyboardEvent, index: number) {
    const to = e.key === "ArrowUp" ? index - 1 : e.key === "ArrowDown" ? index + 1 : -1;
    if (to < 0 || to >= cards.length) return;
    e.preventDefault();
    const card = cards[index]!;
    await saveOrder(moved(cards.map((c) => c.exId), index, to));
    announce = `${card.name} moved to position ${to + 1} of ${cards.length}`;
    await tick();
    listEl?.querySelector<HTMLElement>(`[data-handle="${card.exId}"]`)?.focus();
  }

  // Pointer drag: the card follows the finger; the others slide to show
  // where it will land; the order is saved on release.
  interface Drag {
    from: number;
    to: number;
    pointerId: number;
    startY: number; // page coordinates, so scrolling while dragging counts
    dy: number;
    mids: number[]; // each card's vertical middle when the drag started
    tops: number[]; // and its top and height, relative to the list
    heights: number[];
    size: number; // the dragged card's height plus the gap
  }
  let drag = $state<Drag | null>(null);
  // True for the frame the drop lands in: the list reorders and the drag
  // offsets go together, so no card animates from where it no longer is.
  let settling = $state(false);
  let scrollTimer = 0;

  function onHandleDown(e: PointerEvent, index: number) {
    if (e.button !== 0 || !listEl) return;
    const items = [...listEl.querySelectorAll<HTMLElement>(":scope > li:not(.slot)")];
    const rects = items.map((el) => el.getBoundingClientRect());
    const listTop = listEl.getBoundingClientRect().top;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    drag = {
      from: index,
      to: index,
      pointerId: e.pointerId,
      startY: e.pageY,
      dy: 0,
      mids: rects.map((r) => r.top + window.scrollY + r.height / 2),
      tops: rects.map((r) => r.top - listTop),
      heights: rects.map((r) => r.height),
      size: rects[index]!.height + 10,
    };
  }

  function onHandleMove(e: PointerEvent) {
    if (!drag || e.pointerId !== drag.pointerId) return;
    drag.dy = e.pageY - drag.startY;
    const mid = drag.mids[drag.from]! + drag.dy;
    drag.to = drag.mids.filter((m, i) => i !== drag!.from && m < mid).length;
    // Scroll when held near the top or bottom edge.
    clearInterval(scrollTimer);
    const edge = 72;
    const speed = e.clientY < edge ? -12 : e.clientY > window.innerHeight - edge - 56 ? 12 : 0;
    if (speed) {
      scrollTimer = window.setInterval(() => {
        window.scrollBy(0, speed);
        if (drag) {
          drag.dy += speed;
          const m = drag.mids[drag.from]! + drag.dy;
          drag.to = drag.mids.filter((x, i) => i !== drag!.from && x < m).length;
        }
      }, 16);
    }
  }

  function onHandleUp(e: PointerEvent) {
    if (!drag || e.pointerId !== drag.pointerId) return;
    clearInterval(scrollTimer);
    const { from, to } = drag;
    settling = true;
    drag = null;
    if (from !== to) {
      const card = cards[from]!;
      saveOrder(moved(cards.map((c) => c.exId), from, to));
      announce = `${card.name} moved to position ${to + 1} of ${cards.length}`;
    }
    requestAnimationFrame(() => requestAnimationFrame(() => (settling = false)));
  }

  // Where the dragged card will land, drawn as an empty planned slot.
  const slot = $derived.by(() => {
    if (!drag) return null;
    const { from, to, tops, heights } = drag;
    const height = heights[from]!;
    const top = to > from ? tops[to]! + heights[to]! - height : tops[to]!;
    return { top, height };
  });

  // Where each card is drawn during a drag.
  function shiftOf(i: number): number {
    if (!drag) return 0;
    if (i === drag.from) return drag.dy;
    if (drag.from < drag.to && i > drag.from && i <= drag.to) return -drag.size;
    if (drag.to < drag.from && i >= drag.to && i < drag.from) return drag.size;
    return 0;
  }

  let saving = Promise.resolve();
  function saveDetails(patch: { name?: string; notes?: string; performed_at?: string }) {
    saving = (async () => {
      try {
        const w = await api<Workout>("PUT", `/workouts/${id}`, patch);
        workout = { ...workout!, ...w, exercises: workout!.exercises, sets: workout!.sets };
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
      flash(
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
      flash(confirming === "discard" ? "Workout discarded" : "Workout deleted");
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
      flash(`Saved template “${t.name}”`);
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
      <ol class="cards" bind:this={listEl} class:dragging={drag} class:settling>
        {#each cards as c, i (c.exId)}
          <li
            class:lifted={drag?.from === i}
            style:transform={drag ? `translateY(${shiftOf(i)}px)` : null}
          >
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
            stop={stops[i]!}
            lineIn={i === 0 ? null : segment(i - 1, i)}
            lineOut={i === cards.length - 1 ? null : segment(i, i + 1)}
            {freshSetId}
            onactivate={() => (activeId = c.exId)}
            onlog={(body) => logSet(c, body)}
            onupdate={updateSet}
            ondelete={(sid) => deleteSet(sid)}
            onremove={() => askRemove(c)}
          >
            {#snippet handle()}
              {#if cards.length > 1}
                <button
                  type="button"
                  class="grip"
                  data-handle={c.exId}
                  aria-label="Move {c.name}, position {i + 1} of {cards.length}. Drag, or use the arrow keys."
                  onpointerdown={(e) => onHandleDown(e, i)}
                  onpointermove={onHandleMove}
                  onpointerup={onHandleUp}
                  onpointercancel={onHandleUp}
                  onkeydown={(e) => onHandleKey(e, i)}
                >
                  <svg viewBox="0 0 20 20" aria-hidden="true">
                    <circle cx="7" cy="5" r="1.6" /><circle cx="13" cy="5" r="1.6" />
                    <circle cx="7" cy="10" r="1.6" /><circle cx="13" cy="10" r="1.6" />
                    <circle cx="7" cy="15" r="1.6" /><circle cx="13" cy="15" r="1.6" />
                  </svg>
                </button>
              {/if}
            {/snippet}
          </ExerciseCard>
          </li>
        {/each}
        {#if slot}
          <li class="slot" aria-hidden="true" style:top="{slot.top}px" style:height="{slot.height}px"></li>
        {/if}
      </ol>
      <p class="sr-only" aria-live="polite">{announce}</p>
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

<Sheet
  bind:open={() => removing !== null, (v) => !v && (removing = null)}
  title="Remove {removing?.name ?? 'exercise'}?"
>
  {#if removing}
    {@const n = removing.sets.length}
    <p class="ask">This also deletes the {n} {n === 1 ? "set" : "sets"} logged for it in this workout.</p>
    <div class="two">
      <Button onclick={() => (removing = null)}>Keep it</Button>
      <Button variant="ink" onclick={() => removing && removeExercise(removing)}>Remove</Button>
    </div>
  {/if}
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
    position: relative;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  /* .cards.settling outranks .cards > li's transition below. */
  .cards.settling > li {
    transition: none;
  }
  .cards > li {
    position: relative;
    transition: transform 160ms var(--ease);
  }
  /* The card being dragged follows the finger, lifted over the rest. */
  .cards > li.lifted {
    z-index: 5;
    transition: none;
  }
  /* Lifted: a raised-surface rule (2px, like sheets and the tab bar) on the
     highlight paper; 2.5px stays the active card's. No shadow. */
  .cards > li.lifted :global(.item:not(.active) .card) {
    border-width: 2px;
    background: var(--paper-hi);
  }
  .cards > li.lifted .grip {
    color: var(--ink);
  }
  /* The landing slot: the world's dashed "planned" box, with its own stretch
     of dashed route in the gutter so the line doesn't break. */
  .cards > li.slot {
    position: absolute;
    left: 26px;
    right: 0;
    border: 1.5px dashed var(--ink-2);
    border-radius: var(--r);
    pointer-events: none;
  }
  .cards > li.slot::before {
    content: "";
    position: absolute;
    left: -19px;
    top: 0;
    bottom: -10px; /* through the gap below, as each card's own segment does */
    border-left: 1.5px dashed var(--ink-2);
  }
  .dragging {
    user-select: none;
  }
  .grip {
    flex: none;
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    margin: 2px 2px 0 0;
    border: 0;
    background: none;
    color: var(--ink-2);
    cursor: grab;
    touch-action: none; /* the finger drags the card, not the page */
  }
  .grip:active {
    cursor: grabbing;
    color: var(--ink);
  }
  .grip svg {
    width: 20px;
    height: 20px;
    fill: currentColor;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
  @media (prefers-reduced-motion: reduce) {
    .cards > li {
      transition: none;
    }
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
