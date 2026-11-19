<script lang="ts" module>
  export interface ToastMsg {
    id: number;
    text: string;
    error?: boolean;
    action?: { label: string; run: () => void };
  }
</script>

<script lang="ts">
  let { toast = $bindable() }: { toast: ToastMsg | null } = $props();

  // Messages with an action (Undo) stay long enough to reach for it.
  $effect(() => {
    if (!toast) return;
    const id = toast.id;
    const t = setTimeout(() => {
      if (toast?.id === id) toast = null;
    }, toast.action ? 6000 : 3000);
    return () => clearTimeout(t);
  });
</script>

<div class="slot" role="status" aria-live="polite">
  {#if toast}
    {#key toast.id}
      <div class="toast" class:error={toast?.error}>
        <span>{toast?.text}</span>
        {#if toast.action}
          <button
            type="button"
            onclick={() => {
              const run = toast?.action?.run; // read before clearing
              toast = null;
              run?.();
            }}>{toast?.action?.label}</button
          >
        {/if}
      </div>
    {/key}
  {/if}
</div>

<style>
  .slot {
    position: fixed;
    left: 12px;
    right: 12px;
    bottom: calc(66px + env(safe-area-inset-bottom));
    z-index: 30;
    display: flex;
    justify-content: center;
    pointer-events: none;
  }
  .toast {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: 14px;
    max-width: 32rem;
    width: 100%;
    background: var(--ink);
    color: var(--paper);
    border-radius: var(--r);
    padding: 4px 4px 4px 14px;
    min-height: 48px;
    font-weight: 600;
  }
  .toast span {
    flex: 1;
    padding: 8px 0;
  }
  .toast.error {
    background: var(--paper);
    color: var(--ink);
    border: 2px solid var(--ink);
  }
  button {
    flex: none;
    min-height: 40px;
    padding: 0 14px;
    border: 1.5px solid var(--paper);
    border-radius: 4px;
    background: none;
    color: var(--paper);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    cursor: pointer;
  }
  @media (prefers-reduced-motion: no-preference) {
    .toast {
      animation: up 220ms var(--ease);
    }
    @keyframes up {
      from {
        transform: translateY(10px);
        opacity: 0;
      }
    }
  }
</style>
