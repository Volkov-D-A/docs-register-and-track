# Используем свежий Go (Debian-based)
FROM golang:1.26-bookworm

# 1. Системные зависимости + Node.js 22
RUN apt-get update && apt-get install -y \
    build-essential \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    pkg-config \
    curl \
    && curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# 2. Wails CLI
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest

WORKDIR /app

# 3. Копируем всё
COPY . .

# 4. ВАЖНО: Заходим во frontend и устанавливаем зависимости для LINUX
# Это создаст правильную папку node_modules внутри образа
WORKDIR /app/frontend
# Удаляем package-lock.json, чтобы не конфликтовать с Windows-версией
RUN rm -f package-lock.json
RUN npm install

# Возвращаемся в корень для сборки
WORKDIR /app
CMD ["wails", "build"]