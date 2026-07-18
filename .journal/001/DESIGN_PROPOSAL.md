# Meigma Packages Repository Design Proposal

- Status: Proposal for review
- Session: 001
- Date: 2026-07-17
- Revision: Use one Meigma GitHub App for cross-repository dispatch
- Source contract: `meigma/packages` jumpstart supplied for this session

## 1. Executive summary

`meigma/packages` will publish package-manager metadata around `.deb` and `.rpm`
assets that already exist in Meigma project GitHub Releases. GitHub Releases
remain the durable artifact source. Cloudflare R2 contains a disposable,
rebuildable projection served exclusively through `https://pkgs.meigma.dev`.

The core of the repository should be a small, testable command-line program that
builds and validates a complete candidate repository tree in a local directory.
GitHub Actions workflows should remain thin orchestration layers around that
same program. Incremental publish and full rebuild differ only in how they
select source releases and initialize the candidate tree; metadata generation,
signing, validation, and remote application are shared.

The work should proceed through vertical proofs rather than a complete
up-front implementation. The first proof should generate and verify one APT
repository and one RPM repository locally using fixture packages and a
throwaway key. It should also simulate interruption at each publication stage.
Only after that proof settles the real tool behavior should the durable command
structure and workflows be filled in.

The design preserves the jumpstart's adopted defaults:

- one hostname: `pkgs.meigma.dev`;
- APT under `/apt`, with suite `stable` and one component per project;
- RPM under `/rpm/<project>`;
- metadata signatures in v1, package signatures later;
- five retained versions by default;
- one private Meigma GitHub App for short-lived cross-repository dispatch;
- daily and post-publish smoke tests;
- no new paid service or account without approval.

## 2. Goals and boundaries

### 2.1 Goals

1. Make clean Debian, Ubuntu, Fedora, and RHEL-family installations possible
   using only `pkgs.meigma.dev` URLs.
2. Sign all repository metadata with a CI-held signing-only subkey whose primary
   key remains offline.
3. Make publish idempotent and serialize every production mutation.
4. Make the complete public tree reconstructable from GitHub Releases,
   `projects.yml`, the public key, and repository code.
5. Reduce onboarding to one registry entry plus a standard dispatch step in the
   consumer repository.
6. Exercise the real build/sign/verify path locally and in pull-request CI
   without R2 or production secrets.
7. Detect broken public metadata or installation paths within one publish run
   and again through a daily smoke test.

### 2.2 Non-goals

- private packages;
- package-level RPM or DEB signatures in v1;
- Windows or macOS distribution;
- COPR, PPA, OBS, or other distro-native channels;
- PAT-based dispatch authentication;
- building project binaries or replacing GoReleaser/nfpm;
- using R2 as the artifact of record;
- a package repository web application, database, or control plane.

## 3. Design principles

### 3.1 Release assets are immutable inputs

The publisher never modifies a source release and never builds a project
binary. It resolves a published GitHub Release, downloads only registry-matched
assets, verifies their checksums and package metadata, and projects them into
the repository tree.

### 3.2 Build a candidate before touching remote state

Every operation constructs a complete candidate tree in an isolated local
directory. Metadata and signatures are verified against that directory before
any R2 write begins. A failed candidate build leaves the bucket untouched.

### 3.3 One engine, several entry points

Local fixture tests, incremental publish, rebuild, and staging rehearsal use the
same candidate builder and validators. Workflows select inputs and credentials;
they do not reimplement package logic in YAML.

### 3.4 Prefer deterministic decisions over historical mutation

The desired package set is derived from the registry and qualifying GitHub
Releases each time. Retention is a deterministic selection rule, not a chain of
imperative deletes. A small generated state manifest records why the current
tree exists but is never the sole source of truth.

### 3.5 Prove uncertain tool behavior early

APT by-hash publication, RPM metadata/signature activation, CDN cache behavior,
and Ed25519 compatibility are proof gates. The proposal does not pretend those
details are settled merely because the surrounding architecture is settled.

## 4. System context

```text
meigma/<project> release workflow
    |
    | repository_dispatch: publish-package
    | { project, tag }
    v
meigma/packages
    |-- projects.yml validates and resolves the project
    |-- GitHub Release adapter downloads assets and checksums
    |-- candidate builder applies retention and creates metadata
    |-- signer creates APT and RPM metadata signatures
    |-- verifier proves the candidate locally
    |-- publisher applies the verified tree to R2 in ordered stages
    v
R2 bucket: meigma-packages
    |
    v
https://pkgs.meigma.dev/{apt,rpm,...}
    |
    +-- APT clients
    +-- DNF/YUM clients
    +-- scheduled/post-publish smoke tests
```

### 4.1 Public object tree

