<script lang="ts">
  import { untrack } from "svelte";
  import { api } from "../lib/api";
  import type { Exercise, ExerciseOptions, NewExercise } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";
  import Notice from "../ui/Notice.svelte";

  // Create or edit a custom exercise. On create it first warns (never
  // blocks) when the name is close to a library default, offering that one
  // instead through onuse. onsave does the request; a thrown error is shown
  // in the form.
  let {
    options,
    initial,
    creating,
    submitLabel,
    cancelLabel = "Cancel",
    onsave,
    oncancel,
    onuse,
    useText = (e: Exercise) => `Use ${e.name}`,
  }: {
    options: ExerciseOptions;
    initial?: Partial<NewExercise>;
    creating: boolean;
    submitLabel: string;
    cancelLabel?: string;
    onsave: (body: NewExercise) => Promise<void>;
    oncancel: () => void;
    onuse?: (e: Exercise) => void;
    useText?: (e: Exercise) => string; // the "use the default instead" button
  } = $props();

  let form = $state<NewExercise>({
    name: "",
    tracking_type: "weight_reps",
    equipment: "",
    movement_pattern: "",
    primary_muscle: "",
    secondary_muscles: [],
    mechanic: "compound",
    // Read once: the form owns its edits from here on.
    ...untrack(() => initial),
  });
  let similar = $state<Exercise[] | null>(null);
  let problem = $state<string | null>(null);
  let saving = $state(false);

  async function submit(force = false) {
    problem = null;
    saving = true;
    try {
      const body = { ...form, name: form.name.trim() };
      // The primary muscle can't also be secondary.
      body.secondary_muscles = body.secondary_muscles.filter((m) => m !== body.primary_muscle);
      if (creating && !force) {
        const found = await api<Exercise[]>("GET", "/exercises/similar?name=" + encodeURIComponent(body.name));
        if (found.length) {
          similar = found;
          return;
        }
      }
      await onsave(body);
    } catch (e) {
      problem = e instanceof Error ? e.message : "Couldn't save the exercise";
    } finally {
      saving = false;
    }
  }

  function toggleSecondary(v: string, on: boolean) {
    form.secondary_muscles = on ? [...form.secondary_muscles, v] : form.secondary_muscles.filter((m) => m !== v);
  }
</script>

<form
  class="form"
  onsubmit={(e) => {
    e.preventDefault();
    submit();
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
      {#if onuse}
        {#each similar.slice(0, 1) as e (e.id)}
          <Button onclick={() => onuse(e)}>{useText(e)}</Button>
        {/each}
      {:else}
        <Button onclick={() => (similar = null)}>Rename</Button>
      {/if}
      <Button disabled={saving} onclick={() => submit(true)}>Create anyway</Button>
    </Notice>
  {/if}
  {#if problem}<p class="problem" role="alert">{problem}</p>{/if}

  <div class="two">
    <Button onclick={oncancel}>{cancelLabel}</Button>
    <Button type="submit" variant="ink" disabled={saving}>{saving ? "Saving…" : submitLabel}</Button>
  </div>
</form>

<style>
  .form {
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
  .problem {
    margin: 0;
    padding: 10px 12px;
    border: 2px solid var(--ink);
    border-radius: var(--r);
    font-weight: 700;
  }
  .problem::first-letter {
    text-transform: uppercase;
  }
</style>
