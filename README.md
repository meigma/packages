# meigma/packages

`meigma/packages` builds and publishes the signed APT and RPM repositories used
by Meigma projects. It builds verified candidates from GitHub Release assets,
applies deterministic retention, proves rebuild/no-op behavior, and executes
deletion-safe sync plans. The protected publication path signs with the CI
signing subkey, publishes a verified GitHub Release to either the isolated R2
`_staging/` prefix or the production root, verifies the remote result, and
installs through the public hostname. Publication is confined to registered
projects and exact stable `vX.Y.Z` releases whose GitHub assets, digests,
package identity, and provenance all verify independently.

The `meigma-packages` binary is an implementation detail of this repository. It
is built and run from source in local development and GitHub Actions, and is not
published as a general-purpose release artifact.

To install published software, follow the [APT and RPM installation
guide](docs/docs/install.md). The repository signing-key fingerprint is
`9C74476A669465EEB8D46AD8B0E68773B6E259F6`.

## Local setup

Install [mise](https://mise.jdx.dev), then provision the pinned toolchain:

```sh
mise install
```

`mise.toml` selects Go, Moon, Python, uv, golangci-lint, actionlint, and
ShellCheck. `mise.lock`
records their per-platform URLs and checksums, and locked mode fails closed when
a selected platform has no resolved artifact.

## Common tasks

Moon is the development and CI entrypoint:

```sh
moon run root:format
moon run root:lint
moon run root:build
moon run root:test
moon run root:workflow-check
moon run root:check
```

The aggregate `root:check` task also builds the local documentation. CI runs
the affected equivalent with `moon ci --summary minimal`. Workflow validation
uses pinned `actionlint` and ShellCheck versions. Secrets and deployment
environments are allowed only in the protected staging and production jobs.

The CLI scaffold can be exercised directly:

```sh
go run ./cmd/meigma-packages --help
go run ./cmd/meigma-packages --version
```

The entrypoint under `cmd/meigma-packages` remains thin. Cobra/Viper command
construction lives under `internal/cli`. `MEIGMA_PACKAGES_*` is the live
environment-variable prefix: `MEIGMA_PACKAGES_GITHUB_TOKEN` authenticates
`fetch-release`, and `MEIGMA_PACKAGES_R2_ACCESS_KEY_ID`,
`MEIGMA_PACKAGES_R2_SECRET_ACCESS_KEY`, and
`MEIGMA_PACKAGES_R2_SESSION_TOKEN` credential `apply-sync`.

## Publication

The `Publish` workflow runs on a trusted `publish-package` repository dispatch
from a registered consumer, or manually with a registered `project` and exact
stable `vX.Y.Z` `tag`. An unprivileged job validates the request against
[`projects.yml`](projects.yml) first; prereleases, build metadata, missing or
repeated prefixes, and leading-zero variants fail closed.

The protected `staging` and `production` jobs then run `scripts/publish.sh`:
download the release from GitHub, verify asset digests and pinned
release-workflow attestations, rebuild the signed multi-architecture APT/RPM
tree, apply the ordered sync plan to R2, verify the remote bytes, repeat the
publish as a no-op, and install the package from the public URL on clean
Debian, Ubuntu, and Fedora containers.

A trusted dispatch always publishes through staging to production. Manual runs
select `apply_staging` and `apply_production` explicitly; production only runs
after staging succeeds. Immutable package and content-addressed metadata
objects receive a one-year immutable cache policy; activation metadata, state,
keys, and repo configuration remain `no-store`. Production deletion is not an
operator mode. See [Publish a release](docs/docs/publishing.md) for the
registry contract and workflow inputs, and the
[operations boundary](docs/docs/operations.md) before dispatching it.

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