```text
pkgs.meigma.dev/
├── index.html
├── meigma.asc
├── apt/
│   ├── dists/stable/**
│   └── pool/<project>/*.deb
├── rpm/<project>/**
├── _state/manifest.json
└── _staging/**
```

`index.html` is a small hand-authored/generated landing page explaining the
service and linking to installation and project documentation; R2 public
buckets do not provide a root object listing. `meigma.asc` is the stable public
key URL. `_state/` and `_staging/` are operational paths, not user-facing
contracts. No generated content or documentation exposes an R2 vendor
hostname.

### 4.2 User install contract

The first consumer's published documentation must preserve these interfaces:

```sh
# Debian/Ubuntu
curl -fsSL https://pkgs.meigma.dev/meigma.asc | sudo tee /etc/apt/keyrings/meigma.asc >/dev/null
sudo tee /etc/apt/sources.list.d/meigma.sources <<'EOF'
Types: deb
URIs: https://pkgs.meigma.dev/apt
Suites: stable
Components: incus-gh-runner
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
sudo apt update && sudo apt install incus-gh-runner

# Fedora/RHEL-family
sudo dnf config-manager addrepo --from-repofile=https://pkgs.meigma.dev/rpm/incus-gh-runner/meigma.repo
sudo dnf install incus-gh-runner
```

Future project docs substitute only the registry-derived component/project and
package names. They do not introduce vendor domains or verification bypasses.

## 5. Proposed repository shape

```text
.
├── .github/
│   ├── CODEOWNERS
│   ├── dependabot.yml
│   └── workflows/
│       ├── ci.yml
│       ├── publish.yml
│       ├── rebuild.yml
│       └── smoke-test.yml
├── cmd/meigma-packages/          # CLI entrypoint
├── internal/
│   ├── registry/                 # projects.yml parsing and validation
│   ├── releases/                 # GitHub Release discovery/download contract
│   ├── packages/                 # DEB/RPM inspection and normalized records
│   ├── retention/                # version selection
│   ├── aptrepo/                  # APT layout and metadata generation
│   ├── rpmrepo/                  # RPM layout and metadata generation
│   ├── signing/                  # GPG process boundary
│   ├── publish/                  # candidate plan and ordered application
│   └── verify/                   # offline and remote checks
├── scripts/                      # thin developer/CI wrappers, not business logic
├── testdata/
│   ├── nfpm/                     # tiny fixture package definitions
│   └── releases/                 # fixture GitHub Release-shaped inputs
├── docs/
│   ├── onboarding.md
│   ├── install.md
│   └── signing-runbook.md
├── projects.yml
├── go.mod
└── README.md
```

The exact internal package split should follow the first vertical proof rather
than precede it. The durable boundary is more important than the folder names:
typed orchestration in Go, external package tools behind narrow process
interfaces, and workflows containing no repository-generation logic.

### Why Go for the durable implementation

Go gives the registry, asset validation, retention, planning, filesystem
transforms, and tests a typed and portable home. The implementation should
shell out only where mature ecosystem tools already define the format:

- `apt-ftparchive` for APT package indexes and Release metadata;
- `createrepo_c` for RPM repository metadata;
- `gpg`/`gpgv` for signing and verification;
- `rclone` for R2 transport;
- `gh` for GitHub Release access in CI and operator workflows.

A disposable Bash spike is appropriate for proving the exact commands. Once
the commands work against fixtures, retain only thin wrappers and move stable
policy into Go. This avoids designing abstractions around guessed CLI behavior.

## 6. Project registry

`projects.yml` is the only onboarding registry. A proposed initial shape is:

```yaml
schema: 1

defaults:
  retention: 5

projects:
  incus-gh-runner:
    repository: meigma/incus-gh-runner
    package_name: incus-gh-runner
    retention: 5
    assets:
      checksums: checksums.txt
      deb: "incus-gh-runner_*_linux_*.deb"
      rpm: "incus-gh-runner_*_linux_*.rpm"
    architectures:
      amd64:
        deb: amd64
        rpm: x86_64
      arm64:
        deb: arm64
        rpm: aarch64
```

The asset patterns above are illustrative until checked against a real
`incus-gh-runner` release. That check belongs in the first consumer-integration
slice, not in speculative schema work.

Registry validation must reject:

- unsupported schema versions;
- duplicate project, repository, or package names;
- repositories outside the `meigma` owner;
- absolute paths, path traversal, or path separators in project/package names;
- unsupported architectures;
- non-positive retention;
- patterns that match zero, multiple, or unexpected assets for a required
  format/architecture;
- unknown fields, so misspellings fail closed.

Registry entries are sorted by project key when rendered into metadata and
documentation. Unknown dispatch projects are rejected before the workflow
enters an environment or receives R2/signing secrets.

