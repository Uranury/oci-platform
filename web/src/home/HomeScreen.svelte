<script lang="ts">
  import { onMount } from "svelte";
  import { api, logout } from "../lib/api";
  import { takeFlash } from "../lib/flash";
  import { fmtClock, fmtDay, fmtDaysAgo, fmtNames, fmtSession } from "../lib/format";
  import type { Template, TemplateSummary, Workout, WorkoutSummary } from "../lib/types";
  import BiroRing from "../ui/BiroRing.svelte";
  import Button from "../ui/Button.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import TabBar from "../ui/TabBar.svelte";
  import Toast, { type ToastMsg } from "../ui/Toast.svelte";

  // How many templates the start sheet lists before linking to the rest.
  const SHOWN_TEMPLATES = 3;
  // A workout without an end time counts as in progress for this long.
  const IN_PROGRESS_HOURS = 12;

  let workouts = $state<WorkoutSummary[] | null>(null);
  let templates = $state<TemplateSummary[]>([]);
  let due = $state<Template | null>(null); // the first template, with its exercises
  let loadError = $state<string | null>(null);
  let starting = $state<string | null>(null); // what's being started: a template id or "empty"
  let toast = $state<ToastMsg | null>(null);
  let toastSeq = 0;
  let deleting = $state<WorkoutSummary | null>(null); // asked before deleting
  let busy = $state(false);

  function say(text: string, error = false) {
    toast = { id: ++toastSeq, text, error };
  }

  async function load() {
    loadError = null;
    try {
      const [w, t] = await Promise.all([
        api<WorkoutSummary[]>("GET", "/workouts"),
        api<TemplateSummary[]>("GET", "/templates"),
      ]);
      workouts = w;
      templates = t;
      // The exercise line under the due template; the screen works without it.
      if (t[0]) due = await api<Template>("GET", `/templates/${t[0].id}`).catch(() => null);
    } catch (e) {
      loadError = e instanceof Error ? e.message : "Couldn't load your workouts";
    }
  }

  onMount(() => {
    const f = takeFlash();
    if (f) say(f);
    load();
  });

  const now = new Date();
  const inProgress = $derived(
    workouts?.find(
      (w) => !w.finished_at && now.getTime() - new Date(w.performed_at).getTime() < IN_PROGRESS_HOURS * 3600_000,
    ) ?? null,
  );
  const lead = $derived(templates[0] ?? null);
  // The rows under the lead: the next templates, never the one already in
  // progress (Resume covers it), at most SHOWN_TEMPLATES - 1 of them.
  const others = $derived(
    inProgress ? templates.filter((t) => t.id !== inProgress.template_id) : templates.slice(1),
  );
  const rows = $derived(others.slice(0, SHOWN_TEMPLATES - 1));
  const hidden = $derived(others.length - rows.length);

  // Sessions since Monday.
  const thisWeek = $derived.by(() => {
    const monday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - ((now.getDay() + 6) % 7));
    return (workouts ?? []).filter((w) => new Date(w.performed_at) >= monday && w.set_count > 0).length;
  });

  async function start(templateId: string | null) {
    starting = templateId ?? "empty";
    try {
      const w = templateId
        ? await api<Workout>("POST", `/templates/${templateId}/start`)
        : await api<Workout>("POST", "/workouts", {});
      location.hash = `#/workouts/${w.id}`;
    } catch (e) {
      say(e instanceof Error ? e.message : "Couldn't start the workout", true);
      starting = null;
    }
  }

  const name = (w: { name?: string | null }) => w.name?.trim() || "Workout";

  async function removeWorkout(w: WorkoutSummary) {
    busy = true;
    try {
      await api("DELETE", `/workouts/${w.id}`);
      workouts = workouts?.filter((x) => x.id !== w.id) ?? null;
      deleting = null;
      say(`Deleted “${name(w)}”`);
    } catch (e) {
      say(e instanceof Error ? e.message : "Couldn't delete the workout", true);
    } finally {
      busy = false;
    }
  }
</script>

