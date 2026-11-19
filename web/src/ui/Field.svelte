<script lang="ts">
  import type { Snippet } from "svelte";

  // A labelled form control: put one <input>, <select> or <textarea> inside;
  // it gets the ruled-box styling.
  let { label, children }: { label: string; children: Snippet } = $props();
</script>

<label class="field">
  <span class="cap">{label}</span>
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
  .field :global(textarea) {
    resize: vertical;
  }
  .field :global(::placeholder) {
    color: var(--ink-2);
  }
</style>
