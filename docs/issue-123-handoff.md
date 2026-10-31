# Issue 123 — implementation handoff

Integration branch: `integration/issue-123-transcription-centralisee`.
Spec: https://github.com/thomas-bsn/snitch/issues/123. Decisions: `docs/adr/0020-configuration-de-transcription-centralisee.md`; terminology: `GLOSSARY.md`.

## Ticket graph

- #143: completed on integration at `97d45a9`.
- #144 and #145: checkpointed, incomplete, separate clean worktrees and remote branches.
- #146, #147 and #148: blocked by #145.

## State

- Product decisions and test seams confirmed by user; spec and six tickets published.
- No real library scan or model benchmark authorized. Use synthetic fixtures only.
- Documentation decision commit: `2b87751`.
- Exploration notes for later tickets: `/tmp/snitch-123-exploration.md` (outside the repo).
- #144 includes the fresh-host, one-command installation promised by #89. The current UI only provides compose/env downloads and `deploy/worker/install.py` assumes a cloned repository; filling the current bundle alone does not satisfy #144.
- Draft PR: https://github.com/thomas-bsn/snitch/pull/149.
- #143 verification: Go unit tests, web lint/build, go vet and error-code check passed per implementer. PostgreSQL integration tests need Docker; rootless Docker was unavailable.
- Next: resume and finish #144/#145 on their own branches. Do not merge or close either yet. After review and merge, start #146–#148 from the new integration tip.
- Weekly usage was reported at 11% remaining on 2026-10-31; root stopped scheduling new tickets and saved checkpoints.

## Resume instructions

Read this file, the spec, each ticket and its blockers, `ROADMAP.md`, relevant phase specs, ADR 0020 and `docs/agents/issue-tracker.md`. Keep each ticket in a dedicated branch/worktree. Record its commit, tests and remaining risks below after merge. Do not start phase 7 or scan real media.

## Completed tickets

- #143: `e292ca2` (feature), merged via `97d45a9`. Central transcription settings can be saved before the first Worker with explicit unconfigured/enabled/disabled mode, version, endpoint, model and language; Admin → Transcription supports it. #144 must key off mode rather than the legacy default model/language.

## Incomplete ticket checkpoints

- #144: `ticket/144-worker-onboarding`, worktree `/tmp/snitch-ticket-144`, work commit `3711491`, integration merge `92f75e5`. Branch pushed to origin. Central registration, read-only form, central bundle values and a secure short-lived installation invitation/command are committed. Go unit/vet, web lint/build and diff check passed. Still needed: update existing Playwright admin-workers journey, run PostgreSQL invitation/registration tests with Docker, verify fresh-host installer on Linux/Docker, handle HTTP-origin command UI, review script security. The worktree is clean. Do not claim complete based on static checks.
- #145: `ticket/145-worker-config`, worktree `/tmp/snitch-ticket-145`, work commit `a53509a`, integration merge `a2a1e66`. Branch pushed to origin. Version states, test API, worker-side model/connection test, initial claim gate and Transcription UI are committed. TypeScript/Vite build, Go unit/vet, focused Python tests, error-code check and diff check passed. Still needed: PostgreSQL `TestFoundation` with Docker, Python sync/ack/idempotency test, Playwright status/Test action, Python OpenAPI client regeneration, security/concurrency review, offline/stale/disabled handling. The worktree is clean. `/tmp/snitch-ticket-145-checkpoint.md` has detailed notes. Current claim gate requires active==desired; #147 must implement the approved fallback to the previous active validated version.
- #144 and #145 both touch `server.go` and OpenAPI. #145 uses migration `00049_worker_transcription_config.sql`; #144 uses `00050_worker_install_invites.sql`. Resolve overlaps when merging; neither branch is ready to merge now.
