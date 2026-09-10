.PHONY: dev build-linux build-windows docker-server-push check-docker check-docflow-server-version clean release-assets release-assets-check wails-bindings wails-bindings-check docs-links-check check-integration-env go-test integration-test integration-db-up integration-db-down db-performance-check go-vet govulncheck frontend-ci frontend-build frontend-lint frontend-test npm-audit release-gate storage-up storage-down storage-reset

# Загружаем переменные из .env (если файл существует)
-include .env

# Переменные
TAGS = webkit2_41
WAILS ?= wails
FRONTEND_DIR = frontend
GOCACHE ?= /tmp/go-build-cache
GOVULNCHECK ?= $(shell command -v govulncheck 2>/dev/null || echo "go run golang.org/x/vuln/cmd/govulncheck@latest")
GO_PACKAGES = . ./cmd/... ./internal/... ./tools/...
INTEGRATION_COMPOSE = docker compose -p docflow-integration -f docker-compose.integration.yaml
INTEGRATION_DSN = postgres://docflow_integration:docflow_integration@127.0.0.1:55432/docflow_test_outbox?sslmode=disable
PERFORMANCE_DIR = build/performance
PERFORMANCE_DOCUMENTS ?= 10000
PERFORMANCE_PAGE_SIZE ?= 50
PERFORMANCE_DEEP_PAGE ?= 0
SERVER_DOCKERFILE = build/server/Dockerfile
DOCKER_PLATFORM ?= linux/amd64
DOCKERHUB_IMAGE = hehelf/docflow-service

release-assets:
	GOCACHE=$(GOCACHE) go generate ./internal/releaseassets

release-assets-check:
	GOCACHE=$(GOCACHE) go run ./tools/releasegen -source docs/releases.yaml -out internal/releaseassets/current_release.yaml -wails-config wails.json -check

wails-bindings:
	@command -v $(WAILS) >/dev/null 2>&1 || (echo "Wails CLI is required to generate bindings." >&2; exit 1)
	@expected="$$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)"; actual="$$($(WAILS) version | sed -n '1p')"; test "$$actual" = "$$expected" || (echo "Wails CLI version $$actual does not match go.mod version $$expected." >&2; exit 1)
	GOCACHE=$(GOCACHE) $(WAILS) generate module -nocolour -tags $(TAGS)
	node $(FRONTEND_DIR)/scripts/normalize-wails-bindings.mjs

wails-bindings-check: wails-bindings
	@git diff --exit-code HEAD -- frontend/wailsjs
	@test -z "$$(git ls-files --others --exclude-standard -- frontend/wailsjs)" || (git status --short -- frontend/wailsjs; echo "Untracked Wails bindings detected. Regenerate and commit frontend/wailsjs." >&2; exit 1)

docs-links-check:
	node tools/check-markdown-links.mjs

# Запуск режима разработки с правильным WebKit для Ubuntu 24.04
dev:
	$(MAKE) release-assets
	wails dev -tags $(TAGS)

# Сборка готового бинарника для тестирования в Linux
build-linux:
	$(MAKE) release-assets
	wails build -tags $(TAGS) -platform linux/amd64

# Кросс-компиляция готового .exe для Windows (для конечных пользователей)
build-windows:
	$(MAKE) release-assets
	wails build -platform windows/amd64

check-docker:
	@command -v docker >/dev/null 2>&1 || (echo "docker is required" >&2; exit 1)
	@docker info >/dev/null 2>&1 || (echo "docker daemon is not available" >&2; exit 1)

check-docflow-server-version: release-assets-check
	@test -n "$(DOCFLOW_SERVER_VERSION)" || (echo "DOCFLOW_SERVER_VERSION is required in .env" >&2; exit 1)
	@actual="$$(GOCACHE=$(GOCACHE) go run ./cmd/docflow-server version)"; \
		test "$$actual" = "$(DOCFLOW_SERVER_VERSION)" || (echo "DOCFLOW_SERVER_VERSION $(DOCFLOW_SERVER_VERSION) does not match embedded product version $$actual" >&2; exit 1)

