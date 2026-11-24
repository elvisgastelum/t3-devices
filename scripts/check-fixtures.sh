#!/usr/bin/env bash
# Usage: scripts/check-fixtures.sh [dir ...]
#
# Scans every regular file under each dir (default: <repo root>/testdata)
# and fails if a fixture contains something that looks like a real secret
# or a real tailnet identity. Findings print to stderr as
# `path:line: <rule>: <match>`.
#
# Rules:
#   a. real-looking tailnet name: any host ending in .ts.net other than
#      exactly example-tailnet.ts.net or a name ending in
#      .example-tailnet.ts.net (case-insensitive).
#   b. pairing token: `pair#token=` (or URL-encoded, `%23` for # and `%3D` for =) followed by a value that is not exactly
#      REDACTED or EXAMPLE-TOKEN (case-insensitive, including the prefix).
#   c. tailscale key: tskey-... that does not end in -REDACTED or
#      -EXAMPLE-TOKEN (case-insensitive).
#   d. token-like string: a run of 32 or more characters from
#      [A-Za-z0-9_+/=-] containing at least one letter and one digit.
#
# Exit codes: 0 clean, 1 findings, 2 usage error (missing directory) or unscannable file.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [ "$#" -eq 0 ]; then
  set -- "$root/testdata"
fi

for d in "$@"; do
  if [ ! -d "$d" ]; then
    echo "check-fixtures: directory not found: $d" >&2
    exit 2
  fi
done

found=0

report() {
  echo "$1:$2: $3: $4" >&2
  found=1
}

tmp="$(mktemp)"
errtmp="$(mktemp)"
trap 'rm -f "$tmp" "$errtmp"' EXIT

# scan <rule> <ERE> <dir> <flags>: runs grep and feeds each hit to check_<rule>.
# -Z puts a NUL after the path (turned into a tab) so colons in paths or
# matches cannot confuse parsing. -a scans binary-looking files as text and
# -R follows symlinks. A grep failure (status >= 2) aborts with exit 2.
scan() {
  local rule="$1" re="$2" dir="$3" flags="$4" rc=0 rec path rest line match
  grep -RnoaZ$flags -E -e "$re" -- "$dir" >"$tmp" 2>"$errtmp" || rc=$?
  if [ "$rc" -ge 2 ]; then
    echo "check-fixtures: cannot scan $dir" >&2
    cat "$errtmp" >&2
    exit 2
  fi
  while IFS= read -r rec; do
    path="${rec%%$'\t'*}"
    rest="${rec#*$'\t'}"
    line="${rest%%:*}"
    match="${rest#*:}"
    "check_$rule" "$path" "$line" "$match"
  done < <(tr '\0' '\t' <"$tmp")
}

check_tailnet() {
  local m
  m="$(printf '%s' "$3" | tr 'A-Z' 'a-z')"
  case "$m" in
    example-tailnet.ts.net | *.example-tailnet.ts.net) ;;
    *) report "$1" "$2" "real-looking tailnet name" "$3" ;;
  esac
}

check_pairing() {
  local v
  v="$(printf '%s' "$3" | tr 'A-Z' 'a-z')"
  v="${v#*token}"
  case "$v" in
    =*) v="${v#=}" ;;
    *) v="${v#%3d}" ;;
  esac
  case "$v" in
    redacted | example-token) ;;
    *) report "$1" "$2" "pairing token" "$3" ;;
  esac
}

check_tskey() {
  local m
  m="$(printf '%s' "$3" | tr 'A-Z' 'a-z')"
  case "$m" in
    *-redacted | *-example-token) ;;
    *) report "$1" "$2" "tailscale key" "$3" ;;
  esac
}

check_generic() {
  if [[ "$3" =~ [A-Za-z] && "$3" =~ [0-9] ]]; then
    report "$1" "$2" "token-like string" "$3"
  fi
}

for dir in "$@"; do
  scan tailnet '[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?\.ts\.net' "$dir" i
  scan pairing "pair(#|%23)token(=|%3d)[^[:space:]\"'&<>]*" "$dir" i
  scan tskey 'tskey-[A-Za-z0-9_-]+' "$dir" i
  scan generic '[A-Za-z0-9_+/=-]{32,}' "$dir" ""
done

if [ "$found" -ne 0 ]; then
  exit 1
fi
echo "check-fixtures: OK"
