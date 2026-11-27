<script lang="ts">
  import { api } from "../lib/api";
  import { BW_FIELD, BW_MAX, fmtChange, suspiciousWeight } from "../lib/bodyweight";
  import { fmtDay, fmtDaysAgo, fmtWeight, fromYmd, ymd } from "../lib/format";
  import type { BodyweightEntry } from "../lib/types";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import Stepper from "../ui/Stepper.svelte";

  // Home's bodyweight line: the latest weigh-in and its change, and a sheet
  // to log today's (prefilled with the last one, so usually a tap or two on
  // the stepper and Save) or fix an earlier day's.

  // How many recent entries the sheet lists.
  const SHOWN = 7;

  let { say }: { say: (text: string, error?: boolean) => void } = $props();

  let entries = $state<BodyweightEntry[] | null>(null); // newest first; null until loaded
  let open = $state(false);
  let day = $state("");
  let kg = $state<number | null>(null);
  let warning = $state<string | null>(null);
  let problem = $state<string | null>(null);
  let busy = $state(false);

  const today = ymd(new Date());
  const latest = $derived(entries?.[0] ?? null);
  const change = $derived(entries ? fmtChange(entries) : null);
  const existing = $derived(entries?.find((e) => e.day === day) ?? null);

  async function load() {
    try {
      entries = await api<BodyweightEntry[]>("GET", "/bodyweight?limit=" + SHOWN);
    } catch {
      entries = null; // the line just stays away; workouts matter more here
    }
  }
  load();

  function pick(d: string) {
    day = d;
    kg = entries?.find((e) => e.day === d)?.weight_kg ?? latest?.weight_kg ?? null;
    warning = problem = null;
  }

  function show() {
    pick(today);
    open = true;
  }

  async function save(force = false) {
    problem = null;
    if (kg == null || !Number.isFinite(kg) || kg < BW_FIELD.min || kg > BW_MAX) {
      problem = `Enter a weight between ${BW_FIELD.min} and ${BW_MAX} kg.`;
      return;
    }
    if (!day || day > today) {
      problem = "Pick a day up to today.";
      return;
    }
    const value = Math.round(kg * 100) / 100;
    if (!force) {
      warning = suspiciousWeight(value, day, entries ?? []);
      if (warning) return;
    }
    warning = null;
    busy = true;
    try {
      const saved = await api<BodyweightEntry>("PUT", `/bodyweight/${day}`, { weight_kg: value });
      entries = [saved, ...(entries ?? []).filter((e) => e.day !== saved.day)]
        .sort((a, b) => b.day.localeCompare(a.day))
        .slice(0, SHOWN);
      open = false;
      say(`Logged ${fmtWeight(saved.weight_kg)} kg${saved.day === today ? "" : " for " + fmtDay(saved.day + "T00:00")}`);
    } catch (e) {
      problem = e instanceof Error ? e.message : "Couldn't save your weight";
    } finally {
      busy = false;
    }
  }

  async function remove() {
    if (!existing) return;
    const gone = existing;
    busy = true;
    try {
      await api("DELETE", `/bodyweight/${gone.day}`);
      entries = (entries ?? []).filter((e) => e.day !== gone.day);
      open = false;
      say(`Deleted the weigh-in for ${fmtDay(gone.day + "T00:00")}`);
      load(); // an older entry may now fit in the list
    } catch (e) {
      problem = e instanceof Error ? e.message : "Couldn't delete it";
    } finally {
      busy = false;
    }
  }

  const ago = (d: string) => fmtDaysAgo(fromYmd(d).toISOString());
</script>

{#if entries}
  <section class="bw" aria-labelledby="bw-h">
    <div class="what">
      <h2 id="bw-h" class="cap">Bodyweight</h2>
      {#if latest}
        <p class="now">
          <span class="fig">{fmtWeight(latest.weight_kg)}</span><span class="unit">kg</span>
          <span class="sub">{ago(latest.day)}{change ? ` · ${change}` : ""}</span>
        </p>
      {:else}
        <p class="sub none">Not logged yet</p>
      {/if}
    </div>
    <Button size="sm" onclick={show} aria-label={latest?.day === today ? "Update today's bodyweight" : "Log bodyweight"}>
      {latest?.day === today ? "Update" : "Log"}
    </Button>
  </section>
{/if}

<Sheet bind:open title="Bodyweight">
  <form
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    <div class="two">
      <Stepper field={BW_FIELD} bind:value={kg} disabled={busy} />
      <Field label="Day">
        <input type="date" max={today} bind:value={day} oninput={() => pick(day)} disabled={busy} />
      </Field>
    </div>

    {#if warning}
      <p class="warn" role="alert">{warning}</p>
    {/if}
    {#if problem}
      <p class="warn" role="alert">{problem}</p>
    {/if}

    {#if warning}
      <div class="two">
        <Button onclick={() => (warning = null)}>Fix it</Button>
        <Button variant="ink" disabled={busy} onclick={() => save(true)}>Save anyway</Button>
      </div>
    {:else}
      <Button type="submit" variant="ink" size="lg" wide disabled={busy}>
        {busy ? "Saving…" : `${existing ? "Update" : "Save"}${kg != null && Number.isFinite(kg) ? ` ${fmtWeight(kg)} kg` : ""}`}
      </Button>
    {/if}
  </form>

  {#if entries?.length}
    <h3 class="cap recent-h">Recent</h3>
    <ol class="recent">
      {#each entries as e (e.day)}
        <li>
          <button type="button" class:on={e.day === day} aria-pressed={e.day === day} onclick={() => pick(e.day)}>
            <span>{e.day === today ? "Today" : fmtDay(e.day + "T00:00")}</span>
            <b>{fmtWeight(e.weight_kg)} kg</b>
          </button>
        </li>
      {/each}
    </ol>
  {/if}

  {#if existing}
    <div class="del">
      <Button variant="danger" disabled={busy} onclick={remove}>
        Delete {existing.day === today ? "today's" : fmtDay(existing.day + "T00:00") + "'s"} weigh-in
      </Button>
    </div>
  {/if}
</Sheet>

<style>
  .bw {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 12px;
    padding: 8px 0;
    border-top: 1.5px solid var(--ink);
    border-bottom: 1.5px solid var(--ink);
  }
  .what {
    min-width: 0;
  }
  h2.cap {
    margin: 0;
  }
  .now {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 0 6px;
    margin: 2px 0 0;
  }
  .fig {
    font: 400 32px/1 var(--gothic);
  }
  .unit {
    margin-left: -3px;
    font-size: 11px;
    font-weight: 700;
  }
  .sub {
    color: var(--ink-2);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .none {
    margin: 4px 0 0;
  }

  form {
    display: grid;
    gap: 12px;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    align-items: end;
  }
  .two :global(input[type="date"]) {
    min-height: 66px;
  }
  .warn {
    margin: 0;
    font-weight: 700;
  }
  h3.recent-h {
    margin: 18px 0 4px;
  }
  .recent {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .recent li {
    border-top: 1.5px solid var(--rule);
  }
  .recent button {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    width: 100%;
    min-height: 44px;
    padding: 0 4px;
    border: 0;
    border-radius: var(--r);
    background: none;
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--ink-2);
    cursor: pointer;
  }
  .recent button b {
    color: var(--ink);
    font: 400 22px/1 var(--gothic);
    letter-spacing: 0.01em;
  }
  .recent button.on {
    background: var(--paper-hi);
    color: var(--ink);
  }
  .del {
    display: flex;
    justify-content: center;
    margin-top: 12px;
  }
</style>
