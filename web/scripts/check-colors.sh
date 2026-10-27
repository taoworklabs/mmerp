#!/bin/sh
# Colours live only in @mmerp/ui; the app reads its --mmerp-* tokens.
cd "$(dirname "$0")/../src" || exit 1
if grep -rnE '#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?([0-9a-fA-F]{2})?\b|rgba?\(|hsla?\(' --include='*.ts' --include='*.tsx' --include='*.css' .; then
  echo 'colour literals found in web/src' >&2
  exit 1
fi
