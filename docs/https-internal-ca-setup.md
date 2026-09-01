# Переход Docflow на HTTPS с внутренним CA Caddy

Документ описывает переход для схемы:

```text
Desktop -> HTTPS :443 -> Caddy -> HTTP :8080 -> docflow-server
```

Caddy использует собственный внутренний центр сертификации (`tls internal`),
автоматически выпускает и обновляет серверные сертификаты. Публичный корневой
сертификат CA встраивается в desktop-приложение. Установка сертификата в
системное хранилище Windows при таком варианте не обязательна.

Закрытый ключ CA хранится только в постоянном внешнем Docker volume. Внешний
volume не удаляется командой `make storage-reset`, которая регулярно удаляет
данные PostgreSQL, MinIO и Seq в среде разработки.

## 1. Ограничения и важные правила

- В desktop встраивается только публичный `root.crt`.
- Нельзя встраивать или распространять `root.key`.
- Нельзя использовать `InsecureSkipVerify: true`, `curl -k` или отключать
  проверку сертификата в production.
- DNS-имя в `server.url` должно точно совпадать с именем в сертификате.
- Данные `/data` контейнера Caddy нельзя считать временным кэшем: там находятся
  CA, закрытые ключи и выпущенные сертификаты.
- Обычный `docker compose down` сохраняет volumes, но
  `docker compose down -v` удаляет все обычные volumes проекта.
- Внешний volume Caddy не удаляется Compose, однако его всё ещё можно удалить
  явной командой `docker volume rm` или очисткой неиспользуемых именованных
  volumes. Такие операции следует выполнять с осторожностью.

Клиентские сертификаты и mTLS в этой схеме не используются. Пользователи
продолжают входить через существующие bearer-сессии Docflow. Это исключает
необходимость выпускать, обновлять и отзывать отдельный сертификат для каждого
рабочего места.

## 2. Подготовить DNS-имя

Выбрать постоянное внутреннее имя, например:

```text
docflow.company.lan
```

Создать во внутреннем DNS запись, указывающую на сервер Ubuntu:

```text
docflow.company.lan -> 192.168.10.20
```

Во всех командах ниже нужно заменить `docflow.company.lan` и IP-адрес на
фактические значения.

Проверить имя на сервере:

```bash
getent hosts docflow.company.lan
```

Проверить имя на Windows-рабочем месте:

```powershell
Resolve-DnsName docflow.company.lan
```

Если внутреннего DNS пока нет, имя можно временно добавить в файл
`C:\Windows\System32\drivers\etc\hosts` каждого рабочего места. Для постоянной
эксплуатации предпочтителен DNS.

## 3. Сделать резервные копии конфигурации

Перейти в каталог проекта на сервере:

```bash
cd /path/to/docs-register-and-track
```

Сохранить текущие файлы:

```bash
cp docker-compose.yaml docker-compose.yaml.before-https
cp build/caddy/Caddyfile build/caddy/Caddyfile.before-https
```

## 4. Создать внешний volume для CA

Эта операция выполняется один раз:

```bash
docker volume create docflow-caddy-data
```

Проверить volume:

```bash
docker volume inspect docflow-caddy-data
```

В `docker-compose.yaml` у сервиса `caddy` подключить `/data` и `/config`:

```yaml
  caddy:
    image: caddy:${CADDY_VERSION:?Set CADDY_VERSION in .env}
    container_name: docflow_caddy
    depends_on:
      docflow-server:
        condition: service_healthy
    ports:
      # Временный HTTP оставляется только на время контролируемого перехода.
      - "${DOCFLOW_SERVER_PORT:-8080}:80"
      - "443:443"
    volumes:
      - ./build/caddy:/etc/caddy:ro
      - caddy_data:/data
      - caddy_config:/config
    restart: unless-stopped
```

В нижней секции `volumes` объявить `caddy_data` внешним:

```yaml
volumes:
  pgdata:
  minio_data:
  seq_data:

  caddy_data:
    external: true
    name: docflow-caddy-data

  caddy_config:
```

Ключевое свойство здесь — `external: true`. Compose использует заранее
созданный volume, но не управляет его жизненным циклом. Поэтому
`docker compose down -v` не удаляет `docflow-caddy-data`.

`caddy_config` не содержит корневой ключ и может оставаться обычным volume.
Критически важным для сохранения TLS-идентичности является `caddy_data`.

Проверить итоговую Compose-конфигурацию:

```bash
docker compose config
```

## 5. Временно включить HTTP и HTTPS параллельно

На время перехода заменить `build/caddy/Caddyfile` следующим содержимым:

```caddyfile
http://:80 {
	request_body {
		max_size 1GB
	}

	reverse_proxy docflow-server:8080
}

docflow.company.lan {
	tls internal

	request_body {
		max_size 1GB
	}

	reverse_proxy docflow-server:8080
}
```

Первый блок временно сохраняет старый HTTP endpoint на порту из
`DOCFLOW_SERVER_PORT`. Второй блок включает HTTPS на порту 443 и заставляет
Caddy выпустить сертификат для `docflow.company.lan` через внутренний CA.

