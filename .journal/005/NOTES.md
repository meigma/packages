---
id: 005
title: Phase 4 staging publication rehearsal
started: 2026-07-18
---

## 2026-07-18 11:07 — Kickoff
Goal for the session: Familiarize with session 001's planning artifacts and begin with Phase 4.
Current state of the world: Phases 0 through 3 are complete on `main`; the repository has verified local candidate building, deterministic rebuild and sync planning, and unprivileged validation workflows, while Phase 4 external staging prerequisites and privileged mutation remain outstanding.
Plan: Reconcile the original Phase 4 plan with the merged implementation and current external prerequisites, then advance through the smallest staging mutation and rehearsal slice with proof gates guiding later details.

## 2026-07-18 11:09 — Phase 4 orientation
Read session 001's design proposal, summary, and running notes, then reconciled them with the closed Phase 2–4 handoffs and `main` at `889358b0bec03a243af54cd261e932836033ecb6`. The Phase 4 gate remains a real `_staging/` rehearsal through R2 and public HTTP: clean-client installs, repeat publish as a verified no-op, and rebuild from an empty staging prefix without touching production.

The current implementation deliberately stops before that boundary. Publish and rebuild workflows are manual fixture-backed validation jobs; the CLI can build and verify local fixture candidates and plan ordered filesystem changes, but it has no GitHub Release adapter, R2 transport/apply command, canonical root `projects.yml`, remote verification, or privileged job. Live GitHub repository metadata currently shows zero deployment environments, Actions secrets, and Actions variables, so the documented Josh-owned provisioning prerequisites have not yet been reflected in `meigma/packages`.

The agile starting sequence should keep the Phase 3 read-only validation job intact, provision the staging environment and credentials, and add a separate privileged staging mutation path. The first implementation proof should be the narrowest real `_staging/` upload/verify slice behind that boundary; release discovery, no-op rehearsal, empty-prefix rebuild, deletion, and full clean-client coverage can then be added as evidence-driven increments within Phase 4. Production paths and consumer dispatch remain Phase 5.

## 2026-07-18 11:32 — Confirmed Cloudflare automation surface
Loaded the installed Cloudflare platform skill and inspected its authenticated API capabilities without mutating external state. The connector can create and configure R2 buckets, attach or configure bucket custom domains, enable or disable the managed `r2.dev` endpoint, operate objects, create scoped account API tokens or temporary credentials, and manage zone rulesets for cache policy. Read-only API calls succeeded for the account's R2 bucket list and confirmed the `meigma.dev` zone is active. Cloudflare-side Phase 4 provisioning can therefore be performed through the plugin after explicit authorization; GPG material and GitHub environment configuration remain separate surfaces.

## 2026-07-18 11:35 — Confirmed secret-safe GPG and GitHub path
Verified local GnuPG 2.4.9, 1Password CLI 2.32.0, and GitHub CLI 2.94.0. GitHub is authenticated as `jmgilman` with admin access to `meigma/packages`; `gh secret set` accepts stdin and encrypts values locally, so a signing-only export and passphrase can be installed into a protected staging environment without appearing in command arguments or chat output. `op item create` and `op document create` also accept stdin, allowing the primary-key backup, CI subkey export, and passphrase to flow directly into 1Password from a mode-0700 temporary GPG home. The configured 1Password account is not currently signed in and has no service-account token, so Josh must unlock/authenticate the CLI before execution. The primary key will remain out of GitHub; only the signing-only subkey export and passphrase belong in the staging environment.

## 2026-07-18 12:10 — Provisioned the staging boundary
Created the Cloudflare R2 bucket `meigma-packages`, attached the active custom domain `pkgs.meigma.dev`, required TLS 1.2 or newer, and disabled the managed `r2.dev` endpoint. Created the GitHub `staging` environment with custom branch policies limited to `main` and `feat/phase4-staging`.

Generated the production repository key in a mode-0700 temporary GnuPG home: Ed25519 certify-only primary `9C74476A669465EEB8D46AD8B0E68773B6E259F6` and Ed25519 signing subkey `9DA41FD9DBD38B19AC75454D27CCA9E924245272`, both non-expiring. Stored the passphrase, primary/signing backup, CI signing-only export, and public key in the 1Password `Homelab` vault, then deleted the temporary keyring. GitHub received only the signing-only export and passphrase as `staging` environment secrets plus the public fingerprints as variables.

The existing account-wide R2 credential in 1Password was deliberately not copied to GitHub. Used it to mint a seven-day temporary credential limited to object read/write under `meigma-packages/_staging/`; verified that `_staging/` access succeeds and `_state/` access is denied. Stored that credential in `Homelab` and installed it in the GitHub environment. It expires at `2026-07-25T18:50:20Z` and must be replaced by a durable least-privilege credential before Phase 4 closes.

Cloudflare cache-rule provisioning remains externally blocked. Both the authenticated connector and the existing 1Password Cloudflare API token return authorization failures for zone rulesets, so no cache rule was partially installed. A token with Cache Rules Edit or a manual dashboard change is still required.

## 2026-07-18 12:19 — Proved the first real staging slice
Implemented the work on `feat/phase4-staging` from `main` SHA `889358b0bec03a243af54cd261e932836033ecb6`. The CLI now hydrates an exact R2 prefix, reuses the Phase 2 content/index/activation/state/delete plan, uploads staging objects with `Cache-Control: no-store`, deletes only after activation and state, rehydrates, and byte-compares the result. Prefix and endpoint validation fail closed. GPG signing accepts a mode-0600 passphrase file so the protected passphrase never enters the process arguments, and rejects files exposed to group or others.

The local live rehearsal used the signing-only subkey and temporary prefix credential to publish 21 ordered actions under `_staging/`, rehydrate and verify R2, repeat as a verified zero-action no-op, verify the public-key fingerprint, and clean-install the newest retained fixture through `https://pkgs.meigma.dev/_staging` on Debian 13, Ubuntu 26.04, and Fedora 44. The aggregate `mise exec -- moon run root:check --summary minimal` gate passed.

Opened PR #9, `feat(publish): add protected R2 staging rehearsal`. The first hosted dispatch proved the read-only prerequisite but exposed that Moon hides `runInCI: false` tasks even when explicitly named; it failed before mutation. Kept that exclusion so routine CI cannot select the privileged task and changed the protected job to invoke the same checked-in rehearsal script directly. Hosted run `29657434585` then passed on `44b0b02fb449da78673b3eca20ec4fb370f7f000`: 28 ordered cross-architecture reconciliation actions, remote verification, repeat no-op, and clean installs. PR CI and Kusari Inspector are green.

This is a reviewable Phase 4 increment, not Phase 4 completion. Remaining gates are a durable least-privilege R2 credential, the final Cloudflare immutable/mutable cache rules, and an explicit empty-prefix disaster-recovery rehearsal. GitHub Release discovery, canonical production registry entries, production publication, and consumer dispatch remain outside this slice.
