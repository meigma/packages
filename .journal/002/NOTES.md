---
id: 002
title: Phase 0 format and consistency spike
started: 2026-07-17
---

## 2026-07-17 22:24 — Kickoff
Goal for the session: Familiarize with session 001's planning artifacts and begin Phase 0, the throwaway package-format and publication-consistency spike.
Current state of the world: Session 001 completed the canonical design proposal and merged the repository-local Go CLI and developer-tooling foundation on `main`; Phase 0 is the first incomplete delivery phase and durable package orchestration has not begun.
Plan: Read the Phase 0 contracts and proof gates, identify the smallest disposable experiments, then execute them incrementally and record evidence before deciding what should become durable.

## 2026-07-17 22:26 — Session 001 planning review
Read session 001's summary and canonical `DESIGN_PROPOSAL.md` end to end. Phase 0 is deliberately a disposable Bash/container spike: build tiny DEB/RPM fixtures, generate and sign APT and RPM repositories, install through clean Debian/Ubuntu/Fedora clients, prove Ed25519 compatibility, and fault-inject every ordered publication boundary. The gate is evidence that direct static publication remains viable, or a precise signing/consistency blocker escalated before durable orchestration begins.

The durable constraints to preserve are that GitHub Releases remain authoritative, R2 is reconstructable derived state, a complete candidate is verified before remote mutation, immutable/content-addressed objects activate before mutable metadata, and deletions happen last. No R2, real signing key, GitHub App, or other external provisioning is needed in Phase 0.

Current `main` is clean at `a1fc27bb14279add011ab4afcdc85d47f1d9895d` with the Go/Cobra/Viper and Moon/mise foundation from PR #1. Docker 29.4.0 is available. Host `nfpm`, `dpkg-deb`, `apt-ftparchive`, `rpm`, and `createrepo_c` are absent, so the spike should keep its toolchain inside pinned containers; Podman is installed but its machine is not running.

Next: begin with the smallest format proof—one tiny package payload, throwaway Ed25519 key material, locally served APT and RPM repositories, and clean clients—then layer publication interruption cases onto the proven repository fixtures.

## 2026-07-17 22:38 — Format and Ed25519 baseline passed
Created implementation worktree `.wt/spike-phase-0-format-consistency` on branch `spike/phase-0-format-consistency` from `main` at `a1fc27b`. Commit `7f12eb1` adds an intentionally disposable, container-contained spike under `spikes/phase0/`.

The spike built `all`/`noarch` DEB and RPM fixtures, generated APT by-hash and RPM checksum-named metadata, created an Ed25519 certification primary plus Ed25519 signing subkey, signed `InRelease`, `Release.gpg`, and `repomd.xml.asc`, verified those signatures offline, served the result on an isolated Docker network, and installed the fixture with signature enforcement from clean pinned Debian 13, Ubuntu 26.04 LTS, and Fedora 44 images. All three clients passed. No production key, R2 access, or GitHub credential was used.

Next: generate old/new repository snapshots from the same throwaway key and run clean clients after each ordered-copy interruption point, with particular focus on the two-object `repomd.xml`/`repomd.xml.asc` activation boundary.

## 2026-07-17 22:52 — Phase 0 gate reached; RPM decision required
Completed and pushed the Phase 0 spike on `spike/phase-0-format-consistency` at `93174c7` (preceded by baseline commit `7f12eb1`). `spikes/phase0/EVIDENCE.md` records the pinned images, tool/client versions, reproduction commands, and observed interruption behavior. Verification passed with `run.sh`, `fault-injection.sh`, ShellCheck, Bash syntax checks, `git diff --check`, and `mise exec -- moon run root:check`.

The Ed25519 format/client matrix passed on Debian 13 (APT 3.0.3), Ubuntu 26.04 LTS (APT 3.2.0), and Fedora 44 (DNF5 5.4.2.1). APT initially exposed a proposal defect: current APT requested SHA-512 by-hash content, then fell back to the mutable index because only SHA-256 existed. Publishing and retaining both SHA-256 and SHA-512 by-hash indexes made the ordered flow safe on the tested client: old metadata remained usable through package/content/index/Release staging, and the single `InRelease` write switched cleanly to the new version.

