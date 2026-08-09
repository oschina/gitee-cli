#!/bin/sh
# uninstall.sh — Remove Gitee CLI Agent Skills from the local agent skills directory.
#
# Usage:
#   sh skills/uninstall.sh
#
# Target directory (override with AGENTS_SKILLS_DIR):
#   $HOME/.agents/skills

set -eu

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
LEGACY_SKILLS="gitee-issue-manage gitee-repo-manager"

echo "This will remove the following skills from $TARGET_DIR:"
for skill_path in "$SRC_DIR"/*/; do
  [ -f "${skill_path}SKILL.md" ] || continue
  echo "  - $(basename "$skill_path")"
done
for legacy_name in $LEGACY_SKILLS; do
  if [ -d "$TARGET_DIR/$legacy_name" ]; then
    echo "  - $legacy_name (legacy)"
  fi
done
printf "Continue? [y/N] "
read -r answer
case "$answer" in
  y|Y|yes|YES) ;;
  *) echo "Aborted."; exit 0 ;;
esac

count=0
for skill_path in "$SRC_DIR"/*/; do
  [ -f "${skill_path}SKILL.md" ] || continue
  name="$(basename "$skill_path")"
  if [ -d "$TARGET_DIR/$name" ]; then
    rm -rf "$TARGET_DIR/$name"
    echo "  ✔ Removed $name"
    count=$((count + 1))
  fi
done
for legacy_name in $LEGACY_SKILLS; do
  if [ -d "$TARGET_DIR/$legacy_name" ]; then
    rm -rf "$TARGET_DIR/$legacy_name"
    echo "  ✔ Removed $legacy_name (legacy)"
    count=$((count + 1))
  fi
done

echo
echo "Done. Removed $count skill(s) from $TARGET_DIR"
