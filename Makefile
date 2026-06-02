.PHONY: dev build-linux build-windows clean release-assets release-assets-check check-release-env go-test go-vet govulncheck frontend-ci frontend-build frontend-lint frontend-test performance-baseline npm-audit npm-license-check license-inventory security-gate release-gate

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

npm-license-check:
	node tools/license-report.js

license-inventory:
	mkdir -p $(RELEASE_EVIDENCE_DIR)
	go list -m -json all > $(RELEASE_EVIDENCE_DIR)/go-modules.json
	cd $(FRONTEND_DIR) && npm ls --all --json > ../$(RELEASE_EVIDENCE_DIR)/npm-dependencies.json
	node tools/license-report.js

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
