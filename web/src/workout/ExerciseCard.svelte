<script lang="ts">
  import { untrack, type Snippet } from "svelte";
  import { fmtLast, fmtSets, fmtTarget } from "../lib/format";
  import { FIELDS, chip, overlifted, prefill, suspicious, toBody, type Values } from "../lib/sets";
  import type { Plan, Previous, SetBody, TrackingType, WorkoutSet } from "../lib/types";
  import BiroRing from "../ui/BiroRing.svelte";
  import BiroTick from "../ui/BiroTick.svelte";
  import Button from "../ui/Button.svelte";
  import Notice from "../ui/Notice.svelte";
  import OverMark from "../ui/OverMark.svelte";
  import Stepper from "../ui/Stepper.svelte";

  type Stop = "done" | "active" | "todo";

  let {
    exerciseId,
    code,
    name,
    type,
    plan = null,
    sets,
    last,
    unavailable = false,
    active,
    stop,
    lineIn,
    lineOut,
    freshSetId = null,
    onactivate,
    onlog,
    onupdate,
    ondelete,
    onremove,
    handle,
  }: {
    exerciseId: string;
    code: string;
    name: string;
    type: TrackingType;
    plan?: Plan | null;
    sets: WorkoutSet[];
    last?: Previous;
    unavailable?: boolean; // archived since: sets show, nothing new is logged
    active: boolean;
    stop: Stop; // this exercise's mark on the route
    lineIn: "red" | "plan" | null; // the route segment arriving here
    lineOut: "red" | "plan" | null; // and leaving
    freshSetId?: string | null;
    onactivate: () => void;
    onlog: (body: SetBody) => Promise<void>;
    onupdate: (setId: string, body: SetBody) => Promise<void>;
    ondelete: (setId: string) => Promise<void>;
    onremove: () => void; // take the exercise out of the workout
    handle?: Snippet; // the drag handle, beside the heading
  } = $props();

  let editingId = $state<string | null>(null);
  let values = $state<Values>({});
  let warning = $state<string | null>(null);
  let problem = $state<string | null>(null);
  let busy = $state(false);
  let draft = $state(0); // bumped on each prefill: remounts the steppers with fresh text

  const editing = $derived(editingId ? (sets.find((s) => s.id === editingId) ?? null) : null);
  const fields = $derived(FIELDS[type]);
  const target = $derived(plan?.target_sets ?? 0);
  const done = $derived(sets.length);
  const nextNo = $derived(done + 1);
  // Dashed placeholders for the planned sets still to come after the next one.
  const later = $derived(Math.max(0, target - nextNo));

  // Today's best set beats last session's: the red coach's note by the name.
  // It is drawn with the pen only when earned while the screen is open.
  const over = $derived(overlifted(type, sets, last?.sets));
  let overFresh = $state(false);
  let wasOver = untrack(() => over);
  $effect(() => {
    if (over && !wasOver) {
      overFresh = true;
      setTimeout(() => (overFresh = false), 700); // drawn once, not on every expand
    }
    wasOver = over;
  });

  // Fresh values whenever what the form is for changes: another exercise
  // becomes active, a set is logged or deleted, or one is picked to edit.
  $effect(() => {
    void [active, sets.length, editingId];
    values = prefill(type, { editing, logged: sets, last: last?.sets, plan });
    draft = untrack(() => draft) + 1;
    warning = null;
    problem = null;
  });

  // Leaving the card ends an edit.
  $effect(() => {
    if (!active) editingId = null;
  });

  async function submit(force = false) {
    const body = toBody(exerciseId, type, values);
    if (typeof body === "string") {
      problem = body;
      return;
    }
    problem = null;
    if (!force) {
      const history = [...sets.filter((s) => s.id !== editingId), ...(last?.sets ?? [])];
      warning = suspicious(body, history);
      if (warning) return;
    }
    warning = null;
    busy = true;
    try {
      if (editing) {
        await onupdate(editing.id, body);
        editingId = null;
      } else {
        await onlog(body);
      }
    } finally {
      busy = false;
    }
  }

  async function remove() {
    if (!editing) return;
    busy = true;
    try {
      await ondelete(editing.id);
      editingId = null;
    } finally {
      busy = false;
    }
  }