## 7. Source release contract

A qualifying release is proposed to mean:

- owner is `meigma` and repository exactly matches the registry;
- release is published, not draft, and not a prerelease;
- tag is a valid `v`-prefixed semantic version;
- every required architecture has exactly one matching DEB and RPM asset;
- the configured checksum asset exists and covers every downloaded package;
- the package's internal name, version, and architecture match the registry,
  tag, and expected architecture mapping.

The same retained amd64/arm64 package assets serve every supported distribution
family. The repository does not create Debian-version, Ubuntu-version, Fedora,
or RHEL-specific copies because the nfpm packages contain static Go binaries
and have no distro-specific build contract.

The publisher should inspect packages using `dpkg-deb --field` and `rpm -qp`,
not infer identity solely from filenames. It should reject symlinks, device
files, path traversal, duplicate checksums, checksum mismatches, and unexpected
matched files.

`repository_dispatch` is an instruction to consider a release, not trusted
proof that the release is valid. Both `project` and `tag` are untrusted input
until registry lookup and GitHub Release inspection succeed.

## 8. Candidate-tree model and idempotency

### 8.1 Operation modes

The CLI should expose a small operational surface, for example:

```text
meigma-packages validate-config
meigma-packages publish --project <name> --tag <tag> --root <dir>
meigma-packages rebuild --root <dir>
meigma-packages verify --root <dir>
meigma-packages plan-sync --root <dir> --remote <remote>
meigma-packages apply-sync --root <dir> --remote <remote>
```

Names may change after the spike. Required behavior should not:

- local mode writes only to a caller-provided directory;
- production remote mutation is opt-in and intended for GitHub Actions;
- commands emit a machine-readable JSON result alongside human-readable logs;
- a sync plan lists creates, replacements, and deletions before application;
- no command logs secret material.

### 8.2 Generated state manifest

The tree should contain an internal, reconstructable manifest such as
`_state/manifest.json` with:

- schema version;
- registry digest;
- signing-key fingerprint;
- selected project/tag/package records and SHA-256 digests;
- retention policy used;
- desired-state digest;
- generation tool version.

It contains no secrets and is not a user contract. Its purpose is auditability
and fast no-op detection. If the desired-state digest already matches the
remote manifest and remote verification succeeds, re-publishing the same tag
returns success without regenerating signatures or touching R2.

The manifest accelerates idempotency but does not weaken rebuildability. A full
rebuild can derive it from the registry, GitHub Releases, and public key.

### 8.3 Determinism

Candidate generation should stabilize everything under repository control:

- sort projects, versions, architectures, and package records;
- normalize file permissions;
- generate gzip files without source mtimes;
- use SHA-256 package and metadata digests;
- avoid wall-clock timestamps except where a package format requires them;
- record the latest selected release time as the logical repository revision.

OpenPGP signature packet timestamps may differ during a rebuild with the same
key. Rebuild acceptance is therefore semantic: the same packages are selected,
all generated metadata references the same content, and all signatures verify.
The fast no-op path prevents unnecessary signature changes on a repeated
incremental publish.

## 9. APT repository design

The public layout remains:

```text
apt/
├── dists/stable/
│   ├── InRelease
│   ├── Release
│   ├── Release.gpg
│   └── <project>/
│       ├── binary-amd64/Packages
│       ├── binary-amd64/Packages.gz
│       ├── binary-amd64/by-hash/SHA256/<digest>
│       ├── binary-arm64/Packages
│       ├── binary-arm64/Packages.gz
│       └── binary-arm64/by-hash/SHA256/<digest>
└── pool/<project>/*.deb
```

For each project and architecture:

1. select retained packages;
2. generate `Packages` with repository-relative `Filename` values;
3. generate deterministic `Packages.gz`;
4. create content-addressed `by-hash/SHA256/<digest>` copies;
5. generate the suite `Release` with `apt-ftparchive release`;
6. create detached `Release.gpg` and clearsigned `InRelease`.

The Release file should set:

```text
Origin: Meigma
Label: Meigma
Suite: stable
Codename: stable
Components: <sorted registry project names>
Architectures: amd64 arm64
Acquire-By-Hash: yes
Description: Meigma package repository
```

`Valid-Until` should be omitted in v1 unless a separate scheduled metadata
refresh policy is introduced. Expiring an otherwise valid repository merely
because no project released recently would be an avoidable availability
failure. This trade-off should be documented as replay protection deferred in
favor of static-repository availability.

APT's Release signature authenticates the hashes of the Packages indexes, and
the indexes authenticate package files. The public key is installed through
`Signed-By` at `/etc/apt/keyrings/meigma.asc`; no global `apt-key`,
`[trusted=yes]`, or signature bypass is allowed.

