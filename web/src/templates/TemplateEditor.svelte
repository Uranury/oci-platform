<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { api } from "../lib/api";
  import { flash } from "../lib/flash";
  import { go, setLeaveGuard } from "../lib/nav";
  import { code } from "../lib/format";
  import type { Field as StepField } from "../lib/sets";
  import {
    MAX_NAME,
    MAX_NOTE,
    TARGET_INPUT,
    TARGETS,
    draftItem,
    itemBody,
    itemErrors,
    toDraft,
    type DraftItem,
  } from "../lib/templates";
  import type { Exercise, ExerciseOptions, Template } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import Stepper from "../ui/Stepper.svelte";
  import TabBar from "../ui/TabBar.svelte";
  import Toast, { type ToastMsg } from "../ui/Toast.svelte";
  import ExercisePicker from "../workout/ExercisePicker.svelte";

  // The template editor; id null = a new template. Everything is local until
  // Save, which sends the whole plan (PUT replaces the exercise list).
  let { id }: { id: string | null } = $props();

  const SETS: StepField = { key: "reps", label: "Sets", unit: "", required: true, step: 1, min: 1, integer: true };

  let name = $state("");
  let notes = $state("");
  let items = $state<DraftItem[]>([]);
  let original = ""; // the saved state, to tell whether anything changed
  let loaded = $state(false);
  let loadError = $state<string | null>(null);
  let exercises = $state<Exercise[]>([]);
  let options = $state<ExerciseOptions | null>(null);

  let picking = $state(false);
  let saving = $state(false);
  let tried = $state(false); // Save pressed once: show what's wrong
  let leavingTo = $state<string | null>(null);
  let announce = $state("");
  let toast = $state<ToastMsg | null>(null);
  let toastSeq = 0;

  function fail(e: unknown) {
    toast = { id: ++toastSeq, text: e instanceof Error ? e.message : "Something went wrong", error: true };
  }

  const snapshot = () =>
    JSON.stringify([name.trim(), notes.trim(), items.map(({ key: _, ...rest }) => rest)]);

  async function load() {
    loadError = null;
    try {
      const [t, ex, opts] = await Promise.all([
        id ? api<Template>("GET", `/templates/${id}`) : Promise.resolve(null),
        api<Exercise[]>("GET", "/exercises"),
        api<ExerciseOptions>("GET", "/exercise-options", undefined, { auth: false }),
      ]);
      if (t) {
        name = t.name;
        notes = t.notes ?? "";
        items = t.exercises.map(toDraft);
      }
      exercises = ex;
      options = opts;
      original = snapshot();
      loaded = true;
    } catch (e) {
      loadError = e instanceof Error ? e.message : "Couldn't load this template";
    }
  }

  onMount(() => {
    load();
    setLeaveGuard({
      dirty: () => loaded && !saving && snapshot() !== original,
      ask: (to) => (leavingTo = to),
    });
  });
  onDestroy(() => setLeaveGuard(null));

  const back = $derived(id ? `#/templates/${id}` : "#/templates");
  const errors = $derived(items.map(itemErrors));
  const nameError = $derived(name.trim() ? null : "Give the template a name");
  const problems = $derived(
    (nameError ? 1 : 0) + errors.reduce((n, e) => n + Object.keys(e).length, 0),
  );
  const totalSets = $derived(items.reduce((n, it) => n + (it.sets ?? 0), 0));

  function pick(e: Exercise) {
    picking = false;
    if (!exercises.some((x) => x.id === e.id)) exercises = [...exercises, e];
    items.push(draftItem(e));
    announce = `Added ${e.name} as ${code(items.length - 1)}`;
  }

  async function move(i: number, d: -1 | 1) {
    const j = i + d;
    if (j < 0 || j >= items.length) return;
    const it = items[i]!;
    [items[i], items[j]] = [items[j]!, items[i]!];
    announce = `${it.name} moved to ${code(j)}, position ${j + 1} of ${items.length}`;
    await tick();
    // Keep focus on the moved exercise, on whichever arrow still works.
    const btn =
      document.querySelector<HTMLButtonElement>(`[data-move="${it.key}:${d}"]:not(:disabled)`) ??
      document.querySelector<HTMLButtonElement>(`[data-move="${it.key}:${-d}"]`);
    btn?.focus();
  }

  // Notes are rare: an empty one stays a link until asked for.
  let noting = $state(new Set<number>());
  async function addNote(key: number) {
    noting = new Set(noting).add(key);
    await tick();
    document.getElementById(`note-${key}`)?.focus();
  }

  function remove(i: number) {
    const [it] = items.splice(i, 1);
    if (it) announce = `Removed ${it.name}`;
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    tried = true;
    if (problems) {
      await tick();
      document.querySelector<HTMLElement>("[aria-invalid='true']")?.focus();
      return;
    }
    saving = true;
    try {
      const body = { name: name.trim(), notes: notes.trim(), exercises: items.map(itemBody) };
      const t = id
        ? await api<Template>("PUT", `/templates/${id}`, body)
        : await api<Template>("POST", "/templates", body);
      original = snapshot();
      flash(id ? "Plan saved" : `Created “${t.name}”`);
      go(`#/templates/${t.id}`);
    } catch (err) {
      fail(err);
    } finally {
      saving = false;
    }
  }

  function leave() {
    const to = leavingTo;
    leavingTo = null;
    original = snapshot(); // nothing left to guard
    if (to) go(to);
  }
