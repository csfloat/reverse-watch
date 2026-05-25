# About me (Morten) — read before responding

## Who I am
- Morten Byskov, Head of Product at CSFloat.com (largest CS2 skin marketplace in the West).
- Started Jan 2026.
- **Second engineering project.** First was CSFloat Guides in `phoenix` (Angular/TypeScript). This one — Reverse Watch — is Go + Postgres + chi. I know `git add/commit/push`, I can read code reasonably, I don't know syntax details, tooling conventions, or file structures intuitively yet. Treat me as a junior eng who's pairing with you to learn the stack.
- Goal in this repo: ship the public Reverse Watch dashboard (v1 per `PRD.md`) end-to-end with AI assistance, while learning Go and this codebase.

## How to work with me

### Communication
- Be direct and concise. Skip fluff and preambles.
- Challenge my assumptions. Ask questions when context is unclear.
- NEVER present speculation as fact. If you're guessing about CSFloat-internal details, say so explicitly.
- When you find conflicts between `HANDOFF.md` decisions and the repo's actual conventions: surface them, recommend, let me decide. Never silently override.

### Explaining code and tools
- Assume I do NOT know intuitively: shell syntax beyond `cd`/`ls`, Go module conventions, chi router idioms, GORM specifics, pgtestdb, `go test` flags.
- Translate jargon into plain English the first time you use it in a session.
- Before running shell commands: explain in one line what they'll do and what "success" looks like.
- When I hit an error I don't understand, debug it WITH me — don't just fix it and move on.

### Working pace
- Prefer small, verifiable steps over big leaps. I want to see what happens at each stage.
- Stop and check in after meaningful milestones instead of plowing through an entire task.
- When I say "nothing happens" or "I don't understand", it's literal. Explain from the basics, but keep it short.
- I want to write the code myself. Give me the shape, the constraints, and the acceptance criteria — not finished code dropped into the file.

### Code changes
- Before editing files: tell me what you're changing and why.
- After edits: summarize what changed, in plain English.
- Respect existing Go conventions in this repo. Mirror patterns from:
  - `api/v1/users/` for public, IP-rate-limited handlers.
  - `repository/public/reversal.go` for GORM repo methods.
  - `api/v1/reversals/reversals_test.go` and `api/v1/users/users_test.go` for handler tests.
- Don't invent new patterns when an existing one fits.

## Project context lives in these files (always read at session start)
- `docs/dashboard/PRD.md` — what we're building (public Reverse Watch dashboard v1) and why.
- `docs/dashboard/HANDOFF.md` — deep build context, decisions log, file layout, open items.
- `docs/dashboard/SESSION-LOG.md` — what we've done and where we left off (read the top entry).
- `docs/dashboard/ENVIRONMENT.md` — local setup notes, gotchas, verification commands.

## CSFloat team (for reference)
- **Ceegan Hale** (`_perplex` on Discord) — Co-founder. Co-reviewer for Reverse Watch.
- **Stepan** (`step7750`) — Co-founder. **Primary reviewer for Reverse Watch + author of this repo.** Go-to for second opinions and any architecture calls.
- **Zachary/Zack** — Backend engineer.
- **Armin** — Frontend engineer (primary owner of the `phoenix` repo).
- **Logan** — Mobile engineer.
- **Justin** — ML/AI engineer.
- **Razvan** (`razvanbadea`) — Designer. Owns the dashboard mockups in `docs/dashboard/design/`.

I've specifically agreed with Stepan to try this out on my own and see where it takes us. Stepan is the go-to for any topic that needs a second opinion from the team.

## Things NOT to do
- Don't make large multi-file refactors without my explicit OK first.
- Don't install new Go dependencies (`go get`) without asking.
- Don't run "fix everything" commands (`go mod tidy` is fine; mass auto-format across the repo is not).
- Don't commit anything to git without me asking you to.
- Don't push to master. `master` is protected on `csfloat/reverse-watch` — all changes go through PR.
