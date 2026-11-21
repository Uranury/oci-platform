<script lang="ts">
  // A ruled strip pinned to the top while the device has no connection. The
  // app shell still opens offline (sw.js), but training data lives on the
  // server, so nothing loads or saves until the connection is back.
  let online = $state(navigator.onLine);
</script>

<svelte:window ononline={() => (online = true)} onoffline={() => (online = false)} />

<div class="slot" role="status" aria-live="polite">
  {#if !online}
    <p class="bar">Offline · reconnect to load or save</p>
  {/if}
</div>

<style>
  .slot {
    position: sticky;
    top: 0;
    z-index: 25;
  }
  .bar {
    margin: 0;
    padding: calc(8px + env(safe-area-inset-top)) 12px 8px;
    border-bottom: 2px solid var(--ink);
    background: var(--paper-hi);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-align: center;
    text-transform: uppercase;
  }
</style>
