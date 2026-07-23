---
title: Install Meigma packages
description: Add the signed Meigma APT or RPM repository and install incus-gh-runner.
---

# Install Meigma packages

Use the Meigma package repository to install Meigma software and receive
updates through your operating system's package manager.

The examples below install `incus-gh-runner`. For any other published
project, substitute its name in the same places: each project is its own APT
component (the `Components:` line) and its own RPM path
(`rpm/<project>/meigma.repo`), and its packages install by the project's
package name. The full list of published projects, versions, and packages is
at [`https://pkgs.meigma.dev/_state/manifest.json`](https://pkgs.meigma.dev/_state/manifest.json).

## Verify the signing key

Download the repository key and verify its full primary fingerprint before
installing it:

```sh
curl -fsSL https://pkgs.meigma.dev/meigma.asc -o /tmp/meigma.asc
gpg --show-keys --with-colons /tmp/meigma.asc \
  | awk -F: '$1 == "fpr" { print $10; exit }'
```

The command must print:

```text
9C74476A669465EEB8D46AD8B0E68773B6E259F6
```

Stop if it prints a different fingerprint.

## Debian and Ubuntu

Install the verified key and add the APT source:

```sh
sudo install -d -m 0755 /etc/apt/keyrings
sudo install -m 0644 /tmp/meigma.asc /etc/apt/keyrings/meigma.asc
sudo tee /etc/apt/sources.list.d/meigma.sources >/dev/null <<'EOF'
Types: deb
URIs: https://pkgs.meigma.dev/apt
Suites: stable
Components: incus-gh-runner
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
sudo apt update
sudo apt install incus-gh-runner
```

## Fedora

Install the repository definition, then install the package:

```sh
sudo curl -fsSL \
  https://pkgs.meigma.dev/rpm/incus-gh-runner/meigma.repo \
  -o /etc/yum.repos.d/meigma.repo
sudo dnf --refresh install incus-gh-runner
```

DNF verifies repository metadata with the Meigma key. The package itself is
also verified with the checksums in that signed metadata.

## Remove the package repository

On Debian or Ubuntu:

```sh
sudo apt remove incus-gh-runner
sudo rm /etc/apt/sources.list.d/meigma.sources
sudo rm /etc/apt/keyrings/meigma.asc
sudo apt update
```

On Fedora:

```sh
sudo dnf remove incus-gh-runner
sudo rm /etc/yum.repos.d/meigma.repo
```
