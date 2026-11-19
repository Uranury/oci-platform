<script lang="ts">
  // A ballpoint tick, drawn with the pen when `fresh` (the set just logged).
  let { fresh = false }: { fresh?: boolean } = $props();
</script>

<svg class="tick" class:fresh viewBox="0 0 24 20" aria-hidden="true">
  <path pathLength="1" d="M3 11.5c2.6 1.6 4.4 3.9 6.1 6.6C11.8 11.6 16 5.8 21.6 2.4" />
</svg>

<style>
  .tick {
    width: 22px;
    height: 18px;
    flex: none;
    overflow: visible;
  }
  path {
    fill: none;
    stroke: var(--biro);
    stroke-width: 2.2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .fresh path {
    stroke-dasharray: 1;
    animation: pen 420ms var(--ease) both;
  }
  @keyframes pen {
    from {
      stroke-dashoffset: 1;
    }
    to {
      stroke-dashoffset: 0;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .fresh path {
      animation: none;
    }
  }
</style>
