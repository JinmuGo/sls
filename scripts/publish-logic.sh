#!/bin/bash
set -e

# Install dependencies
echo "Installing dependencies..."
apt-get update > /dev/null
apt-get install -y dpkg-dev createrepo-c jq > /dev/null

echo "Setting up test directories..."
rm -rf package-repo-test
mkdir -p package-repo-test

# Environment variables (simulating workflow env)
export PACKAGE_NAME="sls"
export PACKAGE_REPO="jinmugo/sls"
export PACKAGE_DESC="A smart fuzzy CLI selector for SSH config hosts"

echo "--- Testing APT Repository Setup ---"
PKG_NAME=$PACKAGE_NAME
FIRST_LETTER=$(echo "$PKG_NAME" | cut -c1)

mkdir -p "package-repo-test/deb/pool/main/${FIRST_LETTER}/${PKG_NAME}"
mkdir -p package-repo-test/deb/dists/stable/main/binary-amd64
mkdir -p package-repo-test/deb/dists/stable/main/binary-arm64

echo "Copying deb packages..."
# Ensure artifacts exist
if ! ls /artifacts/sls_*.deb >/dev/null 2>&1; then
    echo "No deb files found in /artifacts!"
    ls -la /artifacts
    exit 1
fi
cp /artifacts/*.deb "package-repo-test/deb/pool/main/${FIRST_LETTER}/${PKG_NAME}/"

cd package-repo-test/deb

echo "Generating Packages files..."
dpkg-scanpackages --arch amd64 pool/ > dists/stable/main/binary-amd64/Packages
gzip -k -f dists/stable/main/binary-amd64/Packages

dpkg-scanpackages --arch arm64 pool/ > dists/stable/main/binary-arm64/Packages
gzip -k -f dists/stable/main/binary-arm64/Packages

echo "Generating Release file..."
cd dists/stable
cat > Release << EOF
Origin: jinmugo
Label: jinmugo
Suite: stable
Codename: stable
Architectures: amd64 arm64
Components: main
Description: jinmugo package repository
EOF
echo "Date: $(date -Ru)" >> Release

# Add checksums to Release file
{
    echo "MD5Sum:"
    find main -type f \( -name "Packages*" \) -exec sh -c 'echo " $(md5sum "$1" | cut -d" " -f1) $(stat -c%s "$1") $1"' _ {} \;
    echo "SHA256:"
    find main -type f \( -name "Packages*" \) -exec sh -c 'echo " $(sha256sum "$1" | cut -d" " -f1) $(stat -c%s "$1") $1"' _ {} \;
} >> Release

cat Release

echo "APT setup complete."

cd /workspace

echo "--- Testing RPM Repository Setup ---"
mkdir -p package-repo-test/rpm/packages

echo "Copying rpm packages..."
cp /artifacts/*.rpm package-repo-test/rpm/packages/

cd package-repo-test/rpm
echo "Generating repository metadata..."
createrepo_c .

echo "RPM setup complete."

cd /workspace/package-repo-test

echo "--- Testing packages.json update ---"
echo '{"packages":{}}' > packages.json

PKG_TAG="v0.3.0-test"

# Debug variables
echo "Debug: PACKAGE_NAME=$PACKAGE_NAME"
echo "Debug: PKG_TAG=$PKG_TAG"

cat packages.json | jq \
    --arg name "$PACKAGE_NAME" \
    --arg version "$PKG_TAG" \
    --arg repo "$PACKAGE_REPO" \
    --arg desc "$PACKAGE_DESC" \
    '.packages[$name] = {"version": $version, "repo": $repo, "description": $desc, "updated": now | strftime("%Y-%m-%dT%H:%M:%SZ")}' \
    > packages.json.tmp && mv packages.json.tmp packages.json

cat packages.json

echo "All tests passed!"
