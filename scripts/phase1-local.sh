#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
tools_image=meigma-packages-phase0-tools:local
debian_image='debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd'
run_id="phase1-local-$$"
network_name="$run_id"
server_name="$run_id-server"
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/meigma-phase1.XXXXXX")

cleanup() {
  docker container rm --force "$server_name" >/dev/null 2>&1 || true
  docker network rm "$network_name" >/dev/null 2>&1 || true
  rm -rf -- "$work_dir"
}
trap cleanup EXIT

docker_arch=$(docker info --format '{{.Architecture}}')
case "$docker_arch" in
  aarch64 | arm64) go_arch=arm64 ;;
  x86_64 | amd64) go_arch=amd64 ;;
  *)
    echo "unsupported Docker architecture: $docker_arch" >&2
    exit 1
    ;;
esac

docker build --quiet --tag "$tools_image" \
  --build-arg "PHASE0_UID=$(id -u)" \
  --file "$repo_root/spikes/phase0/Dockerfile.tools" \
  "$repo_root/spikes/phase0" >/dev/null

docker run --rm --volume "$work_dir:/work" "$tools_image" sh -ceu '
  build-repository /work/source 1.0.0
  mkdir /work/release
  cp /work/source/apt/pool/phase0/*.deb /work/release/
  cp /work/source/rpm/phase0/noarch/*.rpm /work/release/
'

GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
  mise exec -- go build -trimpath -o "$work_dir/meigma-packages" ./cmd/meigma-packages
signing_key=$(tr -d '\n' < "$work_dir/source/signing-fingerprint.txt")

docker run --rm \
  --volume "$work_dir:/work" \
  --volume "$repo_root/testdata/phase1:/fixtures:ro" \
  "$tools_image" \
  /work/meigma-packages build-local \
    --registry /fixtures/projects.yml \
    --project phase1-fixture \
    --release /work/release \
    --root /work/candidate \
    --gnupg-home /work/gnupg \
    --signing-key "$signing_key" \
    --base-url http://phase1-repo:8080

docker network create "$network_name" >/dev/null
docker run --detach --rm \
  --name "$server_name" \
  --network "$network_name" \
  --network-alias phase1-repo \
  --volume "$work_dir/candidate:/srv:ro" \
  "$tools_image" \
  python3 -m http.server 8080 --directory /srv >/dev/null

docker run --rm --network "$network_name" "$debian_image" sh -ceu '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update >/dev/null
  apt-get install -y --no-install-recommends ca-certificates curl >/dev/null
  install -d -m 0755 /etc/apt/keyrings
  curl -fsS http://phase1-repo:8080/meigma.asc -o /etc/apt/keyrings/meigma.asc
  cat > /etc/apt/sources.list.d/meigma.sources <<EOF
Types: deb
URIs: http://phase1-repo:8080/apt
Suites: stable
Components: phase1-fixture
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
  apt-get update -o Acquire::Languages=none >/dev/null
  apt-get install -y --no-install-recommends meigma-phase0 >/dev/null
  test "$(meigma-phase0)" = "meigma-phase0 1.0.0"
  echo "Phase 1 clean install passed: $(meigma-phase0)"
'

echo 'Phase 1 local vertical slice passed.'
