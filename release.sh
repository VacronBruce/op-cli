#!/bin/bash
set -e

# Release script for op-cli
# Usage: bash release.sh v0.4.0

if [ -z "$1" ]; then
  echo "Usage: bash release.sh <version>"
  echo "Example: bash release.sh v0.4.0"
  exit 1
fi

VERSION="$1"
REPO="VacronBruce/op-cli"

echo "=== Releasing op-cli ${VERSION} ==="
echo ""

# Step 1: Build
echo "1/4 Building binaries..."
LDFLAGS="-X github.com/chenhuijun/op-cli/cmd.Version=${VERSION#v}"
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/op-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/op-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/op-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/op-windows-amd64.exe .
# Ad-hoc sign the darwin binaries with a stable identifier so macOS tools
# (Gatekeeper, Little Snitch) see a named code identity instead of "a.out".
# Note: the identity hash still changes per build — only a real Developer ID
# certificate makes it stable across versions.
if command -v codesign &>/dev/null; then
  codesign -s - -f -i com.gmedtn.op-cli dist/op-darwin-arm64 dist/op-darwin-amd64
  echo "    Signed darwin binaries (ad-hoc)."
fi
echo "    Done."

# Step 2: Update version in install.sh + the plugin manifest
echo "2/4 Updating versions..."
sed -i '' "s/^VERSION=.*/VERSION=\"${VERSION#v}\"/" install.sh
sed -i '' "s/^\$Version   = .*/\$Version   = \"${VERSION#v}\"/" install.ps1
sed -i '' "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION#v}\"/" .claude-plugin/plugin.json
echo "    Done."

# Step 3: Commit + tag + push
echo "3/4 Committing and tagging..."
git add -A
git commit -m "release: ${VERSION}" || true
git tag "${VERSION}"
git push origin develop --tags

# Step 4: Create GitHub release with the binaries + installer attached.
# GitHub serves the newest release at /releases/latest/download/<asset>, so
# install.sh always fetches the current version without any per-release edits.
echo "4/4 Creating GitHub release..."
gh release create "${VERSION}" \
  --repo "${REPO}" \
  --target develop \
  --title "${VERSION}" \
  --notes "## op-cli ${VERSION}

### Install
\`\`\`bash
bash <(curl -fsSL https://github.com/${REPO}/releases/latest/download/install.sh)
\`\`\`
" \
  dist/op-darwin-arm64 \
  dist/op-darwin-amd64 \
  dist/op-linux-amd64 \
  dist/op-windows-amd64.exe \
  install.sh \
  install.ps1

echo ""
echo "=== Released ${VERSION} ==="
echo "https://github.com/${REPO}/releases/tag/${VERSION}"
