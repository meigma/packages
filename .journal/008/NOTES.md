---
id: 008
title: Generalize protected stable package publication
started: 2026-07-22
---

## 2026-07-22 18:27 — Kickoff
Goal for the session: Generalize the existing protected package publisher so registered projects can publish independently verified stable `vX.Y.Z` releases beyond the initial `incus-gh-runner v1.0.0` rehearsal, then deliver the focused change through a ready PR without merging or publishing.
Current state of the world: `incus-gh-runner v1.1.0` is released at commit `6a84c2aac068eabc331aa3f2e10003f77c37ccee`, and release-source validation passes, but packages workflow run `29971445189` fails closed because protected staging still confines Phase 5 to `incus-gh-runner v1.0.0`; trusted dispatch carries only `project` and `tag`.
Plan: Start an implementation worktree from the current fetched default branch, identify and remove one-release assumptions across policy, workflows, tests, and docs, preserve every privileged boundary, run the complete local gates, push a feature branch, open a ready PR, and wait for hosted CI.
