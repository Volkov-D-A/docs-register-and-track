# Обновление пакетов проекта
1. Бэкенд
Вы можете посмотреть список доступных обновлений с помощью команды:
go list -u -m all
Обновить все пакеты до последних минорных версий/патчей:
go get -u ./...
go mod tidy
go install github.com/wailsapp/wails/v2/cmd/wails@latest

2. Фронтенд
Проверить устаревшие пакеты:
npm outdated
Безопасное обновление (минорные версии и патчи):
npm update
Обновление до последних мажорных версий (с осторожностью):
npx npm-check-updates -u (npx npm-check-updates -i интерактивный режим)
npm install

# Актуализация рабочего места после обновления:
1. Подтягиваем зависимости Go (Бэкенд)
go mod download

2. Устанавливаем зависимости фронтенда (Чистая установка)
cd frontend
npm ci

# Тестовое обновлени через git branch:
1. Подготовка (Чистый лист)
git status

2. Создание «песочницы»
git switch -c test-major-updates

3. Эксперимент (Обновление)
npx npm-check-updates -i
npm install.

4. Принятие решения (Два пути)
Сценарий А: Успех! Всё работает отлично.
git add .
git commit -m "chore: update vite and typescript to latest major versions"
git switch main
git merge test-major-updates
git branch -d test-major-updates

Сценарий Б: Провал. Всё сломалось.
git switch main
git branch -D test-major-updates
cd frontend
npm ci