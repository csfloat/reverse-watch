# [reverse.watch](https://reverse.watch)

Community-driven open trade reversal tracking database for Steam. Participating entities can report trade reversals to the open database.

## Interested in Participating?

If you're looking to participate by contributing reversal reports (i.e. marketplace, trading tool, community site, etc...), reach out at **join@reverse.watch**.

## Running Locally

1. Ensure Go 1.24+ and PostgreSQL are installed.
2. Create the two local databases and (for tests) a `postgres` superuser:
   ```bash
   createdb private
   createdb public
   psql -d postgres -c "CREATE USER postgres WITH SUPERUSER PASSWORD 'postgres';"
   # If the user already exists:
   # psql -d postgres -c "ALTER USER postgres WITH SUPERUSER PASSWORD 'postgres';"
   ```
   The superuser is required by [`pgtestdb`](https://github.com/peterldowns/pgtestdb), which spins up disposable databases per test.
3. Copy the config template and fill in your local database credentials:
   ```bash
   cp config.example.json config.json
   ```
4. Run the service:
   ```bash
   go run main.go
   ```

The server starts on port `80` by default (configurable via `HTTP_PORT`).

### Running tests

```bash
go test ./...
```

Tests use `pgtestdb` to provision a fresh database per test against the local Postgres. The `postgres/postgres` superuser from step 2 above is required.

## Public read endpoints

| Endpoint | Purpose |
|---|---|
| `GET /api/v1/users/{steamId}` | Single Steam-ID lookup (existing) |
| `GET /api/v1/stats/summary` | Three KPI counts in one call |
| `GET /api/v1/stats/reversals/daily?days={7\|30\|60\|90\|180\|365}` | Daily reversal counts, UTC, zero-filled |
| `GET /api/v1/reversals/recent?limit={1..100}` | Latest non-expunged reversals (slim public projection) |

All four are public, IP-rate-limited, and return JSON. The two `/stats` endpoints have a 60-second in-process cache.

## Configuration

Configuration is loaded from environment variables or a `config.json` file.

## Authentication

All entity endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer reversewatch_live_xxxxxxxx...
```

API keys are scoped to an entity and carry a permission bitfield. Keys are prefixed with `reversewatch_live_` (production) or `reversewatch_test_` (development).

### Permissions

| Permission | Description |
|---|---|
| `admin` | Full administrative access (Service operator only) |
| `manage` | Manage API keys for own entity |
| `write` | Create reversal reports |
| `delete` | Expunge reversal reports for own entity |
| `read` | Read access |
| `export` | List and export reversal data |

## Rate Limiting

Rate limits are enforced in-memory per process. Throttled responses return `429 Too Many Requests` with `X-RateLimit-*` and `Retry-After` headers.
