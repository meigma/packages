# meigma/packages

`meigma/packages` builds and publishes the signed APT and RPM repositories used
by Meigma projects. It builds verified candidates from fixture release sets,
applies deterministic retention, proves rebuild/no-op behavior, and executes
deletion-safe sync plans. The protected publication path signs with the CI
signing subkey, publishes a verified GitHub Release to either the isolated R2
`_staging/` prefix or the production root, verifies the remote result, and
installs through the public hostname. The initial production slice is confined
to `incus-gh-runner` `v1.0.0` while that boundary is rehearsed.

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
moon run root:workflow-check
moon run root:check
```

The aggregate `root:check` task also builds the local documentation. CI runs
the affected equivalent with `moon ci --summary minimal`. Workflow validation
uses pinned `actionlint` and ShellCheck versions plus a repository policy that
keeps routine jobs read-only, GitHub-hosted, and full-SHA pinned. Secrets and
deployment environments are allowed only in dedicated manual staging and
production jobs.

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
activation. This fixture proof remains local and secrets-free; the separate
Phase 5 proof covers GitHub Release discovery. Production R2 transport and
signing material remain later phases.

## Real release source proof

The opt-in Phase 5 proof downloads `incus-gh-runner` `v1.0.0` from its public
GitHub Release, verifies GitHub's asset digests and the pinned release-workflow
attestations, rebuilds the signed multi-architecture repositories, and performs
clean DEB and RPM installs on the current Docker architecture:

```sh
moon run root:phase5-source-proof
```

The source contract is checked into [`projects.yml`](projects.yml). The proof
uses GitHub and Docker but no publishing or production credentials. It does not
write to R2.

The entrypoint under `cmd/meigma-packages` remains thin. Cobra/Viper command
construction lives under `internal/cli`, with `MEIGMA_PACKAGES_*` reserved as
the environment-variable prefix for future configuration.

## Protected publication

The manual `Publish validation` and `Rebuild validation` workflows exercise the
same fixture-backed proofs on GitHub-hosted runners. They use
`meigma-packages validate-request` to reject unknown projects, unsafe project
names, and invalid stable release tags before invoking the local proof.

Both validation jobs are unprivileged. `Publish validation` can additionally
run the protected staging job with `apply_staging=true`. Selecting
`empty_staging=true` requires the exact `empty _staging only` confirmation and
rehearses recovery by emptying and rebuilding that prefix only.

Production is a second protected job that can run only after staging succeeds.
It requires `apply_production=true` and the exact
`publish incus-gh-runner v1.0.0 to production` confirmation. Root publication
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
