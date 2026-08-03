#!/bin/bash

set -euo pipefail

# Extract the changelog section for a single version from a Keep a Changelog file.
# Usage: release-notes.sh <version> [<changelog file>]
#   <version> is the git tag verbatim (e.g. v0.1.0); CHANGELOG headings use the
#   matching "## [v0.1.0]" form, so the tag and the heading line up directly.

if [ $# -lt 1 ]; then
  echo "Usage: $0 <semver version> [<changelog file>]"
  exit 1
fi

version="$1"
changelog_file="${2:-CHANGELOG.md}"

if [ ! -f "$changelog_file" ]; then
  echo "Error: $changelog_file does not exist"
  exit 1
fi

CAPTURE=0
items=""
while IFS= read -r LINE; do
  # Next version heading ends the section (match "## [", not bare "##", so that
  # Keep a Changelog subsections "### Added" / "### Changed" are captured).
  if [[ "${LINE}" == "## ["* ]] && [[ "${CAPTURE}" -eq 1 ]]; then
    break
  fi
  # A reference-link definition like "[Unreleased]: https://..." or
  # "[v0.1.0]: https://..." marks the trailing link block — end the section.
  if [[ "${LINE}" == "["*"]: http"* ]] && [[ "${CAPTURE}" -eq 1 ]]; then
    break
  fi
  # Start capturing at the requested version heading.
  if [[ "${LINE}" == "## [${version}]"* ]] && [[ "${CAPTURE}" -eq 0 ]]; then
    CAPTURE=1
    continue
  fi
  if [[ "${CAPTURE}" -eq 1 ]]; then
    items+="${LINE}"$'\n'
  fi
done <"${changelog_file}"

# Trim leading/trailing blank lines (portable: BSD/macOS + GNU/Linux).
items="$(printf '%s' "${items}" | awk '
  { lines[NR] = $0 }
  END {
    start = 1;  while (start <= NR && lines[start] ~ /^[[:space:]]*$/) start++
    end   = NR; while (end   >= start && lines[end] ~ /^[[:space:]]*$/) end--
    for (i = start; i <= end; i++) print lines[i]
  }')"

if [[ -n "${items}" ]]; then
  printf '%s\n' "${items}"
else
  echo "No changelog items found for version ${version}"
  exit 1
fi
