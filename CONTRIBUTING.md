# Contributing to Overlift

Thanks for wanting to help. Overlift is a small workout tracker: a Go API with
an embedded Svelte frontend, backed by Postgres. Bug reports, fixes and ideas
are all welcome.

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Reporting a bug or suggesting a feature

Open a [GitHub issue](https://github.com/Uranury/oci-platform/issues). For a
bug, say what you did, what you expected and what happened instead (with the
browser or device if it's a frontend problem). For a feature, describe the
problem it solves first; [`PRODUCT.md`](PRODUCT.md) explains what the app is
for and what it deliberately leaves out (ads, feeds, streak nagging).

Found a security problem? Please don't open a public issue: report it
privately to the maintainer through the contact details on
[their GitHub profile](https://github.com/Uranury).

## Setting up

You need Go (version in `go.mod`), Node (version in `web/.nvmrc`) and Docker
for the local database.

```bash
git clone https://github.com/Uranury/oci-platform.git
cd oci-platform
docker compose -f local/compose.yaml up --build   # Postgres + the app on :8080
```

For frontend work, run the Vite dev server on top of that (it hot-reloads and
forwards API calls to :8080):

```bash
cd web && npm ci && npm run dev   # http://localhost:5173
```

The [README](README.md) covers configuration and the full API.
[`CLAUDE.md`](CLAUDE.md) explains how the code is laid out and the conventions
it follows (repo/service/handler layers, transactions, auth, the design
system), so read the part about the area you're changing first.

## Making a change

1. Fork the repository and create a branch from `master`.
2. Keep the change focused: one fix or feature per pull request.
3. Follow the existing patterns:
   - SQL lives in `internal/repo/queries/*.sql`; run `sqlc generate` after
     changing a query or adding a migration, and commit the generated code.
   - Schema changes are a new numbered pair in `migrations/`; never edit an
     existing migration.
   - Build screens from the shared components in `web/src/ui/`.
4. Add or update tests for what you changed.
5. Run the same checks CI runs:

   ```bash
   gofmt -l .                 # should print nothing
   go vet ./...
   golangci-lint run ./...
   docker compose -f local/compose.yaml up -d db
   TEST_DATABASE_URL='postgres://liftlog:liftlog@localhost:5433/postgres?sslmode=disable' go test ./...
   cd web && npm run check && npm run build
   ```

6. Open a pull request against `master` describing what changed and why.
   CI must pass before it's merged.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/): every green
push to `master` is tagged with the next version based on them.

- `feat: ...` a new feature (minor version bump)
- `fix: ...` a bug fix (patch)
- `docs:`, `chore:`, `refactor:`, `test:` ... anything else (patch)
- `feat!: ...` or a `BREAKING CHANGE:` footer: a breaking change (major)

## License

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
