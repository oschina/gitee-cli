#!/bin/sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

if [ -z "${NPM_VERSION:-}" ] || [ "$NPM_VERSION" = "0.0.0" ]; then
  echo "NPM_VERSION must be set to a release version" >&2
  exit 2
fi
if [ -z "${NPM_TOKEN:-}" ]; then
  echo "NPM_TOKEN is required" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 2
fi

if [ -n "${NPM_TAG:-}" ]; then
  DIST_TAG=$NPM_TAG
else
  case "$NPM_VERSION" in
    *-*) DIST_TAG=next ;;
    *) DIST_TAG=latest ;;
  esac
fi
case "$DIST_TAG" in
  *[!A-Za-z0-9._-]*|'')
    echo "NPM_TAG contains unsupported characters: $DIST_TAG" >&2
    exit 2
    ;;
esac

STAGING_DIR=$(mktemp -d "${TMPDIR:-/tmp}/gitee-cli-npm.XXXXXX")
trap 'rm -rf "$STAGING_DIR"' EXIT HUP INT TERM

NPMRC="$STAGING_DIR/npmrc"
printf '%s\n' "//registry.npmjs.org/:_authToken=$NPM_TOKEN" >"$NPMRC"
chmod 600 "$NPMRC"

stage_package() {
  package_dir=$1
  is_main=${2:-false}
  package_name=$(basename "$package_dir")
  stage="$STAGING_DIR/$package_name"

  mkdir -p "$stage"
  cp -R "$package_dir/." "$stage/"
  cp "$ROOT_DIR/LICENSE" "$stage/LICENSE"

  if [ "$is_main" = "true" ]; then
    cp "$ROOT_DIR/README.md" "$stage/README.md"
    jq --arg version "$NPM_VERSION" \
      '.version = $version | .optionalDependencies |= with_entries(.value = $version)' \
      "$stage/package.json" >"$stage/package.json.tmp"
  else
    jq --arg version "$NPM_VERSION" '.version = $version' \
      "$stage/package.json" >"$stage/package.json.tmp"
  fi
  mv "$stage/package.json.tmp" "$stage/package.json"

  STAGED_PACKAGES="$STAGED_PACKAGES $stage"
}

preflight_package() {
  stage=$1
  npm publish "$stage" --dry-run --tag "$DIST_TAG" --userconfig "$NPMRC"
}

publish_package() {
  stage=$1
  package_name=$(jq -r '.name' "$stage/package.json")
  published_version=$(npm view "$package_name@$NPM_VERSION" version --userconfig "$NPMRC" 2>/dev/null || true)
  if [ "$published_version" = "$NPM_VERSION" ]; then
    echo "Skipping $package_name@$NPM_VERSION (already published)"
    return
  fi
  npm publish "$stage" --tag "$DIST_TAG" --userconfig "$NPMRC"
}

STAGED_PACKAGES=""
for platform in \
  darwin-amd64 darwin-arm64 \
  linux-amd64 linux-arm64 \
  windows-amd64 windows-arm64
do
  stage_package "$ROOT_DIR/npm/gitee-cli-$platform"
done

stage_package "$ROOT_DIR/npm/gitee-cli" true

for stage in $STAGED_PACKAGES; do
  preflight_package "$stage"
done

if [ "${NPM_DRY_RUN:-0}" = "1" ]; then
  echo "npm publish dry-run complete (dist-tag: $DIST_TAG)"
  exit 0
fi

for stage in $STAGED_PACKAGES; do
  publish_package "$stage"
done
