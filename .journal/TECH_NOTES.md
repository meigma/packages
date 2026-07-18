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
  and `moon run root:phase2-proof`; Phase 3 CI and unprivileged workflow
  integration is the next incomplete phase.
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
  `Contents: write`, installed only on `meigma/packages`. Approved consumer
  workflows receive selected-repository organization configuration and mint
  short-lived tokens restricted to the `packages` repository.
- R2/domain/cache configuration, signing material, protected environments, and
  GitHub App registration are Josh-owned external actions deferred until the
  secrets-free phases are green.
