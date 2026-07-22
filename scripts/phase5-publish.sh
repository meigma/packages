#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
tools_image=meigma-packages-phase0-tools:local
debian_image='debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd'
ubuntu_image='ubuntu:26.04@sha256:3131b4cc82a783df6c9df078f86e01819a13594b865c2cad47bd1bca2b7063bb'
fedora_image='fedora:44@sha256:6c75d5bf57cb0fa5aa4b92c6a83c86c791644496d9ac230de7711f5b8ec3b898'
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/meigma-phase5-publish.XXXXXX")

cleanup() {
  rm -rf -- "$work_dir"
}
trap cleanup EXIT

required_variables=(
  PUBLICATION_TARGET
  PROJECT
  TAG
  GPG_SIGNING_SUBKEY
  GPG_PASSPHRASE
  GPG_PRIMARY_FINGERPRINT
  GPG_SIGNING_FINGERPRINT
  MEIGMA_PACKAGES_R2_ACCESS_KEY_ID
  MEIGMA_PACKAGES_R2_SECRET_ACCESS_KEY
  R2_BUCKET
  R2_ENDPOINT
  PUBLIC_BASE_URL
)
for variable in "${required_variables[@]}"; do
  if [[ -z ${!variable:-} ]]; then
    echo "$variable is required" >&2
    exit 2
  fi
done
if [[ "$PROJECT" != incus-gh-runner || "$TAG" != v1.0.0 ]]; then
  echo 'Phase 5 initial publication is confined to incus-gh-runner v1.0.0' >&2
  exit 2
fi

apply_arguments=(
  --bucket "$R2_BUCKET"
  --endpoint "$R2_ENDPOINT"
)
case "$PUBLICATION_TARGET" in
  staging)
    if [[ "${R2_PREFIX:-}" != '_staging/' ]]; then
      echo 'staging publication is confined to R2_PREFIX=_staging/' >&2
      exit 2
    fi
    if [[ "$PUBLIC_BASE_URL" != 'https://pkgs.meigma.dev/_staging' ]]; then
      echo 'staging publication requires the canonical staging public URL' >&2
      exit 2
    fi
    apply_arguments+=(--prefix "$R2_PREFIX")
    ;;
  production)
    if [[ -n "${R2_PREFIX:-}" ]]; then
      echo 'production publication requires an empty R2_PREFIX' >&2
      exit 2
    fi
    if [[ "$PUBLIC_BASE_URL" != 'https://pkgs.meigma.dev' ]]; then
      echo 'production publication requires the canonical public URL' >&2
      exit 2
    fi
    if [[ "${PRODUCTION_CONFIRMATION:-}" != 'publish incus-gh-runner v1.0.0 to production' ]]; then
      echo 'production publication requires the exact confirmation phrase' >&2
      exit 2
    fi
    apply_arguments+=(--production-root)
    ;;
  *)
    echo 'PUBLICATION_TARGET must be staging or production' >&2
    exit 2
    ;;
esac

mise exec -- go run ./cmd/meigma-packages fetch-release \
  --registry "$repo_root/projects.yml" \
  --project "$PROJECT" \
  --tag "$TAG" \
  --output "$work_dir/releases/$TAG" > "$work_dir/fetch-result.json"

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

printf '%s' "$GPG_SIGNING_SUBKEY" > "$work_dir/signing-subkey.asc"
chmod 0600 "$work_dir/signing-subkey.asc"
unset GPG_SIGNING_SUBKEY
printf '%s\n' "$GPG_PASSPHRASE" > "$work_dir/gpg-passphrase"
chmod 0600 "$work_dir/gpg-passphrase"
unset GPG_PASSPHRASE

docker run --rm \
  --volume "$work_dir:/work" \
  --env GPG_PRIMARY_FINGERPRINT \
  --env GPG_SIGNING_FINGERPRINT \
  "$tools_image" sh -ceu '
    export GNUPGHOME=/work/gnupg
    install -d -m 0700 "$GNUPGHOME"
    gpg --batch --import /work/signing-subkey.asc >/dev/null 2>&1
    primary=$(gpg --batch --with-colons --list-secret-keys \
      | awk -F: '\''$1 == "fpr" { print $10; exit }'\'')
    signing=$(gpg --batch --with-colons --list-secret-keys \
      | awk -F: '\''$1 == "ssb" { subkey = 1; next } subkey && $1 == "fpr" { print $10; exit }'\'')
    test "$primary" = "$GPG_PRIMARY_FINGERPRINT"
    test "$signing" = "$GPG_SIGNING_FINGERPRINT"
    gpg --batch --list-secret-keys "$primary" 2>/dev/null | grep -q "^sec#"
  '

GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
  mise exec -- go build -trimpath -o "$work_dir/meigma-packages" ./cmd/meigma-packages
cp "$repo_root/projects.yml" "$work_dir/projects.yml"

docker run --rm \
  --volume "$work_dir:/work" \
  "$tools_image" \
  /work/meigma-packages rebuild-local \
    --registry /work/projects.yml \
    --project "$PROJECT" \
    --releases /work/releases \
    --root /work/candidate \
    --gnupg-home /work/gnupg \
    --signing-key "$GPG_SIGNING_FINGERPRINT" \
    --gpg-passphrase-file /work/gpg-passphrase \
    --base-url "$PUBLIC_BASE_URL" > "$work_dir/rebuild-result.json"

if [[ "$PUBLICATION_TARGET" == staging && "${EMPTY_STAGING:-false}" == true ]]; then
  if [[ "${EMPTY_STAGING_CONFIRMATION:-}" != 'empty _staging only' ]]; then
    echo 'empty staging recovery requires the exact confirmation phrase' >&2
    exit 2
  fi
  mkdir "$work_dir/empty"
  mise exec -- go run ./cmd/meigma-packages apply-sync \
    --root "$work_dir/empty" \
    "${apply_arguments[@]}" > "$work_dir/empty-result.json"
  jq -e '.verified == true' "$work_dir/empty-result.json" >/dev/null
fi

mise exec -- go run ./cmd/meigma-packages apply-sync \
  --root "$work_dir/candidate" \
  "${apply_arguments[@]}" > "$work_dir/apply-result.json"
jq -e '.verified == true' "$work_dir/apply-result.json" >/dev/null

mise exec -- go run ./cmd/meigma-packages apply-sync \
  --root "$work_dir/candidate" \
  "${apply_arguments[@]}" > "$work_dir/no-op-result.json"
jq -e '.verified == true and .no_op == true and (.actions | length) == 0' \
  "$work_dir/no-op-result.json" >/dev/null

curl --fail --silent --show-error "$PUBLIC_BASE_URL/meigma.asc" \
  --output "$work_dir/public-meigma.asc"
public_fingerprint=$(gpg --batch --show-keys --with-colons "$work_dir/public-meigma.asc" \
  | awk -F: '$1 == "fpr" { print $10; exit }')
if [[ "$public_fingerprint" != "$GPG_PRIMARY_FINGERPRINT" ]]; then
  echo 'public signing-key fingerprint does not match the protected environment' >&2
  exit 1
fi

for image in "$debian_image" "$ubuntu_image"; do
  docker run --rm \
    --env "PUBLIC_BASE_URL=$PUBLIC_BASE_URL" \
    "$image" sh -ceu '
      export DEBIAN_FRONTEND=noninteractive
      apt-get update >/dev/null
      apt-get install -y --no-install-recommends ca-certificates curl >/dev/null
      install -d -m 0755 /etc/apt/keyrings
      curl -fsS "$PUBLIC_BASE_URL/meigma.asc" -o /etc/apt/keyrings/meigma.asc
      cat > /etc/apt/sources.list.d/meigma.sources <<EOF
Types: deb
URIs: $PUBLIC_BASE_URL/apt
Suites: stable
Components: incus-gh-runner
Signed-By: /etc/apt/keyrings/meigma.asc
EOF
      apt-get update -o Acquire::Languages=none >/dev/null
      apt-get install -y --no-install-recommends incus-gh-runner >/dev/null
      test "$(dpkg-query --show --showformat="\${Version}" incus-gh-runner)" = 1.0.0
    '
done

docker run --rm \
  --env "PUBLIC_BASE_URL=$PUBLIC_BASE_URL" \
  "$fedora_image" sh -ceu '
    curl -fsS "$PUBLIC_BASE_URL/rpm/incus-gh-runner/meigma.repo" \
      -o /etc/yum.repos.d/meigma.repo
    dnf -q --refresh install -y incus-gh-runner >/dev/null
    test "$(rpm --query --queryformat "%{VERSION}" incus-gh-runner)" = 1.0.0
  '

desired_state=$(jq -r '.desired_state_digest' "$work_dir/rebuild-result.json")
actions=$(jq -r '.actions | length' "$work_dir/apply-result.json")
echo "Phase 5 $PUBLICATION_TARGET desired state: $desired_state"
echo "Phase 5 $PUBLICATION_TARGET ordered R2 actions: $actions"
echo "Phase 5 $PUBLICATION_TARGET publication, verification, no-op, and clean installs passed."
