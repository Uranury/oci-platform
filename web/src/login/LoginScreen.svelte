<script lang="ts">
  import { api, store, type TokenPair } from "../lib/api";
  import Button from "../ui/Button.svelte";
  import Field from "../ui/Field.svelte";

  let mode = $state<"login" | "register">("login");
  let username = $state("");
  let email = $state("");
  let password = $state("");
  let error = $state<string | null>(null);
  let busy = $state(false);

  function setMode(m: "login" | "register") {
    mode = m;
    error = null;
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    busy = true;
    try {
      if (mode === "register") {
        await api("POST", "/auth/register", { username: username.trim(), email: email.trim(), password }, { auth: false });
      }
      const pair = await api<TokenPair>("POST", "/auth/login", { email: email.trim(), password }, { auth: false });
      store.set(pair);
      location.hash = "#/";
    } catch (ex) {
      error = ex instanceof Error ? ex.message : "Something went wrong";
    } finally {
      busy = false;
    }
  }
</script>

<main class="auth">
  <h1>Overlift</h1>
  <p class="tag">Your training log</p>

  <div class="modes" role="group" aria-label="Log in or sign up">
    <button type="button" aria-pressed={mode === "login"} onclick={() => setMode("login")}>Log in</button>
    <button type="button" aria-pressed={mode === "register"} onclick={() => setMode("register")}>Sign up</button>
  </div>

  <form onsubmit={submit}>
    {#if mode === "register"}
      <Field label="Username">
        <input bind:value={username} name="username" autocomplete="username" required />
      </Field>
    {/if}
    <Field label="Email">
      <input bind:value={email} type="email" name="email" autocomplete="email" required />
    </Field>
    <Field label="Password">
      <input
        bind:value={password}
        type="password"
        name="password"
        autocomplete={mode === "login" ? "current-password" : "new-password"}
        minlength="8"
        required
      />
    </Field>
    {#if mode === "register"}<p class="hint">At least 8 characters.</p>{/if}

    {#if error}<p class="error" role="alert">{error}</p>{/if}

    <Button type="submit" variant="ink" size="lg" wide disabled={busy}>
      {busy ? "One moment…" : mode === "login" ? "Log in" : "Create account"}
    </Button>
  </form>
</main>

<style>
  .auth {
    max-width: 24rem;
    margin: 0 auto;
    padding: max(14vh, 48px) 16px 48px;
  }
  h1 {
    margin: 0;
    padding-bottom: 8px;
    border-bottom: 2px solid var(--red);
    font: 400 clamp(64px, 22vw, 96px) / 0.9 var(--gothic);
    text-transform: uppercase;
  }
  .tag {
    margin: 8px 0 28px;
    color: var(--ink-2);
    font-size: 13px;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .modes {
    display: grid;
    grid-template-columns: 1fr 1fr;
    margin-bottom: 20px;
    border: var(--line);
    border-radius: var(--r);
    overflow: hidden;
  }
  .modes button {
    min-height: 48px;
    border: 0;
    background: transparent;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .modes button + button {
    border-left: var(--line);
  }
  /* The selected mode reads like the active tab (underlined), so the
     submit stays the only solid ink block. */
  .modes button {
    color: var(--ink-2);
  }
  .modes button[aria-pressed="true"] {
    background: var(--paper-hi);
    color: var(--ink);
    text-decoration: underline;
    text-decoration-thickness: 2px;
    text-underline-offset: 6px;
  }
  form {
    display: grid;
    gap: 14px;
  }
  .hint {
    margin: -6px 0 0;
    color: var(--ink-2);
    font-size: 12px;
  }
  /* Server messages arrive lowercase ("invalid email or password"). */
  .error::first-letter {
    text-transform: uppercase;
  }
  .error {
    margin: 0;
    padding: 10px 12px;
    border: 2px solid var(--ink);
    border-radius: var(--r);
    font-weight: 700;
  }
</style>
