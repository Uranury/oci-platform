<script lang="ts">
  // A count kept as ballpoint tally marks: four strokes and a fifth struck
  // through them, then a new group. Each stroke leans a little differently so
  // it reads as hand-made; the leans are fixed, so it never jitters.
  let { count, label }: { count: number; label: string } = $props();

  const LEAN = [0.6, -0.4, 0.9, -0.7, 0.2, -0.9, 0.5, -0.2];
  const GROUP = 30; // px per group of five

  const strokes = $derived.by(() => {
    const out: string[] = [];
    for (let i = 0; i < count; i++) {
      const g = Math.floor(i / 5);
      const j = i % 5;
      const x = g * GROUP + 3;
      const lean = LEAN[i % LEAN.length] ?? 0;
      if (j < 4) {
        const sx = x + j * 6;
        out.push(`M${sx + lean} 2.5 L${sx - lean} 15.5`);
      } else {
        out.push(`M${x - 3} 13 L${x + 22} 4.5`);
      }
    }
    return out;
  });
  const width = $derived(Math.max(1, Math.ceil(count / 5)) * GROUP - 4);
</script>

<svg class="tally" viewBox="0 0 {width} 18" width={width} height="18" role="img" aria-label={label}>
  {#each strokes as d, i (i)}<path {d} />{/each}
</svg>

<style>
  .tally {
    flex: none;
    overflow: visible;
  }
  path {
    fill: none;
    stroke: var(--biro);
    stroke-width: 2;
    stroke-linecap: round;
  }
</style>
