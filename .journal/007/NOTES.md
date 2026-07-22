---
id: 007
title: New work session
started: 2026-07-21
---

## 2026-07-21 17:35 — Kickoff
Goal for the session: Start a fresh journal session; the substantive work request has not yet been provided.
Current state of the world: Phase 4 staging publication is complete on `main`, and the repository is ready for the next focused increment.
Plan: Bind this session to the current task, then wait for the user's actual request and proceed iteratively.

## 2026-07-21 17:46 — First-release DEB/RPM gap analysis
Goal: Determine what `meigma/incus-gh-runner` and `meigma/packages` need before the first `v1.0.0` release can ship installable DEB/RPM packages.
Current evidence: `incus-gh-runner` is clean on public `master` at `0b7a4b6`; Release Please PR #24 is open and green at `ed3ff0f`, and no tag or release exists. GoReleaser currently emits four raw binaries, four SBOMs, and one checksum file; Melange APKs feed only the OCI image. The release stager is hard-coded to those nine assets.
Findings: Add GoReleaser/nFPM DEB/RPM outputs for Linux amd64/arm64, include the safe systemd deployment surface without enabling the unconfigured service, stage and checksum all four packages, and prove native DEB/RPM installation in both architectures in the dry-run and tag workflows before merging the release PR. If canonical APT/DNF installation through `pkgs.meigma.dev` is required, `meigma/packages` also needs GitHub Release discovery, a real `projects.yml`, multi-architecture ingestion/index generation, a production-safe R2 root apply mode and protected environment, consumer dispatch, and post-publish smoke/recovery. The current publisher accepts exactly one DEB and one RPM and generates only one host-architecture APT index; only the `_staging/` environment exists.
Next: Present the work as small proof-driven slices and keep PR #24 unmerged until the package asset rehearsal passes.

## 2026-07-21 18:38 — Four release packages landed and rehearsed
Implementation: Opened `meigma/incus-gh-runner` PR #46 from `feat/release-packages` at `f73b86a`. GoReleaser/nFPM now builds DEB and RPM packages for Linux amd64 and arm64, including the binary, hardened systemd unit, tmpfiles policy, editable configuration, credential drop-in examples, licenses, and guarded daemon-reload scripts. The service remains disabled and unstarted after installation. Package assets are now staged with the raw binaries and SBOMs, covered by checksum/provenance validation, and documented as the preferred host installation path.
Validation: Pinned GoReleaser 2.17.0 produced all four packages locally. Clean ARM64 installations passed on Debian 13 and Fedora 44. The full local `root:check` gate and `actionlint` passed. Hosted Release Dry Run #29883466029 passed at the exact PR head, including clean DEB and RPM installs on native amd64 and arm64 runners; CI, CodeQL, GitHub Pages, and external repository checks are green. Release Please PR #24 remains unmerged.
Next: Review and merge PR #46 before allowing the first Release Please release PR to merge. Follow-on repository publication through `pkgs.meigma.dev` remains a separate slice.

## 2026-07-21 18:49 — Release package PR merged
Outcome: After maintainer approval, verified PR #46 was still at reviewed head `f73b86a`, its required CI and exact-head hosted package rehearsal were green, and it was mergeable. Squash-merged it on GitHub as `e69fd1c`, fast-forwarded the local `incus-gh-runner` `master`, and removed the clean integrated feature worktree and local/remote branch. Post-merge CI, both CodeQL runs, and the documentation build passed on `e69fd1c`. Release Please PR #24 remains open and was refreshed automatically by Release Please; it was not merged.
Next: The four GitHub Release package assets are ready for the first release. The next Phase 5 slice can freeze the real asset contract and implement `meigma/packages` release discovery and registry onboarding.

## 2026-07-21 19:16 — Published v1.0.0 verified
Outcome: Verified the public, non-prerelease `incus-gh-runner` `v1.0.0` release and tag both resolve to release commit `2ad80f9`; tag-triggered Release run #29884714489 succeeded on that exact SHA. All 13 expected assets exist. All 12 binary/package/SBOM files match `checksums.txt`, SPDX SBOMs parse, the native macOS ARM64 binary reports `1.0.0`, and GitHub SLSA provenance verifies against `.github/workflows/attest.yml` with `refs/tags/v1.0.0` and GitHub-hosted runners. Published DEB/RPM metadata is exactly amd64/arm64 and x86_64/aarch64; clean downloaded ARM64 installs passed on Debian 13 and Fedora 44 without enabling the service, while the tag workflow independently passed all four native installs. OCI digest `sha256:4445f7285ad45914495f0b847a25ac553f4495cfa65ab4120c1a4131d397f726` contains linux/amd64 and linux/arm64; its keyless Cosign signature and SLSA provenance both verify.
Hardening gap: The GitHub release API reports `immutable=false`, so `gh release verify v1.0.0` has no release-level attestation. Artifact checksums, GitHub asset digests, exact-workflow build provenance, and OCI signing are healthy, but future source ingestion should enforce those controls and a follow-up should enable immutable releases plus public copy-paste verification instructions.
Next: Freeze the exact four real package names in the `meigma/packages` source contract and implement the first real GitHub Release discovery/registry-onboarding slice.

## 2026-07-21 20:08 — Real release source proof established
Implementation: On `feat/phase5-release-source`, added the canonical root `projects.yml` entry for `meigma/incus-gh-runner`, including package identity, exact checksum asset, versioned DEB/RPM patterns, amd64/arm64 package metadata mappings, retention, and the pinned attestation signer workflow. Added an atomic `fetch-release` path that accepts only registered stable published releases, validates the complete package set, checks GitHub API SHA-256 digests and sizes, and verifies each package's SLSA provenance before exposing the download. Extended deterministic rebuilds to validate and index both APT and RPM architectures while preserving the fixture-backed legacy path.
Proof: Unit and failure-path tests pass, including incomplete release sets, duplicate architecture mappings, digest mismatch, and no partial output after provenance failure. The live `v1.0.0` fetch selected the checksum file and four packages and passed all four pinned-workflow attestation checks. `root:phase5-source-proof` then ingested those exact public assets, produced `binary-amd64` and `binary-arm64` APT indexes plus `x86_64` and `aarch64` RPM trees, and completed clean native DEB and RPM installs from the newly signed local repositories.
Boundary: This slice remains read-only toward GitHub and R2. The live proof is opt-in, uses no publishing credentials, and does not add an automated privileged workflow, production R2 apply, consumer dispatch, or cache/rollback changes.
Next: Finish documentation and full repository validation, then open the Phase 5 source-onboarding PR for review.
