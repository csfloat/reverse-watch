# Reverse Watch — Session Log

Rolling log of work sessions on the public dashboard build. Newest at top. Each entry is self-contained — read the top entry and you should know where to start.

---

## 2026-05-27 (Wed, morning) — Session #5

**Branch:** `feature/public-dashboard-v1`.
**Theme:** Pre-review cleanup pass.

- Backend: dropped GORM tags from `domain/dto/stats.go`, collapsed the local `bucket` struct in `DailyCounts`, switched `allowedDays` to a slice + `slices.Contains`, collapsed duplicate `100` limit, trimmed verbose narration comments across `internal/devseed/*` and `server/server.go`. Full test suite green.
- Frontend (`static/index.html`): three real correctness fixes — `chartInstance.destroy()` before re-render (was leaking uPlot instances), `mouseleave` listener attached once at boot (was duplicating on every period change), `formatDate` now uses `timeZone: 'UTC'` (table dates now match chart). Dead-code sweep: orphan CSS custom properties, unused class rules, dead element IDs, unused JS variables. Comment hygiene: removed ~15 narration comments / section banners; kept the ~6 high-value "why" comments. 1772 → 1705 lines.
- Docs: appended a "Project Summary (for a reviewer's first pass)" section to `docs/dashboard/ENVIRONMENT.md` — what the branch adds, file map, mermaid request-flow, design decisions, open items, run commands.
- Pending list unchanged: PR #3 (PostHog), Lighthouse audit, Discord pings to Zach + Razvan.

---

## 2026-05-26 (Tue, late evening) — Session #4

**Branch:** `feature/public-dashboard-v1` (background tweak uncommitted).
**Theme:** Background atmosphere.

- Tried adding animated SVG background lines/arcs to match Razvan's mock. Didn't work — reverted via `git restore`.
- Checked prod: the "lines" are just a radial gradient artifact, not real geometry. Adopted the prod approach with a tightened `circle 600px at 50% 80px` so the glow stays around the hero and scrolls away with the page.
- Pending list unchanged: PR #3 (PostHog), Lighthouse audit, Discord pings to Zach + Razvan.

---

## 2026-05-26 (Tue, evening) — Session #3

**Duration:** Single sitting.
**On branch:** `feature/public-dashboard-v1` — now **11 commits ahead** of `master`.
**Tests:** `go test ./api/v1/stats/...` green; rest of suite unchanged from Session #2.
**Theme:** UI polish + denser data for the new chart range.

### What got done

**Synthetic seed (commit `9ef639b`)**

- New `internal/devseed/synthetic.go`. `GenerateSynthetic(now)` is deterministic (fixed RNG seed = 42), produces ~9,800 reversals over the last 180 UTC days, at least 1 per day. Daily counts follow a gentle sinusoid around ~50/day with ~5% spike days (2.5–5×) and ~10% quiet days (0.2–0.5×). Marketplace mix 80% csfloat / 10% tradeit / 5% skinport / 5% swap.gg. Sources: 90% direct, 5% related_user (with valid related_steam_id), 5% user_report. ~1.5% rows expunged.
- Snowflake IDs are constructed from each row's `created_at` (mirroring `domain/models/snowflake.go` bit layout), so they don't collide with the real CSV seed or with each other.
- `internal/devseed/sheet.go` `InsertReversals` now chunks at 1,000 rows per round trip (Postgres 65,535-parameter limit on a single statement).
- `cmd/seed` gains `-synthetic`. Both modes idempotent via `ON CONFLICT (id) DO NOTHING`; they can coexist (different ID ranges).
- Verified: `go run ./cmd/seed -synthetic` inserts 9,800. KPIs jumped from 100 → 9,900 traders_indexed.

**Backend period allow-list (commit `6b8d708`)**

- Extended `allowedDays` in `api/v1/stats/stats.go` from `{30, 60, 90}` to `{7, 30, 60, 90, 180, 365}`. Error string + tests updated. New positive `TestDailyHandler_AcceptedDays` walks every accepted value and asserts the response length equals `days`.
- Restarted dev server (PID 558 was the stale parent; PID 587 was the actual listener — both killed before relaunching).

