#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
tools_image=meigma-packages-phase0-tools:local
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/meigma-phase2.XXXXXX")

cleanup() {
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
  for version in 1.0.0 1.1.0 2.0.0; do
    build-repository /work/source "$version" >/dev/null
    release_dir="/work/releases/v$version"
    mkdir -p "$release_dir"
    cp /work/source/apt/pool/phase0/*.deb "$release_dir/"
    cp /work/source/rpm/phase0/noarch/*.rpm "$release_dir/"
    (
      cd "$release_dir"
      sha256sum -- *.deb *.rpm > checksums.txt
    )
  done
'

GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
  mise exec -- go build -trimpath -o "$work_dir/meigma-packages" ./cmd/meigma-packages
cp "$repo_root/testdata/phase2/projects.yml" "$work_dir/projects-retain2.yml"
sed 's/retention: 2/retention: 3/' \
  "$repo_root/testdata/phase2/projects.yml" > "$work_dir/projects-retain3.yml"
signing_key=$(tr -d '\n' < "$work_dir/source/signing-fingerprint.txt")

run_rebuild() {
  local registry=$1
  local root=$2
  local output=$3

  docker run --rm --volume "$work_dir:/work" --entrypoint sh "$tools_image" -ceu '
    /work/meigma-packages rebuild-local \
      --registry "/work/$1" \
      --project phase2-fixture \
      --releases /work/releases \
      --root "/work/$2" \
      --gnupg-home /work/gnupg \
      --signing-key "$4" \
      --base-url http://phase2-repo:8080 > "/work/$3"
  ' sh "$registry" "$root" "$output" "$signing_key"
}

run_rebuild projects-retain3.yml remote remote-result.json
run_rebuild projects-retain2.yml candidate candidate-result.json
run_rebuild projects-retain2.yml candidate no-op-result.json
run_rebuild projects-retain2.yml rebuilt rebuilt-result.json

docker run --rm --volume "$work_dir:/work" --entrypoint sh "$tools_image" -ceu '
  /work/meigma-packages plan-sync \
    --root /work/candidate \
    --remote /work/remote > /work/sync-plan.json
'

docker run --rm --interactive --volume "$work_dir:/work" "$tools_image" python3 - <<'PY'
import json
from pathlib import Path

work = Path('/work')
candidate = json.loads((work / 'candidate-result.json').read_text())
noop = json.loads((work / 'no-op-result.json').read_text())
rebuilt = json.loads((work / 'rebuilt-result.json').read_text())
plan = json.loads((work / 'sync-plan.json').read_text())

assert candidate['selected_versions'] == ['v2.0.0', 'v1.1.0']
assert candidate['no_op'] is False
assert noop['no_op'] is True
assert noop['desired_state_digest'] == candidate['desired_state_digest']
assert rebuilt['desired_state_digest'] == candidate['desired_state_digest']

actions = plan['actions']
delete_indexes = [index for index, action in enumerate(actions) if action['kind'] == 'delete']
assert delete_indexes
assert delete_indexes == list(range(delete_indexes[0], len(actions)))
assert any('1.0.0' in action['path'] for action in actions if action['kind'] == 'delete')

manifest = json.loads((work / 'candidate/_state/manifest.json').read_text())
assert len(manifest['packages']) == 4
print('Phase 2 retained versions:', ', '.join(candidate['selected_versions']))
print('Phase 2 desired state:', candidate['desired_state_digest'])
print('Phase 2 ordered actions:', len(actions))
PY

mise exec -- go test ./internal/localrepo \
  -run 'TestPlanSyncFailurePointsNeverDeleteCandidateContent' -count=1

echo 'Phase 2 deterministic rebuild and sync-planning proof passed.'
