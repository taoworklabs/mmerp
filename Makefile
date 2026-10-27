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

.PHONY: gen lint test e2e dev ui-dev storybook image smoke db reset-db seed migration

# web links ../packages/ui, whose own dependencies must be installed first.
web/node_modules: web/package.json web/pnpm-lock.yaml packages/ui/node_modules
	pnpm -C web install --frozen-lockfile
	@touch $@

packages/ui/node_modules: packages/ui/package.json packages/ui/pnpm-lock.yaml
	pnpm -C packages/ui install --frozen-lockfile
	@touch $@

gen: web/node_modules
	go build -o bin/sqlc-gen-data ./cmd/sqlc-gen-data
	$(SQLC) generate
	go run ./cmd/openapi > api/openapi.json
	pnpm -C web gen

lint: web/node_modules
	$(GOLANGCI) run ./...
	GOLANGCI='$(GOLANGCI)' scripts/check-go-lint-fixtures.sh
	go vet -tags perf ./internal/modules/hrm/ ./internal/platform/data/...
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

# Creates an empty migration named after the current UTC time: make migration name=hrm_leave_carryover.
migration:
	@echo '$(name)' | grep -qE '^[a-z]+_[a-z0-9_]+$$' || { echo 'usage: make migration name=<module>_<description> (lowercase, digits, _)'; exit 1; }
	@f=migrations/$$(date -u +%Y%m%d%H%M%S)_$(name).sql; touch $$f; echo $$f

# Builds the Demo demo company through the API of a running make dev, on an empty
# database (make reset-db); E2E_ADMIN_LOGIN and E2E_ADMIN_PASSWORD name a tenant administrator.
seed: web/node_modules
	E2E_EXTERNAL=1 E2E_SEED=1 E2E_BASE_URL=$${E2E_BASE_URL:-http://localhost:8080} pnpm -C web exec playwright test --project seed

dev: db web/node_modules
	trap 'kill 0' EXIT; DATABASE_URL='$(DATABASE_URL)' PRODUCTS=hrm,sales go run ./cmd/server & pnpm -C web dev

# Only the Vite dev server, on the component catalogue (/dev/ui); no backend or Postgres.
ui-dev: web/node_modules
	pnpm -C web dev --open /dev/ui

# Storybook of the @mmerp/ui primitives (packages/ui), on :6006; no backend or Postgres.
storybook: packages/ui/node_modules
	pnpm -C packages/ui storybook

image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

smoke:
	APP_IMAGE=$(IMAGE) scripts/smoke.sh
