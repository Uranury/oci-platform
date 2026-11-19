<script lang="ts">
  import type { Snippet } from "svelte";

  // A bottom sheet on phones, a centred panel on wider screens. Built on
  // <dialog>, so focus is trapped and Escape closes it.
  let { open = $bindable(false), title, children }: {
    open: boolean;
    title: string;
    children: Snippet;
  } = $props();

  let dialog: HTMLDialogElement;

  $effect(() => {
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  });
</script>

<dialog
  bind:this={dialog}
  aria-label={title}
  onclose={() => (open = false)}
  onclick={(e) => {
    if (e.target === dialog) open = false; // backdrop
  }}
>
  <div class="panel">
    <header>
      <h2>{title}</h2>
      <button type="button" class="close" aria-label="Close" onclick={() => (open = false)}>
        <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M5 5l10 10M15 5L5 15" /></svg>
      </button>
    </header>
    {#if open}{@render children()}{/if}
  </div>
</dialog>

<style>
  dialog {
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    width: 100%;
    max-width: 34rem;
    max-height: 100dvh;
    margin: auto auto 0;
  }
  dialog::backdrop {
    background: #1c1b1973;
  }
  .panel {
    background: var(--paper);
    border-top: 2px solid var(--ink);
    border-radius: var(--r) var(--r) 0 0;
    padding: 14px 14px max(20px, env(safe-area-inset-bottom));
    max-height: 88dvh;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
  @media (min-width: 600px) {
    dialog {
      margin: auto;
    }
    .panel {
      border: var(--line);
      border-radius: var(--r);
    }
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }
  h2 {
    margin: 0;
    font: 400 32px/1 var(--gothic);
    text-transform: uppercase;
  }
  .close {
    width: 44px;
    height: 44px;
    display: grid;
    place-items: center;
    border: 0;
    background: none;
    cursor: pointer;
    margin-right: -10px;
  }
  .close svg {
    width: 20px;
    height: 20px;
  }
  .close path {
    stroke: var(--ink);
    stroke-width: 2;
    stroke-linecap: round;
  }
  @media (prefers-reduced-motion: no-preference) {
    dialog[open] .panel {
      animation: rise 260ms var(--ease);
    }
    @keyframes rise {
      from {
        transform: translateY(24px);
        opacity: 0.6;
      }
    }
  }
</style>
