# oci-platform

Lift Log: a small workout tracker (log workouts, exercises and sets) as a Go REST API plus an embedded web frontend, with JWT-based auth and refresh-token rotation. Built as the application layer for a Linux/DevOps coursework project (3-tier deploy: Nginx → this API → Postgres).

## Stack

- Go 1.26, Gin for routing
- Frontend: Svelte 5 + TypeScript, built with Vite into `web/dist/app` and embedded into the binary (no separate web server)
- Postgres via `jackc/pgx/v5`
- `golang-migrate` for schema migrations, embedded into the binary via `embed.FS`
- `golang-jwt/jwt/v5` for access tokens, bcrypt for passwords

## Running it

Requires a reachable Postgres instance. Copy `.env.example` to `.env` (or export the same variables in your shell/systemd unit — the app does not load `.env` files itself):

```bash
cp .env.example .env
export $(cat .env | xargs)   # or your preferred way of loading it
npm --prefix web ci && npm --prefix web run build   # the frontend, embedded by go build
go run ./cmd/app
```

Required: `DATABASE_URL`, `JWT_SECRET`. Everything else has a default — see `.env.example`.

Schema migrations run automatically on startup; nothing to apply by hand. Open `http://localhost:8080/` for the web app. Without the npm step the binary still builds and the API works, but `/` shows a "frontend not built" page.

## API

All request/response bodies are JSON.

| Method | Path              | Auth | Description                          |
|--------|-------------------|------|---------------------------------------|
| POST   | `/auth/register`  | –    | Create an account                     |
| POST   | `/auth/login`     | –    | Get an access + refresh token pair    |
| POST   | `/auth/refresh`   | –    | Rotate a refresh token for a new pair |
| POST   | `/auth/logout`    | –    | Revoke one refresh token              |
| POST   | `/auth/logout-all`| Bearer | Revoke every refresh token for the user ("log out everywhere") |
| GET    | `/exercise-options` | –  | The fixed category lists (patterns, muscles, mechanics, equipment, tracking types) |
| GET    | `/exercises`      | Bearer | Library defaults + your own custom exercises (archived hidden) |
| POST   | `/exercises`      | Bearer | Create a custom exercise (all category fields required; 409 if you already have that name) |
| GET    | `/exercises/similar?name=` | Bearer | Defaults whose names closely match, for a "did you mean…" warning |
| PUT    | `/exercises/{id}` | Bearer | Edit your custom exercise (403 for defaults) |
| DELETE | `/exercises/{id}` | Bearer | Delete your custom exercise; archived instead if it has logged sets (`{"result":"deleted"\|"archived"}`) |
| POST   | `/exercises/{id}/duplicate` | Bearer | Copy a default (or your own) into a new custom exercise; optional `{"name"}` |
| GET    | `/workouts`       | Bearer | List your workouts, newest first    |
| POST   | `/workouts`       | Bearer | Start a workout (`{"name"?, "notes"?, "performed_at"?}`; date defaults to now) |
| GET    | `/workouts/{id}`  | Bearer | Get a workout with its sets         |
| PUT    | `/workouts/{id}`  | Bearer | Update name, notes, date            |
| DELETE | `/workouts/{id}`  | Bearer | Delete a workout and its sets       |
| POST   | `/workouts/{id}/sets` | Bearer | Log a set: `{"exercise_id"}` plus the measurements its `tracking_type` takes — `reps`, `weight_kg`, `duration_seconds`, `distance_m` |
| DELETE | `/workouts/{id}/sets/{setID}` | Bearer | Delete a set               |
| GET    | `/` (and static files) | – | The web frontend (built from `web/`) |
| GET    | `/healthz`        | –    | Liveness check                        |

"Bearer" routes require `Authorization: Bearer <access_token>`. Access tokens are short-lived (`ACCESS_TOKEN_TTL`, default 15m); use `/auth/refresh` with the refresh token to get a new pair.

## Development

```bash
go build ./...
go vet ./...
gofmt -l .              # should print nothing
golangci-lint run ./...
go test ./...
```

The repo, service and handler tests need a real Postgres; without one they're skipped. Point them at the local dev database (each test creates and drops its own database there, cloned from a migrated template, so your dev data isn't touched):

```bash
docker compose -f local/compose.yaml up -d db
TEST_DATABASE_URL='postgres://liftlog:liftlog@localhost:5433/postgres?sslmode=disable' go test ./...
```

Frontend work: run the backend (e.g. `docker compose -f local/compose.yaml up`), then the Vite dev server, which hot-reloads and forwards API calls to `:8080` (`API_URL` overrides):

```bash
cd web
npm ci         # once, and after package-lock.json changes
npm run dev    # http://localhost:5173
npm run check  # svelte-check: types and Svelte warnings
```

Queries live in `internal/repo/queries/*.sql`; after editing one (or adding a migration) regenerate the typed Go code and commit it:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1   # once
sqlc generate
```

CI (`.github/workflows/ci.yml`) runs all of the above on every push and PR to `master`, plus `sqlc diff` (fails if generated code is stale), `npm run check` and a frontend build, ansible-lint, and a linux/arm64 release build with the frontend embedded. Every green push to `master` is tagged with the next version (`feat:` → minor, `fix:`/other → patch, `type!:` / `BREAKING CHANGE` → major), so deploys are named after the tag.

## Project layout

See `CLAUDE.md` for the package-by-package breakdown and the conventions the codebase follows (repo/service/handler layering, transaction usage, auth design, etc.).
