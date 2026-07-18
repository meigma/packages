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