## 10. RPM repository design

Each project has an independent repository:

```text
rpm/<project>/
├── meigma.repo
├── x86_64/*.rpm
├── aarch64/*.rpm
└── repodata/
    ├── <checksum>-primary.xml.*
    ├── <checksum>-filelists.xml.*
    ├── <checksum>-other.xml.*
    ├── repomd.xml
    └── repomd.xml.asc
```

`createrepo_c` runs at `rpm/<project>` so one `repomd.xml` indexes both
architecture directories. Unique checksum-bearing metadata filenames remain
enabled. The publisher signs `repodata/repomd.xml` with an armored detached
signature.

Generated `meigma.repo` contains:

```ini
[meigma-<project>]
name=Meigma <project>
baseurl=https://pkgs.meigma.dev/rpm/<project>
enabled=1
repo_gpgcheck=1
gpgcheck=0
gpgkey=https://pkgs.meigma.dev/meigma.asc
```

Package-level `gpgcheck=1` remains a v2 concern. V1 must prove that DNF rejects
tampered `repomd.xml` and accepts the signed repository without
`--nogpgcheck`.

## 11. Retention model

The default retention is the newest five qualifying semantic versions per
project. An override is allowed in the registry.

Selection happens before metadata generation:

1. enumerate qualifying published releases;
2. order by semantic version descending;
3. take the first `N` versions;
4. require the configured package set for every selected version;
5. include those package files for each supported architecture and format;
6. generate metadata only from the selected set.

This version-set approach keeps DEB and RPM views aligned. Older assets remain
on GitHub Releases and reappear if the retention value is increased and a
rebuild runs.

The plan must list every proposed removal. A failed metadata build or remote
copy prevents deletion. Remote deletions happen only after new metadata is
active and verified.

## 12. Publish transaction

### 12.1 Serialization

All production mutations from `publish.yml` and `rebuild.yml` share one static
GitHub Actions concurrency group, conceptually:

```yaml
concurrency:
  group: meigma-packages-r2-production
  queue: max
```

No production workflow uses `cancel-in-progress: true`. Validation that does
not access secrets or remote state may occur before the serialized job. The
hydrate/build/apply/verify sequence occurs under the single mutation lock.

GitHub currently documents that ordinary concurrency retains only one pending
run; `queue: max` is therefore required to preserve late dispatches rather than
silently replacing them.

### 12.2 Publish stages

Within the serialized job:

1. Revalidate the registry and requested source release.
2. Hydrate the current bucket/prefix into a fresh local workspace.
3. Download and verify source assets.
4. Apply retention and build a complete candidate tree.
5. Verify package metadata, signatures, paths, and install configuration
   entirely locally.
6. Compute and display the remote change plan.
7. Upload immutable package objects and content-addressed metadata.
8. Upload canonical index payloads.
9. Upload activation metadata/signatures (`InRelease`, `Release*`,
   `repomd.xml*`) in the proven order.
10. Upload the state manifest and landing/config files.
11. Delete objects no longer referenced, after all copies succeed.
12. Verify the remote tree through the S3 endpoint and public hostname.
13. Run the post-publish install smoke test.

A single unconstrained `rclone sync` should not be trusted as an activation
transaction. Rclone's checksum comparison and delete-after behavior remain
useful, but the publisher should apply filtered copy/activation/delete stages
explicitly so failure boundaries are testable.

### 12.3 Client-visible consistency proof gate

R2 provides strong read-after-write, overwrite, list, and delete consistency
through its S3/API surface. That guarantee is per object, not a multi-object
transaction. Cloudflare's custom-domain cache can also continue serving an
overwritten or deleted object until TTL expiry or purge.

The production design therefore requires:

- immutable long-lived caching for package objects, APT by-hash objects, and
  checksum-named RPM metadata;
- cache bypass or strict revalidation for mutable activation paths such as
  `InRelease`, `Release*`, `Packages*`, `repomd.xml*`, `meigma.asc`, and
  `meigma.repo`;
- packages and content-addressed metadata uploaded before activation metadata;
- deletions last;
- fault-injection tests that stop after each stage and run real APT/DNF clients.

APT by-hash and RPM checksum-named metadata substantially reduce the race
surface. RPM still exposes `repomd.xml` and `repomd.xml.asc` as two mutable
objects. The first spike must prove acceptable DNF behavior during an
interrupted ordered update. If direct R2 publication cannot satisfy the
no-half-publish invariant, stop before production and escalate. The available
choices would be accepting a precisely bounded retry window, adding an atomic
snapshot-routing layer such as a Cloudflare Worker, or revising the invariant;
none should be selected unilaterally.

## 13. Rebuild design

