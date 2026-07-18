---
id: 004
title: Phase 3 CI and unprivileged workflow integration
started: 2026-07-18
---

## 2026-07-18 09:55 — Kickoff
Goal for the session: Familiarize with session 001's planning artifacts and begin Phase 3.
Current state of the world: Phases 0–2 are complete on `main`; Phase 2 added deterministic rebuild, verified no-op behavior, and deletion-safe sync planning. Phase 3 is the next incomplete secrets-free slice and is expected to cover hosted CI, workflow lint/security policy, unprivileged publish/rebuild validation, and initial operator documentation without pulling GitHub Release discovery, R2, or production secrets forward.
Plan: Read the canonical session 001 planning artifacts, reconcile Phase 3 with the current repository, and identify the smallest proof-driven implementation slice.

## 2026-07-18 09:57 — Phase 3 orientation
Read session 001's `DESIGN_PROPOSAL.md` and `SUMMARY.md`, plus the summaries and durable outcomes from sessions 002 and 003. The proposal remains a working direction rather than a fixed specification: GitHub Releases are authoritative, candidates are built and verified before mutation, workflows stay thin, and Phase 3 must remain free of R2, production keys, protected-environment wiring, and real remote mutation.

Reconciled the plan with `main` at `d3b67865c258ce81e3d4e7f40a5f000870e2581f`. The current `ci.yml` is already GitHub-hosted, read-only, secrets-free, and full-SHA pinned. Exact post-merge run 29651724124 passed and its logs confirm that both `root:phase1-proof` and `root:phase2-proof` ran, including clean package installation, verified no-op/rebuild behavior, and ordered sync-plan proof.

The remaining Phase 3 gap is therefore narrower than the original proposal: add pinned workflow and shell linting plus repository-specific workflow policy checks; introduce publish/rebuild workflow validation shells that cannot access secrets, environments, or remote mutation; and update initial operator/developer documentation, especially the stale docs index. Preserve the existing proofs instead of duplicating them. The smallest first experiment should establish the lint/policy gate and a deliberately unprivileged workflow shape, then let hosted PR evidence determine the next increment.

## 2026-07-18 10:10 — Phase 3 local proof
Created isolated Worktrunk branch `feature/phase-3-unprivileged-workflows` from `main` at `d3b67865c258ce81e3d4e7f40a5f000870e2581f`. Added a thin `validate-request` CLI seam over the existing registry and stable-tag validation, manual fixture-backed publish/rebuild validation workflows, pinned actionlint and ShellCheck tooling, executable workflow/container-image policy, and updated developer/operator documentation.

The Phase 3 policy requires empty top-level permissions, no write scopes, GitHub-hosted runners, full-SHA action pins, disabled checkout credential persistence, pinned external container images, and no secrets, environments, or privileged pull-request triggers. The manual workflows intentionally perform no remote mutation and use the existing Phase 1/2 proofs rather than adding duplicate package logic to YAML.

Local verification passed with `mise exec -- moon run root:check root:phase1-proof root:phase2-proof --summary minimal`: 13 tasks completed, including Go format/lint/build/test, strict docs, workflow and shell policy, clean package installation, deterministic rebuild/no-op behavior, and deletion-safe sync planning. Next: commit the isolated implementation, open a PR, and verify hosted checks on the exact head.
