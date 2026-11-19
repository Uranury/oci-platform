<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import { label } from "../lib/exercises";
  import { fmtDay, fmtSets } from "../lib/format";
  import type { Exercise, ExerciseOptions, Previous } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Notice from "../ui/Notice.svelte";

  // What the detail sheet shows for one exercise: its categories, the user's
  // recent sessions with it, and what they can do with it.
  let { exercise, options, onedit, oncustomize, ondelete }: {
    exercise: Exercise;
    options: ExerciseOptions;
    onedit: () => void;
    oncustomize: () => Promise<void>;
    ondelete: () => Promise<void>;
  } = $props();

  let history = $state<Previous[] | null>(null);
  let historyError = $state(false);
  let confirming = $state(false);
  let busy = $state(false);

  onMount(async () => {
    try {
      history = await api<Previous[]>("GET", `/exercises/${exercise.id}/history`);
    } catch {
      historyError = true;
    }
  });

  const also = $derived(exercise.secondary_muscles.map((m) => label(options.muscles, m)));

  async function run(action: () => Promise<void>) {
    busy = true;
    try {
      await action();
    } finally {
      busy = false;
    }
  }
</script>

<dl class="facts">
  <div>
    <dt class="cap">Equipment</dt>
    <dd>{label(options.equipment, exercise.equipment)}</dd>
  </div>
  <div>
    <dt class="cap">Movement</dt>
    <dd>
      {label(options.movement_patterns, exercise.movement_pattern)}
      <span class="also">{label(options.mechanics, exercise.mechanic)}</span>
    </dd>
  </div>
  <div>
    <dt class="cap">Muscles</dt>
    <dd>
      {label(options.muscles, exercise.primary_muscle)}
      {#if also.length}<span class="also">also {also.join(", ")}</span>{/if}
    </dd>
  </div>
  <div>
    <dt class="cap">Logged as</dt>
    <dd>{label(options.tracking_types, exercise.tracking_type)}</dd>
  </div>
</dl>

<h3 class="cap">Your recent sessions</h3>
{#if historyError}
  <p class="quiet">Couldn't load your history.</p>
{:else if !history}
  <p class="quiet" aria-busy="true">Loading…</p>
{:else if !history.length}
  <p class="quiet">Not done yet. Your sets will show up here once you log it.</p>
{:else}
  <ol class="history">
    {#each history as h (h.workout_id)}
      <li>
        <a href="#/workouts/{h.workout_id}">
          <span class="day">{fmtDay(h.performed_at)}</span>
          <b>{fmtSets(h.sets)}</b>
        </a>
      </li>
    {/each}
  </ol>
{/if}

<div class="actions">
  {#if exercise.system}
    <p class="quiet">A library default everyone shares. Customize makes your own editable copy.</p>
    <Button variant="ink" wide disabled={busy} onclick={() => run(oncustomize)}>Customize</Button>
  {:else if confirming}
    <Notice
      message={`Delete “${exercise.name}”? If it has logged sets it's archived instead: hidden from the picker, still in your history.`}
    >
      <Button onclick={() => (confirming = false)}>Keep it</Button>
      <Button disabled={busy} onclick={() => run(ondelete)}>Delete</Button>
    </Notice>
  {:else}
    <Button variant="ink" wide onclick={onedit}>Edit</Button>
    <div class="apart">
      <Button variant="danger" onclick={() => (confirming = true)}>Delete exercise</Button>
    </div>
  {/if}
</div>

<style>
  .facts {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin: 0 0 18px;
  }
  .facts dd {
    margin: 0;
    font-weight: 600;
  }
  .also {
    display: block;
    color: var(--ink-2);
    font-weight: 500;
  }
  h3.cap {
    margin: 0 0 4px;
  }
  .quiet {
    margin: 4px 0 12px;
    color: var(--ink-2);
  }
  .history {
    margin: 0 0 16px;
    padding: 0;
    list-style: none;
  }
  .history a {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    min-height: 44px;
    align-items: center;
    border-top: 1.5px solid var(--rule);
    text-decoration: none;
    font-size: 13px;
    text-transform: uppercase;
  }
  .day {
    color: var(--ink-2);
    white-space: nowrap;
  }
  .history b {
    text-align: right;
  }
  .actions {
    display: grid;
    gap: 10px;
    margin-top: 8px;
  }
  .actions .quiet {
    margin: 0;
    font-size: 13px;
  }
  .apart {
    display: flex;
    justify-content: flex-end;
    margin-top: 14px;
  }
</style>
