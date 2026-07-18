# meigma/packages

`meigma/packages` builds and publishes the signed APT and RPM repositories used
by Meigma projects. The repository currently contains the local CLI and
development foundation; package-repository behavior will be added in small,
proof-driven follow-up slices.

The `meigma-packages` binary is an implementation detail of this repository. It
is built and run from source in local development and GitHub Actions, and is not
published as a general-purpose release artifact.

## Local setup

Install [mise](https://mise.jdx.dev), then provision the pinned toolchain:

```sh
mise install
```

`mise.toml` selects Go, Moon, Python, uv, and golangci-lint. `mise.lock`
records their per-platform URLs and checksums, and locked mode fails closed when
a selected platform has no resolved artifact.

## Common tasks

Moon is the development and CI entrypoint:

```sh
moon run root:format
moon run root:lint
moon run root:build
moon run root:test
moon run root:check
```

The aggregate `root:check` task also builds the local documentation. CI runs
the affected equivalent with `moon ci --summary minimal`.

The CLI scaffold can be exercised directly:

```sh
go run ./cmd/meigma-packages --help
go run ./cmd/meigma-packages --version
```

The entrypoint under `cmd/meigma-packages` remains thin. Cobra/Viper command
construction lives under `internal/cli`, with `MEIGMA_PACKAGES_*` reserved as
the environment-variable prefix for future configuration.

## Documentation

Build the repository-local documentation with:

```sh
moon run docs:build
```

Serve it locally with:

```sh
moon run docs:serve
```

## Intentional exclusions

This repository does not inherit the template's Release Please, GoReleaser,
`ghd`, container-image, attestation, security-scan, or GitHub Pages publication
machinery. The CLI only supports this repository's automation and is not itself
a released product.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and
[SECURITY.md](SECURITY.md) for private vulnerability reporting.
