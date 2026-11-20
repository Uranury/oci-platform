<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import { takeFlash } from "../lib/flash";
  import { fmtDaysAgo, fmtNames, fmtTarget } from "../lib/format";
  import type { Template, TemplateSummary, Workout } from "../lib/types";
  import BiroRing from "../ui/BiroRing.svelte";
  import Button from "../ui/Button.svelte";
  import TabBar from "../ui/TabBar.svelte";
  import Toast, { type ToastMsg } from "../ui/Toast.svelte";

  // How many exercises the due plan lists with their targets.
  const SHOWN_PLAN = 4;

  let templates = $state<TemplateSummary[] | null>(null);
  let due = $state<Template | null>(null); // the first template, with its targets
  let loadError = $state<string | null>(null);
  let starting = $state<string | null>(null);
  let toast = $state<ToastMsg | null>(null);
  let toastSeq = 0;

  function say(text: string, error = false) {
    toast = { id: ++toastSeq, text, error };
  }

  async function load() {
    loadError = null;
    try {
      const t = await api<TemplateSummary[]>("GET", "/templates");
      templates = t;
      // The targets under the due plan; the list works without them.
      if (t[0]) due = await api<Template>("GET", `/templates/${t[0].id}`).catch(() => null);
    } catch (e) {
      loadError = e instanceof Error ? e.message : "Couldn't load your templates";
    }
  }

  onMount(() => {
    const f = takeFlash();
    if (f) say(f);
    load();
  });

  async function start(t: TemplateSummary) {
    starting = t.id;
    try {
      const w = await api<Workout>("POST", `/templates/${t.id}/start`);
      location.hash = `#/workouts/${w.id}`;
    } catch (e) {
      say(e instanceof Error ? e.message : "Couldn't start the workout", true);
      starting = null;
    }
  }

  const lead = $derived(templates?.[0] ?? null);
  const rest = $derived(templates?.slice(1) ?? []);
  const plan = $derived(due && due.id === lead?.id ? due.exercises : null);

  const lastDone = (t: TemplateSummary) => (t.last_used_at ? `Last done ${fmtDaysAgo(t.last_used_at)}` : "Never done");
  const count = (n: number) => `${n} ${n === 1 ? "exercise" : "exercises"}`;
</script>

<div class="screen">
  <header class="top">
    <span class="brand">Lift Log</span>
    {#if templates?.length}
      <Button size="sm" href="#/templates/new">
        <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M10 4v12" /></svg>
        New
      </Button>
    {/if}
  </header>

  <h1>Templates</h1>
  <p class="meta">
    {#if templates?.length}
      <span>{templates.length} {templates.length === 1 ? "template" : "templates"}</span>
      <span>Most due first</span>
    {:else}
      <span>Plans you can repeat</span>
    {/if}
  </p>

  {#if loadError}
    <p class="state" role="alert">{loadError}</p>
    <Button onclick={load}>Try again</Button>
  {:else if !templates}
    <p class="state" aria-busy="true">Loading…</p>
  {:else if !lead}
    <p class="none">
      No templates yet. Plan a workout you can start again in one tap, or open a logged workout and choose
      Save as template.
    </p>
    <Button variant="ink" size="lg" wide href="#/templates/new">New template</Button>
  {:else}
    <section class="due" aria-labelledby="due-h">
      <h2 id="due-h" class="sr-only">Due next</h2>
      <a class="ringed" href="#/templates/{lead.id}">
        <span class="big">{lead.name}</span><BiroRing />
      </a>
      <p class="sub">{lastDone(lead)} · {count(lead.exercise_count)}</p>

      {#if plan?.length}
        <ol class="plan">
          {#each plan.slice(0, SHOWN_PLAN) as e (e.id)}
            <li>{e.exercise_name} <b>{fmtTarget(e)}</b></li>
          {/each}
          {#if plan.length > SHOWN_PLAN}<li class="more">+{plan.length - SHOWN_PLAN} more</li>{/if}
        </ol>
      {:else if lead.exercises.length}
        <p class="plan names">{fmtNames(lead.exercises, SHOWN_PLAN)}</p>
      {:else}
        <p class="plan names">No exercises yet.</p>
      {/if}

      <div class="two">
        <Button href="#/templates/{lead.id}/edit">Edit plan</Button>
        <Button variant="ink" disabled={starting !== null || !lead.exercise_count} onclick={() => start(lead)}>
          {starting === lead.id ? "Starting…" : "Start"}
        </Button>
      </div>
    </section>

    {#if rest.length}
      <ul class="rows">
        {#each rest as t (t.id)}
          <li>
            <a href="#/templates/{t.id}">
              <span class="row-name">{t.name}</span>
              <span class="row-sub names">{t.exercises.length ? fmtNames(t.exercises, 3) : "No exercises yet"}</span>
              <span class="row-sub">{lastDone(t)}</span>
            </a>
            <Button size="sm" disabled={starting !== null || !t.exercise_count} onclick={() => start(t)}>
              {starting === t.id ? "…" : "Start"}
            </Button>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</div>

<TabBar active="templates" />
<Toast bind:toast />

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
  .brand {
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .top path {
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
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
    margin: 0 0 14px;
    padding-top: 8px;
    border-top: 2px solid var(--red);
    color: var(--ink-2);
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .state {
    margin: 24px 0 16px;
    font-weight: 600;
  }
  .none {
    margin: 10px 0 16px;
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

  /* The due plan: one ruled panel on the raised paper, its name ringed. */
  .due {
    padding: 12px;
    border: 2.5px solid var(--ink);
    border-radius: var(--r);
    background: var(--paper-hi);
  }
  /* The loop's tail starts inside its left edge: keep it clear of the
     first letter, however short the name. */
  .ringed {
    position: relative;
    display: inline-block;
    padding: 4px 16px 2px 20px;
    margin-left: -6px;
    text-decoration: none;
  }
  .big {
    font: 400 42px/0.95 var(--gothic);
    text-transform: uppercase;
    overflow-wrap: anywhere;
  }
  .sub {
    margin: 6px 0 0;
    color: var(--ink-2);
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .plan {
    margin: 10px 0 12px;
    padding: 10px 0 0;
    border-top: 1.5px solid var(--rule);
    list-style: none;
  }
  .plan li {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 12px;
    min-height: 30px;
  }
  .plan b {
    flex: none;
    font: 400 22px/1 var(--gothic);
    letter-spacing: 0.01em;
    text-transform: uppercase;
  }
  .plan .more,
  .plan.names {
    color: var(--ink-2);
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .rows {
    margin: 18px 0 0;
    padding: 0;
    list-style: none;
  }
  .rows li {
    display: flex;
    align-items: center;
    gap: 12px;
    border-top: 1.5px solid var(--ink);
  }
  .rows a {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    min-height: 56px;
    padding: 10px 0;
    text-decoration: none;
  }
  .rows a:active {
    background: var(--paper-hi);
  }
  .row-name {
    font: 400 30px/1 var(--gothic);
    text-transform: uppercase;
    overflow-wrap: anywhere;
  }
  .row-sub {
    margin-top: 3px;
    color: var(--ink-2);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .row-sub.names {
    text-wrap: pretty;
    color: var(--ink);
    text-transform: none;
    font-size: 13px;
    letter-spacing: 0;
  }
</style>
