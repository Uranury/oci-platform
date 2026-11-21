<script lang="ts">
  import { onMount } from "svelte";
  import { cueFinish, setSound, soundOn } from "../lib/sound";
  import type { WorkoutSet } from "../lib/types";
  import Button from "../ui/Button.svelte";

  // The end of the session: the red route runs down every exercise done to
  // its last stop, then a DONE stamp lands on the page (with a thump and a
  // chime, if sounds are on). Shown once, right after finishing.
  let {
    title,
    started,
    finished,
    route,
    sets,
    ondone,
  }: {
    title: string;
    started: string;
    finished: string;
    route: { code: string; name: string; sets: number }[];
    sets: WorkoutSet[];
    ondone: () => void;
  } = $props();

  let dialog: HTMLDialogElement;
  let sound = $state(soundOn());

  // The route draws one stop after another, in at most ~0.45 s however long
  // the session; the stamp lands once it reaches the end.
  const LEAD = 180;
  const step = $derived(Math.min(110, 450 / Math.max(1, route.length)));
  const stampAt = $derived(LEAD + route.length * step + 220);
  // The stamp's fall takes this long before it hits the paper.
  const FALL = 170;

  const minutes = $derived(
    Math.max(1, Math.round((new Date(finished).getTime() - new Date(started).getTime()) / 60000)),
  );
  const volume = $derived(
    Math.round(
      sets.reduce(
        (sum, s) => (s.exercise_tracking_type === "weight_reps" ? sum + (s.weight_kg ?? 0) * (s.reps ?? 0) : sum),
        0,
      ),
    ),
  );
  const figures = $derived(
    [
      minutes < 60
        ? { n: String(minutes), unit: "min" }
        : { n: `${Math.floor(minutes / 60)}:${String(minutes % 60).padStart(2, "0")}`, unit: "h" },
      { n: String(sets.length), unit: sets.length === 1 ? "set" : "sets" },
      volume > 0 ? { n: volume.toLocaleString("en-GB").replace(/,/g, " "), unit: "kg" } : null,
    ].filter((f) => f !== null),
  );

  const date = $derived(
    new Date(finished).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" }),
  );

  onMount(() => {
    dialog.showModal();
    const still = matchMedia("(prefers-reduced-motion: reduce)").matches;
    const impact = still ? 200 : stampAt + FALL;
    cueFinish(impact);
    const t = setTimeout(() => {
      if (sound && "vibrate" in navigator) navigator.vibrate(18);
    }, impact);
    return () => clearTimeout(t);
  });

  function toggleSound() {
    sound = !sound;
    setSound(sound);
  }
</script>

<dialog
  bind:this={dialog}
  aria-labelledby="finish-title"
  onclose={ondone}
  style:--step="{step}ms"
  style:--lead="{LEAD}ms"
  style:--stamp="{stampAt}ms"
  style:--fall="{FALL}ms"
