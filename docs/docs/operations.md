---
title: Operations boundary
description: What the read-only workflows and protected publication paths can do.
---

# Operations boundary

## Available workflows

`CI` runs on pull requests and pushes to `main`. It has read-only repository
access, receives no secrets, and runs affected Moon tasks: Go checks,
documentation, and workflow linting.

`Publish` accepts either a manual request or the exact `publish-package`
repository dispatch event. Both paths first validate the registered project
and exact stable `vX.Y.Z` release tag in an unprivileged job without secrets.

Manual requests change no package repository state without the explicit
`apply_staging` and `apply_production` inputs. A trusted consumer dispatch
always continues through protected staging and production after validation; it
carries only `project` and `tag` and cannot skip staging or choose an R2
prefix.

The protected `staging` job imports only the signing subkey, independently
revalidates and builds the selected registered release, applies the ordered
plan under the staging prefix, rehydrates and verifies R2, repeats the publish
as a no-op, and installs the package version derived from the validated tag
through the public hostname from clean Debian, Ubuntu, and Fedora containers.
Because the applied sync plan converges on the verified desired state
(deletions last, verification before completion), recovering staging from any
bad or empty state is just another staging publish.

The protected `production` job runs only after staging succeeds. Production
publishes at the bucket root with separate credentials. Root sync deliberately
excludes `_staging/` from hydration, mutation, and verification, and rejects
an incomplete production candidate before making remote requests. A production
run may safely be a no-op. There is no operator selection that empties or
deletes production independently of the verified desired-state plan.

## Enforced safety boundary

All checked-in workflows keep:

- empty top-level token permissions and no write permission;
- GitHub-hosted runners;
- full commit SHA pins for actions;
- checkout credential persistence disabled;
- secrets and deployment environments only inside the protected staging and
  production jobs;
- no privileged pull-request triggers.

These properties are maintained through code review; `moon run
root:workflow-check` lints workflow syntax and shell.

## Cache and credential boundary

Production package payloads, APT by-hash objects, and checksum-named RPM
metadata are published as immutable for one year. Mutable activation metadata
such as `InRelease`, `Release`, `Packages`, `repomd.xml`, repository
configuration, keys, and logical state use `no-store`. Staging remains
`no-store` throughout.

Staging and production use separate R2 credentials. The staging credential is
confined to `_staging/`; the production credential is bucket-wide because the
root repository spans multiple prefixes. Both jobs receive only the signing
subkey, never the primary signing key.

## Deferred follow-up

The following remains deferred beyond the generalized production slice:

- automated credential renewal or an OIDC broker;
- onboarding additional projects through the existing registry contract.
