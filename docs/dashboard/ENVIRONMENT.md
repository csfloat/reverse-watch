# Reverse Watch — Local Environment

Last verified: 2026-05-23.

## Stack you need installed

- macOS (this doc is macOS-specific; CI is Ubuntu).
- Go 1.24+ (`go version` to check).
- PostgreSQL 18 via Homebrew (`brew install postgresql@18 && brew services start postgresql@18`).
- `gh` (GitHub CLI) for PR work.

## One-time Postgres setup

The repo's tests (`internal/testutil/db.go`) hardcode user `postgres`, password `postgres`, host `localhost:5432`. This matches CI (`.github/workflows/test.yml`). Don't change those test-side credentials — match locally instead:

```bash
psql -d postgres -c "ALTER USER postgres WITH SUPERUSER PASSWORD 'postgres';"
```

(If the `postgres` user doesn't exist yet on your install, swap `ALTER` for `CREATE`.)

Verify with:

```bash
PGPASSWORD=postgres psql -U postgres -h localhost -d postgres -c "SELECT current_user;"
```

Expected: a row showing `postgres`. If you get a password failure, the above ALTER didn't take.

## App config (`config.json`)

`config.json` is gitignored. Current local setup:

```json
{
  "Database": {
    "Host": "localhost", "Port": "5432",
    "User": "byskov", "Password": "devpassword",
    "PrivateDBName": "reverse_watch_private",
    "PublicDBName":  "reverse_watch_public"
  },
  "HTTP":        { "Port": "8080", "AllowedOrigins": ["http://localhost:8080"] },
  "Environment": "development"
}
```

Note the **app** uses Morten's user (`byskov/devpassword`) but **tests** use `postgres/postgres`. Both work; they hit different databases.

## Day-to-day commands

| Goal                           | Command                                                  |
|--------------------------------|----------------------------------------------------------|
| Boot the dev server            | `go run main.go` (runs at `http://localhost:8080/`)      |
| Run full test suite            | `go test ./...` (~15s, all packages)                     |
| Run one package's tests        | `go test ./repository/public/...`                        |
| Run one test by name           | `go test ./repository/public/ -run TestSummaryStats -v`  |
| Compile-only check (no run)    | `go build ./...`                                         |
| Format on save                 | Your editor should do this; `gofmt -w .` if not          |

## Workflow gotchas

- **Two terminals.** Keep one terminal alive for `go run main.go`. Use a second one for shell commands. Pasting `git`/`go test` into the dev-server terminal silently goes to the server's stdin, not the shell.
- **Don't push to `master`.** It's protected on `csfloat/reverse-watch`. Always branch + PR. Current dev branch: `feature/public-dashboard-v1`.
- **CI is identical to local tests.** If `go test ./...` is green locally, CI will be green.

## When something's wrong — debug ladder

1. `brew services list` — is Postgres running?
2. `PGPASSWORD=postgres psql -U postgres -h localhost -d postgres -c "SELECT 1;"` — can I authenticate?
3. `go env GOPATH GOROOT` — is Go on the right version?
4. `go mod download` — are deps in place?
5. Talk to the agent. Don't power through silently.
