<script lang="ts">
  import { api } from "../lib/api";
  import type { Exercise, ExerciseOptions, NewExercise } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";
  import Notice from "../ui/Notice.svelte";

  let { exercises, options, taken, onpick, onerror }: {
    exercises: Exercise[];
    options: ExerciseOptions;
    taken: Set<string>; // already in this workout
    onpick: (e: Exercise) => void;
    onerror: (e: unknown) => void;
  } = $props();

  let query = $state("");
  let creating = $state(false);

  const label = (list: { value: string; label: string }[], v: string) =>
    list.find((o) => o.value === v)?.label ?? v;
  const meta = (e: Exercise) =>
    `${label(options.equipment, e.equipment)} · ${label(options.muscles, e.primary_muscle)}`;

  const q = $derived(query.trim().toLowerCase());
  const matches = $derived(
    exercises.filter((e) => !q || e.name.toLowerCase().includes(q) || meta(e).toLowerCase().includes(q)),
  );
  const mine = $derived(matches.filter((e) => !e.system));
  const library = $derived(matches.filter((e) => e.system));

  // --- new exercise ---
  let form = $state<NewExercise>({
    name: "",
    tracking_type: "weight_reps",
    equipment: "",
    movement_pattern: "",
    primary_muscle: "",
    secondary_muscles: [],
    mechanic: "compound",
  });
  let similar = $state<Exercise[] | null>(null);
  let saving = $state(false);

  function startCreating() {
    form.name = query.trim();
    similar = null;
    creating = true;
  }

  // Warns (never blocks) when the name is close to a library default.
  async function create(force = false) {
    saving = true;
    try {
      const body = { ...form, name: form.name.trim() };
      body.secondary_muscles = body.secondary_muscles.filter((m) => m !== body.primary_muscle);
      if (!force) {
        const found = await api<Exercise[]>("GET", "/exercises/similar?name=" + encodeURIComponent(body.name));
        if (found.length) {
          similar = found;
          return;
        }
      }
      onpick(await api<Exercise>("POST", "/exercises", body));
    } catch (e) {
      onerror(e);
    } finally {
      saving = false;
    }
  }

  function toggleSecondary(v: string, on: boolean) {
    form.secondary_muscles = on
      ? [...form.secondary_muscles, v]
      : form.secondary_muscles.filter((m) => m !== v);
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
    <Button wide onclick={startCreating}>New exercise{q ? ` “${query.trim()}”` : ""}</Button>
  </div>
{:else}
  <form
    class="create"
    onsubmit={(e) => {
      e.preventDefault();
      create();
    }}
  >
    <Field label="Name">
      <input bind:value={form.name} required maxlength="100" placeholder="e.g. Paused squat" />
    </Field>
    <Field label="How do you log it?">
      <select bind:value={form.tracking_type}>
        {#each options.tracking_types as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
      </select>
    </Field>
    <div class="two">
      <Field label="Equipment">
        <select bind:value={form.equipment} required>
          <option value="" disabled>Choose…</option>
          {#each options.equipment as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
      </Field>
      <Field label="Mechanic">
        <select bind:value={form.mechanic}>
          {#each options.mechanics as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
      </Field>
      <Field label="Movement">
        <select bind:value={form.movement_pattern} required>
          <option value="" disabled>Choose…</option>
          {#each options.movement_patterns as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
      </Field>
      <Field label="Primary muscle">
        <select bind:value={form.primary_muscle} required>
          <option value="" disabled>Choose…</option>
          {#each options.muscles as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
      </Field>
    </div>
    <fieldset>
      <legend class="cap">Secondary muscles</legend>
      <div class="tags">
        {#each options.muscles as o (o.value)}
          {@const primary = o.value === form.primary_muscle}
          <label class="tag" class:off={primary}>
            <input
              type="checkbox"
              disabled={primary}
              checked={!primary && form.secondary_muscles.includes(o.value)}
              onchange={(e) => toggleSecondary(o.value, e.currentTarget.checked)}
            />
            {o.label}
          </label>
        {/each}
      </div>
    </fieldset>

    {#if similar}
      <Notice message={`The library already has ${similar.map((e) => `“${e.name}”`).join(", ")}.`}>
        {#each similar.slice(0, 1) as e (e.id)}
          <Button onclick={() => onpick(e)}>Use {e.name}</Button>
        {/each}
        <Button disabled={saving} onclick={() => create(true)}>Create anyway</Button>
      </Notice>
    {/if}

    <div class="two">
      <Button onclick={() => (creating = false)}>Back</Button>
      <Button type="submit" variant="ink" disabled={saving}>{saving ? "Saving…" : "Create & add"}</Button>
    </div>
  </form>
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
  .create {
    display: grid;
    gap: 14px;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  fieldset {
    border: 0;
    margin: 0;
    padding: 0;
  }
  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .tag {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 36px;
    padding: 0 10px;
    border: 1px solid var(--rule);
    border-radius: 4px;
    font-size: 13px;
    cursor: pointer;
  }
  .tag:has(input:checked) {
    border-color: var(--ink);
    background: var(--paper-hi);
    font-weight: 700;
  }
  .tag.off {
    opacity: 0.45;
    cursor: default;
  }
</style>
