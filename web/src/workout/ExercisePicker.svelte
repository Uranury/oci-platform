<script lang="ts">
  import ExerciseForm from "../exercises/ExerciseForm.svelte";
  import { api } from "../lib/api";
  import { exerciseMeta } from "../lib/exercises";
  import type { Exercise, ExerciseOptions, NewExercise } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";

  let { exercises, options, taken, onpick, onerror }: {
    exercises: Exercise[];
    options: ExerciseOptions;
    taken: Set<string>; // already in this workout
    onpick: (e: Exercise) => void;
    onerror: (e: unknown) => void;
  } = $props();

  let query = $state("");
  let creating = $state(false);

  const meta = (e: Exercise) => exerciseMeta(options, e);

  const q = $derived(query.trim().toLowerCase());
  const matches = $derived(
    exercises.filter((e) => !q || e.name.toLowerCase().includes(q) || meta(e).toLowerCase().includes(q)),
  );
  const mine = $derived(matches.filter((e) => !e.system));
  const library = $derived(matches.filter((e) => e.system));

  async function create(body: NewExercise) {
    try {
      onpick(await api<Exercise>("POST", "/exercises", body));
    } catch (e) {
      onerror(e);
    }
  }
</script>

{#snippet row(e: Exercise)}
  <li>
    <button type="button" class="pick" disabled={taken.has(e.id)} onclick={() => onpick(e)}>
      <span class="name">{e.name}</span>
      <span class="sub">{taken.has(e.id) ? "In this workout" : meta(e)}</span>
    </button>
  </li>
{/snippet}

{#if !creating}
  <Field label="Find exercise">
    <input type="search" bind:value={query} placeholder="Name, equipment or muscle" autocomplete="off" />
  </Field>

  {#if mine.length}
    <h3>My exercises</h3>
    <ul>{#each mine as e (e.id)}{@render row(e)}{/each}</ul>
  {/if}
  {#if library.length}
    <h3>Library</h3>
    <ul>{#each library as e (e.id)}{@render row(e)}{/each}</ul>
  {/if}
  {#if !matches.length}
    <p class="none">Nothing matches “{query.trim()}”.</p>
  {/if}

  <div class="more">
    <Button wide onclick={() => (creating = true)}>New exercise{q ? ` “${query.trim()}”` : ""}</Button>
  </div>
{:else}
  <ExerciseForm
    {options}
    initial={{ name: query.trim() }}
    creating
    submitLabel="Create & add"
    cancelLabel="Back"
    onsave={create}
    oncancel={() => (creating = false)}
    onuse={onpick}
  />
{/if}

<style>
  h3 {
    margin: 18px 0 4px;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  li + li {
    border-top: 1px solid var(--rule);
  }
  .pick {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    width: 100%;
    min-height: 56px;
    padding: 8px 2px;
    border: 0;
    background: none;
    text-align: left;
    cursor: pointer;
  }
  .pick:disabled {
    cursor: default;
    opacity: 0.6;
  }
  .name {
    font: 400 24px/1.05 var(--gothic);
    text-transform: uppercase;
  }
  .sub {
    font-size: 12px;
    color: var(--ink-2);
  }
  .none {
    margin: 16px 0;
  }
  .more {
    margin-top: 18px;
  }
</style>