`rebuild.yml` is manual-only and uses the same production concurrency group.
It performs this sequence:

1. validate `projects.yml`;
2. enumerate every qualifying release for every enabled project, with API
   pagination;
3. select retained versions;
4. download and verify all required assets into an empty workspace;
5. build and verify the same candidate layout used by incremental publish;
6. display counts, selected versions, desired-state digest, and remote diff;
7. apply first to `_staging/` unless the operator explicitly selects the
   protected production environment;
8. run clean-container smoke tests;
9. apply to production through the same ordered publisher;
10. verify the final public tree.

The first disaster-recovery exercise should empty only the staging prefix,
rebuild it from GitHub Releases, and compare its logical manifest and generated
metadata to production. Production deletion is not part of routine testing.

## 14. Workflow design

### 14.1 `ci.yml`

Triggers: pull requests and pushes to `main`.

Responsibilities:

- validate `projects.yml`;
- run Go unit and integration tests;
- run `actionlint` and workflow policy checks;
- run ShellCheck for retained shell wrappers;
- generate tiny fixture DEB/RPM assets from pinned nfpm inputs;
- generate a throwaway GPG key;
- execute the complete local publish path into a temporary directory;
- verify APT and RPM metadata/signatures;
- serve the directory locally and install the fixture package in pinned Debian,
  Ubuntu, and Fedora containers;
- run an identical second publish and assert a no-op plan;
- run an empty-root rebuild and compare logical output;
- exercise interruption points with the local-directory transport.

PR CI has `permissions: contents: read`, receives no R2 or production signing
secrets, and uses GitHub-hosted runners.

### 14.2 `publish.yml`

Triggers:

- `repository_dispatch` type `publish-package`;
- `workflow_dispatch` with typed `project`, `tag`, and target inputs.

Job boundaries:

1. `validate`: read-only, no environment, no publisher secrets; normalize the
   event/manual inputs and verify the registry/release contract.
2. `publish`: serialized, references the chosen protected environment, imports
   the signing subkey into an ephemeral `GNUPGHOME`, builds and applies the
   candidate, then verifies the remote result.
3. `smoke`: calls the shared smoke workflow after a successful apply.

`permissions: {}` is set at workflow level and jobs receive only their required
permissions. Checkout uses `persist-credentials: false`. Every third-party
action and cross-repository workflow is pinned to a full commit SHA.

### 14.3 `rebuild.yml`

Trigger: `workflow_dispatch` only.

Inputs should make staging the safe default. Production rebuild requires the
protected production environment and an explicit confirmation input. It calls
the same CLI and smoke workflow as publish.

### 14.4 `smoke-test.yml`

Triggers:

- reusable `workflow_call` from publish/rebuild;
- daily UTC schedule;
- manual dispatch for diagnosis.

The matrix covers current Debian stable, Ubuntu LTS, and Fedora container
images pinned by digest. For each registered project it:

- downloads `meigma.asc` and verifies its expected fingerprint;
- configures the documented APT or DNF repository with metadata checking on;
- refreshes metadata;
- installs the newest package;
- verifies the installed package/version.

Post-publish smoke failure fails the publish run. Scheduled failure opens or
updates one stable tracking issue instead of creating a new issue every day.
Only the notification job needs `issues: write`.

## 15. Authentication and secret handling

### 15.1 Single GitHub App for consumer dispatch

V1 uses one private, Meigma-owned GitHub App as the cross-repository automation
identity. This explicitly supersedes the jumpstart's PAT default.

The App should be registered with:

- repository permission `Contents: write`, which GitHub requires for the
  create-repository-dispatch endpoint;
- no organization permissions;
- no webhook subscriptions, callback URL, or user authorization flow;
- installation access limited to the `meigma/packages` repository;
- no repository-ruleset bypass role.

The App does not need installation access to consumer repositories. Approved
consumer workflows receive the App client ID through an organization Actions
variable and the App private key through an organization Actions secret, both
restricted to the participating consumer repositories. Suggested names are:

- `MEIGMA_PACKAGES_APP_CLIENT_ID`;
- `MEIGMA_PACKAGES_APP_PRIVATE_KEY`.

After a consumer publishes its GitHub Release, its trusted release workflow:

1. invokes `actions/create-github-app-token` pinned to a full commit SHA;
2. requests an installation token for owner `meigma`, repository `packages`,
   and `permission-contents: write`;
3. sends `POST /repos/meigma/packages/dispatches` with event type
   `publish-package` and the `{project, tag}` payload;
4. allows the action to revoke the installation token when the job finishes.

The shared consumer-workflow fragment should have this shape (the delivered
template replaces the action placeholder with a reviewed full commit SHA):

