# Phase 0 format and consistency spike

This directory is intentionally disposable. It proves real package-manager and
signing behavior before any of it is promoted into the durable Go publisher.

The first increment builds a tiny `all`/`noarch` fixture package, generates APT
and RPM repositories, signs their metadata with a throwaway Ed25519 signing
subkey, serves the tree over an isolated Docker network, and installs the
fixture from clean current Debian, Ubuntu LTS, and Fedora containers.

Run it with:

```sh
./spikes/phase0/run.sh
```

All package tools and generated key material stay inside temporary container
state. The script removes that state when it exits. Production R2, signing
keys, and GitHub credentials are neither required nor accepted.
