#!/bin/sh
# Proves the data-access forbidigo/depguard rules in .golangci.yml actually fire, the
# same way check-queries.mjs proves the schema-boundary check: run golangci-lint over
# the planted violations in scripts/lint-fixtures/go and diff against expected.txt.
set -eu
cd "$(dirname "$0")/.."

: "${GOLANGCI:=go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0}"

dir=scripts/lint-fixtures/go
expected="$dir/expected.txt"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

set +e
# shellcheck disable=SC2086 # GOLANGCI is a command, word-splitting is intended.
$GOLANGCI run --build-tags lintfixture "./$dir/..." >"$tmp/out" 2>"$tmp/err"
set -e

if grep -qi 'level=error\|panic:' "$tmp/err"; then
  echo "check-go-lint-fixtures: golangci-lint failed to run" >&2
  cat "$tmp/err" >&2
  exit 1
fi

grep -E '^[^[:space:]]+\.go:[0-9]+:[0-9]+:' "$tmp/out" |
  sed -E 's/^([^:]+):([0-9]+):[0-9]+:.*\(([A-Za-z0-9_-]+)\)$/\1:\2 \3/' |
  sort >"$tmp/got"

sort "$expected" >"$tmp/want"

if ! diff -u "$tmp/want" "$tmp/got" >"$tmp/diff"; then
  echo "check-go-lint-fixtures: golangci-lint output does not match $expected" >&2
  cat "$tmp/diff" >&2
  exit 1
fi

n=$(wc -l <"$expected" | tr -d ' ')
echo "check-go-lint-fixtures: all $n planted violations reported"
