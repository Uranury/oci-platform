<script lang="ts">
  import { tick } from "svelte";
  import { ApiError, download } from "../lib/api";
  import Button from "./Button.svelte";
  import Field from "./Field.svelte";

  // The side menu's "Your data" section: the history as a CSV file, one row
  // per set, for a chosen range of days. Collapsed to one button until asked
  // for; days are the user's own (the API gets the device's time zone).

  type Preset = "30d" | "year" | "all";

  let open = $state(false);
  let preset = $state<Preset | null>("30d");
  let from = $state("");
  let to = $state("");
  let busy = $state(false);
  let problem = $state<string | null>(null);
  let done = $state<string | null>(null);
  let panel = $state<HTMLElement>();

  // YYYY-MM-DD for a local day; what <input type="date"> reads and writes.
  function ymd(d: Date): string {
    const p = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
  }
  const today = ymd(new Date());

  function pick(p: Preset) {
    preset = p;
    const now = new Date();
    to = p === "all" ? "" : today;
    if (p === "30d") from = ymd(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 29));
    else if (p === "year") from = ymd(new Date(now.getFullYear(), 0, 1));
    else from = "";
    problem = done = null;
  }
  pick("30d");

  // Typing a date is a custom range.
  function edited() {
    preset = null;
    problem = done = null;
  }

  let backwards = $derived(from !== "" && to !== "" && from > to);

  async function toggle() {
    open = !open;
    if (open) {
      await tick();
      panel?.querySelector<HTMLButtonElement>('[aria-pressed="true"], button')?.focus();
    }
  }

  function longDay(day: string): string {
    // "T00:00" makes it local midnight; a bare date would be read as UTC.
    return new Date(day + "T00:00").toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
  }

  function rangeText(): string {
    if (!from && !to) return "in your history";
    if (!from) return `up to ${longDay(to)}`;
    if (!to) return `since ${longDay(from)}`;
    return from === to ? `on ${longDay(from)}` : `between ${longDay(from)} and ${longDay(to)}`;
  }

  async function save() {
    busy = true;
    problem = done = null;
    const q = new URLSearchParams({ tz: Intl.DateTimeFormat().resolvedOptions().timeZone });
    if (from) q.set("from", from);
    if (to) q.set("to", to);
    try {
      const { blob, filename } = await download("/workouts/export?" + q);
      // The header is the first line; a range with no sets has nothing else.
      const rows = (await blob.text()).trimEnd().split("\n").length - 1;
      if (rows === 0) {
        problem = `Nothing logged ${rangeText()}. Pick a wider range.`;
        return;
      }
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.append(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 10_000); // after the download has started
      done = `${filename} · ${rows} ${rows === 1 ? "set" : "sets"}`;
    } catch (e) {
      problem =
        e instanceof ApiError && e.status === 0
          ? "You're offline, so there's nothing to export yet. Try again when connected."
          : e instanceof ApiError && e.status === 400
            ? `Couldn't export: ${e.message}.`
            : "Couldn't export your history. Try again.";
    } finally {
      busy = false;
    }
  }
</script>

<Button wide aria-expanded={open} aria-controls="export-panel" onclick={toggle}>
  Export history (CSV)
</Button>

{#if open}
  <div id="export-panel" class="panel" bind:this={panel}>
    <div class="chips" role="group" aria-label="Range">
      <button type="button" class="chip" aria-pressed={preset === "30d"} onclick={() => pick("30d")}>30 days</button>
      <button type="button" class="chip" aria-pressed={preset === "year"} onclick={() => pick("year")}>This year</button>
      <button type="button" class="chip" aria-pressed={preset === "all"} onclick={() => pick("all")}>All time</button>
    </div>

    <div class="dates">
      <Field label="From">
        <input type="date" bind:value={from} max={to || today} oninput={edited} />
      </Field>
      <Field label="To">
        <input type="date" bind:value={to} min={from || undefined} max={today} oninput={edited} />
      </Field>
    </div>
    <p class="hint">{!from && !to ? "Every workout you've logged." : "Leave a date empty for no limit."} One row per set.</p>

    {#if backwards}
      <p class="problem" role="alert">From is after To. Swap the dates.</p>
    {/if}

    <Button variant="ink" wide disabled={busy || backwards} onclick={save}>
      {busy ? "Exporting…" : "Download CSV"}
    </Button>

    <div aria-live="polite">
      {#if problem}<p class="problem">{problem}</p>{/if}
      {#if done}<p class="done">Saved {done}</p>{/if}
    </div>
  </div>
{/if}

<style>
  .panel {
    display: grid;
    gap: 12px;
    margin-top: 12px;
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .chip {
    flex: 1 1 auto;
    min-height: 44px;
    padding: 0 12px;
    border: 1px solid var(--rule);
    border-radius: 4px;
    background: transparent;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .chip[aria-pressed="true"] {
    border: 1.5px solid var(--ink);
    background: var(--paper-hi);
    font-weight: 700;
  }
  .dates {
    display: grid;
    gap: 10px;
  }
  .hint {
    margin: -4px 0 0;
    font-size: 12px;
    color: var(--ink-2);
  }
  .problem {
    margin: 0;
    font-weight: 700;
  }
  .done {
    margin: 0;
    font-size: 12px;
    overflow-wrap: anywhere;
  }
</style>
