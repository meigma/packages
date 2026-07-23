# Technical Notes

- The canonical implementation handoff is
  `.journal/001/DESIGN_PROPOSAL.md`, especially Section 19's Phase 0–5 agile
  delivery sequence. Begin at the first incomplete phase, satisfy its proof
  gate, and let evidence revise later details rather than implementing the full
  proposal as a waterfall specification.
- `main` contains the repository-local `meigma-packages` Go/Cobra/Viper CLI,
  pinned mise toolchain, Moon root/docs projects, read-only CI, Dependabot, and
  Apache-2.0/MIT dual licensing from PR #1. PR #6 added the secrets-free
  `build-local` vertical slice and `moon run root:phase1-proof`. PR #7 added
  deterministic fixture-set `rebuild-local`, `plan-sync`, logical manifests,
  and `moon run root:phase2-proof`. PR #8 added `validate-request`, manual
  fixture-backed publish/rebuild validation workflows, pinned actionlint and
  ShellCheck, executable workflow/image policy, and the initial operations
  boundary. PR #9 added exact-prefix R2 apply and verification, passphrase-file
  signing, and the protected staging rehearsal. PRs #10–#12 completed the first
  Phase 5 path: verified GitHub Release ingestion, multi-architecture APT/RPM
  rebuilds, protected production-root publication, trusted consumer dispatch,
  public installation documentation, no-op verification, and clean installs.
  PR #13 generalized that same path to registered exact stable `vX.Y.Z`
  releases and successfully published `incus-gh-runner v1.1.0`.
- Phase 3 validation remains manual, read-only, GitHub-hosted, and disconnected
  from secrets, deployment environments, R2, and remote mutation. The Phase 4
  publish workflow keeps that job as an unprivileged prerequisite and performs
  mutation only in a separate serialized job protected by the `staging`
  environment. Routine CI cannot select the privileged Moon task.
- Phase 2 rebuild equivalence is the logical manifest digest, not byte equality
  of timestamp-bearing repository metadata or OpenPGP signatures. An unchanged
  rebuild verifies retained package digests and repository signatures before
  returning a no-op. Sync plans order content, indexes, activation metadata,
  state, and finally deletion; no referenced candidate path may be deleted.
- GitHub Releases are authoritative inputs; the APT/RPM tree on R2 is derived
  and reconstructable. Build and verify a candidate tree before any remote
  mutation.
- Phase 0 passed Ed25519 repository signing and clean installation on Debian 13,
  Ubuntu 26.04 LTS, and Fedora 44. APT publication must retain both SHA-256 and
  SHA-512 by-hash indexes; the tested modern clients requested SHA-512 and
  otherwise fell back to the mutable index.
- Direct static RPM publication has an unavoidable fail-closed interval between
  `repomd.xml` and `repomd.xml.asc` writes. The approved v1 contract bypasses
  caching for the mutable pair, writes them consecutively, never bypasses
  verification, converges safely on retry, defers deletion until verification,
  and proves package resolution or installation after publication. DNF5 cache
  refresh success alone is insufficient because it may hide a disabled repo.
- Cross-repository dispatch uses one private Meigma GitHub App with repository
  `Contents: write`. Its installation uses selected-repository access for
  `homebrew-tap`, `scoop-bucket`, and `packages`; the `incus-gh-runner`
  workflow mints a short-lived token restricted to the `packages` repository
  and sends only the fixed project plus a validated stable tag. The packages
  workflow rejects any payload fields beyond `project` and `tag`; dispatch
  always traverses validation, staging, and production and synthesizes its
  production confirmation internally.
- Protected publication accepts only registered exact `vX.Y.Z` tags. After
  validation it derives the expected package version by removing exactly one
  leading `v`; DEB/RPM metadata and clean Debian, Ubuntu, and Fedora installs
  must all report that version. Manual production confirmation is exactly
  `publish <project> <tag> to production`. Run `29972562362` proved this
  contract for `incus-gh-runner v1.1.0` on merge `cdc66905`.
- Staging uses R2 bucket `meigma-packages`, custom domain `pkgs.meigma.dev`, and
  exact prefix `_staging/`; the managed `r2.dev` endpoint is disabled. The
  publisher fails closed unless both the prefix and public staging URL match.
  Cloudflare cache rules were installed manually because the available API
  tokens lacked Rulesets permission; live staging objects return `no-store`
  and remain uncached.
- The production repository key is Ed25519 primary
  `9C74476A669465EEB8D46AD8B0E68773B6E259F6` with signing subkey
  `9DA41FD9DBD38B19AC75454D27CCA9E924245272`. Signing backups, passphrase, and
  staging/production R2 source credentials live in the 1Password `Meigma`
  vault; GitHub receives only the signing-only export, passphrase, and separate
  environment-scoped R2 credentials. Renew or replace the one-year staging
  credential before `2027-07-18T19:51:37Z`.