```yaml
- name: Mint packages dispatch token
  id: packages-app
  uses: actions/create-github-app-token@<reviewed-full-commit-sha> # v3
  with:
    client-id: ${{ vars.MEIGMA_PACKAGES_APP_CLIENT_ID }}
    private-key: ${{ secrets.MEIGMA_PACKAGES_APP_PRIVATE_KEY }}
    owner: meigma
    repositories: packages
    permission-contents: write

- name: Request package publication
  env:
    GH_TOKEN: ${{ steps.packages-app.outputs.token }}
    PROJECT: ${{ github.event.repository.name }}
    RELEASE_TAG: ${{ github.ref_name }}
  run: |
    gh api --method POST repos/meigma/packages/dispatches \
      -f event_type=publish-package \
      -F client_payload[project]="$PROJECT" \
      -F client_payload[tag]="$RELEASE_TAG"
```

Installation tokens currently expire after one hour even if not revoked. The
private key is the only long-lived GitHub dispatch credential and is centrally
rotated once rather than maintaining PATs in every consumer. The token-minting
step must run only from trusted release workflow code, never from pull-request
jobs or attacker-controlled scripts.

### 15.2 R2 credentials

Use an R2 bucket-scoped object read/write token exposed as S3-compatible
credentials:

- `R2_ACCOUNT_ID`;
- `R2_ACCESS_KEY_ID`;
- `R2_SECRET_ACCESS_KEY`.

Store them as `staging`/`production` environment secrets. Restrict production
deployment branches to `main`; use required review and prevent self-review if
the repository plan supports those controls. The publisher configures rclone
through environment variables or a temporary config file with restrictive
permissions, then removes it.

### 15.3 GPG signing key

Josh holds the offline primary key. CI receives only:

- `GPG_SIGNING_SUBKEY`;
- `GPG_PASSPHRASE`;
- an expected primary fingerprint as a non-secret variable.

The signing job:

1. creates a mode-0700 temporary `GNUPGHOME`;
2. imports the secret-subkey export;
3. verifies the primary fingerprint and confirms that no usable primary secret
   key is present;
4. signs by full subkey fingerprint, never by ambiguous short key ID;
5. passes the passphrase through an input descriptor, not a process argument;
6. verifies every generated signature before publication;
7. removes the temporary keyring at job completion.

The public `meigma.asc` is a minimal armored public-key export and has a stable
URL. Its fingerprint appears in the README, install docs, signing runbook, and
smoke test.

The first compatibility spike uses the planned Ed25519 primary/signing subkey
structure in current Debian, Ubuntu, and Fedora containers. If a real supported
RPM client cannot verify it, record the evidence and use the pre-approved
RSA-4096 fallback; announce the deviation before production.

## 16. Cache and HTTP policy

The public domain is part of the correctness boundary.

Recommended object classes:

| Class | Examples | Cache policy |
|---|---|---|
| Immutable packages | `*.deb`, `*.rpm` | one year, immutable |
| Content-addressed metadata | APT `by-hash/**`, checksum-named RPM XML | one year, immutable |
| Mutable activation metadata | `InRelease`, `Release*`, `Packages*`, `repomd.xml*` | bypass cache or always revalidate |
| Key/config/docs | `meigma.asc`, `meigma.repo`, `index.html` | short TTL with revalidation |
| Internal state | `_state/**`, `_staging/**` | no-cache; not documented to users |

The provisioning checklist must explicitly configure and verify this policy.
Do not enable a blanket Cache Everything rule over mutable repository metadata.
Remote verification should test both the S3 endpoint and custom domain so a
correct bucket hidden behind stale CDN objects is detected.

## 17. Observability and failure handling

Every operation should produce a GitHub job summary containing:

- mode and target;
- project/tag or rebuild scope;
- source release URLs;
- retained version set;
- package counts by format/architecture;
- desired-state digest;
- created/replaced/deleted object counts;
- no-op or changed result;
- signing-key fingerprint;
- candidate, S3, public-host, and smoke verification results.

Logs must not include credentials, secret key material, passphrases, or rclone
configuration. Failures before remote application leave R2 untouched. Failures
during application stop immediately, skip deletions, and leave enough summary
data for a queued re-publish or rebuild to converge safely.

The daily smoke workflow is the v1 external monitor. No additional monitoring
service or account is proposed.

## 18. Documentation deliverables

### README

- purpose and ownership model;
- architecture and bucket layout;
- publish/rebuild/smoke workflow overview;
- exact external provisioning checklist;
- cache policy checklist;
- signing-key fingerprint and link to the signing runbook;
- disaster recovery: run staging rebuild, verify, then production rebuild;
- GitHub App ownership, installation, private-key rotation, and incident
  revocation procedure;
- status/limitations, including metadata-only signing in v1.

