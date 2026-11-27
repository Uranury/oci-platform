<script lang="ts">
  import { store } from "../lib/api";

  // What Overlift keeps about a person and why. Reachable logged out (from
  // sign-up) and in (from the side menu). Keep it true to the code and the
  // deployment: change it whenever what's stored, logged or kept changes.

  const UPDATED = "28 November 2026";
  const loggedIn = !!store.access;
</script>

<div class="screen">
  <header class="top">
    <a class="back" href={loggedIn ? "#/" : "#/login"}>
      <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12.5 4.5L7 10l5.5 5.5" /></svg>
      {loggedIn ? "Workouts" : "Log in"}
    </a>
  </header>

  <h1>Privacy</h1>
  <p class="meta">Updated <b>{UPDATED}</b></p>

  <p class="lead">
    Overlift is a small workout log run by one person for themselves and a few friends. It keeps what you put into it so
    it can show it back to you, and nothing else is done with it.
  </p>

  <section>
    <h2 class="cap">What you give it</h2>
    <ul>
      <li><b>Your account:</b> username, email and password. The password is stored only as a one-way hash; nobody can read it back.</li>
      <li><b>Your training:</b> workouts, sets, templates, custom exercises and bodyweight you log.</li>
    </ul>
  </section>

  <section>
    <h2 class="cap">What's recorded along the way</h2>
    <ul>
      <li>
        <b>Server logs:</b> the web server notes each request (your IP address, the time, the address asked for and your
        browser). They're used to keep the app running and find faults, and kept for about two weeks.
      </li>
      <li><b>Sessions:</b> a record of each device you're logged in on, so you stay logged in, until you log out or it expires.</li>
      <li>
        <b>On your device:</b> your login and the Sounds setting, in the browser's storage, and the app's own files for
        offline use. No cookies.
      </li>
    </ul>
  </section>

  <section>
    <h2 class="cap">What it's used for</h2>
    <p>
      Only to run the app for you: your history, "last time", prefilled sets and your progress. It isn't sold or shared,
      and there are no ads, analytics, trackers or third-party scripts.
    </p>
  </section>

  <section>
    <h2 class="cap">Who can see it</h2>
    <p>
      You, through the app. The person who runs Overlift can reach the server and its database to keep it working. Other
      users can't see anything of yours.
    </p>
  </section>

  <section>
    <h2 class="cap">Where it's kept</h2>
    <p>
      On servers rented from Oracle Cloud, reached over an encrypted (HTTPS) connection. The database is backed up every
      night, and each backup is deleted after about a week.
    </p>
  </section>

  <section>
    <h2 class="cap">Your choices</h2>
    <ul>
      <li><b>Take your history with you:</b> menu → Export history downloads your workouts as a spreadsheet file.</li>
      <li>
        <b>Delete your account:</b> menu → Delete account removes your account and everything in it straight away. It
        leaves the last backups within about a week.
      </li>
      <li><b>Questions,</b> or a copy of everything kept about you: ask the person who invited you to Overlift.</li>
    </ul>
  </section>

  <p class="small">If what's kept changes, this page changes with it, and the date at the top moves.</p>
</div>

<style>
  .screen {
    max-width: 34rem;
    margin: 0 auto;
    padding: 0 16px calc(48px + env(safe-area-inset-bottom));
  }
  .top {
    display: flex;
    align-items: center;
    min-height: 56px;
    padding-top: max(8px, env(safe-area-inset-top));
  }
  .back {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    min-height: 44px;
    padding-right: 8px;
    margin-left: -6px;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    text-decoration: none;
  }
  .back svg {
    width: 20px;
    height: 20px;
  }
  .back path {
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  h1 {
    margin: 2px 0 8px;
    font: 400 clamp(52px, 15vw, 72px) / 0.92 var(--gothic);
    text-transform: uppercase;
  }
  .meta {
    margin: 0 0 18px;
    padding-top: 8px;
    border-top: 2px solid var(--ink);
    color: var(--ink-2);
    font-size: 13px;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .meta b {
    color: var(--ink);
  }
  .lead {
    margin: 0 0 8px;
    font-size: 16px;
    font-weight: 600;
    line-height: 1.5;
  }
  section {
    margin-top: 18px;
    padding-top: 8px;
    border-top: 1.5px solid var(--rule);
  }
  h2.cap {
    margin: 0 0 6px;
  }
  p,
  li {
    font-size: 14px;
    line-height: 1.55;
  }
  p {
    margin: 0;
  }
  ul {
    margin: 0;
    padding-left: 18px;
  }
  li + li {
    margin-top: 6px;
  }
  .small {
    margin-top: 24px;
    color: var(--ink-2);
    font-size: 12px;
  }
</style>
