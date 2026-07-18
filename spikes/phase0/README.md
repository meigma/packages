# Phase 0 format and consistency spike

This directory is intentionally disposable. It proves real package-manager and
signing behavior before any of it is promoted into the durable Go publisher.

The first increment builds a tiny `all`/`noarch` fixture package, generates APT
and RPM repositories, publishes both SHA-256 and SHA-512 APT by-hash indexes,
signs their metadata with a throwaway Ed25519 signing
subkey, serves the tree over an isolated Docker network, and installs the
fixture from clean current Debian, Ubuntu LTS, and Fedora containers.

Run it with:

```sh
./spikes/phase0/run.sh
./spikes/phase0/fault-injection.sh
```

The second command publishes a `1.0.0` snapshot, incrementally copies a `1.1.0`
snapshot in the proposed order, and starts clean APT and DNF clients after each
interruption point. It also exercises both possible orderings of RPM's mutable
`repomd.xml` and detached `repomd.xml.asc` activation pair.

All package tools and generated key material stay inside temporary container
state. The script removes that state when it exits. Production R2, signing
keys, and GitHub credentials are neither required nor accepted.
