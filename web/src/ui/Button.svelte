<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLButtonAttributes } from "svelte/elements";

  // ink: the one primary action in a region (solid black).
  // ghost: secondary actions (ruled outline).
  // danger: destructive actions, deliberately quiet (underlined text);
  //   place it apart from the primary action, never beside it.
  let {
    variant = "ghost",
    size = "md",
    wide = false,
    type = "button",
    children,
    ...rest
  }: HTMLButtonAttributes & {
    variant?: "ink" | "ghost" | "danger";
    size?: "sm" | "md" | "lg";
    wide?: boolean;
    children: Snippet;
  } = $props();
</script>

<button {type} class="btn {variant} {size}" class:wide {...rest}>{@render children()}</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    min-height: 52px;
    padding: 0 10px;
    border-radius: var(--r);
    font: 700 14px/1.2 var(--mono);
    letter-spacing: 0.1em;
    text-transform: uppercase;
    cursor: pointer;
    touch-action: manipulation;
  }
  .sm {
    min-height: 44px;
    padding: 0 18px;
  }
  .lg {
    min-height: 56px;
    font-size: 15px;
  }
  .wide {
    width: 100%;
  }
  .ink {
    border: 0;
    background: var(--ink);
    color: var(--paper);
  }
  .ink:active:not(:disabled) {
    transform: translateY(1px);
  }
  .ghost {
    border: var(--line);
    background: transparent;
    color: var(--ink);
  }
  .ghost:active:not(:disabled) {
    background: var(--paper-hi);
  }
  .danger {
    min-height: 44px;
    padding: 0 4px;
    border: 0;
    background: none;
    color: var(--ink);
    font-size: 12px;
    text-decoration: underline;
    text-underline-offset: 4px;
  }
  .btn:disabled {
    opacity: 0.55;
    cursor: default;
  }
  .btn :global(svg) {
    width: 20px;
    height: 20px;
    flex: none;
  }
</style>
