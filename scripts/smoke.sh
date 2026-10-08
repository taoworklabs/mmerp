#!/bin/sh
# Starts the image with Postgres in a throwaway compose project on random ports,
# waits for /healthz and the frontend page, then removes containers and volumes.
set -eu
cd "$(dirname "$0")/.."

export APP_PORT=0 POSTGRES_PORT=0
dc() { docker compose -p mmerp-smoke "$@"; }

ok=0
cleanup() {
  [ "$ok" = 1 ] || dc logs app || true
  dc down -v --remove-orphans
}
trap cleanup EXIT

dc up -d --no-build
url="http://$(dc port app 8080)"

deadline=$(( $(date +%s) + ${SMOKE_TIMEOUT:-60} ))
until curl -fsS "$url/healthz" >/dev/null 2>&1 \
  && curl -fsS "$url/" | grep -q '<div id="root">'; do
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "smoke: app not ready within ${SMOKE_TIMEOUT:-60}s" >&2
    exit 1
  fi
  sleep 1
done
ok=1
echo "smoke: healthz and frontend OK at $url"
