#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APP_NAME="treehouse"
VERSION_FILE="VERSION"

abort() {
  echo "ERROR: $*" >&2
  exit 1
}

require_semver() {
  command -v semver >/dev/null || abort "semver CLI is required. Install it with: brew install semver"
}

validate_version() {
  local version="$1"
  [ -n "$version" ] || return 1
  [[ "$version" != v* ]] || return 1
  semver "$version" >/dev/null 2>&1
}

current_version() {
  local versions=() tag version
  if [ -n "$VERSION_FILE_VERSION" ]; then
    versions+=("$VERSION_FILE_VERSION")
  fi

  while IFS= read -r tag; do
    version="$(tag_version "$tag")"
    if validate_version "$version"; then
      versions+=("$version")
    fi
  done < <(git tag --list 'v*')

  if ((${#versions[@]} == 0)); then
    return 0
  fi

  semver "${versions[@]}" | tail -n 1
}

require_clean_main() {
  local branch local_head remote_head
  branch="$(git branch --show-current)"
  [ "$branch" = "main" ] || abort "release must be created from main, not ${branch:-detached HEAD}"
  [ -z "$(git status --porcelain)" ] || abort "working tree must be clean before release"

  git fetch origin main --tags

  local_head="$(git rev-parse HEAD)"
  remote_head="$(git rev-parse origin/main)"
  [ "$local_head" = "$remote_head" ] || abort "local main must match origin/main before release"
  [ -z "$(git status --porcelain)" ] || abort "working tree changed during preflight"
}

read_version_file() {
  if [ ! -f "$VERSION_FILE" ]; then
    return 0
  fi

  local version
  version="$(tr -d '[:space:]' < "$VERSION_FILE")"
  [ -n "$version" ] || abort "$VERSION_FILE cannot be empty"
  validate_version "$version" || abort "$VERSION_FILE must contain a SemVer version without a leading v"
  printf '%s\n' "$version"
}

tag_version() {
  local tag="$1"
  printf '%s\n' "${tag#v}"
}

require_semver
require_clean_main

VERSION_FILE_VERSION="$(read_version_file)"
CURRENT_VERSION="$(current_version)"
if [ -n "$CURRENT_VERSION" ]; then
  DEFAULT_VERSION="$(semver --increment patch "$CURRENT_VERSION")"
else
  DEFAULT_VERSION="0.1.0"
fi

echo "Current release version: ${CURRENT_VERSION:-none}"
echo "Current $VERSION_FILE: ${VERSION_FILE_VERSION:-none}"
echo ""

read -r -p "New version (without v) [$DEFAULT_VERSION]: " NEW_VERSION
NEW_VERSION="${NEW_VERSION:-$DEFAULT_VERSION}"
validate_version "$NEW_VERSION" || abort "version must be a SemVer version without a leading v"

if [ -n "$CURRENT_VERSION" ]; then
  semver --include-prerelease --range ">$CURRENT_VERSION" "$NEW_VERSION" >/dev/null \
    || abort "$NEW_VERSION must be greater than current version $CURRENT_VERSION"
fi

NEW_TAG="v$NEW_VERSION"
if git rev-parse -q --verify "refs/tags/$NEW_TAG" >/dev/null; then
  abort "tag $NEW_TAG already exists"
fi

require_clean_main
if git rev-parse -q --verify "refs/tags/$NEW_TAG" >/dev/null; then
  abort "tag $NEW_TAG was created while preparing the release"
fi

printf '%s\n' "$NEW_VERSION" > "$VERSION_FILE"
git diff --check -- "$VERSION_FILE"

git add "$VERSION_FILE"
git commit -m "Release $APP_NAME $NEW_VERSION"
git tag -a "$NEW_TAG" -m "$APP_NAME $NEW_VERSION"
git push --atomic --dry-run origin HEAD:main "$NEW_TAG"
git push --atomic origin HEAD:main "$NEW_TAG"

echo "Created and pushed $NEW_TAG."
