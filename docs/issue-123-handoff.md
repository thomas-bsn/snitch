# Issue 123 — implementation handoff

Integration branch: `integration/issue-123-transcription-centralisee`.
Spec: https://github.com/thomas-bsn/snitch/issues/123. Decisions: `docs/adr/0020-configuration-de-transcription-centralisee.md`; terminology: `GLOSSARY.md`.

## Ticket graph

- #143: completed on integration at `97d45a9`.
- #144 and #145: in progress in separate worktrees.
- #146, #147 and #148: blocked by #145.

## State

- Product decisions and test seams confirmed by user; spec and six tickets published.
- No real library scan or model benchmark authorized. Use synthetic fixtures only.
- Documentation decision commit: `2b87751`.
- Exploration notes for later tickets: `/tmp/snitch-123-exploration.md` (outside the repo).
- #144 includes the fresh-host, one-command installation promised by #89. The current UI only provides compose/env downloads and `deploy/worker/install.py` assumes a cloned repository; filling the current bundle alone does not satisfy #144.
- Draft PR: https://github.com/thomas-bsn/snitch/pull/149.
- #143 verification: Go unit tests, web lint/build, go vet and error-code check passed per implementer. PostgreSQL integration tests need Docker; rootless Docker was unavailable.
- Next: merge #144 and #145 after review, then start #146–#148 from the new integration tip.

## Resume instructions

Read this file, the spec, each ticket and its blockers, `ROADMAP.md`, relevant phase specs, ADR 0020 and `docs/agents/issue-tracker.md`. Keep each ticket in a dedicated branch/worktree. Record its commit, tests and remaining risks below after merge. Do not start phase 7 or scan real media.

## Completed tickets

- #143: `e292ca2` (feature), merged via `97d45a9`. Central transcription settings can be saved before the first Worker with explicit unconfigured/enabled/disabled mode, version, endpoint, model and language; Admin → Transcription supports it. #144 must key off mode rather than the legacy default model/language.
