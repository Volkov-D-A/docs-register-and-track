.PHONY: dev build-linux build-windows clean

# Переменные
TAGS = webkit2_41

# Запуск режима разработки с правильным WebKit для Ubuntu 24.04
dev:
	wails dev -tags $(TAGS)

# Сборка готового бинарника для тестирования в Linux
build-linux:
	wails build -tags $(TAGS) -platform linux/amd64

# Кросс-компиляция готового .exe для Windows (для конечных пользователей)
build-windows:
	wails build -platform windows/amd64

# Очистка кэша сборки и папки build/bin
clean:
	rm -rf build/bin/*