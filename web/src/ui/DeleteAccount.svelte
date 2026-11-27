<script lang="ts">
  import { tick } from "svelte";
  import { api, ApiError, store } from "../lib/api";
  import { flash } from "../lib/flash";
  import { go } from "../lib/nav";
  import Button from "./Button.svelte";
  import Field from "./Field.svelte";

  // The side menu's account deletion: collapsed to a quiet button until
  // asked for, then says what goes and wants the password before it deletes
  // the account and everything in it.

  let open = $state(false);
  let password = $state("");
  let busy = $state(false);
  let problem = $state<string | null>(null);
  let panel = $state<HTMLElement>();

  async function toggle(on: boolean) {
    open = on;
    password = "";
    problem = null;
    if (on) {
      await tick();
      panel?.querySelector("input")?.focus();
    }
  }

  async function remove(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    problem = null;
    try {
      await api("POST", "/auth/delete-account", { password });
      store.clear();
      flash("Your account and everything in it were deleted.");
      go("#/login");
    } catch (ex) {
      problem =
        ex instanceof ApiError && ex.status === 403
          ? "That password isn't right."
          : ex instanceof Error
            ? ex.message
            : "Couldn't delete the account. Try again.";
    } finally {
      busy = false;
    }
  }
</script>

{#if open}
  <form class="panel" bind:this={panel} onsubmit={remove} aria-labelledby="delete-h">
    <p id="delete-h" class="what">
      This deletes your account and everything in it: workouts, templates, custom exercises and bodyweight. It can't be
      undone. Export your history first if you want to keep it.
    </p>
    <Field label="Password">
      <input type="password" bind:value={password} autocomplete="current-password" required disabled={busy} />
    </Field>
    {#if problem}<p class="problem" role="alert">{problem}</p>{/if}
    <div class="two">
      <Button disabled={busy} onclick={() => toggle(false)}>Cancel</Button>
      <Button type="submit" variant="ink" disabled={busy || !password}>{busy ? "Deleting…" : "Delete everything"}</Button>
    </div>
  </form>
{:else}
  <Button variant="danger" onclick={() => toggle(true)}>Delete account</Button>
{/if}

<style>
  .panel {
    display: grid;
    gap: 12px;
    padding: 12px;
    border: 2px solid var(--ink);
    border-radius: var(--r);
  }
  .what {
    margin: 0;
    font-weight: 600;
    line-height: 1.45;
  }
  .problem {
    margin: 0;
    font-weight: 700;
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
</style>
