# Точечные проверки и диагностика. Основной запуск — make release-gate.
.PHONY: release-assets-check docs-links-check go-test go-vet govulncheck frontend-ci frontend-build frontend-lint frontend-test npm-audit integration-test integration-db-up integration-db-down db-performance-check storage-smoke-test clean _check-integration-env

GO_PACKAGES = . ./cmd/... ./internal/... ./tools/...
INTEGRATION_COMPOSE = docker compose --env-file $(if $(wildcard .env),.env,/dev/null) -p docflow-integration -f testing/compose/integration.yaml
INTEGRATION_DSN = postgres://docflow_integration:docflow_integration@127.0.0.1:55432/docflow_test_outbox?sslmode=disable
PERFORMANCE_DIR = build/performance
PERFORMANCE_DOCUMENTS ?= 10000
PERFORMANCE_PAGE_SIZE ?= 50
PERFORMANCE_DEEP_PAGE ?= 0
GOVULNCHECK ?= $(shell command -v govulncheck 2>/dev/null || echo "go run golang.org/x/vuln/cmd/govulncheck@latest")

release-assets-check:
	GOCACHE=$(GOCACHE) go run ./tools/releasegen -source docs/releases.yaml -out internal/shared/releaseassets/current_release.yaml -wails-config wails.json -check

docs-links-check:
	node tools/check-markdown-links.mjs

# Удаление локальных бинарников (Go cache и данные Compose сохраняются).
clean:
	rm -rf build/bin/*

# Запуск тестов
go-test:
	$(MAKE) release-assets
	GOCACHE=$(GOCACHE) go test $(GO_PACKAGES)

# Запускает изолированные PostgreSQL и SeaweedFS, затем отдельный стек
# PostgreSQL/S3/Samba для резервирования. Оба стека удаляются после проверки.
_check-integration-env:
	@command -v docker >/dev/null 2>&1 || (echo "docker is required for PostgreSQL integration tests" >&2; exit 1)
	@docker compose version >/dev/null 2>&1 || (echo "docker compose is required for PostgreSQL integration tests" >&2; exit 1)
	@docker info >/dev/null 2>&1 || (echo "docker daemon is not available for PostgreSQL integration tests" >&2; exit 1)
	@test -n "$(POSTGRES_VERSION)" || (echo "POSTGRES_VERSION is required for PostgreSQL integration tests" >&2; exit 1)

integration-test: _check-integration-env
	@set -eu; \
		cleanup() { $(INTEGRATION_COMPOSE) down -v --remove-orphans; }; \
		trap cleanup EXIT INT TERM; \
		$(INTEGRATION_COMPOSE) up -d --build --wait; \
		mkdir -p build/release-evidence; \
		DOCFLOW_INTEGRATION_S3_ENDPOINT=127.0.0.1:58333 DOCFLOW_INTEGRATION_DSN='$(INTEGRATION_DSN)' GOCACHE=$(GOCACHE) go test ./internal/... -run Integration -count=1 -p=1 -coverprofile=build/release-evidence/integration-go-coverage.out; \
		GOCACHE=$(GOCACHE) bash testing/scripts/backup-integration.sh; \
		python3 tools/merge-go-coverage.py build/release-evidence/integration-combined-coverage.out build/release-evidence/integration-go-coverage.out build/release-evidence/backup-integration/server-coverage.out

# Generates a local baseline only. It intentionally has no pass/fail latency
# threshold because Docker and developer hardware are not stable benchmark hosts.
db-performance-check: _check-integration-env
	@set -eu; \
		cleanup() { $(INTEGRATION_COMPOSE) down -v --remove-orphans; }; \
		trap cleanup EXIT INT TERM; \
		mkdir -p $(PERFORMANCE_DIR); \
		$(INTEGRATION_COMPOSE) up -d --build --wait; \
		if ! DOCFLOW_INTEGRATION_DSN='$(INTEGRATION_DSN)' GOCACHE=$(GOCACHE) go test ./internal/server/repository -run '^$$' -bench Integration -benchmem -count=1 -v > $(PERFORMANCE_DIR)/db-performance.txt 2>&1; then cat $(PERFORMANCE_DIR)/db-performance.txt; exit 1; fi; \
		GOCACHE=$(GOCACHE) go run ./tools/dbperf -dsn '$(INTEGRATION_DSN)' -out $(PERFORMANCE_DIR) -documents $(PERFORMANCE_DOCUMENTS) -page-size $(PERFORMANCE_PAGE_SIZE) -deep-page $(PERFORMANCE_DEEP_PAGE) | tee $(PERFORMANCE_DIR)/summary.txt

# Эти цели полезны при ручной отладке интеграционных тестов. Данные не
# предназначены для сохранения: integration-db-down удаляет volume.
integration-db-up: _check-integration-env
	$(INTEGRATION_COMPOSE) up -d --build --wait

integration-db-down: _check-integration-env
	$(INTEGRATION_COMPOSE) down -v --remove-orphans

go-vet:
	GOCACHE=$(GOCACHE) go vet $(GO_PACKAGES)

govulncheck:
	GOCACHE=$(GOCACHE) $(GOVULNCHECK) $(GO_PACKAGES)

frontend-ci:
	cd $(FRONTEND_DIR) && npm ci

frontend-build:
	cd $(FRONTEND_DIR) && npm run build

frontend-lint:
	cd $(FRONTEND_DIR) && npm run lint

frontend-test:
	cd $(FRONTEND_DIR) && npm test

npm-audit:
	cd $(FRONTEND_DIR) && npm audit --audit-level=critical

storage-smoke-test: _check-docker
	bash testing/scripts/prod-compose-smoke.sh
