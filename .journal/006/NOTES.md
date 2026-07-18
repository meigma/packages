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

## 2026-07-18 15:06 — Pre-release work identified
Verified live state after Phase 4 closed: PR #9 is merged at `f2e361e`, staging publication and empty-prefix recovery passed, and production discovery/publication remains Phase 5. `incus-gh-runner` has no published release yet; PR #20 is completing release readiness and its exact-head dry run passed.

The critical pre-release gap is now concrete: the consumer release pipeline stages raw Linux/macOS binaries, SBOMs, `checksums.txt`, an APK-backed container, and a reference VM image, but it produces no `.deb` or `.rpm` release assets. The package repository contract requires checksummed DEB/RPM assets for amd64 and arm64. We do not need to wait: first add and rehearse those packages in `incus-gh-runner`, then implement release discovery and registry validation here against the resulting frozen naming/checksum contract. Production workflow/smoke/docs/dispatch scaffolding can proceed against fixtures, while the actual production publish must wait for a published qualifying release.