**Chart annotation chips (commit pending)**

- Added editorial event timeline chips that float above the chart line at the date of the event — Pricempire-style. Data source: `static/cs2-events.json` (flat JSON, `{date, title, description?, url?}`). Edit the file and refresh — no rebuild.
- Frontend logic in `static/index.html`:
  - `loadCS2Events()` fetches the JSON once at boot with `cache: 'no-store'`.
  - `renderEventChips(chart)` filters events to the chart's visible x-range, computes each chip's pixel position via `chart.valToPos(ts, 'x')`, and row-stacks colliding chips (up to 3 rows; oldest overflow dropped with a console warning).
  - Re-renders on every chart redraw (period change) and on window resize.
- Hover popover shows full date, title, description, and optional link.
- Documented in PRD §6.3 (chart section).

**Dashboard UI (commit `cfac9e8`)**

- Renamed first KPI label "Traders Indexed" → "Steam IDs Searched". JSON contract unchanged (`traders_indexed` stays on the wire); the JS-side comment notes the mapping.
- Added a segmented period picker next to the chart title: `7d / 30d / 3m / 6m / 1y`. Default `30d`. Active state styled with the accent color. Mobile reflow drops `margin-left: auto` so the picker wraps cleanly under the title.
- Removed the static "· Last 30 days" subtitle and rephrased the chart subtitle.
- `loadDaily(days)` now race-safe via a `dailyFetchSeq` counter — rapid picker clicks always settle on the latest selection.
- `renderChart(daily, days)` switches the x-axis label format to month-only when `days > 60`, so 6m and 1y don't overlap.
- PRD updates: §3.1 (label + picker), §3.5 (synthetic seed now in scope — reverses the earlier "no synthetic data" stance), §6.2 (UI-only rename note), §6.3 (new `days` allow-list).

### Decisions logged

- **Synthetic data is dev-only.** `cmd/seed` still refuses to run unless `Environment=development`. PRD §3.5 explicitly carves out production from this change.
- **`60d` stays in the allow-list** even though the picker doesn't expose it. Cost is trivial; keeps v1.1 free to add a "2 months" tile without another PR.
- **The KPI rename is UI-only.** Underlying field stays `traders_indexed` because "Steam IDs Searched" is, strictly, a rebrand of the same definition (distinct steam_ids present in the index). A future "actual public search counter" would be a different metric entirely and is a v1.x scope question, not v1.

### Where we resume — pick from one of these

| Option | Scope | Why |
|---|---|---|
| **A. PR #3 — PostHog analytics** | Wire `dashboard_viewed`, `lookup_submitted`, `lookup_result_shown`, `extension_cta_click` via `science.csfloat.io` proxy (PRD §9). | PRD §11 launch criterion. ~30–60 min. |
| **B. Lighthouse audit** | Run Lighthouse, fix what falls below 90 on Performance + Accessibility. | PRD §11 launch criterion. Open-ended. |
| **C. Draft Discord pings** | Zach (KPI defs confirmation) + Razvan (mobile mocks). | Unblocks the human-gated items so they progress in parallel. |

### Open items still outstanding

Same five items from Session #2, **plus one new**:

- **D-open-4 (new) — Steam display name source for the Trader column.** The original `static/index.html` was rendering the marketplace slug in the "Trader" column, which was incorrect — Razvan's mockup wants the Steam display name there. Local dashboard now renders a deterministic fake name derived from `steam_id` (djb2 hash → adjective + noun + suffix from ~13.5k combos). Real names need to come from somewhere: Steam's `GetPlayerSummaries` (rate-limited, needs cache layer), a `steam_users` table (cleanest but violates "no schema changes"), or punt to v1.1. PRD §6.4 + §14 D-open-4 + HANDOFF §3 all flagged. Needs Zach.

Synthetic data unblocks the chart visually but doesn't replace Zach's KPI sign-off, Razvan's mobile mocks, or this new question.

### Misc state for next session

