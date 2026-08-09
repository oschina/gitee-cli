#!/bin/sh
# install.sh — Install Gitee CLI Agent Skills into the local agent skills directory.
#
# Usage:
#   sh skills/install.sh
#
# Target directory (override with AGENTS_SKILLS_DIR):
#   $HOME/.agents/skills
#
# Each skill shipped here is mirrored: its target directory is replaced in
# full on every run. Apart from names previously shipped by this project,
# skills from other sources in the target are never touched.

set -eu

# Resolve the directory this script lives in (follow symlinks), so it can be
# run from anywhere (e.g. `sh /path/to/install.sh` or `curl ... | sh`).
SCRIPT_PATH="$0"
while [ -h "$SCRIPT_PATH" ]; do
  LINK="$(readlink "$SCRIPT_PATH")"
  case "$LINK" in
    /*) SCRIPT_PATH="$LINK" ;;
    *)  SCRIPT_PATH="$(dirname "$SCRIPT_PATH")/$LINK" ;;
  esac
done
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd -P)"
SRC_DIR="$SCRIPT_DIR"

TARGET_DIR="${AGENTS_SKILLS_DIR:-$HOME/.agents/skills}"

# Names shipped by older releases. Remove only these known aliases so a
# reinstall cannot leave duplicate skills with overlapping triggers.
LEGACY_SKILLS="gitee-issue-manage gitee-repo-manager"

if [ ! -d "$SRC_DIR" ]; then
  echo "Error: skills directory not found at $SRC_DIR" >&2
  exit 1
fi

echo "Installing Gitee CLI Agent Skills"
echo "  from: $SRC_DIR"
echo "  into: $TARGET_DIR"
echo

mkdir -p "$TARGET_DIR"

for legacy_name in $LEGACY_SKILLS; do
  if [ -d "$TARGET_DIR/$legacy_name" ]; then
    rm -rf "$TARGET_DIR/$legacy_name"
    echo "  ✔ Removed legacy $legacy_name"
  fi
done

count=0
for skill_path in "$SRC_DIR"/*/; do
  [ -f "${skill_path}SKILL.md" ] || continue
  name="$(basename "$skill_path")"
  rm -rf "$TARGET_DIR/$name"
  mkdir -p "$TARGET_DIR/$name"
  cp -R "$skill_path." "$TARGET_DIR/$name/"
  echo "  ✔ Installed $name"
  count=$((count + 1))
done

echo
echo "Done. Installed $count skill(s) to $TARGET_DIR"
echo "Restart your agent (or reload skills) to pick them up."
