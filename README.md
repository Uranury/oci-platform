# oci-platform

Lift Log: a small workout tracker (log workouts, exercises and sets) as a Go REST API plus an embedded web frontend, with JWT-based auth and refresh-token rotation. Built as the application layer for a Linux/DevOps coursework project (3-tier deploy: Nginx → this API → Postgres).

## Stack

- Go 1.26, stdlib `net/http` (Go 1.22+ method+pattern routing, no router dependency)
- Postgres via `jackc/pgx/v5`
- `golang-migrate` for schema migrations, embedded into the binary via `embed.FS`
- `golang-jwt/jwt/v5` for access tokens, bcrypt for passwords

## Running it

Requires a reachable Postgres instance. Copy `.env.example` to `.env` (or export the same variables in your shell/systemd unit — the app does not load `.env` files itself):

```bash
cp .env.example .env
export $(cat .env | xargs)   # or your preferred way of loading it
go run ./cmd/app
```

Required: `DATABASE_URL`, `JWT_SECRET`. Everything else has a default — see `.env.example`.

Schema migrations run automatically on startup; nothing to apply by hand. Open `http://localhost:8080/` for the web app.

## API

All request/response bodies are JSON.

| Method | Path              | Auth | Description                          |
|--------|-------------------|------|---------------------------------------|
| POST   | `/auth/register`  | –    | Create an account                     |
| POST   | `/auth/login`     | –    | Get an access + refresh token pair    |
| POST   | `/auth/refresh`   | –    | Rotate a refresh token for a new pair |
| POST   | `/auth/logout`    | –    | Revoke one refresh token              |
| POST   | `/auth/logout-all`| Bearer | Revoke every refresh token for the user ("log out everywhere") |
| GET    | `/exercises`      | Bearer | List your exercises                 |
| POST   | `/exercises`      | Bearer | Create an exercise (`{"name"}`; 409 if the name exists) |
| DELETE | `/exercises/{id}` | Bearer | Delete an exercise (409 while it has logged sets) |
| GET    | `/workouts`       | Bearer | List your workouts, newest first    |
| POST   | `/workouts`       | Bearer | Start a workout (`{"name"?, "notes"?, "performed_at"?}`; date defaults to now) |
| GET    | `/workouts/{id}`  | Bearer | Get a workout with its sets         |
| PUT    | `/workouts/{id}`  | Bearer | Update name, notes, date            |
| DELETE | `/workouts/{id}`  | Bearer | Delete a workout and its sets       |
| POST   | `/workouts/{id}/sets` | Bearer | Log a set (`{"exercise_id", "reps", "weight_kg"}`) |
| DELETE | `/workouts/{id}/sets/{setID}` | Bearer | Delete a set               |
| GET    | `/` (and static files) | – | The web frontend (`web/static`) |
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

CI (`.github/workflows/ci.yml`) runs all of the above on every push and PR to `master`.

## Project layout

See `CLAUDE.md` for the package-by-package breakdown and the conventions the codebase follows (repo/service/handler layering, transaction usage, auth design, etc.).
