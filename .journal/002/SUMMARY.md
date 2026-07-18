---
id: 002
title: Package format proof and local candidate builder
date: 2026-07-17
status: complete
repos_touched: [meigma/packages]
related_sessions: [001]
---

## Goal

Familiarize with session 001's phased plan, execute the disposable Phase 0
package-format and publication-consistency spike, and use its evidence to decide
whether durable implementation could begin.

## Outcome

The goal was met. Phase 0 proved signed Ed25519 APT and RPM repositories against
clean Debian 13, Ubuntu 26.04 LTS, and Fedora 44 clients, corrected the APT
by-hash contract, and isolated the unavoidable two-object RPM activation window.
Josh approved a bounded fail-closed policy for that window, recorded in
`spikes/phase0/EVIDENCE.md`, and [PR #5](https://github.com/meigma/packages/pull/5)
merged the proof at `23f23ddbf55f88efabc1af23b2756f9e5c6d567e`.

The evidence also cleared the way for the smallest Phase 1 vertical slice.
[PR #6](https://github.com/meigma/packages/pull/6) merged a registry-driven local
candidate builder, offline signature verification, and a one-command clean-client
installation proof at `0c4517f356e047f920e944579ca013a7fba8ee3f`. CI and
Kusari Inspector passed on both exact reviewed heads. `main` is clean and current,
and both implementation worktrees and branches were removed.

## Key Decisions

- Publish and retain both SHA-256 and SHA-512 APT by-hash indexes -> current APT
  clients requested SHA-512 and otherwise fell back to the mutable index.
- Accept a tightly bounded, fail-closed RPM metadata/signature window for v1 ->
  independent object writes cannot atomically switch `repomd.xml` and its
  detached signature, and DNF5 does not use a later matching signature from a
  multi-signature bundle.
- Constrain the RPM window with cache bypass, consecutive writes, no verification
  bypass, retryable convergence, deferred deletion, and real package resolution
  or installation after publication -> a refresh exit code alone can hide a
  disabled DNF repository.
- Keep Phase 1 local and secrets-free -> prove durable orchestration and command
  boundaries before introducing GitHub, R2, or production signing credentials.

## Changes

- `spikes/phase0/` - disposable containerized DEB/RPM generation, Ed25519
  signing, clean-client compatibility checks, publication fault injection, and
  recorded evidence.
- `internal/localrepo/` - registry parsing and candidate-tree orchestration for
  signed APT and RPM repositories, including offline verification before success.
- `internal/cli/build_local.go` and `internal/cli/root.go` - the
  `meigma-packages build-local` command and its narrow dependency boundary.
- `internal/cli/root_test.go` and `internal/localrepo/registry_test.go` -
  behavior-focused Testify coverage for command request resolution and registry
  validation.
- `scripts/phase1-local.sh` and `testdata/phase1/` - one-command fixture build,
  candidate generation, clean Debian installation, and payload execution proof.
- `moon.yml` and `README.md` - developer task and usage documentation for
  `moon run root:phase1-proof`.

## Open Threads

- Begin Phase 2 with the smallest deterministic retention, rebuild, and sync-plan
  slice; let the implementation refine session 001's proposed boundaries.
- Preserve the approved RPM activation constraints in future remote publication
  work and assert package resolution or installation, not only DNF refresh status.
- R2, cache configuration, production signing material, protected environments,
  and GitHub App provisioning remain deferred to the later secrets-bearing phase.

## Lessons

- APT by-hash safety depends on publishing the hash algorithms clients actually
  request; SHA-256 alone was insufficient in the tested modern clients.
- Detached RPM metadata signatures make strict no-half-publish semantics
  impossible over independent static-object writes, so the honest v1 contract is
  a short fail-closed and automatically recoverable interval.
- DNF5 can report a successful cache operation while disabling a repository;
  package resolution or installation is the meaningful publication proof.
- The disposable spike paid for itself by correcting the design before the same
  assumptions were embedded in durable orchestration.

## References

- [Session 001 design proposal](../001/DESIGN_PROPOSAL.md)
- [Phase 0 evidence](https://github.com/meigma/packages/blob/0c4517f356e047f920e944579ca013a7fba8ee3f/spikes/phase0/EVIDENCE.md)
- [PR #5: Phase 0 package-format and publication proof](https://github.com/meigma/packages/pull/5)
- [PR #6: Phase 1 verified local package candidates](https://github.com/meigma/packages/pull/6)
