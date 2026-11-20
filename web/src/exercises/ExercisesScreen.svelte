<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "../lib/api";
  import { exerciseMeta, label, toNewExercise } from "../lib/exercises";
  import { takeFlash } from "../lib/flash";
  import { fmtLast, fmtSets } from "../lib/format";
  import type { Exercise, ExerciseOptions, NewExercise, Previous, TrackingType } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import TabBar from "../ui/TabBar.svelte";
  import Toast, { type ToastMsg } from "../ui/Toast.svelte";
  import ExerciseDetail from "./ExerciseDetail.svelte";
  import ExerciseForm from "./ExerciseForm.svelte";

  let exercises = $state<Exercise[] | null>(null);
  let options = $state<ExerciseOptions | null>(null);
  let last = $state<Map<string, Previous>>(new Map());
  let loadError = $state<string | null>(null);

  let tab = $state<"mine" | "library">("library");
  let query = $state("");
  let muscle = $state(""); // "" = all

  // The sheet: one exercise's detail or edit form, or the new-exercise form.
  let openId = $state<string | null>(null);
  let editing = $state(false);
  let creating = $state(false);

  let toast = $state<ToastMsg | null>(null);
  let toastSeq = 0;
  function say(text: string, error = false) {
    toast = { id: ++toastSeq, text, error };
  }
  function fail(e: unknown) {
    say(e instanceof Error ? e.message : "Something went wrong", true);
  }

  async function load() {
    loadError = null;
    try {
      const [ex, opts, prev] = await Promise.all([
        api<Exercise[]>("GET", "/exercises"),
        api<ExerciseOptions>("GET", "/exercise-options", undefined, { auth: false }),
        api<Previous[]>("GET", "/exercises/last"),
      ]);
      exercises = ex;
      options = opts;
      last = new Map(prev.map((p) => [p.exercise_id, p]));
    } catch (e) {
      loadError = e instanceof Error ? e.message : "Couldn't load the exercises";
    }
  }

  onMount(() => {
    const f = takeFlash();
    if (f) say(f);
    load();
  });

  const q = $derived(query.trim().toLowerCase());
  const matches = $derived(
    (exercises ?? []).filter((e) => {
      if (muscle && e.primary_muscle !== muscle && !e.secondary_muscles.includes(muscle)) return false;
      if (!q) return true;
      return e.name.toLowerCase().includes(q) || (options && exerciseMeta(options, e).toLowerCase().includes(q));
    }),
  );
  const mine = $derived(matches.filter((e) => !e.system));
  const library = $derived(matches.filter((e) => e.system));
  const shown = $derived(tab === "mine" ? mine : library);
  const totalMine = $derived((exercises ?? []).filter((e) => !e.system).length);
  const totalLibrary = $derived((exercises ?? []).length - totalMine);
  const open = $derived(openId ? (exercises?.find((e) => e.id === openId) ?? null) : null);

  // ---- actions ----

  function close() {
    openId = null;
    editing = false;
    creating = false;
  }

  async function create(body: NewExercise) {
    const e = await api<Exercise>("POST", "/exercises", body);
    exercises = [...(exercises ?? []), e];
    tab = "mine";
    close();
    say(`Created “${e.name}”`);
  }

  async function update(body: NewExercise) {
    if (!open) return;
    const id = open.id;
    await api("PUT", `/exercises/${id}`, body);
    exercises = (exercises ?? []).map((e) =>
      e.id === id ? { ...e, ...body, tracking_type: body.tracking_type as TrackingType } : e,
    );
    editing = false;
    say("Saved");
  }

  // Your own copy of a default, opened for editing.
  async function customize() {
    if (!open) return;
    try {
      const cp = await api<Exercise>("POST", `/exercises/${open.id}/duplicate`, {});
      exercises = [...(exercises ?? []), cp];
      tab = "mine";
      openId = cp.id;
      editing = true;
      say(`Copied “${cp.name}” to your exercises`);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        say(`You already have an exercise called “${open.name}”`, true);
      } else {
        fail(e);
      }
    }
  }

  async function remove() {
    if (!open) return;
    const { id, name } = open;
    try {
      const r = await api<{ result: "deleted" | "archived" }>("DELETE", `/exercises/${id}`);
      exercises = (exercises ?? []).filter((e) => e.id !== id);
      close();
      say(r.result === "archived" ? `Archived “${name}”: still in your history` : `Deleted “${name}”`);
    } catch (e) {
      fail(e);
    }
  }
</script>

