---
id: 007
title: First-release Linux packages and Phase 5 publication
date: 2026-07-21
status: complete
repos_touched: [meigma/packages, meigma/incus-gh-runner]
related_sessions: [001, 002, 003, 004, 005]
---

## Goal
Ship installable DEB and RPM packages with the first `incus-gh-runner` release, publish them through the signed `pkgs.meigma.dev` APT/RPM repository, and prove the release-to-repository dispatch path end to end.

## Outcome
The goal was met. `incus-gh-runner` v1.0.0 ships native amd64/arm64 DEB and RPM assets, the public APT/RPM repository is live and reconstructable from verified GitHub Release inputs, and a repository-restricted GitHub App dispatch successfully drove protected staging and production publication plus clean Debian, Ubuntu, and Fedora installs.

## Key Decisions
- Build DEB/RPM assets with GoReleaser/nFPM and install the hardened service surface without enabling or starting an unconfigured service.
- Treat GitHub Releases as authoritative and R2 as derived state; verify asset digests, the complete architecture set, and pinned-workflow SLSA provenance before candidate construction or mutation.
- Keep staging and production in separate protected jobs with separate R2 credentials, exact destructive confirmations, verification before activation, repeat no-op proofs, and clean client installs.
- Accept only a fixed `publish-package` dispatch contract and mint a short-lived GitHub App token restricted to `meigma/packages`; consumer payloads cannot select privileged modes or bypass gates.
- Preserve the initial `v1.0.0` publication constraint until the first complete path was proven, deferring general retention/version selection rather than over-generalizing before evidence existed.

## Changes
- `meigma/incus-gh-runner/.goreleaser.yaml` and `packaging/systemd-daemon-reload.sh` - added four native Linux packages and safe system service installation behavior.
- `meigma/incus-gh-runner/.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, and release staging scripts - stage, validate, attest, and clean-install the package assets on native architectures.
- `meigma/incus-gh-runner/.github/workflows/packages.yml` - dispatch stable published releases through a repository-restricted GitHub App token.
- `meigma/packages/projects.yml`, `internal/githubrelease/`, and `internal/localrepo/` - register and atomically ingest the verified real release, then build multi-architecture APT/RPM trees.
- `meigma/packages/internal/r2repo/`, `internal/cli/apply_sync.go`, and `scripts/phase5-publish.sh` - add production-root publication, safe ordering/cache behavior, hydration, verification, recovery, and no-op reconciliation.
- `meigma/packages/.github/workflows/publish.yml` - add trusted consumer dispatch plus protected staging and production publication gates.
- Installation and operations documentation in both repositories - publish copy-paste APT/RPM setup, full key-fingerprint verification, direct-asset fallback, and operator boundaries.

## Open Threads
- Generalize the protected publisher beyond the initial pinned `incus-gh-runner` `v1.0.0` selection while preserving retention and complete-set validation.
- Automate R2 credential renewal and document/test operator rollback beyond the existing staging empty-prefix recovery rehearsal.
- Enable immutable GitHub Releases and add public release-level verification once the repository setting and release workflow support it.
- Resolve the local Moon/Proto Go system-toolchain mismatch so the aggregate gate does not require `PROTO_GO_VERSION=1.26.4` when global Proto selects a newer Go.

## Lessons
- A selected-repository GitHub App installation fails closed before dispatch when the target repository is absent; adding only the intended target restored the path without broadening token permissions.
- Publication success alone is insufficient proof: rehydration, repeat no-op reconciliation, and clean installs caught the operational contract across repository formats and architectures.

## References
- [incus-gh-runner PR #46](https://github.com/meigma/incus-gh-runner/pull/46)
- [incus-gh-runner release PR #24](https://github.com/meigma/incus-gh-runner/pull/24)
- [meigma/packages PR #10](https://github.com/meigma/packages/pull/10)
- [meigma/packages PR #11](https://github.com/meigma/packages/pull/11)
- [meigma/packages PR #12](https://github.com/meigma/packages/pull/12)
- [incus-gh-runner PR #47](https://github.com/meigma/incus-gh-runner/pull/47)
- [End-to-end consumer run #29892388968](https://github.com/meigma/incus-gh-runner/actions/runs/29892388968)
- [End-to-end publisher run #29892394354](https://github.com/meigma/packages/actions/runs/29892394354)
- [Public production manifest](https://pkgs.meigma.dev/_state/manifest.json)
