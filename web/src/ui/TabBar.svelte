<script lang="ts">
  let { active }: { active: "workouts" | "templates" | "exercises" } = $props();
  const tabs = [
    { id: "workouts", href: "#/", label: "Workouts" },
    { id: "templates", href: "#/templates", label: "Templates" },
    { id: "exercises", href: "#/exercises", label: "Exercises" },
  ] as const;
</script>

<nav aria-label="Main">
  {#each tabs as t (t.id)}
    <a href={t.href} aria-current={t.id === active ? "page" : undefined}>{t.label}</a>
  {/each}
</nav>

<style>
  nav {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 20;
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 11.33rem));
    justify-content: center;
    background: var(--paper);
    border-top: 2px solid var(--ink);
    padding-bottom: env(safe-area-inset-bottom);
  }
  a {
    display: grid;
    place-items: center;
    height: 54px;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    text-decoration: none;
    color: var(--ink-2);
  }
  a[aria-current="page"] {
    color: var(--ink);
    text-decoration: underline;
    text-decoration-thickness: 2px;
    text-underline-offset: 6px;
  }
</style>
