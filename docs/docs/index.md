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

## Publication

The `Publish` workflow validates the requested project and tag without
secrets, then hands off to the protected staging and production jobs, which
run `scripts/publish.sh` to rebuild, sign, apply, verify, and clean-install
the release. Manual runs select `apply_staging` and `apply_production`
explicitly; a trusted consumer dispatch carrying `project` and `tag` always
publishes through staging to production. See [Install packages](install.md)
for the public repository commands and [Operations boundary](operations.md)
for the enforced scope and recovery behavior.
