---
title: meigma-packages
slug: /
description: Build and operate Meigma APT and RPM repositories.
---

# meigma-packages

`meigma-packages` is the repository-local automation for building and
publishing signed APT and RPM repositories from approved Meigma GitHub
Releases.

The current secrets-free codebase builds and verifies signed local candidates
from fixture release sets, retains stable versions deterministically, verifies
same-input no-ops, rebuilds from an empty root, and plans ordered filesystem
changes with deletions last. GitHub Release discovery and real remote
publication remain deliberately deferred.

## Developer quick start

```sh
mise install
moon run root:check
moon run root:workflow-check
go run ./cmd/meigma-packages --help
```

The CLI is not distributed independently. Local development and GitHub Actions
run it from this repository.

## Current proof surface

- `root:phase1-proof` builds a signed APT/RPM candidate and installs its fixture
  package from a clean Debian container.
- `root:phase2-proof` proves retention, verified no-op behavior, empty-root
  rebuild equivalence, and deletion-safe sync planning.
- `root:workflow-check` validates workflow syntax, embedded and standalone
  shell, full-SHA action pins, read-only permissions, and the Phase 3
  no-secrets boundary.

The manually dispatched publish and rebuild validation workflows run these
local proofs on GitHub-hosted runners. They do not publish anything. See
[Operations boundary](operations.md) for the exact handoff into staging work.
