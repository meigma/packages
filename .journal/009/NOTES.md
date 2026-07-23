---
id: 009
title: New session
started: 2026-07-22
---

## 2026-07-22 18:56 — Kickoff
Goal for the session: not yet stated; the user opened a new session and has not
made their first request.
Current state of the world: `main` at `cdc6690` with the generalized protected
stable publication path proven end to end (session 008 shipped
`incus-gh-runner v1.1.0` through validation, staging, and production via PR
#13). Open threads from 008: automate or replace long-lived R2 credentials
(renew before 2027-07-18), onboard more projects via `projects.yml`, and
possible multi-release retention work.
Plan: await the user's first request, then update the title and goal here.

## 2026-07-22 18:58 — Goal stated
Goal: audit the repo for over-engineered tests/scripts/gates (especially bash)
left by the previous agent. The repo's only real job is publishing deb/rpm
packages to R2. Produce an assessment of what is useless ceremony vs. what
genuinely protects publication; user wants to strip the rest. Context: they
just fixed a hardcoded-values issue that was blocking releases (PR #13 era).
Assessment first; no changes yet.

## 2026-07-22 19:10 — Exploration complete
Surveyed scripts/ (1,495 lines bash+python), moon.yml tasks, all three
workflows, spikes/phase0, testdata, and Go internals (~4.7k lines). Key
findings: (1) check_workflow_policy.py asserts exact string fragments and
counts of publish.yml/phase5-publish.sh — self-referential, breaks on any
edit; (2) phase1-proof and phase2-proof run docker fixture builds + container
installs on every CI run (confirmed in run 29972545107 logs, ~30s+ each);
(3) "disposable" spikes/phase0 Dockerfile.tools is load-bearing in the real
publish path; (4) phase5-source.sh hardcodes 5-asset/4-package shape — same
hardcoding class that blocked releases; (5) phase4-staging.sh and rebuild.yml
superseded by the phase5 path. Real value: Go CLI + unit tests, protected
environments, validate-request, apply-sync verify/no-op, signing, clean-install
smoke. Assessment delivered; awaiting go-ahead before deleting anything.

## 2026-07-22 19:35 — PR 14 merged, PR 15 open
PR #14 (chore: remove phase proofs and workflow policy checker) merged as
211d286: deleted check_workflow_policy, phase1/2/4 scripts, testdata,
rebuild.yml, spike extras; phase3-ci.sh became lint-workflows.sh (actionlint +
shellcheck only). CI dropped from ~65s to ~32s. PR #15 opened: renames
phase5-publish.sh to publish.sh (env-driven, no hardcoded URLs/prefixes),
deletes phase5-source.sh (had hardcoded 5-asset/4-package asserts),
validate_publish_event.py, spikes/ entirely (Dockerfile.tools →
docker/tools.Dockerfile, TOOLS_UID arg, shellcheck removed); publish.yml
renamed "Publish", inputs cut to project/tag/apply_staging/apply_production,
confirmation phrases and empty_staging dropped. Decision: neither environment
has required reviewers (only branch policies), so the apply_production
checkbox (default false) remains the manual speed bump; trusted dispatch
unchanged. Plan after merge: staging-only dispatch of incus-gh-runner v1.1.0
from main to prove the reworked path. Also noticed stale feat/phase4-staging
branch allowed in staging environment deployment policy — candidate cleanup.
