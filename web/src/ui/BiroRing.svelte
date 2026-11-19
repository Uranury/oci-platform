<script lang="ts">
  // A hand-drawn ballpoint loop around its positioned parent: the next set.
  // Keyed by the caller so it redraws when the next set moves.
</script>

<svg class="ring" viewBox="0 0 100 40" preserveAspectRatio="none" aria-hidden="true">
  <path
    pathLength="1"
    d="M10 24C4 12 30 4 52 4s44 4 44 15c0 12-26 18-48 18S3 33 4 22C5 13 22 7 38 6"
  />
</svg>

<style>
  .ring {
    position: absolute;
    inset: -7px -9px -6px -8px;
    width: calc(100% + 17px);
    height: calc(100% + 13px);
    overflow: visible;
    pointer-events: none;
  }
  path {
    fill: none;
    stroke: var(--biro);
    stroke-width: 2;
    stroke-linecap: round;
    vector-effect: non-scaling-stroke;
    stroke-dasharray: 1;
    animation: pen 560ms var(--ease) both;
  }
  @media (prefers-reduced-motion: reduce) {
    path {
      animation: none;
    }
  }
</style>