</script>

{#snippet arrow(up: boolean)}
  <svg viewBox="0 0 20 20" aria-hidden="true">
    <path d={up ? "M5 12.5L10 7l5 5.5" : "M5 7.5L10 13l5-5.5"} />
  </svg>
{/snippet}

<div class="screen">
  <header class="top">
    <a class="back" href={back}>
      <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12.5 4.5L7 10l5.5 5.5" /></svg>
      {id ? "Plan" : "Templates"}
    </a>
  </header>

  {#if loadError}
    <p class="state" role="alert">{loadError}</p>
    <Button onclick={load}>Try again</Button>
  {:else if !loaded || !options}
    <p class="state" aria-busy="true">Loading…</p>
  {:else}
    <h1>{id ? "Edit plan" : "New template"}</h1>
    <p class="meta">
      <span>{items.length} {items.length === 1 ? "exercise" : "exercises"} · {totalSets} {totalSets === 1 ? "set" : "sets"}</span>
      <span>No weight targets</span>
    </p>

    <form novalidate onsubmit={save}>
      <div class="about">
        <Field label="Name">
          <input
            bind:value={name}
            maxlength={MAX_NAME}
            placeholder="e.g. Push day A"
            autocomplete="off"
            aria-invalid={tried && !!nameError}
            aria-describedby={tried && nameError ? "name-err" : undefined}
          />
        </Field>
        {#if tried && nameError}<p class="err" id="name-err">{nameError}</p>{/if}
        <Field label="Notes · optional">
          <textarea rows="2" bind:value={notes} placeholder="What this session is for"></textarea>
        </Field>
      </div>

      <h2 class="cap list-h">Exercises</h2>
      {#if items.length}
        <ol class="items">
          {#each items as it, i (it.key)}
            {@const err = tried ? errors[i]! : {}}
            <li class="item" aria-labelledby="it-{it.key}">
              <div class="head">
                <h3 id="it-{it.key}"><span class="code">{code(i)}</span>{it.name}</h3>
                <div class="order">
                  <button
                    type="button"
                    data-move="{it.key}:-1"
                    aria-label="Move {it.name} up"
                    disabled={i === 0}
                    onclick={() => move(i, -1)}>{@render arrow(true)}</button
                  >
                  <button
                    type="button"
                    data-move="{it.key}:1"
                    aria-label="Move {it.name} down"
                    disabled={i === items.length - 1}
                    onclick={() => move(i, 1)}>{@render arrow(false)}</button
                  >
                </div>
              </div>
              {#if it.archived}<p class="quiet">Archived · a workout starts without it</p>{/if}

              <div class="targets">
                <div class="sets" class:bad={!!err.sets}>
                  <Stepper field={SETS} bind:value={it.sets} />
                </div>
                {#each TARGETS[it.type] as k (k)}
                  {@const inp = TARGET_INPUT[k]}
                  <label class="well" class:bad={!!err[k]}>
                    <span class="cap">{inp.label} · optional</span>
                    <span class="box">
                      <input
                        bind:value={it[k]}
                        inputmode={inp.inputmode}
                        placeholder={inp.placeholder}
                        autocomplete="off"
                        aria-invalid={!!err[k]}
                        aria-describedby={err[k] ? `err-${it.key}-${k}` : undefined}
                      />
                      {#if inp.unit}<span class="unit">{inp.unit}</span>{/if}
                    </span>
                  </label>
                {/each}
              </div>
              {#each Object.entries(err) as [k, msg] (k)}
                <p class="err" id="err-{it.key}-{k}">{msg}</p>
              {/each}

              {#if it.notes || noting.has(it.key)}
                <Field label="Note · optional">
                  <input
                    id="note-{it.key}"
                    bind:value={it.notes}
                    maxlength={MAX_NOTE}
                    placeholder="e.g. pause at the bottom"
                    autocomplete="off"
                  />
                </Field>
              {/if}
              <div class="foot">
                {#if !it.notes && !noting.has(it.key)}
                  <button type="button" class="text" onclick={() => addNote(it.key)}>Add note</button>
                {/if}
                <Button variant="danger" onclick={() => remove(i)}>Remove {code(i)}</Button>
              </div>
            </li>
          {/each}
        </ol>
      {:else}
        <p class="none">No exercises yet. Add the first one; you can set sets and reps for each.</p>
      {/if}
      <p class="sr-only" aria-live="polite">{announce}</p>

      <Button variant={items.length ? "ghost" : "ink"} wide onclick={() => (picking = true)}>
        <svg viewBox="0 0 20 20" aria-hidden="true" class="plus"><path d="M4 10h12M10 4v12" /></svg>
        Add exercise
      </Button>

      <div class="submit">
        {#if tried && problems}
          <p class="err" role="alert">
            {problems === 1 ? "One thing to fix" : `${problems} things to fix`} before saving.
          </p>
        {/if}
        <Button type="submit" variant={items.length ? "ink" : "ghost"} size="lg" wide disabled={saving}>
          {saving ? "Saving…" : id ? "Save plan" : "Create template"}
        </Button>
      </div>
    </form>
  {/if}
</div>

<TabBar active="templates" />
<Toast bind:toast />

{#if options}
  <Sheet bind:open={picking} title="Add exercise">
    <ExercisePicker
      {exercises}
      {options}
      taken={new Set(items.map((it) => it.exercise_id))}
      takenLabel="In this plan"
      onpick={pick}
      onerror={fail}
    />
  </Sheet>
{/if}

<Sheet bind:open={() => leavingTo !== null, (v) => !v && (leavingTo = null)} title="Discard changes?">
  <p class="ask">You've changed this plan without saving.</p>
  <div class="two">
    <Button onclick={() => (leavingTo = null)}>Keep editing</Button>
    <Button variant="ink" onclick={leave}>Discard</Button>
  </div>
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
    min-height: 56px;
    padding-top: max(8px, env(safe-area-inset-top));
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
  svg {
    width: 20px;
    height: 20px;
  }
  path {
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
  }
  .meta {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 4px 12px;
    margin: 0 0 16px;
    padding-top: 8px;
    border-top: 2px solid var(--red);
    color: var(--ink-2);
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .state {
    margin: 24px 0 16px;
    font-weight: 600;
  }
  .about {
    display: grid;
    gap: 12px;
  }
  .list-h {
    margin: 24px 0 8px;
  }
  .none {
    margin: 0 0 14px;
    color: var(--ink-2);
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }

  .items {
    margin: 0 0 12px;
    padding: 0;
    list-style: none;
  }
  .item {
    display: grid;
    gap: 10px;
    margin-bottom: 10px;
    padding: 8px 12px 4px;
    border: var(--line);
    border-radius: var(--r);
  }
  .item:focus-within {
    border-width: 2.5px;
    padding: 7px 11px 3px;
    background: var(--paper-hi);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  h3 {
    flex: 1;
    min-width: 0;
    margin: 0;
    font: 400 30px/1 var(--gothic);
    text-transform: uppercase;
    letter-spacing: 0.01em;
    overflow-wrap: anywhere;
  }
  .code {
    display: inline-grid;
    place-items: center;
    width: 22px;
    height: 22px;
    margin-right: 8px;
    border: var(--line);
    border-radius: 3px;
    font: 700 13px/1 var(--mono);
    vertical-align: 0.32em;
  }
  .order {
    display: flex;
    flex: none;
    margin-right: -8px;
  }
  .order button {
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    padding: 0;
    border: 0;
    border-radius: var(--r);
    background: none;
    color: var(--ink);
    cursor: pointer;
    touch-action: manipulation;
  }
  .order button:active:not(:disabled) {
    background: var(--paper);
  }
  .order button:disabled {
    color: var(--ink-2);
    opacity: 0.4;
    cursor: default;
  }
  .quiet {
    margin: -6px 0 0;
    color: var(--ink-2);
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }

  .targets {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  /* A target typed as text, in the same 64px gothic well as the stepper. */
  .well {
    display: block;
    min-width: 0;
  }
  .box {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 3px;
    height: 64px;
    padding: 0 10px;
    box-sizing: border-box;
    border: var(--line);
    border-radius: var(--r);
    background: var(--paper-hi);
    cursor: text;
  }
  .box:has(input:focus-visible) {
    outline: 2px solid var(--biro);
    outline-offset: 2px;
  }
  .box input {
    width: 100%;
    min-width: 0;
    padding: 0;
    border: 0;
    background: none;
    text-align: center;
    font: 400 46px/62px var(--gothic);
    letter-spacing: 0.01em;
  }
  .box input:focus {
    outline: none;
  }
  .box input::placeholder {
    color: var(--ink-2);
  }
  .unit {
    flex: none;
    font-size: 11px;
    font-weight: 700;
  }
  .bad :global(.row),
  .bad .box {
    border-width: 2px;
  }
  .err {
    margin: -4px 0 0;
    font-weight: 700;
  }
  .foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: -4px -4px 0;
  }
  .foot > :global(:last-child) {
    margin-left: auto;
  }
  .text {
    min-height: 44px;
    padding: 0 4px;
    border: 0;
    background: none;
    font: 700 12px/1.2 var(--mono);
    letter-spacing: 0.1em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .plus {
    flex: none;
  }
  .submit {
    display: grid;
    gap: 10px;
    margin-top: 24px;
  }
  .submit .err {
    margin: 0;
  }
  .ask {
    margin: 0 0 14px;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
</style>
