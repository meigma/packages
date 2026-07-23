---
id: 008
title: Generalize protected stable package publication
date: 2026-07-22
status: complete
repos_touched: [meigma/packages]
related_sessions: [001, 004, 005, 007]
---

## Goal
Generalize the existing protected package publisher beyond the initial `incus-gh-runner v1.0.0` safety rail so registered projects can publish independently verified exact stable `vX.Y.Z` releases without weakening staging, production, provenance, credential, deletion, or R2 boundaries.

## Outcome
The goal was met. PR #13 generalized the single protected publication path and was squash-merged as `cdc66905ff0398a24635f9b0a6e5efa60b8ce25f`. A fresh trusted consumer dispatch then published `incus-gh-runner v1.1.0` through unprivileged validation, protected staging, and protected production. Both public manifests report package version `1.1.0` for DEB/RPM amd64 and arm64 mappings, and the workflow passed remote verification, no-op replay, signing verification, and clean Debian, Ubuntu, and Fedora installs.

## Key Decisions
- Accept only exact `vX.Y.Z` tags and derive the expected package version by removing exactly one leading `v` after validation -> avoids loose, prerelease, repeated-prefix, and build-metadata interpretations.
- Return registered package identity and derived version from `validate-request` -> keeps shell publication and clean-install checks on the same registry-backed contract.
- Require repository-dispatch payloads to contain exactly `project` and `tag` -> extra deletion, staging-bypass, confirmation, or R2-target controls fail before protected jobs.
- Derive `publish <project> <tag> to production` from validated values -> manual runs must type the exact phrase while trusted dispatch synthesizes it internally and cannot override it.
- Preserve the validation -> staging -> production chain and independently revalidate the release in each protected job -> no privileged step trusts dispatch metadata or a prior job as artifact authority.
- Retry the consumer release run rather than the failed packages run -> the fresh repository dispatch selected the newly merged packages default-branch workflow instead of replaying the obsolete bootstrap implementation.

## Changes
- `internal/localrepo/validation.go` and version tests - expose registered package identity and the exact tag-derived version while tightening stable tags to `vX.Y.Z`.
- `scripts/phase5-source.sh` and `scripts/phase5-publish.sh` - remove project/release pins, consume validated identity, derive confirmation, and assert dynamic package versions in clean installs.
- `.github/workflows/publish.yml` - generalize manual and trusted-dispatch confirmation while preserving protected job ordering and separate environments.
- `scripts/validate_publish_event.py` and workflow policy tests - reject privileged dispatch fields and regressions that bypass staging or alter R2 targeting.
- `README.md` and `docs/docs/` - document the generalized release, confirmation, and operations contracts.
- Production operation - reran `meigma/incus-gh-runner` run `29971438115` attempt 2, which created successful packages run `29972562362` for `v1.1.0`.

## Open Threads
- Automate R2 credential renewal or replace long-lived credentials with an OIDC broker.
- Onboard additional projects through the existing `projects.yml` registry contract.
- Multi-release discovery remains separate work if protected publication should retain more than the explicitly selected release across runs.

## Lessons
- Rerunning a failed target repository run reuses its original workflow revision; retry cross-repository publication from the trusted consumer so the new dispatch resolves the target repository's current default branch.
- Workflow success is not sufficient handoff evidence: verify the exact target SHA, inspect protected-job proof summaries, and read the public staging and production manifests after publication.

## References
- [PR #13](https://github.com/meigma/packages/pull/13)
- [PR CI run 29972342517](https://github.com/meigma/packages/actions/runs/29972342517)
- [Merged-main CI run 29972545107](https://github.com/meigma/packages/actions/runs/29972545107)
- [Successful publisher run 29972562362](https://github.com/meigma/packages/actions/runs/29972562362)
- [Consumer run 29971438115](https://github.com/meigma/incus-gh-runner/actions/runs/29971438115)
- [Released source v1.1.0](https://github.com/meigma/incus-gh-runner/releases/tag/v1.1.0)
- [Production manifest](https://pkgs.meigma.dev/_state/manifest.json)
- [Session 007 summary](../007/SUMMARY.md)
