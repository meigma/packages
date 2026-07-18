# Technical Notes

- The canonical implementation handoff is
  `.journal/001/DESIGN_PROPOSAL.md`, especially Section 19's Phase 0–5 agile
  delivery sequence. Begin at the first incomplete phase, satisfy its proof
  gate, and let evidence revise later details rather than implementing the full
  proposal as a waterfall specification.
- `main` contains the repository-local `meigma-packages` Go/Cobra/Viper CLI,
  pinned mise toolchain, Moon root/docs projects, read-only CI, Dependabot, and
  Apache-2.0/MIT dual licensing from PR #1. The CLI itself has no release or
  container-publication workflow.
- GitHub Releases are authoritative inputs; the APT/RPM tree on R2 is derived
  and reconstructable. Build and verify a candidate tree before any remote
  mutation.
- Client-visible consistency is an explicit Phase 0 proof gate: serialization
  alone does not make multi-object R2 publication atomic, and Cloudflare custom
  domain caching can expose stale activation metadata.
- Cross-repository dispatch uses one private Meigma GitHub App with repository
  `Contents: write`, installed only on `meigma/packages`. Approved consumer
  workflows receive selected-repository organization configuration and mint
  short-lived tokens restricted to the `packages` repository.
- R2/domain/cache configuration, signing material, protected environments, and
  GitHub App registration are Josh-owned external actions deferred until the
  secrets-free phases are green.
