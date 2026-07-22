#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
tools_image=meigma-packages-phase0-tools:local
debian_image='debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd'
fedora_image='fedora:44@sha256:6c75d5bf57cb0fa5aa4b92c6a83c86c791644496d9ac230de7711f5b8ec3b898'
run_id="phase5-source-$$"
network_name="$run_id"
server_name="$run_id-server"
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/meigma-phase5.XXXXXX")

cleanup() {
  docker container rm --force "$server_name" >/dev/null 2>&1 || true
  docker network rm "$network_name" >/dev/null 2>&1 || true
  rm -rf -- "$work_dir"
}
trap cleanup EXIT

command -v gh >/dev/null

mise exec -- go run ./cmd/meigma-packages fetch-release \
  --registry "$repo_root/projects.yml" \
  --project incus-gh-runner \
  --tag v1.0.0 \
  --output "$work_dir/releases/v1.0.0" > "$work_dir/fetch-result.json"

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

docker run --rm --volume "$work_dir:/work" "$tools_image" \
  build-repository /work/signing-source 0.0.1 >/dev/null

GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
  mise exec -- go build -trimpath -o "$work_dir/meigma-packages" ./cmd/meigma-packages
cp "$repo_root/projects.yml" "$work_dir/projects.yml"
signing_key=$(tr -d '\n' < "$work_dir/signing-source/signing-fingerprint.txt")

docker run --rm --volume "$work_dir:/work" "$tools_image" \
  /work/meigma-packages rebuild-local \
    --registry /work/projects.yml \
    --project incus-gh-runner \
    --releases /work/releases \
    --root /work/candidate \
    --gnupg-home /work/gnupg \
    --signing-key "$signing_key" \
    --base-url http://phase5-repo:8080 > "$work_dir/rebuild-result.json"

docker run --rm --interactive --volume "$work_dir:/work" "$tools_image" python3 - <<'PY'
import json
from pathlib import Path

root = Path('/work')
fetch = json.loads((root / 'fetch-result.json').read_text())
manifest = json.loads((root / 'candidate/_state/manifest.json').read_text())

assert len(fetch['assets']) == 5
assert manifest['selected_versions'] == ['v1.0.0']
assert len(manifest['packages']) == 4
assert {
    (package['format'], package['repository_architecture'])
    for package in manifest['packages']
} == {
    ('deb', 'amd64'),
    ('deb', 'arm64'),
    ('rpm', 'amd64'),
    ('rpm', 'arm64'),
}
for architecture in ('amd64', 'arm64'):
    assert (
        root
        / 'candidate/apt/dists/stable/incus-gh-runner'
        / f'binary-{architecture}/Packages'
    ).is_file()
for architecture in ('x86_64', 'aarch64'):
    assert (root / f'candidate/rpm/incus-gh-runner/{architecture}').is_dir()
PY

docker network create "$network_name" >/dev/null
docker run --detach --rm \
  --name "$server_name" \
  --network "$network_name" \
  --network-alias phase5-repo \
  --volume "$work_dir/candidate:/srv:ro" \
  "$tools_image" \
  python3 -m http.server 8080 --directory /srv >/dev/null

docker run --rm --network "$network_name" "$debian_image" sh -ceu '
  export DEBIAN_FRONTEND=noninteractive
  apt-get update >/dev/null
  apt-get install -y --no-install-recommends ca-certificates curl >/dev/null
  install -d -m 0755 /etc/apt/keyrings
  curl -fsS http://phase5-repo:8080/meigma.asc -o /etc/apt/keyrings/meigma.asc
  cat > /etc/apt/sources.list.d/meigma.sources <<EOF
Types: deb
URIs: http://phase5-repo:8080/apt
Suites: stable
Components: incus-gh-runner
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
  apt-get update -o Acquire::Languages=none >/dev/null
  apt-get install -y --no-install-recommends incus-gh-runner >/dev/null
  test "$(dpkg-query --show --showformat="\${Version}" incus-gh-runner)" = 1.0.0
'

docker run --rm --network "$network_name" "$fedora_image" sh -ceu '
  curl -fsS http://phase5-repo:8080/rpm/incus-gh-runner/meigma.repo \
    -o /etc/yum.repos.d/meigma.repo
  dnf -q --refresh install -y incus-gh-runner >/dev/null
  test "$(rpm --query --queryformat "%{VERSION}" incus-gh-runner)" = 1.0.0
'

echo 'Phase 5 real release discovery, rebuild, and clean DEB/RPM installs passed.'
