GOLANGCI := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
SQLC     := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE    ?= mmerp:dev
# Personal overrides, not committed: e.g. point at a Postgres already running on this machine
# by setting both DATABASE_URL and TEST_DATABASE_URL (TEST_DATABASE_URL also skips compose).
-include local.mk
export TEST_DATABASE_URL
DATABASE_URL ?= postgres://mmerp:mmerp@localhost:5433/mmerp?sslmode=disable
# Dev-only key, also used by e2e; never use it for real data.
export ENCRYPTION_KEYS ?= 1:bW1lcnAgZGV2IGtleSAtIG5ldmVyIHByb2R1Y3Rpb24=

.PHONY: gen lint test e2e dev image smoke db reset-db seed

web/node_modules: web/package.json web/pnpm-lock.yaml
	pnpm -C web install --frozen-lockfile
	@touch $@

gen: web/node_modules
	$(SQLC) generate
	go run ./cmd/openapi > api/openapi.json
	pnpm -C web gen

lint: web/node_modules
	$(GOLANGCI) run ./...
	pnpm -C web typecheck
	pnpm -C web lint
	node web/scripts/check-i18n.mjs
	node scripts/check-queries.mjs

# CI provides Postgres itself and sets TEST_DATABASE_URL.
db:
	@if [ -z "$$TEST_DATABASE_URL" ]; then docker compose up -d --wait postgres; fi

test: db web/node_modules
	go test ./...
	pnpm -C web test

# Builds the app, serves it on a fresh mmerp_e2e database and runs Playwright.
e2e: db web/node_modules
	pnpm -C web e2e

# Empties the dev database and creates the admin (password "e2e password"); a running make dev reconnects.
reset-db: db
	docker run --rm --network host postgres:18 psql -q '$(subst /mmerp?,/postgres?,$(DATABASE_URL))' -c 'DROP DATABASE IF EXISTS mmerp WITH (FORCE)' -c 'CREATE DATABASE mmerp'
	echo 'e2e password' | DATABASE_URL='$(DATABASE_URL)' go run ./cmd/server create-admin admin 'Quản trị viên'

# Builds the Demo demo company through the API of a running make dev, on an empty
# database (make reset-db); E2E_ADMIN_LOGIN and E2E_ADMIN_PASSWORD name a tenant administrator.
seed: web/node_modules
	E2E_EXTERNAL=1 E2E_SEED=1 E2E_BASE_URL=$${E2E_BASE_URL:-http://localhost:8080} pnpm -C web exec playwright test --project seed

dev: db web/node_modules
	trap 'kill 0' EXIT; DATABASE_URL='$(DATABASE_URL)' PRODUCTS=hrm,sales go run ./cmd/server & pnpm -C web dev

image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

smoke:
	APP_IMAGE=$(IMAGE) scripts/smoke.sh
