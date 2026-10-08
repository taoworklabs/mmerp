#!/bin/sh
# Colours live only in shared/ui (theme.ts and CSS modules); everywhere else uses tokens.
cd "$(dirname "$0")/../src" || exit 1
if grep -rnE '#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?([0-9a-fA-F]{2})?\b|rgba?\(|hsla?\(' --include='*.ts' --include='*.tsx' --include='*.css' . | grep -v '^\./shared/ui/'; then
  echo 'colour literals found outside shared/ui' >&2
  exit 1
fi
