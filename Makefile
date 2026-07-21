.PHONY: dev build-linux build-windows clean release-assets release-assets-check check-release-env go-test integration-test integration-db-up integration-db-down db-performance-check go-vet govulncheck frontend-ci frontend-build frontend-lint frontend-test npm-audit release-gate

# Загружаем переменные из .env (если файл существует)
-include .env
export ENCRYPTION_KEY

# Переменные
TAGS = webkit2_41
FRONTEND_DIR = frontend
RELEASE_EVIDENCE_DIR = build/release-evidence
GOCACHE ?= /tmp/go-build-cache
GOVULNCHECK ?= $(shell command -v govulncheck 2>/dev/null || echo "go run golang.org/x/vuln/cmd/govulncheck@latest")
GO_PACKAGES = $(shell go list ./... | grep -v '/frontend/node_modules/')
INTEGRATION_COMPOSE = docker compose -p docflow-integration -f docker-compose.integration.yaml
INTEGRATION_DSN = postgres://docflow_integration:docflow_integration@127.0.0.1:55432/docflow_test_outbox?sslmode=disable
PERFORMANCE_DIR = build/performance
PERFORMANCE_DOCUMENTS ?= 10000
PERFORMANCE_PAGE_SIZE ?= 50
PERFORMANCE_DEEP_PAGE ?= 0

# Ключ шифрования конфигурации (из .env → ENCRYPTION_KEY)
LDFLAGS = -X 'github.com/Volkov-D-A/docs-register-and-track/internal/config.rawEncryptionKey=$(ENCRYPTION_KEY)'

release-assets:
	GOCACHE=$(GOCACHE) go generate ./internal/releaseassets

release-assets-check:
	GOCACHE=$(GOCACHE) go run ./tools/releasegen -source docs/releases.yaml -out internal/releaseassets/current_release.yaml -wails-config wails.json -check

check-release-env:
	@test -n "$(ENCRYPTION_KEY)" || (echo "ENCRYPTION_KEY is required for production build; provide it via approved release secret injection." >&2; exit 1)

# Запуск режима разработки с правильным WebKit для Ubuntu 24.04
dev:
	$(MAKE) release-assets
	wails dev -tags $(TAGS) -ldflags "$(LDFLAGS)"

# Сборка готового бинарника для тестирования в Linux
build-linux:
	$(MAKE) check-release-env
	$(MAKE) release-assets
	wails build -tags $(TAGS) -platform linux/amd64 -ldflags "$(LDFLAGS)"

# Кросс-компиляция готового .exe для Windows (для конечных пользователей)
build-windows:
	$(MAKE) check-release-env
	$(MAKE) release-assets
	wails build -platform windows/amd64 -ldflags "$(LDFLAGS)"

# Очистка кэша сборки и папки build/bin
clean:
	rm -rf build/bin/*

# Запуск тестов
go-test:
	$(MAKE) release-assets
	GOCACHE=$(GOCACHE) go test $(GO_PACKAGES)

# Запускает изолированный PostgreSQL, выполняет тесты с суффиксом Integration
# и всегда удаляет контейнер вместе с тестовым volume.
integration-test:
	@set -eu; \
		cleanup() { $(INTEGRATION_COMPOSE) down -v --remove-orphans; }; \
		trap cleanup EXIT INT TERM; \
		$(INTEGRATION_COMPOSE) up -d --wait; \
		DOCFLOW_INTEGRATION_DSN='$(INTEGRATION_DSN)' GOCACHE=$(GOCACHE) go test ./internal/... -run Integration -count=1 -p=1

# Generates a local baseline only. It intentionally has no pass/fail latency
# threshold because Docker and developer hardware are not stable benchmark hosts.
db-performance-check:
	@set -eu; \
		cleanup() { $(INTEGRATION_COMPOSE) down -v --remove-orphans; }; \
		trap cleanup EXIT INT TERM; \
		mkdir -p $(PERFORMANCE_DIR); \
		$(INTEGRATION_COMPOSE) up -d --wait; \
		if ! DOCFLOW_INTEGRATION_DSN='$(INTEGRATION_DSN)' GOCACHE=$(GOCACHE) go test ./internal/repository -run '^$$' -bench Integration -benchmem -count=1 -v > $(PERFORMANCE_DIR)/db-performance.txt 2>&1; then cat $(PERFORMANCE_DIR)/db-performance.txt; exit 1; fi; \
		GOCACHE=$(GOCACHE) go run ./tools/dbperf -dsn '$(INTEGRATION_DSN)' -out $(PERFORMANCE_DIR) -documents $(PERFORMANCE_DOCUMENTS) -page-size $(PERFORMANCE_PAGE_SIZE) -deep-page $(PERFORMANCE_DEEP_PAGE) | tee $(PERFORMANCE_DIR)/summary.txt

# Эти цели полезны при ручной отладке интеграционных тестов. Данные не
# предназначены для сохранения: integration-db-down удаляет volume.
integration-db-up:
	$(INTEGRATION_COMPOSE) up -d --wait

integration-db-down:
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

release-gate:
	@./tools/release-gate.sh

# ==========================================
# УПРАВЛЕНИЕ БАЗОЙ ДАННЫХ (DOCKER)
# ==========================================

# Обычный запуск базы данных в фоновом режиме
storage-up:
	docker compose up -d

# Остановка контейнера (данные СОХРАНЯЮТСЯ)
storage-down:
	docker compose down

# ПОЛНЫЙ СБРОС: удаляет контейнер, УНИЧТОЖАЕТ ВСЕ ДАННЫЕ (том) и поднимает чистую БД
storage-reset:
	docker compose down -v
	docker compose up -d
