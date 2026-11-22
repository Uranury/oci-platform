<script lang="ts">
  // The coach's note in red pen beside an exercise whose best set today beats
  // last session's: a quick rising arrow. Drawn with the pen when `fresh`
  // (just earned), otherwise simply there.
  let { fresh = false }: { fresh?: boolean } = $props();
</script>

<span class="over">
  <svg class:fresh viewBox="0 0 22 22" aria-hidden="true">
    <path pathLength="1" d="M3.5 18.5C7 15.2 11.4 10.4 17.2 4.3M10.6 4.6c2.4-.5 4.6-.7 6.8-.6.3 2.1.2 4.3-.2 6.6" />
  </svg>
  <span class="sr-only">Beat last time</span>
</span>

<style>
  .over {
    display: inline-block;
    margin-left: 8px;
    vertical-align: 0.18em;
  }
  svg {
    display: block;
    width: 22px;
    height: 22px;
    overflow: visible;
  }
  path {
    fill: none;
    stroke: var(--red);
    stroke-width: 2.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .fresh path {
    stroke-dasharray: 1;
    animation: pen 520ms var(--ease) both;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
  @media (prefers-reduced-motion: reduce) {
    .fresh path {
      animation: none;
    }
  }
</style>
