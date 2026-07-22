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
