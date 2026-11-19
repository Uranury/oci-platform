<script lang="ts">
  import { parseDuration } from "../lib/format";
  import { fmtValue, type Field } from "../lib/sets";

  let { field, value = $bindable(), disabled = false }: {
    field: Field;
    value: number | null | undefined;
    disabled?: boolean;
  } = $props();

  // The typed text; resynced whenever the value changes from outside
  // (a step, a prefill), left alone while the user is typing in it.
  let text = $state("");
  let input: HTMLInputElement;
  $effect(() => {
    const v = value;
    if (document.activeElement !== input) text = fmtValue(field.key, v);
  });

  function parse(t: string): number | null {
    if (field.key === "duration_seconds") return parseDuration(t);
    const s = t.trim().replace(",", ".");
    return s === "" ? null : Number(s);
  }

  function oninput() {
    value = parse(text);
  }

  function onblur() {
    if (value != null && Number.isFinite(value)) text = fmtValue(field.key, value);
  }

  function step(dir: 1 | -1) {
    const cur = value != null && Number.isFinite(value) ? value : null;
    if (cur == null) {
      value = dir > 0 ? (field.key === "weight_kg" && !field.required ? field.step : field.min || field.step) : null;
      return;
    }
    const next = Math.round((cur + dir * field.step) / field.step) * field.step;
    if (next < field.min || (next === 0 && field.min === 0 && !field.required)) {
      value = field.required ? field.min : null;
    } else {
      value = next;
    }
  }

  const name = $derived(field.label.toLowerCase());
  const empty = $derived(value == null);
</script>

<div class="stepper" class:empty>
  <span class="cap">{field.label}{field.required ? "" : " · optional"}</span>
  <div class="row">
    <button type="button" aria-label="Less {name}" {disabled} onclick={() => step(-1)}>
      <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12" /></svg>
    </button>
    <label class="fig">
      <input
        bind:this={input}
        bind:value={text}
        size={Math.max(2, text.length)}
        {oninput}
        {onblur}
        {disabled}
        inputmode={field.integer ? "numeric" : "decimal"}
        placeholder={field.key === "duration_seconds" ? "0:00" : "0"}
        aria-label={field.label + (field.unit ? ` (${field.unit})` : "")}
        autocomplete="off"
      />
      {#if field.unit}<span class="unit">{field.unit}</span>{/if}
    </label>
    <button type="button" aria-label="More {name}" {disabled} onclick={() => step(1)}>
      <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h12M10 4v12" /></svg>
    </button>
  </div>
</div>

<style>
  .stepper {
    min-width: 0;
  }
  .row {
    display: grid;
    grid-template-columns: 44px 1fr 44px;
    align-items: stretch;
    height: 64px;
    border: var(--line);
    border-radius: var(--r);
    background: var(--paper-hi);
  }
  button {
    display: grid;
    place-items: center;
    background: none;
    border: 0;
    padding: 0;
    cursor: pointer;
    touch-action: manipulation;
  }
  button:active:not(:disabled) {
    background: var(--paper);
  }
  button:disabled {
    cursor: default;
    opacity: 0.4;
  }
  button:first-child {
    border-radius: var(--r) 0 0 var(--r);
  }
  button:last-child {
    border-radius: 0 var(--r) var(--r) 0;
  }
  svg {
    width: 20px;
    height: 20px;
  }
  svg path {
    stroke: var(--ink);
    stroke-width: 2.2;
    stroke-linecap: round;
  }
  .fig {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 3px;
    min-width: 0;
    cursor: text;
  }
  input {
    width: auto;
    field-sizing: content;
    min-width: 2ch;
    max-width: 100%;
    text-align: center;
    border: 0;
    background: none;
    padding: 0;
    font-family: var(--gothic);
    font-size: 46px;
    line-height: 62px;
    letter-spacing: 0.01em;
  }
  input:focus {
    outline: none;
  }
  .row:has(input:focus-visible) {
    outline: 2px solid var(--biro);
    outline-offset: 2px;
  }
  input::placeholder {
    color: var(--ink-2);
  }
  .unit {
    font-size: 11px;
    font-weight: 700;
    margin-left: -2px;
    flex: none;
    max-width: 3ch;
  }
  .fig:has(input:placeholder-shown) .unit {
    display: none;
  }
</style>
