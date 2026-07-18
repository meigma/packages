---
id: 006
title: Phase 5 consumer integration
started: 2026-07-18
---

## 2026-07-18 14:53 — Kickoff
Goal for the session: Familiarize with session 001's planning artifacts and begin from Phase 5.
Current state of the world: Phases 0–3 are merged on `main`; the journal records Phase 4 staging provisioning and rehearsal as the immediate predecessor to Phase 5.
Plan: Review session 001's planning artifacts, reconcile the Phase 5 scope and proof gate with the current repository state, and identify the smallest useful first slice before implementation.

## 2026-07-18 14:54 — Phase 5 orientation
Read session 001's `NOTES.md`, `SUMMARY.md`, and complete `DESIGN_PROPOSAL.md`. Phase 5 is the production-and-first-consumer slice: publish `incus-gh-runner`, add post-publish and daily clean-client smoke tests, land the consumer's short-lived GitHub App dispatch, validate onboarding against that real integration, and rehearse documented recovery. Its proof gate is canonical-snippet installation plus reliable queued publication from the consumer release workflow.

The current implementation boundary is concrete. `main` contains Phases 0–3. The Phase 4 branch adds ordered R2 staging mutation and protected `_staging/` rehearsal, while intentionally retaining fixture inputs and excluding GitHub Release discovery, production selection, smoke automation, production registry entries, recovery rehearsal, and consumer dispatch. Phase 5 should therefore begin with the smallest real-consumer/source-discovery slice after the Phase 4 prerequisite lands, using the real `incus-gh-runner` release shape to refine the illustrative registry contract before expanding production workflow behavior.
