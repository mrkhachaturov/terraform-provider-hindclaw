#!/usr/bin/env bash

set -euo pipefail

tag="${1:?tag is required}"
changelog_path="${2:?changelog path is required}"
output_path="${3:?output path is required}"

version="${tag#v}"

awk -v version="$version" '
BEGIN {
  in_section = 0
  found = 0
}
$0 ~ "^## \\[" version "\\]" {
  in_section = 1
  found = 1
  next
}
$0 ~ "^## \\[" && in_section {
  exit
}
in_section {
  print
}
END {
  if (!found) {
    exit 2
  }
}
' "$changelog_path" | sed '/^[[:space:]]*$/N;/^\n$/D' > "$output_path"

if [[ ! -s "$output_path" ]]; then
  echo "No release notes found for $tag in $changelog_path" >&2
  exit 3
fi