### `docs/onboarding.md`

- nfpm expectations and supported architectures;
- asset/checksum naming contract;
- registry-entry example;
- required organization variable/secret names and selected-repository access;
- copy-paste GitHub App token and dispatch steps with the action pinned to a
  full commit SHA;
- explanation that the App is installed only on `meigma/packages`, not granted
  write access to consumers;
- manual re-publish procedure;
- onboarding validation checklist.

### `docs/install.md`

- canonical APT Deb822 source configuration;
- canonical DNF `.repo` installation;
- per-project install commands generated from the registry;
- public key fingerprint verification;
- uninstall/repository removal instructions;
- no signature-bypass alternatives.

### `docs/signing-runbook.md`

- offline primary and signing-only subkey generation;
- primary backup and revocation certificate handling;
- minimal public export;
- secret-subkey export for GitHub Actions;
- environment secret installation;
- CI import/fingerprint verification;
- rotation and revocation procedure;
- Ed25519 compatibility proof and RSA fallback record.

## 19. Agile delivery sequence

Each phase should be independently reviewable. Later phases should adapt to
what earlier proofs reveal.

### Phase 0 — Throwaway format and consistency spike

Build tiny fixture DEB/RPM packages, generate/sign both repository formats,
serve them locally, and install from clean Debian/Ubuntu/Fedora containers.
Exercise Ed25519. Simulate interruption between ordered copy stages and observe
APT/DNF behavior. Keep only the commands and evidence worth carrying forward.

Gate: either the direct-static publication model is proven viable or the exact
consistency/signing blocker is escalated before durable implementation.

### Phase 1 — Local vertical slice

Implement one registry entry and one local command that takes fixture release
assets to a verified candidate tree. Include APT, RPM, signatures, install
files, and one end-to-end container install. No GitHub API, R2, or real key.

Gate: a developer can reproduce the verified repository locally with one
command.

### Phase 2 — Determinism, retention, and rebuild

Add package inspection, checksum validation, semantic-version retention,
state-manifest/no-op behavior, full rebuild from fixture release sets, ordered
sync planning, and failure-injection tests.

Gate: same-tag publish is a no-op; empty-root rebuild reproduces the logical
tree; interruption never deletes referenced content.

### Phase 3 — Repository CI and unprivileged workflows

Add PR CI, workflow linting/security policy, pinned actions/images, fixture
generation, local publish tests, and initial documentation. Add publish/rebuild
workflow validation jobs without connecting secrets.

Gate: everything buildable without external provisioning is green on a PR.

### Phase 4 — Staging provisioning and rehearsal

Josh provisions the R2 bucket/domain/token and GPG material. Configure the
`_staging/` prefix and protected environments. Run the real publisher against
staging, verify through R2 and HTTP, run clean-container installs, repeat for a
no-op, and perform an empty-prefix rebuild.

Gate: all acceptance behavior passes without touching production paths.

### Phase 5 — Production and first consumer

Publish `incus-gh-runner`, run post-publish and daily smoke tests, prepare and
land the consumer dispatch change, validate the onboarding doc against that
real change, and perform a documented recovery rehearsal.

Gate: users can install from the canonical snippets and the consumer release
workflow reliably triggers queued publication.

## 20. Ownership and approval boundaries

| Area | Repository work | Josh/external action |
|---|---|---|
| Candidate builder, validators, tests | Implemented here | Review behavior |
| Workflows and docs | Implemented here | Review privileged boundaries |
| R2 bucket and custom domain | Checklist only | Create and attach |
| R2 API token | Consumption code only | Create and store secrets |
| GPG primary/subkey | Runbook and CI importer | Generate offline and store secrets |
| Cache rules | Document and verify | Apply in Cloudflare |
| Dispatch GitHub App | Document token/dispatch contract | Register App, install only on `meigma/packages`, and scope the organization variable/secret to consumers |
| `incus-gh-runner` workflow change | Ready-to-apply patch/PR | Approve and merge in that repo |
| Spend/new account/invariant deviation | Stop and explain | Decide |

## 21. Risks and proof gates

| Risk | Early proof or mitigation | Escalation condition |
|---|---|---|
| Multi-object metadata activation is not atomic | By-hash/checksum filenames, ordered writes, cache bypass, interruption tests | APT/DNF can observe an unrecoverable half-published state |
| CDN serves stale overwrite/delete results | Per-path cache policy and public-host verification | Required cache control needs new paid infrastructure |
| Ed25519 fails on a supported RPM client | Run real client matrix in Phase 0 | Use documented RSA fallback and announce evidence |
| Asset patterns do not match real releases | Inspect `incus-gh-runner` release before finalizing registry entry | Source release contract requires a broader redesign |
| Release checksum file is incomplete/ambiguous | Fail closed and inspect package metadata | Consumer release workflow must change materially |
| App private key, signing subkey, or R2 credentials leak | Selected-repository secrets, protected environments, ephemeral files, minimal job permissions | Any secret appears in logs/artifacts or untrusted job context |
| Queue overflow or stuck publish blocks later releases | Job timeout, visible queue, manual re-dispatch/rebuild runbook | Queue cannot preserve all legitimate release events |
| Retention removes a still-referenced object | Candidate validation, activation before delete, skip delete on any error | Fault injection finds a 404 reachable from valid metadata |