<div class="screen">
  <header class="top">
    <span class="brand">Overlift</span>
    {#if options}
      <Button size="sm" onclick={() => (creating = true)}>
        <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M10 4v12" /></svg>
        New
      </Button>
    {/if}
  </header>

  <h1>Exercises</h1>
  {#if exercises}
    <p class="meta">
      <span>{totalLibrary} in the library · {totalMine} yours</span>
      <b>{last.size} trained</b>
    </p>
  {/if}

  {#if loadError}
    <p class="state" role="alert">{loadError}</p>
    <Button onclick={load}>Try again</Button>
  {:else if !exercises || !options}
    <p class="state" aria-busy="true">Loading…</p>
  {:else}
    <Field label="Find exercise" hideLabel>
      <input type="search" bind:value={query} placeholder="Search name, equipment, muscle" autocomplete="off" />
    </Field>

    <div class="chips" role="group" aria-label="Muscle">
      <button type="button" class="chip" aria-pressed={muscle === ""} onclick={() => (muscle = "")}>All</button>
      {#each options.muscles as m (m.value)}
        <button type="button" class="chip" aria-pressed={muscle === m.value} onclick={() => (muscle = m.value)}
          >{m.label}</button
        >
      {/each}
    </div>

    <div class="switch" role="group" aria-label="Whose exercises">
      <button type="button" aria-pressed={tab === "mine"} onclick={() => (tab = "mine")}>Mine · {mine.length}</button>
      <button type="button" aria-pressed={tab === "library"} onclick={() => (tab = "library")}
        >Library · {library.length}</button
      >
    </div>

    {#if shown.length}
      <ul class="rows">
        {#each shown as e (e.id)}
          {@const l = last.get(e.id)}
          <li>
            <button type="button" class="row" onclick={() => (openId = e.id)}>
              <span class="name">{e.name}</span>
              <span class="cat">{exerciseMeta(options, e)}</span>
              {#if l}
                <span class="last">{fmtLast(l.performed_at)} · {fmtSets(l.sets)}</span>
              {:else}
                <span class="last none">Not done yet</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {:else if q || muscle}
      <p class="empty">
        Nothing in {tab === "mine" ? "your exercises" : "the library"} matches{q ? ` “${query.trim()}”` : ""}{muscle
          ? ` for ${label(options.muscles, muscle).toLowerCase()}`
          : ""}.
      </p>
    {:else if tab === "mine"}
      <p class="empty">
        None yet. Customize one from the library to make it yours, or create your own with New.
      </p>
    {/if}
  {/if}
</div>

<TabBar active="exercises" />
<Toast bind:toast />

{#if options}
  <Sheet
    bind:open={() => open !== null, (v) => !v && close()}
    title={editing ? "Edit exercise" : (open?.name ?? "")}
  >
    {#if open && editing}
      <ExerciseForm
        {options}
        initial={toNewExercise(open)}
        creating={false}
        submitLabel="Save"
        onsave={update}
        oncancel={() => (editing = false)}
      />
    {:else if open}
      <ExerciseDetail
        exercise={open}
        {options}
        onedit={() => (editing = true)}
        oncustomize={customize}
        ondelete={remove}
      />
    {/if}
  </Sheet>

  <Sheet bind:open={creating} title="New exercise">
    <ExerciseForm
      {options}
      initial={{ name: query.trim() }}
      creating
      submitLabel="Create"
      onsave={create}
      oncancel={() => (creating = false)}
      onuse={(e) => {
        // The default it's close to: open it, where Customize is.
        creating = false;
        tab = "library";
        openId = e.id;
      }}
      useText={() => "Open it"}
    />
  </Sheet>
{/if}

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
    min-height: 56px;
    padding-top: max(8px, env(safe-area-inset-top));
  }
  .top svg path {
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
  }
  .brand {
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  h1 {
    margin: 2px 0 8px;
    font: 400 clamp(52px, 15vw, 72px) / 0.92 var(--gothic);
    text-transform: uppercase;
  }
  .meta {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin: 0 0 12px;
    padding-top: 8px;
    border-top: 2px solid var(--red);
    color: var(--ink-2);
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .meta b {
    color: var(--ink);
    white-space: nowrap;
  }
  .state {
    margin: 24px 0 16px;
    font-weight: 600;
  }

  /* One scrolling row; the edge fades so it's clear there's more. */
  .chips {
    display: flex;
    gap: 6px;
    margin: 10px -12px 0;
    padding: 0 12px 2px;
    overflow-x: auto;
    scrollbar-width: none;
    mask-image: linear-gradient(to right, #000 calc(100% - 24px), transparent);
  }
  .chip {
    flex: none;
    min-height: 40px;
    padding: 0 12px;
    border: 1px solid var(--rule);
    border-radius: 4px;
    background: transparent;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .chip[aria-pressed="true"] {
    border: 1.5px solid var(--ink);
    background: var(--paper-hi);
    font-weight: 700;
  }

  .switch {
    display: grid;
    grid-template-columns: 1fr 1fr;
    margin-top: 12px;
    border: var(--line);
    border-radius: var(--r);
    overflow: hidden;
  }
  .switch button {
    min-height: 48px;
    border: 0;
    background: transparent;
    color: var(--ink-2);
    font-size: 13px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .switch button + button {
    border-left: var(--line);
  }
  .switch button[aria-pressed="true"] {
    background: var(--paper-hi);
    color: var(--ink);
    text-decoration: underline;
    text-decoration-thickness: 2px;
    text-underline-offset: 6px;
  }

  .rows {
    margin: 8px 0 0;
    padding: 0;
    list-style: none;
  }
  .row {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    width: 100%;
    min-height: 56px;
    padding: 8px 0;
    border: 0;
    border-top: 1.5px solid var(--rule);
    background: none;
    text-align: left;
    cursor: pointer;
  }
  .row:active {
    background: var(--paper-hi);
  }
  .name {
    font: 400 26px/1 var(--gothic);
    text-transform: uppercase;
    overflow-wrap: anywhere;
  }
  .cat,
  .last {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .cat {
    color: var(--ink-2);
  }
  .last {
    font-weight: 700;
  }
  .last.none {
    color: var(--ink-2);
    font-weight: 500;
  }
  .empty {
    margin: 20px 0;
    color: var(--ink-2);
  }
</style>
