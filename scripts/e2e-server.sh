#!/bin/sh
# Serves the built app for Playwright on a fresh database: :8090 with PRODUCTS=hrm,sales,
# :8091 with no product enabled. Creates the e2e admin. Runs until killed.
set -eu
cd "$(dirname "$0")/.."

base=${TEST_DATABASE_URL:-postgres://mmerp:mmerp@localhost:5433/mmerp?sslmode=disable}
db_url=$(echo "$base" | sed -E 's#/[^/?]*(\?|$)#/mmerp_e2e\1#')
tmp=$(mktemp -d)
bin=$tmp/mmerp
export FILES_DIR=$tmp/files
export ENCRYPTION_KEYS=${ENCRYPTION_KEYS:-1:bW1lcnAgZGV2IGtleSAtIG5ldmVyIHByb2R1Y3Rpb24=}

pnpm -C web build >/dev/null
go build -tags embedweb -o "$bin" ./cmd/server

# psql from the Postgres image, so the host needs only Docker.
docker run --rm --network host postgres:18 psql -q "$base" \
  -c 'DROP DATABASE IF EXISTS mmerp_e2e WITH (FORCE)' -c 'CREATE DATABASE mmerp_e2e'
echo 'e2e password' | DATABASE_URL=$db_url "$bin" create-admin admin 'Quản trị viên'

trap 'kill 0' EXIT INT TERM
# Only the :8090 server works jobs, so they run with its products.
DATABASE_URL=$db_url HTTP_ADDR=:8091 PRODUCTS= RUN_JOBS=false LOG_LEVEL=warn "$bin" &
until curl -fsS http://localhost:8091/healthz >/dev/null 2>&1; do sleep 0.2; done
DATABASE_URL=$db_url HTTP_ADDR=:8090 PRODUCTS=hrm,sales LOG_LEVEL=warn "$bin"