- Local dev server is **running in the background** with the new binary (re-spawned during this session, PID at restart was 33315).
- DB now contains 9,800 synthetic rows on top of the 98 real CSV rows and the 2 manual `BulkCreate` rows from Session #1.
- Branch `feature/public-dashboard-v1` is 11 commits ahead of `master`. All local. **Do not push** until Morten explicitly says v1 is ready for Zach.

### Kick-off prompt for next session

> Continuing v1 of the Reverse Watch dashboard. Read `docs/dashboard/SESSION-LOG.md` top entry — that's where we stopped. Branch `feature/public-dashboard-v1` is 11 commits ahead of master, all local. KPI rename, period picker, and synthetic seed all in. Next codable items are PR #3 (PostHog analytics) and the Lighthouse pass — both PRD §11 launch criteria. Recommend starting with PR #3.

---

## 2026-05-24 → 2026-05-26 (Sun–Tue) — Session #2

**Duration:** Three sittings across three days.
**On branch:** `feature/public-dashboard-v1` — now **7 commits ahead** of `master`.
**Tests:** `go test ./...` green (full suite + new repo/handler tests).
**Workflow shift:** Per the updated `ABOUT-MORTEN.md`, Cursor now writes the code; Morten reviews. Local commits at each meaningful milestone, no push until v1 is fully complete (one PR to Zach, not piecemeal).

### What got done

**PR #1 — backend (commit `66f75c5`)**

All 8 build-order steps from Session #1 complete:

1. `domain/dto/stats.go` — `SummaryStats` and `DailyCount` DTOs.
2. Extended `domain/repository/public.go` `ReversalRepository` with `SummaryStats()`, `DailyCounts(days)`, `ListRecent(limit)`.
3. Implemented those three in `repository/public/reversal.go`. `SummaryStats` is a single SQL query with `FILTER (WHERE …)` for all three counts. `DailyCounts` zero-fills in Go after the aggregate.
4. Repo tests in `repository/public/reversal_test.go` covering happy paths, zero-fill, date boundaries, soft-delete exclusion, and `traders_indexed` including expunged.
5. New `api/v1/stats/{router.go,stats.go}` with `/summary` and `/reversals/daily` handlers and a 60s in-process `sync.Map`-based cache (per-`days` key for daily; single key for summary).
6. Restructured `api/v1/reversals/router.go` — wrapped existing auth-gated routes in `chi.Group`, added public `/recent` outside.
7. Mounted `/stats` in `api/v1/v1.go`.
8. Handler tests for all three endpoints (happy path, cache hit, invalid params, sorting/limit).

KPI definitions follow PRD §6.2 verbatim. `traders_flagged_24h` is `created_at`-bucketed.

**Dev seed (commit `311fe96`)**

