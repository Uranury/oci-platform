<script lang="ts">
  import type { Snippet } from "svelte";

  // A heavy-ruled box asking for a decision before going on ("1000 kg?",
  // "the library already has …"). children: the two choice buttons.
  let { message, children }: { message: string; children: Snippet } = $props();
</script>

<div class="notice" role="alert">
  <p>{message}</p>
  <div class="choices">{@render children()}</div>
</div>

<style>
  .notice {
    padding: 10px 12px 12px;
    border: 2px solid var(--ink);
    border-radius: var(--r);
  }
  p {
    margin: 0 0 10px;
    font-weight: 700;
  }
  .choices {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
</style>
