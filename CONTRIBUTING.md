# Contributing

Keep changes small and proof-driven. Package tooling has several external
format boundaries, so prefer a narrow fixture-backed experiment before adding
durable abstractions or expanding the command surface.

For private vulnerability reporting, use [SECURITY.md](SECURITY.md) instead of
public channels.

## Pull requests

Contributors should:

1. Keep each change focused on one problem or proof.
2. Add or update tests when behavior changes.
3. Update documentation when user-facing behavior changes.
4. Use Conventional Commit subjects.
5. Make sure `moon run root:check` passes before requesting review.

## Local setup

```sh
mise install
moon run root:check
```

Useful focused commands:

```sh
moon run root:format
moon run root:lint
moon run root:build
moon run root:test
moon run docs:build
go run ./cmd/meigma-packages --help
```

The CLI is repository-local and has no independent release process. Changes
land through reviewed pull requests and are consumed from the repository
checkout by local development and GitHub Actions.
