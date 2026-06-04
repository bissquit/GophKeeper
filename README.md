# GophKeeper

GophKeeper представляет собой клиент-серверную систему, позволяющую пользователю надёжно и безопасно хранить логины, пароли, бинарные данные и прочую приватную информацию

## Сервер

### Запуск

```bash
# .env (опционально) — переопределить порт Postgres:
#   POSTGRES_PORT=5433
docker compose up -d postgres

# для dev создаем самоподписанный сертификат
mkdir -p ./tls
openssl req -x509 -newkey rsa:2048 -keyout ./tls/key.pem -out ./tls/cert.pem \
  -days 365 -nodes -subj '/CN=localhost' \
  -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1'

DATABASE_URI='postgres://gophkeeper:gophkeeper@localhost:5433/gophkeeper?sslmode=disable' \
JWT_SECRET='change-me-because-it-should-be-long-and-strong-secret' \
TLS_CERT_FILE=./tls/cert.pem \
TLS_KEY_FILE=./tls/key.pem \
RUN_ADDRESS=':8443' \
go run ./cmd/gophkeeper-server
```

Нюансы:
- Миграции применяются автоматически при старте (реализовано внутри приложения)
- `JWT_SECRET`, `TLS_CERT_FILE`, `TLS_KEY_FILE` обязательны — без них сервер не стартует
- TLS включён всегда

## Клиент

### Сборка

```bash
# текущая ОС
go build -o gophkeeper ./cmd/gophkeeper-client

# с пробросом версии
go build -ldflags "-X main.version=1.0.0 -X main.buildDate=$(date +%F)" \
    -o gophkeeper ./cmd/gophkeeper-client

# сборка для разных ОС
GOOS=linux   GOARCH=amd64 go build -o gophkeeper-linux   ./cmd/gophkeeper-client
GOOS=darwin  GOARCH=arm64 go build -o gophkeeper-darwin  ./cmd/gophkeeper-client
GOOS=windows GOARCH=amd64 go build -o gophkeeper.exe     ./cmd/gophkeeper-client
```

### Использование

```bash
export GOPHKEEPER_SERVER=https://localhost:8443
# для self-signed dev-сертификата — указать CA в env (по умолчанию есть доверие сертам
# из системы):
export GOPHKEEPER_CA=./tls/cert.pem

gophkeeper register alice         # пароль спросит интерактивно
gophkeeper login alice
gophkeeper whoami
gophkeeper logout
gophkeeper version
gophkeeper ping

# секреты (после login — спрашивает мастер-пароль)
gophkeeper add credentials gmail --login alice@ex.com --meta "main"
gophkeeper add text recovery --text "the quick brown fox"
gophkeeper add binary key --file ./id_rsa
gophkeeper add card visa --number 4111111111111111 --expiry 12/29 --cvv 123

gophkeeper list
gophkeeper get gmail
gophkeeper delete gmail
```

Сессия (URL сервера + JWT) лежит в `$HOME/.config/gophkeeper/session.json`
(на macOS — `~/Library/Application Support/gophkeeper/`).

## Безопасность

- Секреты шифруется на клиенте (сервер и СУБД видят только ciphertext)
- Мастер-пароль нигде не сохраняется и не передаётся. **Если забыл — данные не восстановить**
- Транспорт — TLS. Сервер не запустится без `TLS_CERT_FILE`/`TLS_KEY_FILE`
- JWT-токен в файле сессии — кто читает файл, тот действует от имени пользователя

## Тесты

```bash
# быстрый прогон — только юнит-тесты
go test ./...

# с интеграционными (поднимает Postgres-контейнер через testcontainers-go)
go test -tags=integration ./...

# покрытие
go test -tags=integration -coverpkg=./internal/... -coverprofile=cov.out ./...
go tool cover -func=cov.out | tail -1
```

Интеграционные тесты для `internal/repository/db` помечены `//go:build integration`, в обычный прогон не попадают.
