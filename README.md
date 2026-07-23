# meigma/packages

`meigma/packages` builds and publishes the signed APT and RPM repositories used
by Meigma projects. It builds verified candidates from fixture release sets,
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

## Real release source proof

The opt-in Phase 5 proof downloads the selected registered project's exact
stable release from GitHub, verifies GitHub's asset digests and the pinned
release-workflow attestations, rebuilds the signed multi-architecture
repositories, and performs clean DEB and RPM installs on the current Docker
architecture:

```sh
PROJECT=incus-gh-runner TAG=v1.1.0 moon run root:phase5-source-proof
```

`PROJECT` and `TAG` are required, so another registered exact stable release
uses the same proof without changing the script. The proof derives the expected
package version by removing exactly one leading `v` from the validated tag and
requires both package formats to report it.

The source contract is checked into [`projects.yml`](projects.yml). The proof
uses GitHub and Docker but no publishing or production credentials. It does not
write to R2.

The entrypoint under `cmd/meigma-packages` remains thin. Cobra/Viper command
construction lives under `internal/cli`, with `MEIGMA_PACKAGES_*` reserved as
the environment-variable prefix for future configuration.

## Protected publication

The `Publish validation` and manual `Rebuild validation` workflows exercise the
same fixture-backed proofs on GitHub-hosted runners. They use
`meigma-packages validate-request` to reject unknown projects, unsafe project
names, and invalid stable release tags before invoking the local proof.
Accepted publish tags have exactly the `vX.Y.Z` shape; prereleases, build
metadata, missing or repeated prefixes, and leading-zero variants fail closed.

`Publish validation` also accepts the exact `publish-package` repository
dispatch event from a trusted consumer. Its payload is restricted to `project`
and `tag`; a valid dispatch always enters protected staging and production and
cannot request staging deletion.

Both validation jobs are unprivileged. `Publish validation` can additionally
run the protected staging job with `apply_staging=true`. Selecting
`empty_staging=true` requires the exact `empty _staging only` confirmation and
rehearses recovery by emptying and rebuilding that prefix only.

Production is a second protected job that can run only after staging succeeds.
It requires `apply_production=true` and the exact confirmation derived from the
validated selection: `publish <project> <tag> to production`. Trusted dispatch
synthesizes that phrase internally; manual runs must supply it exactly. Root publication
preserves every `_staging/` object and rejects incomplete candidates before
remote access. Immutable package and content-addressed metadata objects receive
a one-year immutable cache policy; activation metadata, state, keys, and repo
configuration remain `no-store`. Production deletion is not an operator mode.
See the [operations boundary](docs/docs/operations.md) before dispatching it.

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
