# Phase 0 evidence

- Date: 2026-07-17
- Host: macOS arm64 with Docker 29.4.0
- Scope: disposable local fixture generation, Ed25519 metadata signing, clean
  client installation, and ordered-copy fault injection
- Production systems or credentials used: none

## Reproduction

```sh
./spikes/phase0/run.sh
./spikes/phase0/fault-injection.sh
```

Both commands completed successfully on the recorded host after the findings
below were incorporated into the spike.

## Pinned client matrix

| Client | Image digest | Package-manager version | Result |
|---|---|---|---|
| Debian 13 | `debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd` | APT 3.0.3 | Signed APT install passed |
| Ubuntu 26.04 LTS | `ubuntu:26.04@sha256:3131b4cc82a783df6c9df078f86e01819a13594b865c2cad47bd1bca2b7063bb` | APT 3.2.0 | Signed APT install passed |
| Fedora 44 | `fedora:44@sha256:6c75d5bf57cb0fa5aa4b92c6a83c86c791644496d9ac230de7711f5b8ec3b898` | DNF5 5.4.2.1 | Signed repository-metadata install passed |

The tool image uses APT 3.0.3, `createrepo_c` 1.2.0, GnuPG 2.4.7, RPM
4.20.1, and `dpkg-deb` 1.22.22. The fixture is an architecture-independent
shell payload indexed in the host-native APT architecture (`arm64`) and RPM
`noarch` repository.

## Format and signing proof

The spike generates a throwaway Ed25519 certification primary key and a
separate Ed25519 signing subkey. It uses that subkey to produce and verify:

- APT `InRelease`;
- APT `Release.gpg` over `Release`;
- RPM `repomd.xml.asc` over `repomd.xml`.

Clean clients imported the public key, enforced repository metadata
verification, installed `meigma-phase0` version `1.0.0`, and executed its
payload. No signature bypass was used. RPM package-level verification remains
disabled as intended for v1 and DNF reports that package OpenPGP checks were
skipped.

## APT interruption result

The first fault run published only APT `by-hash/SHA256` objects, matching the
session 001 proposal. APT 3.0.3 instead requested the SHA-512 by-hash URI. When
that returned 404, it fell back to mutable `Packages.gz`; overwriting that file
before activation produced a hash mismatch and made the repository refresh
fail.

Publishing both SHA-256 and SHA-512 by-hash objects corrected the model. With
old and new snapshots representing versions `1.0.0` and `1.1.0`, a clean
Debian client observed:

| Interruption point | Install result |
|---|---|
| Old snapshot active | `1.0.0` |
| New packages and by-hash objects copied | `1.0.0` |
| Mutable `Packages*` copied | `1.0.0` through old SHA-512 by-hash object |
| New `Release` and `Release.gpg` copied while old `InRelease` remained | `1.0.0` |
| New `InRelease` copied | `1.1.0` |

This proves the APT activation sequence on the tested client, provided all
strong hash algorithms APT may select are published and retained until they
are no longer referenced.

## RPM interruption result

RPM's checksum-named metadata remained safe while the old `repomd.xml` and
signature were active. The final activation cannot be made atomic with two
independent object writes:

| Copy order | Intermediate clean-client result | Completed-pair result |
|---|---|---|
| New `repomd.xml`, then new `repomd.xml.asc` | Bad PGP signature; repository disabled; package unresolved | `1.1.0` installs |
| New `repomd.xml.asc`, then new `repomd.xml` | Bad PGP signature; repository disabled; package unresolved | `1.1.0` installs |

DNF5's `makecache` command can log the bad signature and still return success
after disabling the repository. The fault assertion therefore attempts a real
package installation; that operation fails until the matching pair is active.

## Gate assessment

- Package format generation: **passed**.
- Ed25519 compatibility on the pinned clean-client matrix: **passed**.
- APT ordered static publication: **passed after revising the by-hash contract**.
- RPM no-half-publish invariant with direct multi-object publication:
  **blocked by a demonstrated signature-failure window**.

Phase 1 should not encode RPM publication semantics until the owner chooses
one of the proposal's escalation paths: accept and document a tightly bounded
retry/unavailability window, introduce an atomic snapshot-routing design, or
revise the invariant. This spike does not choose among them.