HTTP нельзя использовать для передачи production credentials через
недоверенную или маршрутизируемую сеть. Параллельный режим предназначен только
для короткого контролируемого перехода в доверенном контуре.

Проверить Caddyfile:

```bash
docker compose run --rm --no-deps caddy \
  caddy validate \
  --config /etc/caddy/Caddyfile \
  --adapter caddyfile
```

Запустить обновлённый Caddy:

```bash
docker compose up -d --force-recreate caddy
docker compose ps
docker compose logs --tail=100 caddy
```

Проверить публикацию HTTPS-порта:

```bash
docker compose port caddy 443
```

## 6. Извлечь публичный корневой сертификат

После первого успешного запуска Caddy создаёт локальный CA во внешнем volume.
Скопировать только его публичный сертификат:

```bash
docker compose cp \
  caddy:/data/caddy/pki/authorities/local/root.crt \
  /tmp/docflow-root-ca.crt
```

Проверить сертификат и записать SHA-256 fingerprint:

```bash
openssl x509 \
  -in /tmp/docflow-root-ca.crt \
  -noout \
  -subject \
  -issuer \
  -dates \
  -fingerprint \
  -sha256
```

Закрытый ключ расположен внутри volume по пути
`/data/caddy/pki/authorities/local/root.key`. Его нельзя копировать в проект,
встраивать в приложение или передавать на рабочие места.

## 7. Проверить HTTPS до изменения desktop

На сервере выполнить:

```bash
curl \
  --cacert /tmp/docflow-root-ca.crt \
  https://docflow.company.lan/health/live
```

Ожидаемый ответ:

```json
{"status":"live"}
```

Проверить readiness:

```bash
curl \
  --cacert /tmp/docflow-root-ca.crt \
  https://docflow.company.lan/health/ready
```

Ожидаемый ответ при исправных зависимостях и совместимой схеме:

```json
{"status":"ready"}
```

Проверить TLS-цепочку и имя сервера:

```bash
openssl s_client \
  -connect docflow.company.lan:443 \
  -servername docflow.company.lan \
  -CAfile /tmp/docflow-root-ca.crt \
  -verify_return_error </dev/null
```

В конце вывода должно быть:

```text
Verify return code: 0 (ok)
```

## 8. Встроить CA в desktop

Создать каталог:

```bash
mkdir -p internal/serverclient/certs
```

Скопировать публичный сертификат:

```bash
cp /tmp/docflow-root-ca.crt \
  internal/serverclient/certs/docflow-root-ca.crt
```

Публичный сертификат не является секретом. Закрытый ключ CA при этом остаётся
только во внешнем volume на сервере.

Добавить в пакет `internal/serverclient` файл `tls_roots.go`:

```go
package serverclient

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"net/http"
	"time"
)

//go:embed certs/docflow-root-ca.crt
var embeddedDocflowRootCA []byte

func newHTTPClient() (*http.Client, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if ok := roots.AppendCertsFromPEM(embeddedDocflowRootCA); !ok {
		return nil, fmt.Errorf("embedded Docflow root CA is invalid")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   3 * time.Minute,
	}, nil
}
```

В `internal/serverclient/migrations.go` функция `NewWithOptions` сейчас создаёт
обычный `http.Client`. Заменить создание клиента на:

```go
httpClient, err := newHTTPClient()
if err != nil {
	return nil, err
}

return &Client{
	baseURL: rawURL,
	http:    httpClient,
}, nil
```

Проверка hostname остаётся включённой. Значение `InsecureSkipVerify` добавлять
нельзя.

Рекомендуется добавить тесты:

- embedded PEM успешно разбирается;
- TLS-сервер с сертификатом от другого CA отклоняется;
- сертификат с неправильным DNS-именем отклоняется;
- HTTP вне loopback отклоняется при `allowInsecureHttp=false`;
- существующие HTTP/client contract tests продолжают проходить.

Запустить проверки:

```bash
GOCACHE=/tmp/docflow-go-cache go test ./internal/serverclient ./internal/app
cd frontend && npm test
```

## 9. Переключить desktop-конфигурацию

Использовать конфигурацию:

```json
{
  "server": {
    "url": "https://docflow.company.lan",
    "allowInsecureHttp": false
  }
}
```

Собрать и опубликовать desktop обычным централизованным способом. До замены
общего executable необходимо закрыть все ранее запущенные desktop-процессы.

Проверить на рабочем месте:

1. запуск приложения;
2. первичную настройку или вход;
3. открытие списков и карточки документа;
4. регистрацию тестового документа;
5. загрузку и скачивание вложения;
6. logout и повторный вход.

Браузер и `curl.exe` на рабочем месте могут продолжать считать сертификат
недоверенным: встроенный CA используется только HTTP-клиентом Docflow. Это
ожидаемое поведение. Если системное доверие также требуется, `root.crt` нужно
отдельно установить в Trusted Root Certification Authorities Windows.

## 10. Отключить временный HTTP

После успешного переключения всех рабочих мест удалить HTTP-блок из Caddyfile:

