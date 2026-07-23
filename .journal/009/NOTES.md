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
