# Reverse Watch

Public dashboard for `reverse.watch` — the open trade-reversal database for Steam. Adds aggregate stats (KPI cards, daily reversal graph, recent reversals table) on top of the existing single-Steam-ID lookup, plus a full UI rewrite per Razvan's mockups.

## OKR link

None. This is extracurricular product work — it does not map to any Q2 2026 OKR. See [`GOALS.md`](../../GOALS.md) for the formal Q2 list. Flagging because the workspace convention assumes everything in `Projects/` ties to an OKR; this is the exception.

## Status

- **2026-05-22** — PRD and engineering handoff drafted in this folder. Repo not yet cloned, branch not yet cut.
- Existing service (`csfloat/reverse-watch`) is live at https://reverse.watch and exposes a single-user lookup at `/api/v1/users/{steamId}`.
- Razvan mockups in [`design/`](design): desktop default / clear / flagged + mobile clear. Mobile default + flagged still pending. Open items in [`HANDOFF.md`](HANDOFF.md) §10.

## Repo

[`csfloat/reverse-watch`](https://github.com/csfloat/reverse-watch). Public. Go 1.24+ chi/GORM/Postgres. Master is protected — branch off `master` and PR back.

## Folder map

- [`PRD.md`](PRD.md) — v1 product spec for the dashboard rewrite.
- [`HANDOFF.md`](HANDOFF.md) — engineering handoff: starter prompt, local setup, PR breakdown, open items. Travels with `PRD.md` into the cloned-repo Cursor workspace.
- `decisions/` — ADR-style entries as decisions get locked. Currently empty.

## Principles

- Keep it stupid simple. The repo today is one Go binary serving one self-contained `static/index.html` from chi. Don't introduce a frontend build step in v1.
- Do not change existing endpoints, models, or schemas. Only add.
- Public endpoints follow the existing precedent: IP-rate-limited, no auth, conservative response shapes (no full row dumps).
- Every CSFloat-internal claim in this folder must be either verified or flagged with "**TBD / unverified**" so we don't ship the PRD with speculation in it.

## Linear

TBD — Morten to file a parent issue and link it from `PRD.md` §13 + `HANDOFF.md` §11. CSFloat team prefix is `CSF-` per [Knowledge/Linear/](../../Knowledge/Linear).
