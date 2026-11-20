<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import { flash, takeFlash } from "../lib/flash";
  import { code, fmtDaysAgo, fmtLast, fmtSets, fmtTarget } from "../lib/format";
  import type { Previous, Template, TemplateSummary, Workout } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import TabBar from "../ui/TabBar.svelte";
  import Toast, { type ToastMsg } from "../ui/Toast.svelte";

  // A template before it's done: its plan drawn as the route still to walk,
  // with last time under each stop. Editing is a separate screen.
  let { id }: { id: string } = $props();

  let template = $state<Template | null>(null);
  let lastUsed = $state<string | null | undefined>(undefined);
  let last = $state<Map<string, Previous>>(new Map());
  let loadError = $state<string | null>(null);
  let busy = $state(false);
  let confirming = $state(false);
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
      const [t, prev, list] = await Promise.all([
        api<Template>("GET", `/templates/${id}`),
        api<Previous[]>("GET", "/exercises/last").catch(() => []),
        api<TemplateSummary[]>("GET", "/templates").catch(() => null),
      ]);
      template = t;
      last = new Map(prev.map((p) => [p.exercise_id, p]));
      lastUsed = list ? (list.find((s) => s.id === id)?.last_used_at ?? null) : undefined;
    } catch (e) {
      loadError = e instanceof Error ? e.message : "Couldn't load this template";
    }
  }

  onMount(() => {
    const f = takeFlash();
    if (f) say(f);
    load();
  });

  const sets = $derived(template?.exercises.reduce((n, e) => n + e.target_sets, 0) ?? 0);

  async function start() {
    busy = true;
    try {
      const w = await api<Workout>("POST", `/templates/${id}/start`);
      location.hash = `#/workouts/${w.id}`;
    } catch (e) {
      fail(e);
      busy = false;
    }
  }

  async function remove() {
    busy = true;
    try {
      await api("DELETE", `/templates/${id}`);
      flash(`Deleted “${template?.name}”`);
      location.hash = "#/templates";
    } catch (e) {
      fail(e);
      busy = false;
    }
  }
</script>

<div class="screen">
  <header class="top">
    <a class="back" href="#/templates">
      <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12.5 4.5L7 10l5.5 5.5" /></svg>
      Templates
    </a>
  </header>

  {#if loadError}
    <p class="state" role="alert">{loadError}</p>
    <Button onclick={load}>Try again</Button>
  {:else if !template}
    <p class="state" aria-busy="true">Loading template…</p>
  {:else}
    {@const n = template.exercises.length}
    <h1>{template.name}</h1>
    <p class="meta">
      <span>
        {#if lastUsed === undefined}&nbsp;{:else if lastUsed}Last done {fmtDaysAgo(lastUsed)}{:else}Never done{/if}
      </span>
      {#if n}<b>{n} {n === 1 ? "exercise" : "exercises"} · {sets} {sets === 1 ? "set" : "sets"}</b>{/if}
    </p>
    {#if template.notes?.trim()}<p class="notes">{template.notes}</p>{/if}

    {#if n}
      <ol class="route">
        {#each template.exercises as e, i (e.id)}
          {@const prev = last.get(e.exercise_id)}
          <li>
            <span class="track" aria-hidden="true">
              {#if i > 0}<i class="seg in"></i>{/if}
              {#if i < n - 1}<i class="seg out"></i>{/if}
              <i class="stop"></i>
            </span>
            <div class="card">
              <div class="head">
                <h2><span class="code">{code(i)}</span>{e.exercise_name}</h2>
                <span class="target">{fmtTarget(e)}</span>
              </div>
              {#if e.exercise_archived}
                <p class="line quiet">Archived · starts without it</p>
              {:else if prev}
                <p class="line"><b>{fmtLast(prev.performed_at)}</b> · {fmtSets(prev.sets)}</p>
              {:else}
                <p class="line quiet">Not done yet</p>
              {/if}
              {#if e.notes}<p class="note">{e.notes}</p>{/if}
            </div>
          </li>
        {/each}
      </ol>

      <div class="two">
        <Button href="#/templates/{id}/edit">Edit plan</Button>
        <Button variant="ink" disabled={busy} onclick={start}>{busy ? "Starting…" : "Start"}</Button>
      </div>
    {:else}
      <p class="none">No exercises in this plan yet. Add some to start a workout from it.</p>
      <Button variant="ink" wide href="#/templates/{id}/edit">Edit plan</Button>
    {/if}

    <div class="apart">
      <Button variant="danger" onclick={() => (confirming = true)}>Delete template</Button>
    </div>
  {/if}
</div>

<TabBar active="templates" />
<Toast bind:toast />

<Sheet bind:open={confirming} title="Delete template?">
  <p class="ask">This deletes “{template?.name}”. Workouts you started from it are kept.</p>
  <div class="two">
    <Button onclick={() => (confirming = false)}>Keep it</Button>
    <Button variant="ink" disabled={busy} onclick={remove}>Delete</Button>
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
    justify-content: space-between;
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
  .back svg {
    width: 20px;
    height: 20px;
  }
  .back path {
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
  .notes {
    margin: -4px 0 16px;
    white-space: pre-line;
  }
  .state {
    margin: 24px 0 16px;
    font-weight: 600;
  }
  .none {
    margin: 10px 0 16px;
    color: var(--ink-2);
  }

  /* The plan as a route not yet walked: dashed, every stop open. */
  .route {
    margin: 0 0 16px;
    padding: 0;
    list-style: none;
  }
  .route li {
    position: relative;
    padding-left: 26px;
    margin-bottom: 10px;
  }
  .track {
    position: absolute;
    left: 0;
    top: 0;
    bottom: -10px;
    width: 18px;
  }
  .seg {
    position: absolute;
    left: 8px;
    border-left: 1.5px dashed var(--ink-2);
  }
  .seg.in {
    top: 0;
    height: 25px;
  }
  .seg.out {
    top: 25px;
    bottom: 0;
  }
  .stop {
    position: absolute;
    left: 3px;
    top: 19px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    box-sizing: border-box;
    background: var(--paper);
    border: 1.5px solid var(--ink-2);
  }
  .card {
    padding: 10px 12px;
    border: var(--line);
    border-radius: var(--r);
  }
  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }
  h2 {
    margin: 0;
    min-width: 0;
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
  .target {
    flex: none;
    font: 400 26px/1 var(--gothic);
    letter-spacing: 0.01em;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .line {
    margin: 6px 0 0;
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .line.quiet {
    color: var(--ink-2);
  }
  .note {
    margin: 4px 0 0;
    color: var(--ink-2);
    font-size: 13px;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .apart {
    display: flex;
    justify-content: flex-end;
    margin-top: 28px;
  }
  .ask {
    margin: 0 0 14px;
  }
</style>