- `internal/devseed/fixtures/reversals_seed.csv` — 98 real rows fetched via the Sheets MCP from `1ccGoHiqXTpjy_jtHSOW3QmrNFP2jvfOsqyWFpNBz-UA`. Two rows from the original 100 dropped due to steam_id precision loss in the source sheet (HANDOFF §10).
- `internal/devseed/sheet.go` — CSV → `[]*models.Reversal`, validates header order, uses `INSERT … ON CONFLICT (id) DO NOTHING` for idempotency.
- `cmd/seed/main.go` — CLI (`go run ./cmd/seed`). Refuses to run unless `Environment=development`.
- Verified: first run inserted 98, second run inserted 0. Stats endpoints reflect the seed (KPIs: 100 / 100 / 2 with Morten's two earlier test rows).

**PR #2 — frontend (commit `56a3fbc`)**

Full `static/index.html` rewrite per PRD §§6.1–6.7. Single self-contained file, inline CSS + JS, no build step:

- Hero with CSFloat logo (`static/csfloat-logo.png`), title, lede, restyled search (icon-only arrow button).
- Search-result chip (clear/flagged) below the input. Background tints follow the verdict via `body.clear-result` / `body.flagged` classes.
- Three KPI cards populated from `/stats/summary`.
- 30-day uPlot line chart (CDN, `uplot@1.6.32`) with custom hover tooltip showing day + count, position-clamped to chart bounds.
- Recently Reported Reversals table — paginated **client-side in 10-row chunks** via "Load More" (button auto-hides when all are shown). Server-side pagination still v1.1 per PRD §6.4.
- Footer with What is reverse.watch? / Want to contribute? columns + Powered-by-CSFloat lockup (using the same logo image).
- Hardcoded `MARKETPLACES` slug-map in JS (D9). Currently just `csfloat`; fallback uses the slug verbatim and a generic `storefront` icon.
- Mobile reflow built against the one mobile mockup we have. Razvan still owes default + flagged mobile mocks.

**Static-file serving workaround (commit `f3fd493`)**

`http.ServeFile` / `http.FileServer` were truncating every static response at exactly 512 bytes (first TCP segment) on Morten's local macOS. JSON endpoints were unaffected (different write path). Replaced both static handlers in `server/server.go` with small in-memory variants (`os.ReadFile` + `w.Write`). New `GET /static/*` route added — same handler. Path traversal rejected. Static payloads are tiny (HTML + logo + future icons), so the read-once cost is negligible. Documented in a code comment.

**Doc updates (commits `2f6bb7e`, `cdb2458`, `5c53975`)**

- `ABOUT-MORTEN.md`: added Git workflow rules (commit-after-bigger-updates, propose-then-commit, never push without explicit ask, never push to master); added Review workflow ("v1 ships as ONE PR to Zach"); added "private code I don't own" caution.
- `README.md`: dashboard intro, explicit DB + `postgres/postgres` superuser setup (the gotcha Morten hit Session #1), `go run ./cmd/seed` instructions, public API endpoint table, links to PRD + HANDOFF.

### Decisions logged (not yet captured anywhere else)

- **Chart library: uPlot** (HANDOFF D7 locked). ~40KB, zero deps, fast. Loaded via CDN.
- **"Load More" pagination is purely client-side reveal** — the API still returns 100 rows in one call. v1.1 will add server-side cursor pagination per PRD §6.4.
- **Search-result chip ignores the marketplace icon from Razvan's mockups** — the existing `/api/v1/users/{steamId}` doesn't return marketplace info, and PRD §6.5 says the lookup endpoint is unchanged. Punt to v1.1 if we want to enrich the response.
- **No favicon / OG image work yet.** Current SVG favicon is the existing one (blue eye).

### Where we resume — pick from one of these

| Option | Scope | Why |
|---|---|---|
| **A. PR #3 — PostHog analytics** | Wire `dashboard_viewed`, `lookup_submitted`, `lookup_result_shown`, `extension_cta_click` via `science.csfloat.io` proxy (PRD §9). | PRD §11 launch criterion. ~30–60 min. |
| **B. Lighthouse audit** | Run Lighthouse, fix what falls below 90 on Performance + Accessibility. | PRD §11 launch criterion. Open-ended. |
| **C. Draft Discord pings** | Zach (KPI defs confirmation) + Razvan (mobile mocks). | Unblocks the human-gated items so they progress in parallel. |

A or B is the more productive next step. C is quick prep work.

### Open items still outstanding

- **Zach — KPI defs sign-off (D-open-1)**, especially `traders_flagged_24h` (currently `created_at`-based). Discord/Slack ping not sent yet.
- **Razvan — mobile default + flagged mockups**. Mobile reflow is best-effort until they arrive.
- **Marketplace registry** (PRD §14.4) — confirm nezha doesn't already maintain a slug → name/icon map we should reuse instead of hardcoding.
- **Linear parent issue** CSF-1518 — still no sub-issues filed.
- **Razvan's mocks show a marketplace chip in the lookup result** — defer to v1.1 (would require API extension).

### Misc state for next session

- Local dev server is **still running in the background** (PID was 558 at last commit, on port 8080). Restart only needed if `server/server.go` is touched again.
- `config.json` unchanged.
- Branch `feature/public-dashboard-v1` is 7 commits ahead of `master` (`66f75c5`, `2f6bb7e`, `311fe96`, `cdb2458`, `f3fd493`, `56a3fbc`, `5c53975`). All local. **Do not push** until Morten explicitly says v1 is ready for Zach.
- All `docs/dashboard/*` files are tracked in git as of commit `66f75c5` (they rode along with the PR #1 backend commit).

### Todo state at session end

- [x] All 8 PR #1 backend steps
- [x] Dev seed (sheet ingest)
- [x] PR #2 frontend rewrite (desktop done; mobile reflow done as far as possible)
- [x] Static-file truncation workaround
- [x] README update (PRD §11 criterion)
- [ ] **A — PR #3 analytics (PostHog)** ← strongest candidate for next session
- [ ] B — Lighthouse pass
- [ ] C — Discord pings to Zach + Razvan
- [ ] (Blocked) Razvan delivers missing mobile mockups
- [ ] (Blocked) Zach confirms KPI definitions
- [ ] (Last) Push branch + open ONE PR to Zach

### Kick-off prompt for next session

> Continuing v1 of the Reverse Watch dashboard. Read `docs/dashboard/SESSION-LOG.md` top entry — that's where we stopped. Branch `feature/public-dashboard-v1` is 7 commits ahead of master, all local. PR #1 backend + dev seed + PR #2 frontend are done. Next codable items are PR #3 (PostHog analytics) and the Lighthouse pass — both PRD §11 launch criteria. Recommend starting with PR #3.

---

## 2026-05-23 (Sat eve) — Session #1

**Duration:** ~15 min of dialogue, no application code written yet (intentional).
**On branch:** `feature/public-dashboard-v1` (already existed at session start).
**Tests:** `go test ./...` green (~13s, 14 packages with tests).

### What got done

- Cursor agent read end-to-end: `HANDOFF.md`, `PRD.md`, `README.md`, plus the repo files HANDOFF §9 listed (api/v1/users, api/v1/reversals, repository/public/reversal.go, domain/repository/public.go, server/server.go, main.go, factory, testutil, render, errors, ratelimit, middleware, dto/reversal, models/reversal, models/steamid, the existing reversals_test.go, etc.).
- No `AGENTS.md` and no `.cursor/rules/` in the repo — nothing extra to obey beyond what HANDOFF/PRD say.
- Local Postgres setup verified working: created a `postgres` superuser with password `postgres` (via `ALTER USER postgres WITH SUPERUSER PASSWORD 'postgres';`) so `pgtestdb` in `internal/testutil/db.go` can spin up temporary test DBs.
- `go test ./...` confirmed green end-to-end. Baseline before any edits.

### Conflicts surfaced vs HANDOFF (none silently overridden — Morten to decide)

| # | Issue | Recommendation | Status |
|---|---|---|---|
| C1 | `api/v1/reversals/router.go` opens with `r.Use(middleware.AuthMiddleware)`; chi panics if `Use()` is called after routes are registered, so we can't just add a peer public `/recent` route. | Wrap the existing auth-gated routes in `r.Group(func(r chi.Router) { r.Use(AuthMiddleware); … })` and add `/recent` outside the group. Smallest diff. | Pending Morten ack |
| C2 | KPI definitions (HANDOFF D-open-1) still unconfirmed by Stepan. PRD §6.2 proposes `created_at`-bucketed 24h KPI; both flagged counts filter `expunged_at IS NULL`; `traders_indexed` includes expunged. | Implement to PRD §6.2 spec; one-line repo tweak if Stepan corrects. **Fire HANDOFF §10 Discord message to Stepan ASAP.** | Pending Morten outreach |
| C3 | No DB indexes on `reversed_at` or `created_at`. `ListRecent` does full-table sort; `DailyCounts` scans the window. | PRD §1.4 + HANDOFF §12 explicitly defer this. Ship without; benchmark + add `CREATE INDEX CONCURRENTLY` post-launch. | Accepted, defer to v1.1 |
| C4 | `/api/v1/stats/reversals/daily` semantically adjacent to `/api/v1/reversals/*` (different chi mounts, no real collision). | Accept; add a one-line comment in `api/v1/v1.go` when mounting `/stats`. | Accepted |
| C5 | `models.SteamID.MarshalJSON` already encodes as string — PRD §6.4 response shape is satisfied automatically. **Not a conflict, just confirmed.** | n/a | ✅ |

### Decisions logged in dialogue (not yet codified anywhere)

- Build order is **bottom-up**: DTOs → interface → repo impl → repo tests → handlers → router → mount → handler tests. (8 steps; see todos below.)
- Daily-bucket zero-fill will be done in Go after the SQL aggregate returns, not via Postgres `generate_series`. Simpler, portable, ~30 days of map lookups is free.
- `dto.DailyCount.Date` typed as `string` in `"2006-01-02"` UTC, not `time.Time`. Avoids custom MarshalJSON.
- Counts typed as `uint64` to match the existing `ReversedAt`/`CreatedAt` convention.
- `[]DailyCount` (values), not `[]*DailyCount`. `SummaryStats` returned by pointer from the repo (`*dto.SummaryStats`) to match the existing `reversalRepository.Read` signature style.
- In-process 60s cache will live in the `api/v1/stats` package (not in `ratelimit/`). Per-`days` key for the daily endpoint; single key for summary.
- `/reversals/recent` will NOT be cached (PRD §7.2).

### Where we resume — Step 1 of 8

**Next task:** Create `domain/dto/stats.go` with two structs:

1. `SummaryStats` — three `uint64` fields with JSON tags `traders_indexed`, `traders_flagged`, `traders_flagged_24h`. Matches PRD §6.6.
2. `DailyCount` — `Date string` (JSON `date`) + `Count uint64` (JSON `count`). Matches PRD §6.3.

Acceptance: `go build ./...` clean. No tests yet — pure types.

Morten writes the file himself. Agent gave a skeleton with comments-as-prompts in the previous message; **drop those comments** in the real file (code shouldn't narrate itself).

### Open items still outstanding (parallel — don't block Step 1 on these)

- HANDOFF §10 Discord pings to **Stepan** (`step7750`) — KPI defs, recent-table Steam ID masking, PostHog proxy origin.
- HANDOFF §10 Discord ping to **Razvan** (`razvanbadea`) — missing mobile default + flagged mockups (only blocks PR #2, not PR #1).
- HANDOFF §10 Discord ping to **Ceegan** (`_perplex`) — sanity check on PRD.
- Linear parent issue: not filed yet.

### Misc state for tomorrow

- `config.json` is set up: port 8080, user `byskov`, password `devpassword`, DBs `reverse_watch_private` / `reverse_watch_public`. `go run main.go` boots cleanly.
- `docs/dashboard/` files (HANDOFF, PRD, design PNGs, this log) are still untracked in git — undecided whether they ride along on PR #1 or get their own commit. Default: probably their own commit, since PR #1 is backend-only.
- Existing `feature/public-dashboard-v1` branch — was already there before today's session; no commits on it yet.
- Two terminals in play locally: one running `go run main.go` (kept alive), one for shell commands. Don't paste shell commands into the dev-server terminal.

### Todo state at session end

- [x] Local test creds + `go test ./...` green
- [x] On feature branch
- [ ] **Step 1** — `domain/dto/stats.go` (IN PROGRESS — Morten to write)
- [ ] Step 2 — extend `domain/repository/public.go` `ReversalRepository` interface (`SummaryStats`, `DailyCounts`, `ListRecent`)
- [ ] Step 3 — implement the three methods in `repository/public/reversal.go`
- [ ] Step 4 — repo tests in `repository/public/reversal_test.go`
- [ ] Step 5 — `api/v1/stats/{router.go,stats.go}` with summary + daily handlers + 60s in-process cache
- [ ] Step 6 — restructure `api/v1/reversals/router.go` (Group existing auth routes) + add public `/recent` handler in `reversals.go`
- [ ] Step 7 — mount `stats` in `api/v1/v1.go`
- [ ] Step 8 — handler tests for all three endpoints

### Kick-off prompt for next session

> Continuing PR #1 on Reverse Watch dashboard. Read `docs/dashboard/SESSION_LOG.md` top entry — that's where we stopped. We're on Step 1 of 8: I need to write `domain/dto/stats.go`. Walk me through it again briefly, then I'll write it.

---

*End of log. Add new sessions above this line.*
