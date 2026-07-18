#!/usr/bin/env bash
set -euo pipefail

repo_root=${1:?repository output path is required}
version=${2:?package version is required}

case "$repo_root" in
  /work/*) ;;
  *)
    echo "repository output must be below /work" >&2
    exit 2
    ;;
esac

export GNUPGHOME=/work/gnupg
key_uid='Meigma Phase 0 <phase0@meigma.dev>'
package_name=meigma-phase0
deb_arch=$(dpkg --print-architecture)

rm -rf -- "$repo_root"
mkdir -p "$repo_root" "$GNUPGHOME" /work/build/payload
chmod 0700 "$GNUPGHOME"

if ! gpg --batch --list-secret-keys "$key_uid" >/dev/null 2>&1; then
  gpg --batch --pinentry-mode loopback --passphrase '' \
    --quick-generate-key "$key_uid" ed25519 cert 0
  primary_fingerprint=$(gpg --batch --with-colons --list-secret-keys "$key_uid" \
    | awk -F: '$1 == "fpr" { print $10; exit }')
  gpg --batch --pinentry-mode loopback --passphrase '' \
    --quick-add-key "$primary_fingerprint" ed25519 sign 0
fi

primary_fingerprint=$(gpg --batch --with-colons --list-secret-keys "$key_uid" \
  | awk -F: '$1 == "fpr" { print $10; exit }')
signing_fingerprint=$(gpg --batch --with-colons --list-secret-keys "$key_uid" \
  | awk -F: '$1 == "ssb" { subkey = 1; next } subkey && $1 == "fpr" { print $10; exit }')

if [[ -z "$primary_fingerprint" || -z "$signing_fingerprint" ]]; then
  echo 'failed to resolve the Ed25519 primary or signing-subkey fingerprint' >&2
  exit 1
fi

cat > /work/build/payload/meigma-phase0 <<EOF
#!/bin/sh
echo 'meigma-phase0 ${version}'
EOF
chmod 0755 /work/build/payload/meigma-phase0

deb_root=/work/build/deb-root
rm -rf -- "$deb_root"
install -D -m 0755 /work/build/payload/meigma-phase0 "$deb_root/usr/bin/meigma-phase0"
mkdir -p "$deb_root/DEBIAN"
cat > "$deb_root/DEBIAN/control" <<EOF
Package: ${package_name}
Version: ${version}
Section: utils
Priority: optional
Architecture: all
Maintainer: Meigma <phase0@meigma.dev>
Description: Disposable package fixture for the Meigma Phase 0 proof
EOF

apt_root="$repo_root/apt"
binary_dir="dists/stable/phase0/binary-${deb_arch}"
mkdir -p "$apt_root/pool/phase0" "$apt_root/$binary_dir"
dpkg-deb --root-owner-group --build "$deb_root" \
  "$apt_root/pool/phase0/${package_name}_${version}_all.deb" >/dev/null

rpm_top=/work/build/rpmbuild
rm -rf -- "$rpm_top"
mkdir -p "$rpm_top"/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
cat > "$rpm_top/SPECS/${package_name}.spec" <<EOF
Name: ${package_name}
Version: ${version}
Release: 1
Summary: Disposable package fixture for the Meigma Phase 0 proof
License: MIT
BuildArch: noarch

%description
Disposable package fixture for proving repository metadata and signatures.

%install
install -D -m 0755 /work/build/payload/meigma-phase0 %{buildroot}/usr/bin/meigma-phase0

%files
/usr/bin/meigma-phase0
EOF
rpmbuild --define "_topdir $rpm_top" -bb "$rpm_top/SPECS/${package_name}.spec" >/dev/null 2>&1

rpm_root="$repo_root/rpm/phase0"
mkdir -p "$rpm_root/noarch"
cp "$rpm_top/RPMS/noarch/${package_name}-${version}-1.noarch.rpm" "$rpm_root/noarch/"

(
  cd "$apt_root"
  apt-ftparchive packages pool/phase0 > "$binary_dir/Packages"
  gzip -9n -c "$binary_dir/Packages" > "$binary_dir/Packages.gz"

  for index in Packages Packages.gz; do
    index_path="$binary_dir/$index"
    for algorithm in SHA256 SHA512; do
      case "$algorithm" in
        SHA256) digest=$(sha256sum "$index_path" | awk '{ print $1 }') ;;
        SHA512) digest=$(sha512sum "$index_path" | awk '{ print $1 }') ;;
      esac
      mkdir -p "$binary_dir/by-hash/$algorithm"
      cp "$index_path" "$binary_dir/by-hash/$algorithm/$digest"
    done
  done

  apt-ftparchive \
    -o APT::FTPArchive::Release::Origin=Meigma \
    -o APT::FTPArchive::Release::Label=Meigma \
    -o APT::FTPArchive::Release::Suite=stable \
    -o APT::FTPArchive::Release::Codename=stable \
    -o APT::FTPArchive::Release::Architectures="$deb_arch" \
    -o APT::FTPArchive::Release::Components=phase0 \
    -o APT::FTPArchive::Release::Acquire-By-Hash=yes \
    -o APT::FTPArchive::Release::Description='Meigma Phase 0 repository' \
    release dists/stable > dists/stable/Release

  gpg --batch --yes --local-user "${signing_fingerprint}!" --armor --detach-sign \
    --output dists/stable/Release.gpg dists/stable/Release
  gpg --batch --yes --local-user "${signing_fingerprint}!" --clearsign \
    --output dists/stable/InRelease dists/stable/Release
)

createrepo_c "$rpm_root" >/dev/null
gpg --batch --yes --local-user "${signing_fingerprint}!" --armor --detach-sign \
  --output "$rpm_root/repodata/repomd.xml.asc" "$rpm_root/repodata/repomd.xml"

cat > "$rpm_root/meigma.repo" <<'EOF'
[meigma-phase0]
name=Meigma Phase 0
baseurl=http://phase0-repo:8080/rpm/phase0
enabled=1
repo_gpgcheck=1
gpgcheck=0
gpgkey=http://phase0-repo:8080/meigma.asc
EOF

gpg --batch --armor --export "$primary_fingerprint" > "$repo_root/meigma.asc"
printf '%s\n' "$primary_fingerprint" > "$repo_root/primary-fingerprint.txt"
printf '%s\n' "$signing_fingerprint" > "$repo_root/signing-fingerprint.txt"
printf '%s\n' "$deb_arch" > "$repo_root/deb-architecture.txt"

verify_home=/work/verify-gnupg
rm -rf -- "$verify_home"
mkdir -m 0700 "$verify_home"
GNUPGHOME="$verify_home" gpg --batch --import "$repo_root/meigma.asc" >/dev/null 2>&1
GNUPGHOME="$verify_home" gpg --batch --verify \
  "$apt_root/dists/stable/InRelease" >/dev/null 2>&1
GNUPGHOME="$verify_home" gpg --batch --verify \
  "$apt_root/dists/stable/Release.gpg" "$apt_root/dists/stable/Release" >/dev/null 2>&1
GNUPGHOME="$verify_home" gpg --batch --verify \
  "$rpm_root/repodata/repomd.xml.asc" "$rpm_root/repodata/repomd.xml" >/dev/null 2>&1

printf 'built version=%s deb_arch=%s primary=%s signing=%s\n' \
  "$version" "$deb_arch" "$primary_fingerprint" "$signing_fingerprint"
