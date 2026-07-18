#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
tools_image=meigma-packages-phase0-tools:local
debian_image='debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd'
ubuntu_image='ubuntu:26.04@sha256:3131b4cc82a783df6c9df078f86e01819a13594b865c2cad47bd1bca2b7063bb'
fedora_image='fedora:44@sha256:6c75d5bf57cb0fa5aa4b92c6a83c86c791644496d9ac230de7711f5b8ec3b898'
run_id="phase0-$$"
network_name="$run_id"
server_name="$run_id-server"
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/meigma-phase0.XXXXXX")

cleanup() {
  docker rm -f "$server_name" >/dev/null 2>&1 || true
  docker network rm "$network_name" >/dev/null 2>&1 || true
  rm -rf -- "$work_dir"
}
trap cleanup EXIT

docker build --quiet --tag "$tools_image" \
  --build-arg "PHASE0_UID=$(id -u)" \
  --file "$script_dir/Dockerfile.tools" "$script_dir" >/dev/null
docker run --rm --volume "$work_dir:/work" "$tools_image" \
  build-repository /work/repo 1.0.0

docker network create "$network_name" >/dev/null
docker run --detach --rm \
  --name "$server_name" \
  --network "$network_name" \
  --network-alias phase0-repo \
  --volume "$work_dir/repo:/srv:ro" \
  "$tools_image" \
  python3 -m http.server 8080 --directory /srv >/dev/null

for image in "$debian_image" "$ubuntu_image"; do
  echo "testing APT client: $image"
  docker run --rm --network "$network_name" "$image" sh -ceu '
    export DEBIAN_FRONTEND=noninteractive
    apt-get update >/dev/null
    apt-get install -y --no-install-recommends ca-certificates curl >/dev/null
    install -d -m 0755 /etc/apt/keyrings
    curl -fsS http://phase0-repo:8080/meigma.asc -o /etc/apt/keyrings/meigma.asc
    cat > /etc/apt/sources.list.d/meigma.sources <<EOF
Types: deb
URIs: http://phase0-repo:8080/apt
Suites: stable
Components: phase0
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
    apt-get update -o Acquire::Languages=none >/dev/null
    apt-get install -y --no-install-recommends meigma-phase0 >/dev/null
    test "$(meigma-phase0)" = "meigma-phase0 1.0.0"
    echo "APT install passed: $(meigma-phase0)"
  '
done

echo "testing DNF client: $fedora_image"
  docker run --rm --network "$network_name" "$fedora_image" sh -ceu '
  curl -fsS http://phase0-repo:8080/rpm/phase0/meigma.repo \
    -o /etc/yum.repos.d/meigma-phase0.repo
  dnf -y --disablerepo="*" --enablerepo=meigma-phase0 \
    --setopt=install_weak_deps=False install meigma-phase0 >/dev/null
  test "$(meigma-phase0)" = "meigma-phase0 1.0.0"
  echo "DNF install passed: $(meigma-phase0)"
'

echo 'Phase 0 format and Ed25519 compatibility proof passed.'
