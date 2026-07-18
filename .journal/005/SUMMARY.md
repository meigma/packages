---
id: 005
title: Phase 4 staging publication rehearsal
date: 2026-07-18
status: complete
repos_touched: [meigma/packages]
related_sessions: [001, 002, 003, 004]
---

## Goal
Advance the package repository plan through Phase 4 by provisioning the protected staging boundary, applying a real candidate to R2, proving remote verification and clean-client installation, and rehearsing recovery from an empty staging prefix without touching production.

## Outcome
The goal was met. PR #9 added the protected R2 staging publisher and was squash-merged into `main` as `f2e361e28f8bd87e63bd654333f64381ad2987c1`. The R2 bucket, custom domain, signing material, protected GitHub environment, prefix-scoped credential, and cache policy were provisioned. Hosted runs proved ordered publication, remote byte verification, immediate no-op convergence, and installation on Debian 13, Ubuntu 26.04, and Fedora 44. A final drill deleted only `_staging/`, confirmed it was empty, and rebuilt and reverified the complete 21-object fixture repository.

## Key Decisions
- Keep privileged staging mutation in a separate protected job -> preserves the Phase 3 secrets-free validation boundary and prevents pull-request-derived mutation.
- Fail closed on the exact `_staging/` prefix and canonical public URL -> the rehearsal credential and code cannot address production paths.
- Keep the certify-only primary GPG key in 1Password and install only the signing-only subkey in GitHub -> limits hosted-run signing authority while retaining an offline recovery root.
- Use a locally signed one-year R2 credential scoped to `meigma-packages/_staging/` -> avoids copying the account-wide parent credential and removes the seven-day release blocker while preserving least privilege.
- Perform the disaster-recovery drill by explicitly emptying only staging before dispatching the protected workflow -> proves reconstruction without risking production data.
- Accept manual Cloudflare cache-rule provisioning -> the available API credentials lacked Rulesets permission; live staging requests confirmed the expected uncached behavior.

## Changes
- `.github/workflows/publish.yml` - added a protected, serialized staging mutation job while retaining the unprivileged prerequisite.
- `internal/r2repo/` and `internal/cli/apply_sync.go` - added exact-prefix R2 hydration, ordered apply, deletion-last behavior, and remote verification.
- `internal/localrepo/` and CLI commands - added passphrase-file signing and the adapters needed by the staging workflow.
- `scripts/phase4-staging.sh` and `moon.yml` - added the real staging rehearsal, repeat no-op proof, public-key verification, and clean-client matrix.
- `scripts/check_workflow_policy.py`, documentation, and tests - revised the policy boundary and documented staging operations.
- Cloudflare - created `meigma-packages`, attached `pkgs.meigma.dev`, disabled `r2.dev`, and configured immutable/mutable cache rules.
- GitHub and 1Password - provisioned the `staging` environment, signing material, fingerprints, R2 configuration, and the prefix credential stored in the `Homelab` vault.

## Open Threads
- Renew the staging R2 credential or introduce an OIDC broker before `2027-07-18T19:51:37Z`.
- Positively verify the immutable production cache rule once Phase 5 publishes a production object; Phase 4 verified only the staging bypass behavior through live headers.
- Phase 5 owns GitHub Release discovery, canonical production registry entries, production publication, the first consumer dispatch, and production smoke/recovery operations.

## Lessons
- Moon tasks marked `runInCI: false` are hidden even when explicitly named in GitHub Actions; the privileged job must invoke the checked-in wrapper directly while policy keeps the Moon task unavailable to routine CI.
- Cloudflare's Temporary Credentials API capped issuance at seven days, while a locally signed token from the R2 parent successfully preserved the same bucket/prefix scope for one year.
- The fixture workflow regenerates disposable package bytes, so recovery runs can produce a new desired-state digest; selected versions, signatures, remote verification, no-op convergence, and clean installs are the relevant recovery evidence.

## References
- [PR #9](https://github.com/meigma/packages/pull/9)
- [Empty-prefix recovery run 29661991447](https://github.com/meigma/packages/actions/runs/29661991447)
- `.journal/001/DESIGN_PROPOSAL.md`
- `.journal/005/NOTES.md`