## 22. Acceptance mapping

| Jumpstart acceptance criterion | Proposed evidence |
|---|---|
| Fixture pipeline passes in PR CI | Phase 1/3 local candidate generation, signature verification, and container installs |
| Staged end-to-end publish | Phase 4 `_staging/` publish through R2 and public HTTP |
| Debian stable install | Smoke matrix with signature enforcement |
| Ubuntu LTS install | Smoke matrix with signature enforcement |
| Fedora install | Smoke matrix with `repo_gpgcheck=1` and no bypass |
| Same-tag publish is a no-op | Desired-state manifest plus zero-change sync plan |
| Empty staging rebuild reproduces tree | Phase 2 local test and Phase 4 R2 staging drill |
| `incus-gh-runner` onboarding works | Phase 5 ready-to-apply consumer patch and real release dispatch |
| Full tree reconstructable from Releases | Rebuild ignores prior state and derives selected assets from registry/releases |

## 23. External provisioning checklist

When the secrets-free work is ready, ask Josh to:

1. create R2 bucket `meigma-packages`;
2. attach `pkgs.meigma.dev` as the production custom domain;
3. keep the `r2.dev` development URL disabled for the production bucket;
4. create a bucket-only object read/write R2 API token;
5. add account ID/access key/secret key to staging and production environments;
6. configure cache bypass/revalidation for mutable metadata paths and immutable
   caching for package/content-addressed paths;
7. generate the offline primary and CI signing subkey from the reviewed runbook;
8. add the signing-subkey/passphrase secrets and expected fingerprint;
9. configure environment protection and allowed deployment branches;
10. register the private Meigma dispatch GitHub App with only repository
    `Contents: write` permission and install it only on `meigma/packages`;
11. store its client ID as an organization Actions variable and private key as
    an organization Actions secret, each limited to approved consumer repos;
12. add `incus-gh-runner` to that selected-repository access only after its
    pinned token/dispatch workflow is ready.

## 24. Evidence anchors

The following current primary documentation informed the proposal:

- [GitHub Actions concurrency](https://docs.github.com/en/actions/concepts/workflows-and-actions/concurrency): concurrency groups and queued runs.
- [GitHub repository dispatch REST endpoint](https://docs.github.com/en/rest/repos/repos#create-a-repository-dispatch-event): GitHub App installation-token support, payload limits, and required repository `Contents: write` permission.
- [GitHub App authentication in Actions](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/making-authenticated-api-requests-with-a-github-app-in-a-github-actions-workflow): minting installation tokens from workflow jobs.
- [GitHub App installation token endpoint](https://docs.github.com/en/rest/apps/apps#create-an-installation-access-token-for-an-app): one-hour token lifetime and repository restriction.
- [GitHub deployment environments](https://docs.github.com/en/actions/concepts/workflows-and-actions/deployment-environments): protected jobs and environment-secret access.
- [Cloudflare R2 consistency](https://developers.cloudflare.com/r2/reference/consistency/): strong object consistency and relaxed cached custom-domain behavior.
- [Cloudflare R2 public buckets](https://developers.cloudflare.com/r2/buckets/public-buckets/): production custom domains and cache controls.
- [Rclone sync](https://rclone.org/commands/rclone_sync/): checksum comparison, delete-after default, and dry-run guidance.
- [Debian `apt-ftparchive`](https://manpages.debian.org/testing/apt-utils/apt-ftparchive.1.en.html): Packages/Release generation and Release fields.
- [Debian `apt-secure`](https://manpages.debian.org/unstable/apt/apt-secure.8.en.html): Release signature verification and `Signed-By` key placement.
- [createrepo_c](https://rpm-software-management.github.io/createrepo_c/): RPM repository metadata generation.
- [GnuPG operational commands](https://gnupg.org/documentation/manuals/gnupg/Operational-GPG-Commands.html): signing-subkey export behavior.

## 25. Recommendation

Approve this proposal as the working direction, not as an immutable build
specification. Begin with Phase 0 and allow its evidence to revise command
boundaries, cache details, and publication ordering while preserving the
jumpstart invariants and user-facing URL/install contracts.
