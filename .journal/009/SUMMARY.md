---
id: 009
title: Strip delivery scaffolding and document publication
date: 2026-07-22
status: complete
repos_touched: [meigma/packages]
related_sessions: [001, 007, 008]
---

## Goal
Audit the repository for over-engineered tests, scripts, and gates left over
from the phased delivery process, strip everything that does not serve the
repo's one job (publishing signed deb/rpm packages to R2), and then verify and
fix the documentation against the cleaned-up state.

## Outcome
The goal was met. PRs #14–#16 removed ~1,850 lines of scaffolding and
ceremony, the reworked publish path was proven with a successful staging-only
dispatch of `incus-gh-runner v1.1.0` (run 29973794881: verification, no-op
replay, and clean Debian/Ubuntu/Fedora installs all passed; production
correctly skipped), and a 7-agent docs-verification workflow drove a final
docs PR. The repo now contains the Go CLI, two workflows (`ci.yml`,
`publish.yml`), two scripts (`publish.sh`, `lint-workflows.sh`),
`docker/tools.Dockerfile`, the registry, and accurate docs. CI dropped from
~65s to ~21s.

## Key Decisions
- Delete `check_workflow_policy.py` outright -> exact string-fragment
  assertions against `publish.yml` broke on every legitimate edit and protect
  nothing a reviewer doesn't; anyone who can edit the workflow can edit the
  checker in the same PR.
- Delete the phase 1/2 fixture proofs, `phase4-staging.sh`, `rebuild.yml`,
  `testdata/`, and `spikes/` -> delivery-phase scaffolding fully superseded by
  Go unit tests and the real publish path; the proofs ran docker fixture
  builds and container installs on every CI run.
- Keep the clean-install smoke test, apply-sync verify/no-op replay, protected
  environments, and pinned actionlint/ShellCheck -> these are the real gates.
- Drop the typed confirmation phrases and `empty_staging` input -> neither
  environment has required reviewers, so the phrase was only a self-typed echo
  of already-typed inputs; the `apply_production` checkbox (default false)
  remains the manual speed bump, and staging recovery is just another
  convergent staging publish.
- Drive staging/production URLs and prefixes from environment-scoped vars
  instead of hardcoded literals in the script -> a domain or prefix change no
  longer touches three files in lockstep.
- Promote `Dockerfile.tools` out of the "disposable" spike directory to
  `docker/tools.Dockerfile` -> it was load-bearing in the publish path.
- Verify docs with a small multi-agent workflow (sonnet reviewers, opus
  adversarial verification) before writing fixes -> all flagged issues were
  confirmed against source; the dispatch example was corrected against the
  real consumer workflow rather than documented from assumption.

## Changes
- PR #14 (`211d286`) - deleted `check_workflow_policy.py`(+test), phase
  1/2/4 scripts, `testdata/`, `rebuild.yml`, spike extras; `phase3-ci.sh`
  became `lint-workflows.sh`; moon tasks and docs updated.
- PR #15 (`fa6accf`) - `phase5-publish.sh` became env-driven
  `scripts/publish.sh`; deleted `phase5-source.sh` (hardcoded 5-asset/4-package
  asserts), `validate_publish_event.py`(+test), and `spikes/`; moved the tools
  image to `docker/tools.Dockerfile`; `publish.yml` renamed `Publish` with
  inputs cut to `project`/`tag`/`apply_staging`/`apply_production`.
- PR #16 (`08a07dd`) - fixed stale README claims (fixture wording, mise tool
  list, live `MEIGMA_PACKAGES_*` prefix); new `docs/docs/publishing.md`
  documenting the `projects.yml` contract, upstream release requirements, the
  project-key -> APT component / `rpm/<project>/` convention, and
  manual/dispatch publish invocation; generalized `install.md`.
- GitHub settings - removed the stale `feat/phase4-staging` deployment branch
  allowance from the staging environment (both environments now allow `main`
  only).

## Open Threads
- Automate R2 credential renewal or replace long-lived credentials with an
  OIDC broker (renew the staging credential before `2027-07-18T19:51:37Z`).
- Onboard additional projects; the registry contract is now documented in
  `docs/docs/publishing.md`.
- Optional: add required reviewers to the `production` environment if a harder
  manual gate is ever wanted — noting it would also gate the automated
  consumer dispatch.
- Trivial help-text assertions in `internal/cli/root_test.go` were left as-is.

## Lessons
- Neither GitHub environment had required reviewers — the elaborate
  confirmation-phrase ceremony was compensating for a protection that was
  never configured. Check the actual enforcement surface before trusting
  in-workflow gates.
- The "disposable" spike directory had become load-bearing (the publish path
  built its Docker image from it); labeling code disposable does not keep it
  disposable.
- `gh pr merge --delete-branch` fails to delete when the PR branch's worktree
  blocks the local checkout switch; delete remote branches explicitly and
  clean worktrees with `wt remove`.

## References
- [PR #14](https://github.com/meigma/packages/pull/14)
- [PR #15](https://github.com/meigma/packages/pull/15)
- [PR #16](https://github.com/meigma/packages/pull/16)
- [Staging proof run 29973794881](https://github.com/meigma/packages/actions/runs/29973794881)
- [Session 008 summary](../008/SUMMARY.md)
