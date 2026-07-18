---
title: Operations boundary
description: What the read-only workflows and protected Phase 4 staging path can do.
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

`Publish validation` is manually dispatched with a fixture project and stable
release tag. It validates the request and runs the Phase 1 local candidate
proof.

`Rebuild validation` is manually dispatched with a fixture project. It
validates the project and runs the Phase 2 deterministic rebuild proof.

Without the explicit `apply_staging` input, the validation workflows change no
package repository state.

`Publish validation` can additionally run a manually selected, protected
`staging` job after its read-only validation succeeds. That job is the only
checked-in privileged boundary: it imports the signing-only subkey, builds the
same fixture candidate, applies the ordered plan under `_staging/`, rehydrates
and verifies R2, repeats the publish as a no-op, and installs through the public
hostname from clean Debian, Ubuntu, and Fedora containers.

## Enforced safety boundary

All checked-in workflows currently require:

- empty top-level token permissions and no write permission;
- GitHub-hosted runners;
- full commit SHA pins for actions;
- checkout credential persistence disabled;
- secrets and deployment environments only inside the dedicated manual staging job;
- no privileged pull-request triggers.

The policy is executable through `moon run root:workflow-check`. It permits
secrets only in the manual staging workflow, requires that job to depend on
read-only validation, and continues to reject pull-request-derived privileged
triggers, production configuration, write permissions, unpinned actions, and
self-hosted runners.

## Deferred after the first staging slice

The following remain deferred beyond the first staging slice:

- automated renewal or an OIDC broker before the prefix-scoped staging credential expires in July 2027;
- the final Cloudflare cache ruleset for immutable and mutable object classes;
- GitHub Release discovery and production registry entries;
- an explicit empty-prefix disaster-recovery rehearsal;
- production remote apply and deletion;
- consumer repository dispatch.

Production selection is not present in Phase 4. It remains behind a later,
separately reviewed protected-environment boundary.