>
  <div class="page">
    <header class="top">
      <span class="cap">Workout finished</span>
      <button
        type="button"
        class="sound"
        aria-pressed={sound}
        aria-label="Sounds"
        title={sound ? "Sounds on" : "Sounds off"}
        onclick={toggleSound}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path class="body" d="M4 9.5h3.5L12 5.5v13l-4.5-4H4z" />
          {#if sound}
            <path d="M15.5 9a4.2 4.2 0 0 1 0 6M18 6.5a7.6 7.6 0 0 1 0 11" />
          {:else}
            <path d="M16 9.5l5 5M21 9.5l-5 5" />
          {/if}
        </svg>
      </button>
    </header>

    <h2 id="finish-title">{title}</h2>

    <p class="figures">
      {#each figures as f, i (f.unit)}
        {#if i}<span class="dot" aria-hidden="true">·</span>{/if}
        <span class="fig"><b>{f.n}</b> {f.unit}</span>
      {/each}
    </p>

    <ol class="route">
      {#each route as r, i (r.code)}
        <li style:--i={i}>
          <i class="seg" aria-hidden="true"></i>
          <i class="stop" aria-hidden="true"></i>
          <span class="code">{r.code}</span>
          <span class="name">{r.name}</span>
          <span class="n"><b>{r.sets}</b> {r.sets === 1 ? "set" : "sets"}</span>
        </li>
      {/each}
      <li class="end" style:--i={route.length} aria-hidden="true">
        <i class="bar"></i>
      </li>
    </ol>

    <div class="stamp" aria-hidden="true">
      <b>Done</b>
      <small>{date}</small>
    </div>

    <div class="go">
      <!-- svelte-ignore a11y_autofocus -->
      <Button variant="ink" size="lg" wide autofocus onclick={() => dialog.close()}>Back to workouts</Button>
    </div>
  </div>
</dialog>

<style>
  dialog {
    width: 100%;
    max-width: none;
    height: 100%;
    max-height: none;
    margin: 0;
    padding: 0;
    border: 0;
    background: var(--paper);
    color: var(--ink);
    overscroll-behavior: contain;
  }
  dialog::backdrop {
    background: var(--paper);
  }
  .page {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    max-width: 34rem;
    min-height: 100%;
    margin: 0 auto;
    padding: max(8px, env(safe-area-inset-top)) 12px max(20px, env(safe-area-inset-bottom));
    overflow-x: clip; /* the stamp starts large, partly off the page */
  }

  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 56px;
  }
  .top .cap {
    margin: 0;
    color: var(--ink-2);
  }
  .sound {
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    margin-right: -8px;
    border: 0;
    border-radius: var(--r);
    background: none;
    color: var(--ink);
    cursor: pointer;
  }
  .sound svg {
    width: 24px;
    height: 24px;
  }
  .sound path {
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  h2 {
    margin: 8px 0 8px;
    font: 400 clamp(52px, 15vw, 72px) / 0.92 var(--gothic);
    text-transform: uppercase;
    letter-spacing: 0.005em;
    overflow-wrap: anywhere;
    text-wrap: balance;
  }

  /* A dated rubber stamp, struck at an angle in the one ink. */
  .stamp {
    align-self: flex-end;
    display: grid;
    justify-items: center;
    margin: 26px 10px 0 0;
    padding: 6px 18px 8px;
    border: 3.5px solid var(--ink);
    border-radius: var(--r);
    outline: 1.5px solid var(--ink);
    outline-offset: 3px;
    transform: rotate(-8deg);
  }
  .stamp b {
    font: 400 72px / 0.9 var(--gothic);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .stamp small {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .figures {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 2px 10px;
    margin: 0 0 22px;
    padding-top: 8px;
    border-top: 2px solid var(--red);
    font-size: 13px;
    color: var(--ink-2);
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .fig b {
    font: 400 34px / 1 var(--gothic);
    color: var(--ink);
    letter-spacing: 0.01em;
  }

  /* Every exercise done, joined by the red route; it ends in a bar. */
  .route {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .route li {
    position: relative;
    display: grid;
    grid-template-columns: 22px 1fr auto;
    align-items: baseline;
    gap: 8px;
    min-height: 44px;
    padding-left: 26px;
  }
  .seg {
    position: absolute;
    left: 8px;
    top: 0;
    width: 2px;
    height: 100%;
    background: var(--red);
    transform-origin: top;
  }
  .route li:first-child .seg {
    top: 18px;
    height: calc(100% - 18px);
  }
  .stop {
    position: absolute;
    left: 3px;
    top: 12px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--red);
  }
  .code {
    font-weight: 700;
    letter-spacing: 0.04em;
  }
  .name {
    font: 400 24px / 1.1 var(--gothic);
    text-transform: uppercase;
    overflow-wrap: anywhere;
  }
  .n {
    color: var(--ink-2);
    font-size: 12px;
    text-transform: uppercase;
  }
  .n b {
    font: 400 24px / 1 var(--gothic);
    color: var(--ink);
  }
  .route .end {
    min-height: 0;
    height: 4px;
  }
  .bar {
    position: absolute;
    left: 0;
    top: 0;
    width: 18px;
    height: 4px;
    background: var(--red);
    transform-origin: center;
  }

  .go {
    margin-top: auto;
    padding-top: 28px;
  }

  @media (prefers-reduced-motion: no-preference) {
    .page {
      animation: settle 240ms var(--ease) both;
    }
    .seg {
      animation: draw var(--step) linear calc(var(--lead) + var(--i) * var(--step)) both;
    }
    .route li:first-child .seg {
      animation-duration: calc(var(--step) * 0.6);
    }
    .stop {
      animation: dot 200ms var(--ease) calc(var(--lead) + var(--i) * var(--step)) both;
    }
    .bar {
      animation: rule 200ms var(--ease) calc(var(--lead) + var(--i) * var(--step)) both;
    }
    /* Raised and tilted off the page, then pressed down hard. */
    .stamp {
      animation:
        stamp var(--fall) cubic-bezier(0.55, 0, 1, 0.45) var(--stamp) both,
        press 260ms var(--ease) calc(var(--stamp) + var(--fall)) forwards;
    }
    .page > :not(.stamp, .go) {
      animation: jolt 220ms var(--ease) calc(var(--stamp) + var(--fall)) both;
    }
    .go {
      animation: settle 260ms var(--ease) calc(var(--stamp) + var(--fall) + 120ms) both;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .stamp {
      animation: ink 200ms ease-out both;
    }
  }

  @keyframes settle {
    from {
      opacity: 0;
    }
  }
  @keyframes draw {
    from {
      transform: scaleY(0);
    }
  }
  @keyframes dot {
    from {
      transform: scale(0.2);
      opacity: 0;
    }
  }
  @keyframes rule {
    from {
      transform: scaleX(0);
    }
  }
  @keyframes stamp {
    from {
      opacity: 0;
      transform: translate(-18px, -26px) rotate(-16deg) scale(1.9);
    }
    to {
      opacity: 1;
      transform: rotate(-8deg) scale(0.96);
    }
  }
  @keyframes press {
    from {
      transform: rotate(-8deg) scale(0.96);
    }
    to {
      transform: rotate(-8deg) scale(1);
    }
  }
  @keyframes jolt {
    30% {
      transform: translateY(3px);
    }
  }
  @keyframes ink {
    from {
      opacity: 0;
    }
  }
</style>
