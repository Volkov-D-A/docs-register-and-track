# Основные команды Docflow. Без аргументов make показывает справку.
.DEFAULT_GOAL := help
-include .env

TAGS = webkit2_41
FRONTEND_DIR = frontend
GOCACHE ?= /tmp/go-build-cache
DOCKER_PLATFORM ?= linux/amd64
DOCKERHUB_IMAGE = hehelf/docflow-service

.PHONY: help help-checks dev-client dev-server storage-up storage-down storage-reset build-linux build-windows docker-server-push release-gate release-assets _build-compiler _check-docker _check-docflow-server-version

help:
	@printf '%s\n' \
	  'Разработка:' \
	  '  make dev-client          Запустить клиент Wails' \
	  '  make dev-server          Собрать локальный сервер и обновить Compose-стек' \
	  'Локальный стек:' \
	  '  make storage-up          Собрать сервер и запустить весь стек' \
	  '  make storage-down        Остановить стек, сохранив данные' \
	  '  make storage-reset       Удалить данные Compose и запустить стек заново' \
	  'Продакшен:' \
	  '  make build-linux         Собрать клиент Linux' \
	  '  make build-windows       Собрать клиент Windows' \
	  '  make docker-server-push  Собрать и опубликовать сервер в Docker Hub' \
	  'Проверки:' \
	  '  make release-gate        Выполнить основной набор проверок' \
	  '  make help-checks         Показать точечные проверки и обслуживание'

help-checks:
	@printf '%s\n' \
	  'Этапы release-gate (можно запускать отдельно):' \
	  '  release-assets-check docs-links-check go-test integration-test go-vet govulncheck' \
	  '  frontend-ci frontend-lint frontend-test frontend-build npm-audit' \
	  'Дополнительные проверки, не входящие в release-gate:' \
	  '  storage-smoke-test       Проверка API, хранилищ и backup/restore в отдельном стеке' \
	  '  db-performance-check     Измерения производительности PostgreSQL' \
	  '  integration-db-up/down   Ручной запуск/удаление интеграционной БД' \
	  'Обслуживание:' \
	  '  release-assets           Обновить встроенные сведения о релизе' \
	  '  clean                    Удалить локальные бинарники из build/bin'

# Разработка
# Сервер обновляется отдельно; dev-client не перезапускает контейнеры.
dev-client: _build-compiler
	$(MAKE) release-assets
	DOCFLOW_LOCAL_BUILD=1 wails dev -tags $(TAGS) -compiler "$(CURDIR)/build/bin/docflow-go"

dev-server: release-assets _build-compiler _check-docker
	GOCACHE=$(GOCACHE) DOCKER_PLATFORM=$(DOCKER_PLATFORM) bash tools/dev-server.sh

# Весь локальный стек: PostgreSQL, SeaweedFS, Seq, сервер и Caddy.
storage-up: dev-server

storage-down:
	docker compose -f docker-compose.yaml down

# Удаляются все именованные тома Compose, включая БД, файлы, Seq и staging бэкапов.
storage-reset:
	docker compose -f docker-compose.yaml down -v
	$(MAKE) storage-up

# Продакшен
build-linux: _build-compiler
	$(MAKE) release-assets
	wails build -tags $(TAGS) -platform linux/amd64 -compiler "$(CURDIR)/build/bin/docflow-go"

build-windows: _build-compiler
	$(MAKE) release-assets
	wails build -platform windows/amd64 -compiler "$(CURDIR)/build/bin/docflow-go"

# Сборка и публикация используют один идентификатор. Перед запуском — docker login.
docker-server-push: override export DOCFLOW_LOCAL_BUILD=0
docker-server-push: _check-docker _check-docflow-server-version _build-compiler
	@set -eu; \
	 identity="$$(build/bin/docflow-go identity)"; \
	 tag="$(DOCFLOW_SERVER_VERSION).$$(printf '%s' "$$identity" | cut -d: -f1)-$$(printf '%s' "$$identity" | cut -d: -f2)"; \
	 docker build --platform $(DOCKER_PLATFORM) --build-arg VERSION=$(DOCFLOW_SERVER_VERSION) --build-arg BUILD_IDENTITY="$$identity" --build-arg LOCAL_BUILD=0 -f build/server/Dockerfile -t $(DOCKERHUB_IMAGE):$$tag .; \
	 test "$$identity" = "$$(build/bin/docflow-go identity)" || { echo 'Sources changed during image build; publication cancelled' >&2; exit 1; }; \
	 docker push $(DOCKERHUB_IMAGE):$$tag; \
	 docker tag $(DOCKERHUB_IMAGE):$$tag $(DOCKERHUB_IMAGE):latest; \
	 docker push $(DOCKERHUB_IMAGE):latest

release-gate:
	@./tools/release-gate.sh

# Служебные зависимости; вручную запускать не требуется.
release-assets:
	GOCACHE=$(GOCACHE) go generate ./internal/releaseassets

_build-compiler:
	GOCACHE=$(GOCACHE) go build -o build/bin/docflow-go ./tools/buildmeta

_check-docker:
	@command -v docker >/dev/null 2>&1 || (echo "docker is required" >&2; exit 1)
	@docker info >/dev/null 2>&1 || (echo "docker daemon is not available" >&2; exit 1)

_check-docflow-server-version: release-assets-check
	@test -n "$(DOCFLOW_SERVER_VERSION)" || (echo "DOCFLOW_SERVER_VERSION is required in .env" >&2; exit 1)
	@actual="$$(GOCACHE=$(GOCACHE) go run ./cmd/docflow-server version)"; \
		test "$$actual" = "$(DOCFLOW_SERVER_VERSION)" || (echo "DOCFLOW_SERVER_VERSION $(DOCFLOW_SERVER_VERSION) does not match embedded product version $$actual" >&2; exit 1)

include build/make/checks.mk
