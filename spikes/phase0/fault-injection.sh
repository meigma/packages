#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
tools_image=meigma-packages-phase0-tools:local
debian_image='debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd'
fedora_image='fedora:44@sha256:6c75d5bf57cb0fa5aa4b92c6a83c86c791644496d9ac230de7711f5b8ec3b898'
run_id="phase0-fault-$$"
network_name="$run_id"
server_name="$run_id-server"
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/meigma-phase0-fault.XXXXXX")

cleanup() {
  docker rm -f "$server_name" >/dev/null 2>&1 || true
  docker network rm "$network_name" >/dev/null 2>&1 || true
  rm -rf -- "$work_dir"
}
trap cleanup EXIT

run_tools() {
  docker run --rm --volume "$work_dir:/work" "$tools_image" "$@"
}

apt_expect_version() {
  local expected=$1
  docker run --rm \
    --network "$network_name" \
    --volume "$work_dir/live/meigma.asc:/etc/apt/keyrings/meigma.asc:ro" \
    "$debian_image" sh -ceu '
      export DEBIAN_FRONTEND=noninteractive
      rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*
      cat > /etc/apt/sources.list.d/meigma.sources <<EOF
Types: deb
URIs: http://phase0-repo:8080/apt
Suites: stable
Components: phase0
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
      apt-get update -o Acquire::Languages=none >/dev/null
      apt-get install -y --no-install-recommends meigma-phase0 >/dev/null
      actual=$(meigma-phase0)
      test "$actual" = "meigma-phase0 $1"
      echo "APT sees $actual"
    ' sh "$expected"
}

dnf_expect_version() {
  local expected=$1
  docker run --rm \
    --network "$network_name" \
    --volume "$work_dir/live/rpm/phase0/meigma.repo:/etc/yum.repos.d/meigma-phase0.repo:ro" \
    "$fedora_image" sh -ceu '
      dnf -y --disablerepo="*" --enablerepo=meigma-phase0 \
        --setopt=install_weak_deps=False install meigma-phase0 >/dev/null
      actual=$(meigma-phase0)
      test "$actual" = "meigma-phase0 $1"
      echo "DNF sees $actual"
    ' sh "$expected"
}

dnf_expect_signature_failure() {
  local label=$1
  if docker run --rm \
    --network "$network_name" \
    --volume "$work_dir/live/rpm/phase0/meigma.repo:/etc/yum.repos.d/meigma-phase0.repo:ro" \
    "$fedora_image" sh -ceu '
      dnf -y --disablerepo="*" --enablerepo=meigma-phase0 \
        --setopt=install_weak_deps=False install meigma-phase0
    '; then
    echo "expected DNF signature rejection at: $label" >&2
    exit 1
  fi
  echo "DNF rejected mismatched activation objects: $label"
}

docker build --quiet --tag "$tools_image" --file "$script_dir/Dockerfile.tools" "$script_dir" >/dev/null
run_tools sh -ceu '
  build-repository /work/old 1.0.0
  build-repository /work/new 1.1.0
'

run_tools sh -ceu 'mkdir /work/live; cp -a /work/old/. /work/live/'
docker network create "$network_name" >/dev/null
docker run --detach --rm \
  --name "$server_name" \
  --network "$network_name" \
  --network-alias phase0-repo \
  --volume "$work_dir/live:/srv:ro" \
  "$tools_image" \
  python3 -m http.server 8080 --directory /srv >/dev/null

echo 'stage 0: old repository is fully active'
apt_expect_version 1.0.0
dnf_expect_version 1.0.0

echo 'stage 1: new packages and content-addressed metadata copied'
# The mounted tool container, not this host shell, expands the path variables.
# shellcheck disable=SC2016
run_tools sh -ceu '
  deb_arch=$(cat /work/old/deb-architecture.txt)
  cp /work/new/apt/pool/phase0/* /work/live/apt/pool/phase0/
  cp -a "/work/new/apt/dists/stable/phase0/binary-$deb_arch/by-hash/." \
    "/work/live/apt/dists/stable/phase0/binary-$deb_arch/by-hash/"
  cp /work/new/rpm/phase0/noarch/* /work/live/rpm/phase0/noarch/
  find /work/new/rpm/phase0/repodata -type f \
    ! -name repomd.xml ! -name repomd.xml.asc \
    -exec cp {} /work/live/rpm/phase0/repodata/ \;
'
apt_expect_version 1.0.0
dnf_expect_version 1.0.0

echo 'stage 2: mutable APT package indexes copied before activation'
# The mounted tool container, not this host shell, expands the path variables.
# shellcheck disable=SC2016
run_tools sh -ceu '
  deb_arch=$(cat /work/old/deb-architecture.txt)
  cp "/work/new/apt/dists/stable/phase0/binary-$deb_arch/Packages" \
    "/work/new/apt/dists/stable/phase0/binary-$deb_arch/Packages.gz" \
    "/work/live/apt/dists/stable/phase0/binary-$deb_arch/"
'
apt_expect_version 1.0.0
dnf_expect_version 1.0.0

echo 'stage 3: detached APT Release pair copied while old InRelease remains'
run_tools sh -ceu '
  cp /work/new/apt/dists/stable/Release \
    /work/new/apt/dists/stable/Release.gpg \
    /work/live/apt/dists/stable/
'
apt_expect_version 1.0.0
dnf_expect_version 1.0.0

echo 'stage 4: atomic InRelease activation copied'
run_tools cp /work/new/apt/dists/stable/InRelease /work/live/apt/dists/stable/InRelease
apt_expect_version 1.1.0
dnf_expect_version 1.0.0

echo 'stage 5a: new repomd.xml copied before its detached signature'
run_tools cp /work/new/rpm/phase0/repodata/repomd.xml /work/live/rpm/phase0/repodata/repomd.xml
dnf_expect_signature_failure 'new repomd.xml with old repomd.xml.asc'
run_tools cp /work/new/rpm/phase0/repodata/repomd.xml.asc /work/live/rpm/phase0/repodata/repomd.xml.asc
dnf_expect_version 1.1.0

echo 'stage 5b: reset, then copy the detached signature before repomd.xml'
run_tools sh -ceu '
  cp /work/old/rpm/phase0/repodata/repomd.xml \
    /work/old/rpm/phase0/repodata/repomd.xml.asc \
    /work/live/rpm/phase0/repodata/
  cp /work/new/rpm/phase0/repodata/repomd.xml.asc \
    /work/live/rpm/phase0/repodata/repomd.xml.asc
'
dnf_expect_signature_failure 'new repomd.xml.asc with old repomd.xml'
run_tools cp /work/new/rpm/phase0/repodata/repomd.xml /work/live/rpm/phase0/repodata/repomd.xml
dnf_expect_version 1.1.0

echo 'Fault-injection result: APT remained installable; both RPM activation orders had a signature-failure window.'
