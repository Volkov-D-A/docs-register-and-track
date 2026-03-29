.PHONY: dev build-linux build-windows clean release-assets

# Загружаем переменные из .env (если файл существует)
-include .env
export

# Переменные
TAGS = webkit2_41

# Ключ шифрования конфигурации (из .env → ENCRYPTION_KEY)
LDFLAGS = -X 'github.com/Volkov-D-A/docs-register-and-track/internal/config.rawEncryptionKey=$(ENCRYPTION_KEY)'

release-assets:
	go generate ./internal/releaseassets

# Запуск режима разработки с правильным WebKit для Ubuntu 24.04
dev:
	$(MAKE) release-assets
	wails dev -tags $(TAGS) -ldflags "$(LDFLAGS)"

# Сборка готового бинарника для тестирования в Linux
build-linux:
	$(MAKE) release-assets
	wails build -tags $(TAGS) -platform linux/amd64 -ldflags "$(LDFLAGS)"

# Кросс-компиляция готового .exe для Windows (для конечных пользователей)
build-windows:
	$(MAKE) release-assets
	wails build -platform windows/amd64 -ldflags "$(LDFLAGS)"

# Очистка кэша сборки и папки build/bin
clean:
	rm -rf build/bin/*

# Запуск тестов
test:
	$(MAKE) release-assets
	go test ./...

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
