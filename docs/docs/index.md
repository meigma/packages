---
title: meigma-packages
slug: /
description: Build and operate Meigma APT and RPM repositories.
---

# meigma-packages

`meigma-packages` is the repository-local automation for building and
publishing signed APT and RPM repositories from approved Meigma GitHub
Releases.

The current codebase intentionally provides only the CLI and development
foundation. Follow-up sessions will add the registry, release discovery,
repository generation, verification, and publication behavior as separately
testable vertical slices.

## Developer quick start

```sh
mise install
moon run root:check
go run ./cmd/meigma-packages --help
```

The CLI is not distributed independently. Local development and GitHub Actions
run it from this repository.
