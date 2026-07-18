---
id: 004
title: Phase 3 CI and unprivileged workflow integration
date: 2026-07-18
status: complete
repos_touched: [meigma/packages]
related_sessions: [001, 002, 003]
---

## Goal

Reconcile session 001's Phase 3 plan with the implementation already delivered
through Phase 2, then land the smallest secrets-free CI and workflow slice that
made everything buildable without external provisioning green on a PR.

## Outcome

The goal was met. [PR #8](https://github.com/meigma/packages/pull/8) added
unprivileged publish/rebuild workflow validation, request validation in the
repository CLI, pinned workflow and shell tooling, executable workflow/image
policy, and current operator documentation. CI and Kusari Inspector passed on
the exact reviewed head, and the PR squash-merged to `main` as
`889358b0bec03a243af54cd261e932836033ecb6`.

Local `main` is clean and synchronized. The implementation worktree and local
and remote feature branches were removed. Post-merge CI run
[29654406090](https://github.com/meigma/packages/actions/runs/29654406090)
passed, followed by successful merged-main dispatches of
[Publish validation](https://github.com/meigma/packages/actions/runs/29654453308)
and [Rebuild validation](https://github.com/meigma/packages/actions/runs/29654454030).

## Key Decisions

- Expose `validate-request` through the Go CLI -> workflows reuse registry and
  stable-tag rules instead of duplicating package policy in YAML or shell.
- Keep publish/rebuild workflows manual and fixture-backed -> Phase 3 proves
  the orchestration surface without implying that R2 publication exists.
- Enforce the Phase 3 privilege boundary as code -> all workflows remain
  GitHub-hosted, read-only, full-SHA pinned, credential-persistence-free, and
  disconnected from secrets and deployment environments.
- Reuse Phase 1 and Phase 2 proofs -> the validation workflows exercise the
  same clean-install, rebuild/no-op, and deletion-safe planning paths already
  maintained locally and in PR CI.

## Changes

- `internal/localrepo/validation.go` and `internal/cli/validate_request.go` -
  added registered-project and optional stable release-tag validation.
- `.github/workflows/publish.yml` and `.github/workflows/rebuild.yml` - added
  manual secrets-free validation workflows over fixture inputs.
- `scripts/check_workflow_policy.py` and `scripts/phase3-ci.sh` - added
  action/image pinning and unprivileged workflow policy proof.
- `mise.toml`, `mise.lock`, and `moon.yml` - pinned actionlint and ShellCheck and
  integrated the workflow check into the full CI task graph.
- `README.md` and `docs/` - documented the current proof surface, manual
  workflows, and the explicit boundary before staging credentials are added.

## Open Threads

- Phase 4 begins with Josh-owned R2/domain/token, signing material, cache rules,
  and protected-environment provisioning.
- Add GitHub Release discovery and remote transport behind a separate
  privileged mutation job; preserve read-only validation before that boundary
  and rehearse against `_staging/` before any production path.
- Preserve the approved RPM fail-closed activation constraints when the real
  transport adapter is introduced.
- Consumer repository dispatch and the first real project remain Phase 5.

## Lessons

- Phase 3 was narrower than the original proposal because existing Moon CI
  already ran both end-to-end fixture proofs on hosted runners. The useful
  increment was enforceable workflow policy and real dispatch surfaces, not a
  duplicate CI pipeline.

## References

- [Session 001 design proposal](../001/DESIGN_PROPOSAL.md)
- [Session 003 summary](../003/SUMMARY.md)
- [PR #8](https://github.com/meigma/packages/pull/8)
- [PR CI run](https://github.com/meigma/packages/actions/runs/29653434897)
- [Post-merge CI run](https://github.com/meigma/packages/actions/runs/29654406090)
- [Publish validation run](https://github.com/meigma/packages/actions/runs/29654453308)
- [Rebuild validation run](https://github.com/meigma/packages/actions/runs/29654454030)