# Перед публикацией выполните docker login. Токен Docker Hub не передаётся Makefile.
docker-server-push: check-docker check-docflow-server-version
	docker build --platform $(DOCKER_PLATFORM) --build-arg VERSION=$(DOCFLOW_SERVER_VERSION) -f $(SERVER_DOCKERFILE) -t $(DOCKERHUB_IMAGE):$(DOCFLOW_SERVER_VERSION) .
	docker push $(DOCKERHUB_IMAGE):$(DOCFLOW_SERVER_VERSION)

# Очистка кэша сборки и папки build/bin
clean:
	rm -rf build/bin/*

# Запуск тестов
go-test:
	$(MAKE) release-assets
	GOCACHE=$(GOCACHE) go test $(GO_PACKAGES)

# Запускает изолированные PostgreSQL и SeaweedFS, выполняет тесты Integration
# и всегда удаляет контейнер вместе с тестовым volume.
check-integration-env:
	@command -v docker >/dev/null 2>&1 || (echo "docker is required for PostgreSQL integration tests" >&2; exit 1)
	@docker compose version >/dev/null 2>&1 || (echo "docker compose is required for PostgreSQL integration tests" >&2; exit 1)
	@docker info >/dev/null 2>&1 || (echo "docker daemon is not available for PostgreSQL integration tests" >&2; exit 1)
	@test -n "$(POSTGRES_VERSION)" || (echo "POSTGRES_VERSION is required for PostgreSQL integration tests" >&2; exit 1)

integration-test: check-integration-env
	@set -eu; \
		cleanup() { $(INTEGRATION_COMPOSE) down -v --remove-orphans; }; \
		trap cleanup EXIT INT TERM; \
		$(INTEGRATION_COMPOSE) up -d --build --wait; \
		$(INTEGRATION_COMPOSE) run --rm s3-ready; \
		DOCFLOW_INTEGRATION_S3_ENDPOINT=127.0.0.1:58333 DOCFLOW_INTEGRATION_DSN='$(INTEGRATION_DSN)' GOCACHE=$(GOCACHE) go test ./internal/... -run Integration -count=1 -p=1

# Generates a local baseline only. It intentionally has no pass/fail latency
# threshold because Docker and developer hardware are not stable benchmark hosts.
db-performance-check: check-integration-env
	@set -eu; \
		cleanup() { $(INTEGRATION_COMPOSE) down -v --remove-orphans; }; \
		trap cleanup EXIT INT TERM; \
		mkdir -p $(PERFORMANCE_DIR); \
		$(INTEGRATION_COMPOSE) up -d --build --wait; \
		$(INTEGRATION_COMPOSE) run --rm s3-ready; \
		if ! DOCFLOW_INTEGRATION_DSN='$(INTEGRATION_DSN)' GOCACHE=$(GOCACHE) go test ./internal/repository -run '^$$' -bench Integration -benchmem -count=1 -v > $(PERFORMANCE_DIR)/db-performance.txt 2>&1; then cat $(PERFORMANCE_DIR)/db-performance.txt; exit 1; fi; \
		GOCACHE=$(GOCACHE) go run ./tools/dbperf -dsn '$(INTEGRATION_DSN)' -out $(PERFORMANCE_DIR) -documents $(PERFORMANCE_DOCUMENTS) -page-size $(PERFORMANCE_PAGE_SIZE) -deep-page $(PERFORMANCE_DEEP_PAGE) | tee $(PERFORMANCE_DIR)/summary.txt

# Эти цели полезны при ручной отладке интеграционных тестов. Данные не
# предназначены для сохранения: integration-db-down удаляет volume.
integration-db-up: check-integration-env
	$(INTEGRATION_COMPOSE) up -d --build --wait

integration-db-down: check-integration-env
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
# УПРАВЛЕНИЕ ЛОКАЛЬНЫМ СТЕКОМ (DOCKER)
# ==========================================

# Запуск PostgreSQL, SeaweedFS, Seq, docflow-server и Caddy в фоновом режиме
storage-up:
	docker compose up -d

# Остановка локального стека (данные СОХРАНЯЮТСЯ)
storage-down:
	docker compose down

# СБРОС ДАННЫХ DEV: PostgreSQL и хранилище; тома Seq/Caddy сохраняются.
storage-reset:
	bash scripts/reset-dev-storage.sh --execute

.PHONY: storage-smoke-test
storage-smoke-test: check-docker
	bash scripts/integration-smoke.sh
