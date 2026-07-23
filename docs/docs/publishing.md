---
title: Publish a release
description: Register a project in projects.yml and run the Publish workflow.
---

# Publish a release

Publication has two halves: a one-time registration of the project in
[`projects.yml`](https://github.com/meigma/packages/blob/main/projects.yml),
and a per-release run of the `Publish` workflow. Registered projects publish
automatically when their release workflow dispatches to this repository;
manual runs use the same path.

## Register a project

Add an entry under `projects:` in `projects.yml`. The registry is strict YAML
(unknown fields are rejected) with `schema: 1` at the top level. The existing
entry is a complete example:

```yaml
schema: 1

defaults:
  retention: 5

projects:
  incus-gh-runner:
    repository: meigma/incus-gh-runner
    package_name: incus-gh-runner
    assets:
      checksums: checksums.txt
      deb: 'incus-gh-runner_${version}_*.deb'
      rpm: 'incus-gh-runner-${version}-1.*.rpm'
    architectures:
      amd64:
        deb: amd64
        rpm: x86_64
      arm64:
        deb: arm64
        rpm: aarch64
    provenance:
      signer_workflow: meigma/incus-gh-runner/.github/workflows/attest.yml
```

Field by field:

- **Project key** — lowercase letters, digits, and single hyphens
  (`^[a-z0-9]+(-[a-z0-9]+)*$`). This key becomes the project's published
  location: the APT component and the `rpm/<project>/` path (see
  [layout](#published-layout) below).
- **`repository`** (required) — the `meigma/<name>` GitHub repository whose
  Releases are the authoritative package source. Only `meigma/`-owned
  repositories are accepted.
- **`package_name`** (required) — the package identity that every DEB and RPM
  in the release must report in its metadata, and the name users pass to
  `apt install` / `dnf install`.
- **`retention`** (optional) — how many stable versions the published
  repository keeps. Falls back to `defaults.retention`, then to 5.
- **`assets.checksums`** (required) — the exact checksum asset name in the
  GitHub Release.
- **`assets.deb`** / **`assets.rpm`** (required) — bare file-name glob
  patterns (no paths) that select the release's package assets.
  `${version}` expands to the tag with its leading `v` removed, so tag
  `v1.1.0` selects `incus-gh-runner_1.1.0_*.deb`.
- **`architectures`** (at least one) — maps each repository architecture key
  to the architecture strings recorded inside the DEB and RPM packages
  (`amd64`/`x86_64`, `arm64`/`aarch64`). Every architecture needs both
  mappings, and no two architectures may share one. The release must contain
  exactly one DEB and one RPM per registered architecture.
- **`provenance.signer_workflow`** (required) — the exact workflow inside
  `repository` (`<repository>/.github/workflows/<file>.yml`) trusted to have
  produced the release's artifact attestations. Publication verifies every
  package asset's attestation against this workflow.

What the source repository must therefore ship in each release:

- an exact stable `vX.Y.Z` tag (no prereleases, build metadata, or
  leading-zero components);
- the checksum asset plus one DEB and one RPM per registered architecture,
  matching the asset patterns;
- GitHub artifact attestations from the pinned signer workflow;
- package metadata whose name is `package_name` and whose version is the tag
  without its leading `v` — clean installs assert this exact version.

Asset digests are verified against GitHub's recorded SHA-256 values after
download; any mismatch, missing architecture, or failed attestation fails the
run before anything is published.

## Published layout

The project key determines where packages land under
`https://pkgs.meigma.dev`:

- APT: suite `stable`, component `<project>` — consumers put the project key
  in the `Components:` line of their sources entry.
- RPM: repository definition at `rpm/<project>/meigma.repo`, with per-arch
  package trees below it.
- `/_state/manifest.json` lists every published project, selected version,
  and package, and is the quickest way to see what is currently live.

## Run a publish

### Automatic (trusted dispatch)

A registered project's release workflow dispatches to this repository after
its GitHub Release is published. It mints a short-lived token from the private
Meigma GitHub App (installed only on `meigma/packages` with
`Contents: write`) via `actions/create-github-app-token`, then sends:

```sh
gh api \
  --method POST \
  repos/meigma/packages/dispatches \
  --raw-field event_type=publish-package \
  --raw-field 'client_payload[project]=<project>' \
  --raw-field "client_payload[tag]=$PACKAGE_TAG"
```

The payload carries only `project` and `tag`. A trusted dispatch always runs
validation, then protected staging, then protected production; it cannot skip
staging or select any other behavior. See `meigma/incus-gh-runner`'s
`.github/workflows/packages.yml` for the complete working example, including
the stable-tag guard and app-token step.

### Manual

Run the `Publish` workflow from the Actions tab or the CLI:

```sh
# Validation only (no secrets, nothing published)
gh workflow run publish.yml -f project=<project> -f tag=vX.Y.Z

# Staging-only rehearsal
gh workflow run publish.yml -f project=<project> -f tag=vX.Y.Z \
  -f apply_staging=true

# Full publish through staging to production
gh workflow run publish.yml -f project=<project> -f tag=vX.Y.Z \
  -f apply_staging=true -f apply_production=true
```

`apply_production` only takes effect after staging succeeds in the same run;
there is no staging bypass. Every publish is convergent: re-running the same
release is a verified no-op, and recovering staging from a bad or empty state
is just another staging publish. See the
[operations boundary](operations.md) for what the protected jobs can and
cannot do.
