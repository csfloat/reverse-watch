# [reverse.watch](https://reverse.watch)

Community-driven open trade reversal tracking database for Steam. Participating entities can report trade reverals to the open database.

## Interested in Participating?

If you're looking to participate by contributing reversal reports (i.e. marketplace, trading tool, community site, etc...), reach out at **join@reverse.watch**.

## Running Locally

1. Ensure Go 1.24+ and PostgreSQL are installed.
2. Copy the config template and fill in your local database credentials:
   ```bash
   cp config.example.json config.json
   ```
3. Run the service:
   ```bash
   go run main.go
   ```

The server starts on port `80` by default (configurable via `HTTP_PORT`).

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

## Seeding local data

Production ingests live data from contributing marketplaces. For local development, a synthetic dataset (~6 months, ~2k rows with realistic daily variance) can be loaded with:

```bash
go run ./cmd/seed
```

The seed:

- Must be run in a development environment against a local database.
- Generates a 6-month dataset ending today, so the dashboard at `/` has enough data to exercise every period (7d / 30d / 3m / 6m / 1y).
- Uses Steam IDs from the unallocated top of the 32-bit account space (`76561201986797568`–`76561202255233023`) and `dev-`-prefixed marketplace slugs, so synthetic rows never collide with real accounts or real ingestors.
- Skips rows that already exist, so a plain rerun inserts nothing. To refresh the data so it ends at today, delete the existing synthetic rows first:

```bash
go run ./cmd/seed -reset
```