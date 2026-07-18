---
title: Operations boundary
description: What the Phase 3 workflows prove and what remains deliberately disabled.
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

The validation workflows are intentionally named for the path they exercise,
not for an external side effect. A successful run changes no package repository
state.

## Enforced safety boundary

All checked-in workflows currently require:

- empty top-level token permissions and no write permission;
- GitHub-hosted runners;
- full commit SHA pins for actions;
- checkout credential persistence disabled;
- no secrets or deployment environments;
- no privileged pull-request triggers.

The policy is executable through `moon run root:workflow-check`. Phase 4 must
change that policy deliberately and review the privileged job boundary rather
than quietly adding credentials to an existing unprivileged job.

## Deferred until staging

The following are not configured in Phase 3:

- GitHub Release discovery and download;
- Cloudflare R2 credentials or object transport;
- production or staging signing material;
- protected deployment environments;
- remote apply, deletion, or public-host verification;
- consumer repository dispatch.

Before staging rehearsal, provision the external resources, introduce a
separate privileged mutation job, preserve read-only validation before that
job, and keep production selection behind explicit environment protection.
