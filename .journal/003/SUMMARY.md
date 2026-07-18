---
id: 003
title: Phase 2 deterministic rebuild and sync planning
date: 2026-07-18
status: complete
repos_touched: [meigma/packages]
related_sessions: [001, 002]
---

## Goal

Reconcile session 001's phased plan with the exact repository state, then
advance through the first incomplete implementation phase without pulling later
GitHub, R2, or production-secret concerns forward.

## Outcome

The goal was met. Orientation confirmed that Phase 1 had already merged in
session 002, so this session implemented and proved Phase 2. [PR #7](https://github.com/meigma/packages/pull/7)
passed hosted CI and Kusari Inspector, squash-merged to `main` as
`d3b67865c258ce81e3d4e7f40a5f000870e2581f`, and passed post-merge CI.

The repository now validates fixture release sets, retains stable semantic
versions deterministically, inspects and verifies package assets, records a
logical state manifest, returns a verified no-op for unchanged input, rebuilds
the same logical tree from an empty root, and emits an ordered filesystem sync
plan whose deletions occur only after content and metadata activation.

## Key Decisions

- Preserve `build-local` and add `rebuild-local` -> Phase 1 remains a stable,
  independently reproducible contract while Phase 2 accepts multi-release
  fixture sets.
- Define rebuild equivalence through a logical manifest digest -> OpenPGP and
  repository metadata timestamps may differ while the selected packages,
  configuration, and signing identity remain equivalent.
- Verify retained package digests and repository signatures before returning a
  no-op -> a matching manifest alone must not hide local candidate corruption.
- Keep sync planning pure and transport-independent -> R2 credentials and
  remote mutation remain deferred while ordered content, index, activation,
  state, and deletion stages are testable locally.
- Exercise every planned interruption point -> retained content remains
  available and expired content is not removed before activation completes.

## Changes

- `internal/localrepo/rebuild.go` - added release discovery, stable semantic
  version retention, checksum/package inspection, logical manifests, rebuild,
  and verified no-op behavior.
- `internal/localrepo/syncplan.go` - added deterministic filesystem snapshots
  and ordered create/replace/delete planning with deletions last.
- `internal/cli/rebuild_local.go` and `internal/cli/plan_sync.go` - exposed the
  Phase 2 behavior as thin JSON-emitting Cobra commands.
- `internal/localrepo/*_test.go` and `internal/cli/root_test.go` - added
  behavior-focused coverage, including simulated interruption at every action.
- `scripts/phase2-local.sh` and `testdata/phase2/projects.yml` - added the
  disposable three-release end-to-end proof.
- `moon.yml` and `README.md` - added and documented
  `moon run root:phase2-proof`.

## Open Threads

- Begin Phase 3 with the smallest remaining secrets-free CI/workflow slice:
  run the Phase 2 proof on hosted PR CI where appropriate, add workflow
  lint/security policy and unprivileged publish/rebuild validation, and extend
  the initial operator documentation.
- Keep GitHub Release discovery, R2 transport, production signing material,
  protected environments, and external provisioning out of Phase 3.
- Preserve the Phase 0 RPM fail-closed activation policy when a real transport
  adapter is introduced later.

## Lessons

- Determinism for signed repositories is a logical-content property, not
  necessarily byte-for-byte equality of timestamp-bearing metadata and
  signatures.
- On macOS Docker mounts, proof result files consumed immediately by another
  container should be written from the container view to avoid transient empty
  host-redirection observations.

## References

- [Session 001 design proposal](../001/DESIGN_PROPOSAL.md)
- [Session 002 summary](../002/SUMMARY.md)
- [PR #7: deterministic rebuild planning](https://github.com/meigma/packages/pull/7)
- [PR #7 CI run](https://github.com/meigma/packages/actions/runs/29651685211)
- [Post-merge CI run](https://github.com/meigma/packages/actions/runs/29651724124)
