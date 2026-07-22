---
title: Operations boundary
description: What the read-only workflows and protected publication paths can do.
---

# Operations boundary

Phase 3 proves the complete repository behavior that does not require external
provisioning. It keeps workflow orchestration thin and runs the same local
candidate and rebuild commands used by developers.

## Available workflows

`CI` runs on pull requests and pushes to `main`. It has read-only repository
access, receives no secrets, and runs affected Moon tasks. The current task
graph includes Go checks, documentation, signed fixture candidates, clean
package installation, deterministic rebuild/no-op proof, ordered sync-plan
fault coverage, and workflow policy enforcement.

`Publish validation` is manually dispatched with a registered project and
stable release tag. It validates the request and proves the real GitHub Release
source path without secrets.

`Rebuild validation` is manually dispatched with a fixture project. It
validates the project and runs the Phase 2 deterministic rebuild proof.

Without the explicit `apply_staging` input, the validation workflows change no
package repository state.

`Publish validation` can additionally run a manually selected, protected
`staging` job after its read-only validation succeeds. It imports only the
signing subkey, builds the verified `incus-gh-runner` `v1.0.0` release, applies
the ordered plan under `_staging/`, rehydrates and verifies R2, repeats the
publish as a no-op, and installs through the public hostname from clean Debian,
Ubuntu, and Fedora containers.

The optional staging recovery rehearsal requires both `empty_staging=true` and
the exact `empty _staging only` phrase. It empties only `_staging/`, verifies
that operation, and immediately rebuilds the prefix from GitHub Releases.

The protected `production` job runs only after staging succeeds and requires
the exact `publish incus-gh-runner v1.0.0 to production` phrase. It publishes at
the bucket root with separate credentials. Root sync deliberately excludes
`_staging/` from hydration, mutation, and verification, and rejects an
incomplete production candidate before making remote requests. A production
run may safely be a no-op. There is no operator selection that empties or
deletes production independently of the verified desired-state plan.

## Enforced safety boundary

All checked-in workflows currently require:

- empty top-level token permissions and no write permission;
- GitHub-hosted runners;
- full commit SHA pins for actions;
- checkout credential persistence disabled;
- secrets and deployment environments only inside dedicated manual staging and production jobs;
- no privileged pull-request triggers.

The policy is executable through `moon run root:workflow-check`. It permits
secrets only in those manual jobs, requires staging to depend on read-only
validation and production to depend on both validation and staging, and
continues to reject pull-request-derived privileged triggers, write
permissions, unpinned actions, and self-hosted runners.

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

The following remain deferred beyond the first staging slice:

- automated credential renewal or an OIDC broker;
- a general production release selector after the initial `v1.0.0` rehearsal;
- consumer repository dispatch.
