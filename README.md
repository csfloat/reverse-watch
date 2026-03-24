# reverse.watch

A shared trade reversal tracking service for CS2 skin marketplaces. Marketplaces have had an influx of users reversing trades due to recent price volatility. This has resulted in an unsatisfactory user experience and a great amount of overhead for support staff. reverse.watch provides a better experience for both staff and users by disincentivizing trade reversals through a shared, cross-marketplace reversal database.

Participating marketplaces report trade reversals to the service, and any marketplace (or end user) can query whether a given Steam account has a history of reversals. This shared visibility discourages abuse and reduces support burden across the ecosystem.

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

All marketplace endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer reversewatch_live_xxxxxxxx...
```

API keys are scoped to a marketplace and carry a permission bitfield. Keys are prefixed with `reversewatch_live_` (production) or `reversewatch_test_` (development). The key ID stored in the database is the SHA-256 hash of the full secret.

### Permissions

| Permission | Description |
|---|---|
| `admin` | Full administrative access (CSFloat only) |
| `manage` | Manage API keys for own marketplace |
| `write` | Create reversal reports |
| `delete` | Expunge reversal reports for own marketplace |
| `read` | Read access |
| `export` | List and export reversal data |

## Rate Limiting

Rate limits are enforced in-memory per process. Throttled responses return `429 Too Many Requests` with `X-RateLimit-*` and `Retry-After` headers.

---

# API Reference

Base URL: `/api/v1`

## Public Endpoints

### Health Check

```
GET /api/v1/health
```

Returns `OK` with status `200`. No authentication required.

---

### Get User Reversal Status

```
GET /api/v1/users/{steamId}
```

Look up whether a Steam user has any active (non-expunged) trade reversals on record. No authentication required.

**Rate limit:** 100 requests/minute per IP.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `steamId` | string | Steam ID 64 of the user |

**Response:**

```json
{
  "steam_id": "76561198012345678",
  "has_reversed": true,
  "last_reversal_timestamp": 1711234567890
}
```

| Field | Type | Description |
|---|---|---|
| `steam_id` | string | The queried Steam ID 64 |
| `has_reversed` | boolean | `true` if the user has any active (non-expunged) reversals |
| `last_reversal_timestamp` | number \| null | Unix timestamp in milliseconds of the most recent reversal, present only when `has_reversed` is `true` |

---

## Marketplace Endpoints

All marketplace endpoints require `Authorization: Bearer <api_key>`.

### Create Reversals

```
POST /api/v1/reversals
```

Report one or more trade reversals in bulk. The reversal is automatically associated with the marketplace that owns the authenticating key.

**Required permission:** `write`
**Rate limit:** 2,000 requests/hour per API key.

**Request body:**

```json
{
  "data": [
    {
      "steam_id": "76561198012345678",
      "source": "direct",
      "reversed_at": 1711234567890
    },
    {
      "steam_id": "76561198087654321",
      "source": "related_user",
      "related_steam_id": "76561198012345678",
      "reversed_at": 1711234567890
    }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `steam_id` | string | Yes | Steam ID 64 of the user who reversed |
| `source` | string | No | How the reversal was identified: `direct`, `related_user`, or `user_report` |
| `related_steam_id` | string | Conditional | Required when `source` is `related_user`. The Steam ID of the related user |
| `reversed_at` | number | No | Unix timestamp in milliseconds when the reversal occurred. Defaults to current time. Cannot be in the future |

**Response:**

```json
{
  "data": [
    {
      "id": "123456789",
      "created_at": 1711234567890,
      "updated_at": 1711234567890,
      "steam_id": "76561198012345678",
      "marketplace_slug": "example-marketplace",
      "source": "direct",
      "reversed_at": 1711234567890
    }
  ]
}
```

---

### Expunge Reversal

```
DELETE /api/v1/reversals/{id}
```

Expunge (soft-delete) a reversal that was previously reported by the authenticating marketplace. Expunged reversals no longer count toward a user's active reversal status.

**Required permission:** `delete`
**Rate limit:** 2,000 requests/hour per API key.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Snowflake ID of the reversal to expunge |

**Response:** `200 OK` with empty body on success.

**Errors:**
- Cannot expunge a reversal belonging to another marketplace.
- Cannot expunge a reversal that has already been expunged.

---

### List Reversals

```
GET /api/v1/reversals
```

List reversal records with optional filtering and cursor-based pagination.

**Required permission:** `export`
**Rate limit:** 300 requests/minute per API key.

**Query parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `steam_id` | string | — | Filter by Steam ID 64 |
| `marketplace_slug` | string | — | Filter by marketplace slug |
| `limit` | number | `5000` | Number of records to return (max `10000`) |
| `cursor` | string | — | Pagination cursor from a previous response |

**Response:**

```json
{
  "data": [
    {
      "id": "123456789",
      "created_at": 1711234567890,
      "updated_at": 1711234567890,
      "steam_id": "76561198012345678",
      "marketplace_slug": "example-marketplace",
      "source": "direct",
      "reversed_at": 1711234567890,
      "expunged_at": null
    }
  ],
  "metadata": {
    "count": 1,
    "next_cursor": "eyJpZCI6MTIzfQ=="
  }
}
```

| Field | Type | Description |
|---|---|---|
| `metadata.count` | number | Number of records in the current page |
| `metadata.next_cursor` | string \| null | Cursor to pass as `cursor` query parameter for the next page. Absent on the last page |

---

### Export Reversals (CSV)

```
GET /api/v1/reversals/export
```

Export reversal records as CSV. Supports the same query parameters as List Reversals with higher limits.

**Required permission:** `export`
**Rate limit:** 60 requests/minute per API key.

**Query parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `steam_id` | string | — | Filter by Steam ID 64 |
| `marketplace_slug` | string | — | Filter by marketplace slug |
| `limit` | number | `10000` | Number of records to return (max `50000`) |
| `cursor` | string | — | Pagination cursor from a previous response |

**Response:** `text/csv` with columns: `id`, `created_at`, `updated_at`, `steam_id`, `marketplace_slug`, `source`, `related_steam_id`, `reversed_at`, `expunged_at`.

The `X-Next-Cursor` response header contains the pagination cursor for the next page, if more results are available.

---

### List Marketplace Keys

```
GET /api/v1/marketplace/keys
```

List all API keys belonging to the authenticating marketplace.

**Required permission:** `manage`
**Rate limit:** 100 requests/minute per API key.

**Response:**

```json
{
  "data": [
    {
      "id": "a1b2c3...",
      "created_at": 1711234567890,
      "updated_at": 1711234567890,
      "environment": "production",
      "marketplace_slug": "example-marketplace",
      "permissions": ["write", "delete", "export"]
    }
  ]
}
```

---

### Create Marketplace Key

```
POST /api/v1/marketplace/keys
```

Create a new API key for the authenticating marketplace.

**Required permission:** `manage`
**Rate limit:** 100 requests/minute per API key.

**Request body:**

```json
{
  "permissions": ["write", "delete"]
}
```

**Response:**

```json
{
  "id": "a1b2c3...",
  "environment": "production",
  "secret_key": "reversewatch_live_xxxxxxxx...",
  "marketplace_slug": "example-marketplace",
  "permissions": ["write", "delete"]
}
```

> **Important:** The `secret_key` is only returned once at creation time. Store it securely.

---

### Delete Marketplace Key

```
DELETE /api/v1/marketplace/keys/{id}
```

Delete an API key belonging to the authenticating marketplace.

**Required permission:** `manage`
**Rate limit:** 100 requests/minute per API key.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Hash ID of the key to delete |

**Response:** `200 OK` with empty body on success.

**Errors:**
- Cannot delete a key belonging to another marketplace.
- Cannot delete a key from a different environment.
- Cannot delete the key currently being used for authentication.

---

## Admin Endpoints

All admin endpoints require `Authorization: Bearer <api_key>` with the `admin` permission. Admin keys can only be created for the `csfloat` marketplace.

**Rate limit:** 2,000 requests/hour per API key (applies to all admin endpoints unless otherwise noted).

### Onboard Marketplace

```
POST /api/v1/admin/marketplace
```

Register a new marketplace and generate its initial `manage`-scoped API key.

**Request body:**

```json
{
  "marketplace_slug": "new-marketplace",
  "name": "New Marketplace"
}
```

| Field | Type | Description |
|---|---|---|
| `marketplace_slug` | string | Unique slug (1-25 chars, alphanumeric and hyphens only) |
| `name` | string | Display name (1-50 chars) |

**Response:**

```json
{
  "marketplace": {
    "slug": "new-marketplace",
    "created_at": 1711234567890,
    "updated_at": 1711234567890,
    "name": "New Marketplace",
    "is_active": true
  },
  "key": {
    "id": "a1b2c3...",
    "environment": "production",
    "secret_key": "reversewatch_live_xxxxxxxx...",
    "marketplace_slug": "new-marketplace",
    "permissions": ["manage"]
  }
}
```

---

### Update Marketplace

```
PATCH /api/v1/admin/marketplace/{slug}
```

Update a marketplace's name or active status.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `slug` | string | Marketplace slug |

**Request body:**

```json
{
  "name": "Updated Name",
  "is_active": false
}
```

All fields are optional; at least one must be provided.

**Response:** The updated marketplace object.

---

### Delete Marketplace

```
DELETE /api/v1/admin/marketplace/{slug}
```

Soft-delete a marketplace. The `csfloat` marketplace cannot be deleted.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `slug` | string | Marketplace slug |

**Response:** `200 OK` with empty body on success.

---

### Create Key (Admin)

```
POST /api/v1/admin/keys
```

Create an API key for any marketplace.

**Request body:**

```json
{
  "marketplace_slug": "example-marketplace",
  "permissions": ["write", "export"]
}
```

**Response:** A `RawKey` object including the `secret_key` (returned only once).

---

### Delete Key (Admin)

```
DELETE /api/v1/admin/keys/{id}
```

Delete any API key by its hash ID. Cannot delete the key currently being used for authentication.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Hash ID of the key |

**Response:** `200 OK` with empty body on success.

---

### Modify Reversal (Admin)

```
PATCH /api/v1/admin/reversals/{id}
```

Update fields on any reversal record.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Snowflake ID of the reversal |

**Request body:**

```json
{
  "source": "user_report",
  "related_steam_id": null,
  "reversed_at": 1711234567890,
  "expunged_at": 1711234567890
}
```

All fields are optional; at least one must be provided.

**Response:** The updated reversal object.

---

### Delete Reversal (Admin)

```
DELETE /api/v1/admin/reversals/{id}
```

Hard-delete a reversal record.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Snowflake ID of the reversal |

**Response:** `200 OK` with empty body on success.

---

### Delete All User Reports (Admin)

```
DELETE /api/v1/admin/users/{steamId}
```

Delete all reversal records for a given Steam user.

**Path parameters:**

| Parameter | Type | Description |
|---|---|---|
| `steamId` | string | Steam ID 64 of the user |

**Response:** `200 OK` with empty body on success.

---

## Error Responses

All error responses follow a consistent JSON format:

```json
{
  "code": 1,
  "message": "malformed request",
  "details": "invalid steam id"
}
```

| Code | HTTP Status | Message |
|---|---|---|
| 1 | 400 | Malformed request |
| 2 | 403 | Forbidden |
| 3 | 404 | Resource not found |
| 4 | 500 | Internal server error |
| 5 | 400 | Unknown resource |
| 6 | 401 | Invalid permission |
| 7 | 401 | Invalid API key |
| 8 | 500 | Failed to create resource |
| 9 | 500 | Failed to read resource |
| 10 | 500 | Failed to update resource |
| 11 | 500 | Failed to delete resource |
| 12 | 400 | Failed to decode JSON |
| 14 | 429 | Rate limited |
| 15 | 409 | Resource already exists |
| 16 | 400 | Invalid resource reference |