```caddyfile
docflow.company.lan {
	tls internal

	request_body {
		max_size 1GB
	}

	reverse_proxy docflow-server:8080
}
```

В `docker-compose.yaml` удалить HTTP mapping:

```yaml
ports:
  - "443:443"
```

Применить изменения:

```bash
docker compose config
docker compose up -d --force-recreate caddy
docker compose logs --tail=100 caddy
```

Убедиться, что старый endpoint недоступен, а HTTPS работает:

```bash
curl --connect-timeout 5 http://docflow.company.lan:8080/health/live

curl \
  --cacert /tmp/docflow-root-ca.crt \
  https://docflow.company.lan/health/live
```

## 11. Поведение `make storage-reset`

Target выполняет:

```make
docker compose down -v
docker compose up -d
```

После описанной настройки он удаляет обычные volumes:

- `pgdata` — база данных, пользователи, документы и server sessions;
- `minio_data` — вложения;
- `seq_data` — технические логи;
- `caddy_config` — восстанавливаемое runtime-состояние конфигурации Caddy.

Внешний `docflow-caddy-data` сохраняется. Поэтому остаются прежними:

- корневой CA;
- закрытый ключ CA;
- TLS-идентичность сервера;
- доверие встроенного в desktop `root.crt`.

После сброса БД все bearer-сессии становятся недействительными, сервер заново
bootstrap-ит схему и потребуется повторная первичная настройка администратора.
Пересобирать desktop из-за HTTPS не потребуется.

Проверка после сброса:

```bash
make storage-reset
docker volume inspect docflow-caddy-data
docker compose logs --tail=50 caddy

curl \
  --cacert /tmp/docflow-root-ca.crt \
  https://docflow.company.lan/health/live
```

## 12. Резервное копирование CA

Volume содержит закрытый ключ CA и должен резервироваться как секрет. Пример
создания архива:

```bash
mkdir -p /secure-backup/docflow

docker run --rm \
  -v docflow-caddy-data:/source:ro \
  -v /secure-backup/docflow:/backup \
  ubuntu:26.04 \
  tar -C /source -czf /backup/caddy-data.tar.gz .
```

Архив необходимо зашифровать утверждённым средством резервного копирования,
ограничить права доступа и хранить отдельно от Docker-сервера.

Проверить наличие файла:

```bash
sudo ls -l /secure-backup/docflow/caddy-data.tar.gz
```

Восстанавливать CA следует только в заранее созданный пустой внешний volume и
до запуска Caddy. После восстановления нужно сверить SHA-256 fingerprint
`root.crt` со значением, встроенным в опубликованный desktop.

## 13. Ротация корневого CA

Caddy автоматически обновляет серверные leaf-сертификаты и intermediate.
Обновление desktop для этих операций не требуется. Плановый выпуск desktop
потребуется только при замене самого корневого CA или при компрометации его
закрытого ключа.

Безопасный порядок ротации root CA:

1. Создать новый CA, не удаляя старый.
2. Выпустить desktop, доверяющий одновременно старому и новому `root.crt`.
3. Закрыть все процессы старой desktop-версии и опубликовать новый executable.
4. Переключить Caddy на новый CA.
5. Проверить HTTPS и основные пользовательские сценарии.
6. В следующем плановом desktop-релизе удалить старый CA.

При потере или компрометации `docflow-caddy-data` требуется внеплановая ротация
CA и повторная публикация desktop. Поэтому внешний volume и его защищённая
резервная копия являются обязательной частью этой схемы.

## 14. Финальный сетевой cutover

HTTPS не закрывает прямой сетевой доступ к PostgreSQL и MinIO автоматически.
В production Compose у сервисов `postgres` и `minio` следует удалить внешние
`ports`, оставив доступ только через внутреннюю Docker network.

После этого необходимо:

1. проверить, что с рабочего места доступны только HTTPS-порт Docflow и
   разрешённые операторские endpoints;
2. закрыть PostgreSQL `5432` и MinIO `9000/9001` от пользовательской сети;
3. ротировать прежние PostgreSQL/MinIO credentials;
4. удалить старые desktop-конфигурации, содержащие инфраструктурные secrets;
5. выполнить production-like login, document, attachment и restore smoke.

На Linux опубликованные Docker-порты могут обходить обычные правила UFW.
Поэтому основной мерой должно быть отсутствие `ports` у PostgreSQL и MinIO, а
не только добавление запретов UFW.

## 15. Критерии завершения

Переход на HTTPS завершён, если одновременно выполнено следующее:

- desktop использует `https://docflow.company.lan`;
- `allowInsecureHttp` выключен;
- сертификат проверяется через встроенный CA без `InsecureSkipVerify`;
- старый HTTP endpoint закрыт;
- `docflow-caddy-data` объявлен внешним и переживает `make storage-reset`;
- защищённая резервная копия CA создана и проверена;
- после `make storage-reset` HTTPS продолжает работать с прежним desktop;
- PostgreSQL и MinIO не опубликованы в пользовательскую сеть;
- старые инфраструктурные credentials ротированы;
- production-like smoke и восстановление backup-набора выполнены.
