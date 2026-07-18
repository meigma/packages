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

## Local candidate proof

The Phase 1 vertical slice builds fixture release assets into a signed and
verified APT/RPM candidate tree, then installs the fixture from a clean Debian
container:

```sh
moon run root:phase1-proof
```

The proof uses only Docker and the pinned local toolchain. It creates a
throwaway signing key and temporary output, invokes the durable
`meigma-packages build-local` command, and removes all generated state when it
finishes. It does not access GitHub Releases, R2, or production credentials.

## Deterministic rebuild proof

The Phase 2 slice adds fixture release-set validation, semantic-version
retention, checksum and package-metadata inspection, logical state manifests,
verified same-input no-ops, and ordered filesystem sync planning:

```sh
moon run root:phase2-proof
```

The proof builds three fixture releases, retains the newest two, verifies that
the same input is a no-op, rebuilds the same logical tree from an empty root,
and confirms that every planned deletion follows content and metadata
activation. It remains local and secrets-free; GitHub Release discovery, R2
transport, and production signing material are later phases.

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

## License

`meigma/packages` is dual-licensed under either of:

- [Apache License, Version 2.0](LICENSE-APACHE)
- [MIT License](LICENSE-MIT)

at your option. See [LICENSE](LICENSE) for the dual-license notice and
contribution terms.
