<script lang="ts">
  import { tick } from "svelte";
  import { api, ApiError, logout } from "../lib/api";
  import { setSound, soundOn } from "../lib/sound";
  import Button from "./Button.svelte";
  import Notice from "./Notice.svelte";

  // The Overlift name in a screen's header, which opens the side menu: a
  // drawer from the left holding device settings and the account. Built on
  // <dialog>, so focus is trapped and Escape closes it. Profile goes here too.

  let open = $state(false);
  let sound = $state(soundOn());
  let confirming = $state(false);
  let busy = $state(false);
  let problem = $state<string | null>(null);
  let dialog: HTMLDialogElement;
  let account: HTMLElement;

  // Swapping the buttons for the question (and back) would drop focus to the
  // top of the dialog; keep it where the hand is.
  async function ask(on: boolean) {
    confirming = on;
    problem = null;
    await tick();
    account.querySelector<HTMLButtonElement>(on ? ".notice button" : ".apart button")?.focus();
  }

  $effect(() => {
    if (open && !dialog.open) {
      sound = soundOn(); // may have changed on the workout screen
      dialog.showModal();
    }
    if (!open && dialog.open) dialog.close();
  });

  function closed() {
    open = false;
    confirming = false;
    problem = null;
  }

  async function logoutAll() {
    busy = true;
    problem = null;
    try {
      await api("POST", "/auth/logout-all");
      logout();
    } catch (e) {
      problem =
        e instanceof ApiError && e.status === 0
          ? "You're offline, so the other devices can't be logged out yet. Try again when connected."
          : "Couldn't log out the other devices. Try again.";
    } finally {
      busy = false;
    }
  }
</script>

<button type="button" class="trigger" aria-haspopup="dialog" aria-expanded={open} onclick={() => (open = true)}>
  <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 5.5h14M3 10h14M3 14.5h9" /></svg>
  <span>Overlift</span>
  <span class="sr-only">menu</span>
</button>

<dialog
  bind:this={dialog}
  aria-labelledby="menu-h"
  onclose={closed}
  onclick={(e) => {
    if (e.target === dialog) open = false; // backdrop
  }}
>
  <div class="panel">
    <header>
      <h2 id="menu-h">Overlift</h2>
      <button type="button" class="close" aria-label="Close menu" onclick={() => (open = false)}>
        <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M5 5l10 10M15 5L5 15" /></svg>
      </button>
    </header>

    <section aria-labelledby="menu-device">
      <h3 id="menu-device" class="cap">This device</h3>
      <label class="toggle">
        <input
          type="checkbox"
          checked={sound}
          onchange={(e) => {
            sound = e.currentTarget.checked;
            setSound(sound);
          }}
        />
        Sounds when logging and finishing
      </label>
    </section>

    <section aria-labelledby="menu-account" bind:this={account}>
      <h3 id="menu-account" class="cap">Account</h3>
      {#if confirming}
        <Notice message="Log out on every device, this one included?">
          <Button disabled={busy} onclick={() => ask(false)}>Cancel</Button>
          <Button variant="ink" disabled={busy} onclick={logoutAll}>{busy ? "Logging out…" : "Log out all"}</Button>
        </Notice>
        {#if problem}<p class="problem" role="alert">{problem}</p>{/if}
      {:else}
        <Button wide onclick={logout}>Log out</Button>
        <div class="apart">
          <Button variant="danger" onclick={() => ask(true)}>Log out everywhere</Button>
        </div>
      {/if}
    </section>
  </div>
</dialog>

<style>
  .trigger {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    min-height: 44px;
    margin-left: -6px;
    padding: 0 6px;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--ink);
    font: inherit;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .trigger svg {
    width: 20px;
    height: 20px;
    flex: none;
  }
  path {
    fill: none;
    stroke: var(--ink);
    stroke-width: 2;
    stroke-linecap: round;
  }

  dialog {
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    width: min(21rem, 86vw);
    max-width: none;
    height: 100dvh;
    max-height: none;
    margin: 0 auto 0 0;
  }
  dialog::backdrop {
    background: #1c1b1973;
  }
  .panel {
    box-sizing: border-box;
    height: 100%;
    overflow-y: auto;
    overscroll-behavior: contain;
    background: var(--paper);
    border-right: 2px solid var(--ink);
    padding: max(8px, env(safe-area-inset-top)) 14px max(20px, env(safe-area-inset-bottom))
      max(14px, env(safe-area-inset-left));
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    min-height: 56px;
    border-bottom: 2px solid var(--ink);
    margin-bottom: 18px;
  }
  h2 {
    margin: 0;
    font: 400 32px/1 var(--gothic);
    text-transform: uppercase;
    letter-spacing: 0.01em;
  }
  h3 {
    margin: 0 0 6px;
  }
  section + section {
    margin-top: 26px;
  }
  .close {
    width: 44px;
    height: 44px;
    display: grid;
    place-items: center;
    margin-right: -4px;
    border: 0;
    background: none;
    cursor: pointer;
  }
  .close svg {
    width: 20px;
    height: 20px;
  }
  .toggle {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 44px;
    font-weight: 600;
    cursor: pointer;
  }
  .toggle input {
    width: 20px;
    height: 20px;
    margin: 0;
  }
  .apart {
    display: flex;
    justify-content: flex-end;
    margin-top: 22px;
  }
  .problem {
    margin: 10px 0 0;
    font-weight: 700;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
  @media (prefers-reduced-motion: no-preference) {
    dialog[open] .panel {
      animation: slide 260ms var(--ease);
    }
    @keyframes slide {
      from {
        transform: translateX(-24px);
        opacity: 0.6;
      }
    }
  }
</style>
