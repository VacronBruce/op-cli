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
REPO="gmedtn/op-cli"
PKG_BASE="https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli"

echo "=== Releasing op-cli ${VERSION} ==="
echo ""

# Step 1: Build
echo "1/4 Building binaries..."
LDFLAGS="-X github.com/chenhuijun/op-cli/cmd.Version=${VERSION#v}"
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/op-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/op-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/op-linux-amd64 .
echo "    Done."

# Step 2: Update version in install.sh + the plugin manifest
echo "2/4 Updating versions..."
sed -i '' "s/^VERSION=.*/VERSION=\"${VERSION#v}\"/" install.sh
sed -i '' "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION#v}\"/" .claude-plugin/plugin.json
echo "    Done."

# Step 3: Commit + tag + push
echo "3/4 Committing and tagging..."
git add -A
git commit -m "release: ${VERSION}" || true
git tag "${VERSION}"
git push origin develop --tags

# Step 4: Upload to package registry (both versioned + latest)
echo "4/4 Uploading to package registry..."
for path in "${VERSION#v}" "latest"; do
  for file in dist/op-darwin-arm64 dist/op-darwin-amd64 dist/op-linux-amd64 install.sh; do
    name=$(basename $file)
    curl -s --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" \
      --upload-file "$file" \
      "${PKG_BASE}/${path}/${name}?status=default" > /dev/null
    echo "    Uploaded ${name} → ${path}"
  done
done

# Step 5: Create GitLab release
echo ""
echo "Creating GitLab release..."
GITLAB_HOST=gitlab-tw.ddns.net glab release create "${VERSION}" \
  --repo "${REPO}" \
  --ref develop \
  --name "${VERSION}" \
  --notes "## op-cli ${VERSION}

### Install
\`\`\`bash
mkdir -p /tmp/op-cli && cd /tmp/op-cli && GITLAB_HOST=gitlab-tw.ddns.net glab release download --repo gmedtn/op-cli --include-external --asset-name=\"install.sh\" && bash install.sh
\`\`\`

Or click **install.sh** on the release page and run: \`bash ~/Downloads/install.sh\`
"

# Step 6: Add package registry links to release
echo "Adding download links..."
BASE_API="https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/releases/${VERSION}/assets/links"
PKG="${PKG_BASE}/latest"

curl -s -X POST --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" --header "Content-Type: application/json" \
  -d "{\"name\":\"op-darwin-arm64\",\"url\":\"${PKG}/op-darwin-arm64\",\"link_type\":\"package\"}" "$BASE_API" > /dev/null
curl -s -X POST --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" --header "Content-Type: application/json" \
  -d "{\"name\":\"op-darwin-amd64\",\"url\":\"${PKG}/op-darwin-amd64\",\"link_type\":\"package\"}" "$BASE_API" > /dev/null
curl -s -X POST --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" --header "Content-Type: application/json" \
  -d "{\"name\":\"op-linux-amd64\",\"url\":\"${PKG}/op-linux-amd64\",\"link_type\":\"package\"}" "$BASE_API" > /dev/null
curl -s -X POST --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" --header "Content-Type: application/json" \
  -d "{\"name\":\"install.sh\",\"url\":\"${PKG}/install.sh\",\"link_type\":\"other\"}" "$BASE_API" > /dev/null

echo ""
echo "=== Released ${VERSION} ==="
echo "https://gitlab-tw.ddns.net/${REPO}/-/releases/${VERSION}"
