#!/bin/sh
# Install sos. Usage:
#   curl -fsSL https://raw.githubusercontent.com/LeoManrique/semi-open-source/main/install.sh | sh
# Stop at the first failed command or unset variable.
set -eu

REPO="LeoManrique/semi-open-source"
# A BINDIR from the environment wins, else /usr/local/bin.
BINDIR="${BINDIR:-/usr/local/bin}"

# Darwin and Linux become darwin and linux, as in the asset names.
os=$(uname -s | tr '[:upper:]' '[:lower:]')
# Map uname's names to Go's: x86_64 is amd64 and aarch64 is arm64.
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

# Red Hat's sosreport installs /usr/sbin/sos. Where it exists, install as sosx
# and leave the short name alone rather than shadowing a system tool.
name="sos"
if command -v sosreport >/dev/null 2>&1 || [ -x /usr/sbin/sos ]; then
  name="sosx"
  echo "sosreport detected, installing as 'sosx' to leave /usr/sbin/sos alone"
fi

# Keep only the value of "tag_name", the newest release's tag.
version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
[ -n "$version" ] || { echo "could not determine the latest release" >&2; exit 1; }

asset="sos_${version}_${os}_${arch}"
# Download into a temporary directory, removed however the script exits.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading $asset"
base="https://github.com/$REPO/releases/download/$version"
# -f fails on an HTTP error instead of saving the error page as the binary.
curl -fsSL -o "$tmp/$name" "$base/$asset"

# Verify against the published checksums before making it executable.
curl -fsSL -o "$tmp/SHA256SUMS" "$base/SHA256SUMS"
# Match the name at the end of the line, so no other asset's line is taken.
want=$(grep " $asset\$" "$tmp/SHA256SUMS" | cut -d' ' -f1)
# No checksum published means no install, never an unverified one.
[ -n "$want" ] || { echo "no checksum published for $asset" >&2; exit 1; }

# Linux ships sha256sum, macOS ships shasum.
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$tmp/$name" | cut -d' ' -f1)
else
  got=$(shasum -a 256 "$tmp/$name" | cut -d' ' -f1)
fi
[ "$want" = "$got" ] || { echo "checksum mismatch for $asset" >&2; exit 1; }

chmod +x "$tmp/$name"

# A fresh Mac may have no /usr/local/bin yet.
mkdir -p "$BINDIR" 2>/dev/null || sudo mkdir -p "$BINDIR"

# Use sudo only when BINDIR isn't writable.
if [ -w "$BINDIR" ]; then
  mv "$tmp/$name" "$BINDIR/$name"
else
  sudo mv "$tmp/$name" "$BINDIR/$name"
fi

echo "installed $BINDIR/$name"
# Run it once, to show the installed binary works.
"$BINDIR/$name" version
