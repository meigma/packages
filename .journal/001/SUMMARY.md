---
id: 001
title: Package repository design and CLI foundation
date: 2026-07-17
status: complete
repos_touched: [meigma/packages]
related_sessions: []
---

## Goal

Turn the supplied `meigma/packages` jumpstart into a practical, proof-driven
implementation direction, establish the repository-local CLI foundation, and
leave later sessions with clear phase boundaries rather than attempting the
full package-publishing system at once.

## Outcome

The goal was met. The session produced the standalone
[design proposal](DESIGN_PROPOSAL.md), revised cross-repository dispatch to use
one centrally managed GitHub App, and merged the Go CLI/developer-tooling
foundation through [PR #1](https://github.com/meigma/packages/pull/1). `main` is
clean at squash commit `a1fc27bb14279add011ab4afcdc85d47f1d9895d`, with
post-merge CI green.

Future implementation sessions should treat
[Section 19, Agile delivery sequence](DESIGN_PROPOSAL.md#19-agile-delivery-sequence)
as the execution plan. Start with **Phase 0 — Throwaway format and consistency
spike**, satisfy its proof gate, and then advance through Phases 1–5 one
independently reviewable slice at a time. The proposal is a working direction,
not an immutable specification; evidence from an earlier phase may revise later
command boundaries or publication details while preserving the stated
invariants and user-facing contracts.

## Key Decisions

- Use an agile Phase 0–5 sequence -> uncertain APT/RPM signing, client behavior,
  and multi-object publication semantics must be proven before durable design is
  expanded.
- Keep GitHub Releases as the reconstructable source of truth -> R2 contains a
  derived static repository, not authoritative package history.
- Build and verify a complete candidate tree before remote mutation -> failed
  generation must not expose partial repository state.
- Use one private Meigma GitHub App for consumer dispatch -> consumers mint
  short-lived installation tokens while the App remains installed only on
  `meigma/packages`.
- Keep `meigma-packages` repository-local -> retain the Go/Cobra/Viper, mise,
  Moon, CI, lint, tests, and local docs foundation, but omit CLI release and
  container-publication machinery.
- Dual-license under Apache-2.0 or MIT -> match the established Meigma license
  structure and contribution terms.

## Changes

- `.journal/001/DESIGN_PROPOSAL.md` - comprehensive architecture and phased
  execution proposal covering registry/release contracts, candidate-tree
  generation, APT/RPM metadata, retention, ordered publication, workflows,
  authentication, proof gates, ownership, and provisioning.
- `cmd/meigma-packages` and `internal/cli` - thin signal-aware Cobra/Viper CLI
  foundation with help/version behavior and tests.
- `mise.toml`, `mise.lock`, `.moon/`, and `moon.yml` - pinned local toolchain and
  format/lint/build/test/docs task graph.
- `.github/workflows/ci.yml` and `.github/dependabot.yml` - read-only,
  SHA-pinned CI and dependency maintenance without release workflows.
- `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, and `docs/` - repository-local
  development and documentation baseline.
- `LICENSE`, `LICENSE-APACHE`, and `LICENSE-MIT` - Apache-2.0/MIT dual licensing.

## Open Threads

- Execute [Phase 0](DESIGN_PROPOSAL.md#phase-0--throwaway-format-and-consistency-spike):
  generate tiny DEB/RPM fixtures, sign repositories, install from clean distro
  containers, exercise Ed25519, and fault-inject ordered publication.
- Do not begin durable package orchestration until Phase 0 either proves the
  direct-static model or records the precise blocker and approved deviation.
- Implement Phases 1–3 without production secrets: local vertical slice,
  deterministic retention/rebuild/sync planning, then unprivileged CI/workflow
  validation.
- Josh-owned R2, cache, signing-key, protected-environment, and GitHub App
  provisioning begins only in Phase 4 using the proposal checklist.
- Inspect a real `meigma/incus-gh-runner` release before finalizing asset
  patterns; production onboarding and consumer dispatch remain Phase 5.

## Lessons

- Workflow serialization does not make a multi-object R2 update atomic. APT
  by-hash, RPM checksum-named metadata, activation ordering, CDN cache behavior,
  and interruption tests collectively determine client-visible consistency.
- The correct first implementation is intentionally disposable: learn the
  exact tool and client behavior, then retain only the evidence and interfaces
  worth making durable.

## References

- [Canonical phased implementation plan](DESIGN_PROPOSAL.md)
- [PR #1: CLI and repository tooling foundation](https://github.com/meigma/packages/pull/1)
- [Post-merge `main` CI run](https://github.com/meigma/packages/actions/runs/29631867648)
- Original jumpstart attachment:
  `/Users/josh/.codex/attachments/753865f2-0865-457c-b7d4-f0c41887b9b8/pasted-text.txt`
