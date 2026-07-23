---
id: 008
title: Generalize protected stable package publication
started: 2026-07-22
---

## 2026-07-22 18:27 — Kickoff
Goal for the session: Generalize the existing protected package publisher so registered projects can publish independently verified stable `vX.Y.Z` releases beyond the initial `incus-gh-runner v1.0.0` rehearsal, then deliver the focused change through a ready PR without merging or publishing.
Current state of the world: `incus-gh-runner v1.1.0` is released at commit `6a84c2aac068eabc331aa3f2e10003f77c37ccee`, and release-source validation passes, but packages workflow run `29971445189` fails closed because protected staging still confines Phase 5 to `incus-gh-runner v1.0.0`; trusted dispatch carries only `project` and `tag`.
Plan: Start an implementation worktree from the current fetched default branch, identify and remove one-release assumptions across policy, workflows, tests, and docs, preserve every privileged boundary, run the complete local gates, push a feature branch, open a ready PR, and wait for hosted CI.

## 2026-07-22 18:37 — Generalized contract and local proof
Created `feat/generalize-stable-publisher` from current `main` at `f05cfdb`. The protected publisher now consumes registered package identity and the package version derived by removing exactly one leading `v` from an exact validated `vX.Y.Z` tag. Source proof, staging, production, Debian, Ubuntu, and Fedora paths no longer pin `incus-gh-runner v1.0.0`; production confirmation is derived as `publish <project> <tag> to production`.

Added exact dispatch-payload validation so any field beyond `project` and `tag` fails before protected jobs, plus executable policy regressions for confirmation mismatch, staging bypass, R2 target changes, privileged dispatch controls, malformed/prerelease tags, unknown projects, and tag/package-version mismatch. Staging/production environments, separate credentials, serialized ordering, `_staging/` isolation, production-root mode, release digest and provenance verification, no-op verification, and deletion controls remain unchanged.

Validation: the repository aggregate gate passed with the documented local `PROTO_GO_VERSION=1.26.4` workaround; Actionlint, ShellCheck, policy/unit tests, and all Go tests passed. The real `incus-gh-runner v1.1.0` proof independently fetched five release assets, verified digests and attestations, rebuilt both architectures and formats, and clean-installed version `1.1.0` on Debian and Fedora. Next: review the complete diff, commit, push, open the ready PR, and wait for hosted CI.

## 2026-07-22 18:40 — Ready PR and hosted CI
Committed the focused implementation as `4e7e0d1a6f9301dbe36bc4817bc57401b0d0b0b7` with subject `fix(publish): accept verified stable release versions`, pushed `feat/generalize-stable-publisher`, and opened ready PR #13. The PR body records the exact tag-to-package-version derivation and preserved release, staging, production, credential, R2, signing, provenance, no-op, and deletion boundaries.

Hosted CI run `29972342517` and Kusari Inspector both passed on the exact PR head; GitHub reports the PR clean and mergeable. No merge, R2 mutation, or production publication was performed. After explicit merge approval, the correct publication retry is consumer run `meigma/incus-gh-runner` `29971438115` (`Publish Packages` for release `v1.1.0`), not failed packages run `29971445189`, because rerunning the consumer emits a new repository dispatch against the then-current packages default branch.

## 2026-07-22 18:52 — Merge and completed v1.1.0 publication
After explicit approval, squash-merged PR #13 on the exact reviewed head as `cdc66905ff0398a24635f9b0a6e5efa60b8ce25f`, fast-forwarded the clean local `main`, and confirmed merged-main CI run `29972545107` passed. Removed the integrated `feat/generalize-stable-publisher` Worktrunk worktree and local branch; local `main` is clean and synchronized with `origin/main`.

Reran trusted consumer run `meigma/incus-gh-runner` `29971438115` as attempt 2 for release commit `6a84c2aac068eabc331aa3f2e10003f77c37ccee`. It emitted fresh packages repository-dispatch run `29972562362` on exact packages merge SHA `cdc66905`. Unprivileged validation passed in 1m58s, protected staging passed in 2m42s, and protected production passed in 2m45s.

Staging desired state `5391844b281c2fc565fa2e6111a1a0ce8d48b1e36157c85993e2cc5e1c23d824` and production desired state `0bde2f03c62360dabbf5fbee4148d39fe05291e4ed5ad0fdb5db37b774d18dd4` each applied 40 ordered R2 actions, then passed remote verification, immediate no-op reconciliation, signing-key verification, and clean Debian/Ubuntu/Fedora installs. Independent public reads of both manifests show only selected version `v1.1.0` and DEB/RPM package version `1.1.0` for amd64/arm64 repository mappings. The original publishing attempt is complete; session 008 remains open until explicitly closed.
