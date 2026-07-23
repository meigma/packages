---
title: meigma-packages
slug: /
description: Build and operate Meigma APT and RPM repositories.
---

# meigma-packages

`meigma-packages` is the repository-local automation for building and
publishing signed APT and RPM repositories from approved Meigma GitHub
Releases.

The current codebase builds and verifies signed candidates, retains stable
versions deterministically, verifies same-input no-ops, rebuilds from an empty
root, and plans ordered remote changes with deletions last. The canonical
registry discovers and verifies GitHub Release packages. Protected staging and
production jobs publish registered exact stable `vX.Y.Z` releases.

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
- `root:phase5-source-proof` requires a registered project and exact stable tag
  through `PROJECT` and `TAG`, then performs clean DEB and RPM installs without
  publishing; `incus-gh-runner` `v1.1.0` is the current exercised example.
- `root:phase5-publish` is the secret-bearing protected entrypoint used to
  publish, verify, repeat as a no-op, and install from staging or production.
- `root:workflow-check` validates workflow syntax, embedded and standalone
  shell, full-SHA action pins, read-only permissions, and the Phase 3
  no-secrets boundary.

The publish workflow validates without secrets, then hands off to protected
staging and production. Manual runs require explicit inputs and exact
confirmation phrases derived from the validated project and tag. A trusted
consumer dispatch must contain exactly `project` and `tag`; it cannot select
deletion behavior, staging bypass, confirmation text, or R2 targeting. See [Install
packages](install.md) for the public repository commands and [Operations
boundary](operations.md) for the enforced scope and recovery behavior.