</script>

<div class="item" class:active>
  <span class="route" aria-hidden="true">
    {#if lineIn}<i class="seg in {lineIn}"></i>{/if}
    {#if lineOut}<i class="seg out {lineOut}"></i>{/if}
    <i class="stop {stop}"></i>
  </span>

  <section class="card" aria-labelledby="ex-{code}">
    <div class="headrow">
      {#if active}
        <header class="head">
          <h2 id="ex-{code}"><span class="code">{code}</span>{name}{#if over}<OverMark fresh={overFresh} />{/if}</h2>
          {#if plan}<span class="target">{fmtTarget(plan)}</span>{/if}
        </header>
      {:else}
        <button type="button" class="head toggle" onclick={onactivate} aria-expanded="false">
          <h2 id="ex-{code}"><span class="code">{code}</span>{name}{#if over}<OverMark fresh={overFresh} />{/if}</h2>
          <span class="count">{target ? `${done} / ${target}` : `${done} set${done === 1 ? "" : "s"}`}</span>
        </button>
      {/if}
      {@render handle?.()}
    </div>

    {#if !active && done}
      <p class="today">Today <b>{fmtSets(sets)}</b></p>
    {/if}
    {#if last}
      <p class="meta">{fmtLast(last.performed_at)} <b>{fmtSets(last.sets)}</b></p>
    {:else if active}
      <p class="meta">First time logging this</p>
    {/if}
    {#if active && plan?.notes}<p class="meta note">{plan.notes}</p>{/if}
    {#if unavailable}<p class="meta">Archived: logged sets stay, new ones can't be added</p>{/if}

    {#if active}
      <ol class="chips" aria-label="Sets">
        {#each sets as s, i (s.id)}
          <li>
            <button
              type="button"
              class="chip logged"
              class:editing={s.id === editingId}
              aria-pressed={s.id === editingId}
              aria-label="Set {code}{i + 1}: {chip(s)}. Edit"
              onclick={() => (editingId = editingId === s.id ? null : s.id)}
            >
              <span class="no">{code}{i + 1}</span>
              <span class="fig">{chip(s)}</span>
              <BiroTick fresh={s.id === freshSetId} />
              {#if s.id === editingId}{#key s.id}<BiroRing />{/key}{/if}
            </button>
          </li>
        {/each}
        {#if !unavailable && !editingId}
          <li class="chip next" aria-current={editingId ? undefined : "step"}>
            <span class="no">{code}{nextNo}</span>
            <span class="lbl">next</span>
            {#key done}<BiroRing />{/key}
          </li>
          {#each { length: later } as _, i (i)}
            <li class="chip planned"><span class="no">{code}{nextNo + i + 1}</span></li>
          {/each}
        {/if}
      </ol>

      {#if !unavailable}
        <form
          class="entry"
          onsubmit={(e) => {
            e.preventDefault();
            submit();
          }}
        >
          {#key draft}
            <div class="steps" style:--n={fields.length}>
              {#each fields as f (f.key)}
                <Stepper field={f} bind:value={values[f.key]} disabled={busy} />
              {/each}
            </div>
          {/key}

          {#if problem}<p class="problem" role="alert">{problem}</p>{/if}
          <!-- While a warning is up, its own buttons are the choice. -->
          {#if warning}
            <div class="gap">
              <Notice message={warning}>
                <Button onclick={() => (warning = null)}>Fix it</Button>
                <Button onclick={() => submit(true)}>{editing ? "Save anyway" : "Log anyway"}</Button>
              </Notice>
            </div>
          {:else if editing}
            <div class="pair">
              <Button disabled={busy} onclick={() => (editingId = null)}>Cancel</Button>
              <Button type="submit" variant="ink" disabled={busy}>Save {code}{sets.indexOf(editing) + 1}</Button>
            </div>
            <div class="apart">
              <Button variant="danger" disabled={busy} onclick={remove}>
                Delete set {code}{sets.indexOf(editing) + 1}
              </Button>
            </div>
          {:else}
            <div class="gap">
              <Button type="submit" variant="ink" size="lg" wide disabled={busy}>
                {busy ? "Logging…" : `Log set ${code}${nextNo}`}
              </Button>
            </div>
            <div class="apart">
              <Button variant="danger" disabled={busy} onclick={onremove}>Remove {code} from workout</Button>
            </div>
          {/if}
        </form>
      {:else}
        <div class="entry apart">
          <Button variant="danger" onclick={onremove}>Remove {code} from workout</Button>
        </div>
      {/if}
    {/if}
  </section>
</div>

<style>
  .item {
    position: relative;
    padding-left: 26px;
    margin-bottom: 10px;
  }
  .route {
    position: absolute;
    left: 0;
    top: 0;
    bottom: -10px;
    width: 18px;
  }
  .seg {
    position: absolute;
    left: 8px;
    width: 2px;
  }
  .seg.in {
    top: 0;
    height: 25px;
  }
  .seg.out {
    top: 25px;
    bottom: 0;
  }
  .seg.red {
    background: var(--red);
  }
  .seg.plan {
    width: 0;
    border-left: 1.5px dashed var(--ink-2);
  }
  .stop {
    position: absolute;
    left: 3px;
    top: 19px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    box-sizing: border-box;
  }
  .stop.done {
    background: var(--red);
  }
  .stop.active {
    background: var(--paper);
    border: 3px solid var(--red);
  }
  .stop.todo {
    background: var(--paper);
    border: 1.5px solid var(--ink-2);
  }

  .card {
    border: var(--line);
    border-radius: var(--r);
    background: var(--paper-hi);
  }
  .item:not(.active) .card {
    background: transparent;
  }
  .active .card {
    border-width: 2.5px;
  }
  .headrow {
    display: flex;
    align-items: flex-start;
  }
  .headrow > :first-child {
    flex: 1;
    min-width: 0;
  }
  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 12px 0;
  }
  .toggle {
    width: 100%;
    border: 0;
    background: none;
    text-align: left;
    cursor: pointer;
    min-height: 44px;
  }
  h2 {
    margin: 0;
    font: 400 30px/1 var(--gothic);
    text-transform: uppercase;
    letter-spacing: 0.01em;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .active h2 {
    font-size: 36px;
  }
  .code {
    display: inline-grid;
    place-items: center;
    width: 22px;
    height: 22px;
    margin-right: 8px;
    border: var(--line);
    border-radius: 3px;
    font: 700 13px/1 var(--mono);
    vertical-align: 0.32em;
  }
  .target,
  .count {
    flex: none;
    font-weight: 700;
    letter-spacing: 0.04em;
  }
  .count {
    font: 400 26px/1 var(--gothic);
    white-space: nowrap;
  }
  .today {
    margin: 6px 12px 0;
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .today b {
    font-weight: 700;
  }
  .item:not(.active) .today:last-child {
    margin-bottom: 10px;
  }
  .meta {
    margin: 4px 12px 0;
    color: var(--ink-2);
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .item:not(.active) .meta:last-child {
    margin-bottom: 10px;
  }
  .meta b {
    color: var(--ink);
    font-weight: 700;
  }
  .note {
    text-transform: none;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 12px;
    margin: 12px 0 0;
    padding: 0 12px;
  }
  .chips > li {
    list-style: none;
  }
  .chip {
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 44px;
    padding: 0 10px;
    border-radius: 4px;
  }
  button.chip {
    border: var(--line);
    background: transparent;
    cursor: pointer;
  }
  .no {
    font-weight: 700;
    font-size: 12px;
    letter-spacing: 0.04em;
  }
  .fig {
    font: 400 26px/1 var(--gothic);
  }
  .chip.next {
    padding: 0 14px;
  }
  .next .lbl {
    font-size: 12px;
    font-weight: 700;
    color: var(--biro);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .planned {
    border: 1.5px dashed var(--ink-2);
    color: var(--ink-2);
  }

  .entry {
    padding: 14px 12px 12px;
  }
  .steps {
    display: grid;
    grid-template-columns: repeat(var(--n), minmax(0, 1fr));
    gap: 10px;
  }
  .problem {
    margin: 10px 0 0;
    font-weight: 700;
  }
  .gap {
    margin-top: 12px;
  }
  .pair {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-top: 12px;
  }
  /* Delete stays well away from Save. */
  .apart {
    display: flex;
    justify-content: flex-end;
    margin-top: 28px;
  }
</style>
