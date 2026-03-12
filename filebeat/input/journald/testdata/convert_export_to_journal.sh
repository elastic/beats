#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${1:-$SCRIPT_DIR}"

if [[ ! -d "$TARGET_DIR" ]]; then
  echo "target directory does not exist: $TARGET_DIR" >&2
  exit 1
fi

if command -v systemd-journal-remote >/dev/null 2>&1; then
  JOURNAL_REMOTE_BIN="$(command -v systemd-journal-remote)"
elif [[ -x /usr/lib/systemd/systemd-journal-remote ]]; then
  JOURNAL_REMOTE_BIN="/usr/lib/systemd/systemd-journal-remote"
else
  echo "systemd-journal-remote not found in PATH or /usr/lib/systemd/systemd-journal-remote" >&2
  exit 1
fi

shopt -s nullglob
exports=("$TARGET_DIR"/*.export)
shopt -u nullglob

if (( ${#exports[@]} == 0 )); then
  echo "no .export files found in: $TARGET_DIR" >&2
  exit 1
fi

for export_file in "${exports[@]}"; do
  journal_file="${export_file%.export}.journal"

  # systemd-journal-remote appends if output exists; remove to regenerate.
  rm -f -- "$journal_file" "${journal_file}.gz"

  echo "building: $(basename -- "$journal_file")"
  "$JOURNAL_REMOTE_BIN" -o "$journal_file" "$export_file"

  echo "gzipping: $(basename -- "$journal_file").gz"
  gzip -f -- "$journal_file"
done

echo "done: converted ${#exports[@]} export files in $TARGET_DIR"
