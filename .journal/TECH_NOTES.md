# Technical Notes

- The phased delivery plan in `.journal/001/DESIGN_PROPOSAL.md` is complete
  and its scaffolding was removed in session 009 (PRs #14–#16). The repo now
  contains only: the Go CLI (`cmd/meigma-packages`, `internal/`), two
  workflows (`ci.yml`, `publish.yml`), two scripts (`scripts/publish.sh`,
  `scripts/lint-workflows.sh`), `docker/tools.Dockerfile`, `projects.yml`,
  and the mkdocs site. Read the proposal only as historical context.
- Publication contract: the `Publish` workflow accepts a trusted
  `publish-package` repository dispatch carrying only `project` and `tag`
  (always validation -> protected staging -> protected production), or a
  manual run with inputs `project`, `tag`, `apply_staging`,
  `apply_production` (both booleans default false; production requires
  staging success in the same run). There are no confirmation phrases and no
  wired-in staging-deletion mode. `scripts/publish.sh` is env-driven
  (`PUBLICATION_TARGET`, environment-scoped `R2_PREFIX`/`PUBLIC_BASE_URL`);
  it validates the request against `projects.yml`, fetches and verifies the
  GitHub Release (digests + attestations), rebuilds the signed tree in the
  tools container, applies and remotely verifies the sync, replays a no-op,
  and clean-installs on Debian/Ubuntu/Fedora asserting the exact package
  version. Proven by staging run 29973794881 (`incus-gh-runner v1.1.0`).
- Neither GitHub environment has required reviewers — protection is
  deployment branch policy only (`main`). The `apply_production` checkbox is
  the only manual-run speed bump; the automated dispatch path is gated by the
  GitHub App token. Adding required reviewers to `production` would also gate
  automated consumer dispatches.
- GitHub Releases are authoritative inputs; the APT/RPM tree on R2 is derived
  and reconstructable. Rebuild equivalence is the logical manifest digest,
  not byte equality of timestamp-bearing metadata or signatures. Sync plans
  order content, indexes, activation metadata, state, then deletion; recovery
  from any bad or empty staging state is just another convergent staging
  publish.
- Registry and layout conventions are documented in
  `docs/docs/publishing.md` (field-by-field `projects.yml` contract, upstream
  release requirements, project key = APT component = `rpm/<project>/` path,
  `_state/manifest.json` discovery). The consumer dispatch example there was
  verified against `meigma/incus-gh-runner/.github/workflows/packages.yml`
  (app token via `actions/create-github-app-token` + `gh api .../dispatches`).
- APT publication must retain both SHA-256 and SHA-512 by-hash indexes;
  tested modern clients requested SHA-512 and otherwise fell back to the
  mutable index. Direct static RPM publication has an unavoidable fail-closed
  interval between `repomd.xml` and `repomd.xml.asc` writes; the publisher
  bypasses caching for the mutable pair, writes them consecutively, defers
  deletion until verification, and proves package resolution after
  publication (DNF5 cache refresh success alone can hide a disabled repo).
- Cross-repository dispatch uses one private Meigma GitHub App with
  repository `Contents: write`, installed with selected-repository access
  including `packages`; the consumer mints a short-lived token restricted to
  `packages` and sends the validated stable tag.
- Staging uses R2 bucket `meigma-packages`, custom domain `pkgs.meigma.dev`,
  exact prefix `_staging/`; the managed `r2.dev` endpoint is disabled.
  Cloudflare cache rules were installed manually (API tokens lacked Rulesets
  permission); staging objects return `no-store`, production package payloads
  are immutable for one year.
- The production repository key is Ed25519 primary
  `9C74476A669465EEB8D46AD8B0E68773B6E259F6` with signing subkey
  `9DA41FD9DBD38B19AC75454D27CCA9E924245272`. Signing backups, passphrase,
  and R2 source credentials live in the 1Password `Meigma` vault; GitHub
  receives only the signing-only export, passphrase, and environment-scoped
  R2 credentials. Renew or replace the one-year staging credential before
  `2027-07-18T19:51:37Z`.
