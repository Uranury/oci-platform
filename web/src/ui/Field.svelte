<script lang="ts">
  import type { Snippet } from "svelte";

  // A labelled form control: put one <input>, <select> or <textarea> inside;
  // it gets the ruled-box styling.
  // hideLabel keeps the caption for screen readers only (a search box whose
  // placeholder already says what it is).
  let { label, hideLabel = false, children }: { label: string; hideLabel?: boolean; children: Snippet } = $props();
</script>

<label class="field">
  <span class={hideLabel ? "sr-only" : "cap"}>{label}</span>
  {@render children()}
</label>

<style>
  .field {
    display: block;
    min-width: 0;
  }
  .field :global(:is(input:not([type="checkbox"]):not([type="radio"]), select, textarea)) {
    display: block;
    width: 100%;
    box-sizing: border-box;
    min-height: 48px;
    padding: 10px 12px;
    border: var(--line);
    border-radius: var(--r);
    background: var(--paper-hi);
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
  .field :global(textarea) {
    resize: vertical;
  }
  .field :global(::placeholder) {
    color: var(--ink-2);
  }
</style>
