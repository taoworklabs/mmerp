#!/bin/sh
# Colours live only in tokens (tokens.ts and its generated tokens.css, plus the
# literal values render.test.ts asserts against); everywhere else uses var(--mmerp-...).
cd "$(dirname "$0")/../src" || exit 1
if grep -rnE '#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?([0-9a-fA-F]{2})?\b|rgba?\(|hsla?\(' --include='*.ts' --include='*.tsx' --include='*.css' . | grep -v '^\./tokens/'; then
  echo 'colour literals found outside src/tokens' >&2
  exit 1
fi