RPM demonstrated the anticipated consistency blocker. Copying new `repomd.xml` before its old detached signature, or copying the new signature before old `repomd.xml`, produced `Bad PGP signature`; DNF disabled the repository and could not resolve the package. The repository recovered and installed the new version once both objects matched. DNF5 `makecache` may still exit successfully after disabling the bad repository, so the durable smoke/fault test must assert package resolution or installation rather than only the refresh exit code.

Phase 0 assessment: formats and Ed25519 pass; APT direct-static publication passes with the revised by-hash contract; the strict RPM no-half-publish invariant fails under independent object writes. Pause before Phase 1 for Josh to choose whether to accept and document a tightly bounded retry/unavailability window, require an atomic snapshot-routing design, or revise the invariant in another explicit way.

## 2026-07-17 22:55 — Transition-signature escape hatch rejected
Before pausing, tested whether an armored `repomd.xml.asc` containing both old and new detached signatures could bridge the activation. DNF5 accepted the old `repomd.xml` when the old signature was the first armor block, but after switching to the new XML it rejected the first bad signature and did not continue to the matching second signature. Reversing the order would only reverse which snapshot works. Commit `1b39202` records this final probe and updated evidence. The RPM owner decision is still required.

## 2026-07-17 23:17 — Phase 0 landed; Phase 1 local slice passed
Josh approved the bounded fail-closed RPM policy for v1. Recorded the operational constraints in the Phase 0 evidence, addressed Kusari's non-root container finding, and squash-merged PR #5 after CI and Kusari passed on exact head `952241f`. `main` is now `23f23dd`; the Phase 0 branch/worktree and remote branch were cleaned up.

Created `feature/phase-1-local-vertical-slice` from that main tip. Commits `eba7093` and `58fa0f9` add the durable `meigma-packages build-local` command, a minimal one-entry fixture registry, Go orchestration for APT/RPM generation and Ed25519 signing/verification, behavior-focused Testify tests, and `moon run root:phase1-proof` as the one-command developer gate. External package tools remain narrow process boundaries; no GitHub API, R2, or production key is involved.

The Phase 1 proof generated fixture release assets, cross-compiled the CLI for the Docker architecture, produced a fresh signed candidate tree, verified `InRelease`, `Release.gpg`, and `repomd.xml.asc` offline, then installed and executed the package from a clean Debian 13 client. `root:phase1-proof` and `root:check` both passed. Next: push the branch, open the independently reviewable Phase 1 PR, confirm hosted CI/security on its exact head, and merge if green.

## 2026-07-17 23:21 — Phase 1 merged and cleaned up
Opened PR #6, `feat(cli): build verified local package candidates`, from `feature/phase-1-local-vertical-slice`. GitHub CI and Kusari Inspector both passed on exact reviewed head `58fa0f9df2ccfa76476ff358825095f676f83194`. Squash-merged the PR as `0c4517f356e047f920e944579ca013a7fba8ee3f`; the merge command reported the known local-main-worktree cleanup warning after GitHub had already completed the merge, so the remote PR state and merge commit were verified independently.

Fast-forwarded the main checkout from `23f23dd` to `0c4517f`, then removed the merged Phase 1 Worktrunk worktree/local branch and deleted the remote feature branch. Phase 1's bounded deliverable is complete: a local, verified candidate builder and a clean-client installation proof are now on main. Session 002 remains active for the next delivery phase.

## 2026-07-18 08:49 — Close
Closed session 002 after confirming [PR #5](https://github.com/meigma/packages/pull/5) and [PR #6](https://github.com/meigma/packages/pull/6) are squash-merged with CI and Kusari Inspector green on their exact reviewed heads. Local `main` is clean and synchronized at `0c4517f`; no session implementation worktrees or branches remain. The durable APT by-hash and bounded fail-closed RPM publication findings are recorded in `TECH_NOTES.md`, and Phase 2 deterministic retention/rebuild/sync planning is the next implementation slice.
