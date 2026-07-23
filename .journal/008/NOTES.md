---
id: 008
title: Generalize protected stable package publication
started: 2026-07-22
---

## 2026-07-22 18:27 — Kickoff
Goal for the session: Generalize the existing protected package publisher so registered projects can publish independently verified stable `vX.Y.Z` releases beyond the initial `incus-gh-runner v1.0.0` rehearsal, then deliver the focused change through a ready PR without merging or publishing.
Current state of the world: `incus-gh-runner v1.1.0` is released at commit `6a84c2aac068eabc331aa3f2e10003f77c37ccee`, and release-source validation passes, but packages workflow run `29971445189` fails closed because protected staging still confines Phase 5 to `incus-gh-runner v1.0.0`; trusted dispatch carries only `project` and `tag`.
Plan: Start an implementation worktree from the current fetched default branch, identify and remove one-release assumptions across policy, workflows, tests, and docs, preserve every privileged boundary, run the complete local gates, push a feature branch, open a ready PR, and wait for hosted CI.

## 2026-07-22 18:37 — Generalized contract and local proof
Created `feat/generalize-stable-publisher` from current `main` at `f05cfdb`. The protected publisher now consumes registered package identity and the package version derived by removing exactly one leading `v` from an exact validated `vX.Y.Z` tag. Source proof, staging, production, Debian, Ubuntu, and Fedora paths no longer pin `incus-gh-runner v1.0.0`; production confirmation is derived as `publish <project> <tag> to production`.

Added exact dispatch-payload validation so any field beyond `project` and `tag` fails before protected jobs, plus executable policy regressions for confirmation mismatch, staging bypass, R2 target changes, privileged dispatch controls, malformed/prerelease tags, unknown projects, and tag/package-version mismatch. Staging/production environments, separate credentials, serialized ordering, `_staging/` isolation, production-root mode, release digest and provenance verification, no-op verification, and deletion controls remain unchanged.

Validation: the repository aggregate gate passed with the documented local `PROTO_GO_VERSION=1.26.4` workaround; Actionlint, ShellCheck, policy/unit tests, and all Go tests passed. The real `incus-gh-runner v1.1.0` proof independently fetched five release assets, verified digests and attestations, rebuilt both architectures and formats, and clean-installed version `1.1.0` on Debian and Fedora. Next: review the complete diff, commit, push, open the ready PR, and wait for hosted CI.
