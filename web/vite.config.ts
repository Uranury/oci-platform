import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// Paths the Go API answers; `npm run dev` forwards them to the app on :8080
// (`docker compose -f local/compose.yaml up` or `go run ./cmd/app`), so the
// frontend gets hot reload while talking to the real backend.
const api = ["/auth", "/exercise-options", "/exercises", "/workouts", "/templates", "/healthz"];
const backend = process.env.API_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [svelte()],
  build: {
    // Embedded by web/embed.go. A subdirectory, so emptying it on every
    // build leaves the committed dist/.gitkeep (which lets `go build` work
    // before the frontend has ever been built) alone.
    outDir: "dist/app",
    emptyOutDir: true,
  },
  server: {
    proxy: Object.fromEntries(api.map((p) => [p, backend])),
  },
});