<div class="screen">
  <header class="top">
    <span class="brand">Overlift</span>
    <button type="button" class="quiet" onclick={logout}>Log out</button>
  </header>

  <h1>Workouts</h1>
  <p class="meta">
    <span>{thisWeek} {thisWeek === 1 ? "session" : "sessions"} this week</span>
    <b>{fmtDay(now.toISOString())}</b>
  </p>

  {#if loadError}
    <p class="state" role="alert">{loadError}</p>
    <Button onclick={load}>Try again</Button>
  {:else if !workouts}
    <p class="state" aria-busy="true">Loading…</p>
  {:else}
    <section class="sheet" aria-labelledby="start-h">
      <!-- Named for screen readers; visually the ringed name and the button say it. -->
      <h2 id="start-h" class="sr-only">{inProgress ? "Workout in progress" : "Start a workout"}</h2>

      {#if inProgress}
        <div class="lead">
          <span class="ringed"><span class="big">{name(inProgress)}</span><BiroRing /></span>
          <p class="sub">
            In progress · started {fmtClock(inProgress.performed_at)} ·
            <b>{inProgress.set_count} {inProgress.set_count === 1 ? "set" : "sets"}</b>
          </p>
        </div>
        <Button href="#/workouts/{inProgress.id}" variant="ink" size="lg" wide>Resume {name(inProgress)}</Button>
      {:else if lead}
        <div class="lead">
          <span class="ringed"><span class="big">{lead.name}</span><BiroRing /></span>
          <p class="sub">
            {lead.last_used_at ? `Last done ${fmtDaysAgo(lead.last_used_at)}` : "Never done"}
            {#if due?.id === lead.id && due.exercises.length}
              · <b>{fmtNames(due.exercises.map((e) => e.exercise_name))}</b>
            {:else}
              · {lead.exercise_count} {lead.exercise_count === 1 ? "exercise" : "exercises"}
            {/if}
          </p>
        </div>
        <Button variant="ink" size="lg" wide disabled={starting !== null} onclick={() => start(lead.id)}>
          {starting === lead.id ? "Starting…" : `Start ${lead.name}`}
        </Button>
      {:else}
        <p class="sub none">No templates yet. Start an empty workout, or plan one you can repeat.</p>
        <Button variant="ink" size="lg" wide disabled={starting !== null} onclick={() => start(null)}>
          {starting === "empty" ? "Starting…" : "Start empty workout"}
        </Button>
        <a class="link center" href="#/templates/new">Create a template</a>
      {/if}

      {#if rows.length}
        <ul class="rows">
          {#each rows as t (t.id)}
            <li>
              <div>
                <span class="row-name">{t.name}</span>
                <span class="row-sub">
                  {t.last_used_at ? fmtDaysAgo(t.last_used_at) : "never done"} · {t.exercise_count}
                  {t.exercise_count === 1 ? "exercise" : "exercises"}
                </span>
              </div>
              <Button size="sm" disabled={starting !== null} onclick={() => start(t.id)}>
                {starting === t.id ? "…" : "Start"}
              </Button>
            </li>
          {/each}
        </ul>
      {/if}

      {#if templates.length}
        <div class="links">
          <button type="button" class="link" disabled={starting !== null} onclick={() => start(null)}>
            {starting === "empty" ? "Starting…" : "Empty workout"}
          </button>
          {#if hidden}<a class="link" href="#/templates">All templates ({templates.length})</a>{/if}
        </div>
      {/if}
    </section>

    <h2 class="cap history-h">History</h2>
    {#if workouts.length}
      <ol class="history">
        {#each workouts as w (w.id)}
          <li>
            <a href="#/workouts/{w.id}">
              <span class="date">
                {new Date(w.performed_at).toLocaleDateString("en-GB", { weekday: "short" })}
                <span>{new Date(w.performed_at).getDate()}</span>
              </span>
              <span class="what">
                <span class="row-name">{name(w)}</span>
                <span class="row-sub">
                  {#if w.set_count}
                    {fmtNames(w.exercises, 2)} · {w.set_count} {w.set_count === 1 ? "set" : "sets"}
                  {:else}
                    No sets logged
                  {/if}
                </span>
              </span>
              <span class="took">
                {w.finished_at
                  ? fmtSession(w.performed_at, w.finished_at)
                  : w.id === inProgress?.id
                    ? "now"
                    : ""}
              </span>
            </a>
            <button
              type="button"
              class="del"
              aria-label="Delete {name(w)}, {fmtDay(w.performed_at)}"
              onclick={() => (deleting = w)}
            >
              <svg viewBox="0 0 20 20" aria-hidden="true">
                <path d="M3.5 5.5h13M8 5.5V3.5h4v2M5.5 5.5l.8 11h7.4l.8-11M8.5 8.5v5.5M11.5 8.5v5.5" />
              </svg>
            </button>
          </li>
        {/each}
      </ol>
    {:else}
      <p class="sub none">No workouts yet. Your sessions will show up here.</p>
    {/if}
  {/if}
</div>

<TabBar active="workouts" />
<Toast bind:toast />

<Sheet bind:open={() => deleting !== null, (v) => !v && (deleting = null)} title="Delete workout?">
  {#if deleting}
    {@const n = deleting.set_count}
    <p class="ask">
      This deletes “{name(deleting)}” from {fmtDay(deleting.performed_at)}{n
        ? ` and all ${n} of its ${n === 1 ? "set" : "sets"}`
        : ""}. It can't be undone.
    </p>
    <div class="two">
      <Button onclick={() => (deleting = null)}>Keep it</Button>
      <Button variant="ink" disabled={busy} onclick={() => deleting && removeWorkout(deleting)}>Delete</Button>
    </div>
  {/if}
</Sheet>

<style>
  .screen {
    max-width: 34rem;
    margin: 0 auto;
    padding: 0 12px calc(90px + env(safe-area-inset-bottom));
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 56px;
    padding-top: max(8px, env(safe-area-inset-top));
  }
  .brand,
  .quiet {
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .quiet {
    min-height: 44px;
    margin-right: -6px;
    padding: 0 6px;
    border: 0;
    background: none;
    cursor: pointer;
  }
  h1 {
    margin: 2px 0 8px;
    font: 400 clamp(52px, 15vw, 72px) / 0.92 var(--gothic);
    text-transform: uppercase;
  }
  .meta {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin: 0 0 14px;
    padding-top: 8px;
    border-top: 2px solid var(--red);
    color: var(--ink-2);
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .meta b {
    color: var(--ink);
  }
  .state {
    margin: 24px 0 16px;
    font-weight: 600;
  }

  .sheet {
    padding: 12px;
    border: 2.5px solid var(--ink);
    border-radius: var(--r);
    background: var(--paper-hi);
  }
  h2.cap {
    margin: 0;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
  .lead {
    margin: 0 0 12px;
  }
  /* Room inside the biro loop so it circles the name instead of crossing
     it; the loop's tail starts inside its left edge, so more on the left. */
  .ringed {
    position: relative;
    display: inline-block;
    padding: 4px 16px 2px 20px;
    margin-left: -6px;
  }
  .big {
    font: 400 42px/0.95 var(--gothic);
    text-transform: uppercase;
    overflow-wrap: anywhere;
  }
  .sub {
    margin: 6px 0 0;
    color: var(--ink-2);
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .sub b {
    color: var(--ink);
  }
  .none {
    margin: 10px 0 12px;
    text-transform: none;
    font-size: 14px;
  }
  .rows {
    margin: 12px 0 0;
    padding: 0;
    list-style: none;
  }
  .rows li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 0;
    border-top: 1.5px solid var(--rule);
  }
  .rows li > div,
  .what {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .row-name {
    font: 400 28px/1 var(--gothic);
    text-transform: uppercase;
    overflow-wrap: anywhere;
  }
  .row-sub {
    margin-top: 3px;
    color: var(--ink-2);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .links {
    display: flex;
    justify-content: center;
    gap: 24px;
    margin-top: 8px;
  }
  .link {
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    padding: 0 4px;
    border: 0;
    background: none;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    text-decoration: underline;
    text-underline-offset: 4px;
    cursor: pointer;
  }
  .link:disabled {
    opacity: 0.55;
  }
  .center {
    display: flex;
    justify-content: center;
    margin-top: 6px;
  }

  h2.history-h {
    margin: 24px 0 6px;
  }
  .history {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .history li {
    display: flex;
    align-items: stretch;
    border-top: 1.5px solid var(--ink);
  }
  .history a {
    flex: 1;
    min-width: 0;
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr) auto;
    gap: 10px;
    align-items: start;
    min-height: 56px;
    padding: 8px 0;
    text-decoration: none;
  }
  .history a:active {
    background: var(--paper-hi);
  }
  .date {
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    line-height: 1.2;
  }
  .date span {
    display: block;
    font: 400 26px/1 var(--gothic);
  }
  .took {
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .del {
    flex: none;
    display: grid;
    place-items: center;
    width: 44px;
    margin-right: -10px;
    border: 0;
    border-radius: var(--r);
    background: none;
    color: var(--ink-2);
    cursor: pointer;
  }
  .del:active {
    background: var(--paper-hi);
    color: var(--ink);
  }
  .del svg {
    width: 20px;
    height: 20px;
  }
  .del path {
    fill: none;
    stroke: currentColor;
    stroke-width: 1.6;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .ask {
    margin: 0 0 16px;
    font-weight: 600;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .history .row-name {
    font-size: 24px;
  }
</style>
