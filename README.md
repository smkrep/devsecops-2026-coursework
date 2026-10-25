# API gateway

Минимальный шлюз на Go 1.26: HTTP/JSON → gRPC-Gateway → gRPC-сервер
шлюза → gRPC backend. Также поддерживается прямой вход по gRPC.

Пока реализован один метод `catalog.v1.CatalogService/GetItem` с HTTP-маршрутом
`GET /api/v1/items/{id}`. Backend должен реализовывать тот же protobuf-контракт;
его реализация в этот проект пока не входит.

## Сборка и запуск

```sh
make build
./bin/gateway
```

Настройки задаются переменными окружения:

| Переменная | Значение по умолчанию |
| --- | --- |
| `HTTP_ADDR` | `127.0.0.1:8080` |
| `GRPC_ADDR` | `127.0.0.1:9090` |
| `BACKEND_ADDR` | `127.0.0.1:9092` |
| `BACKEND_TIMEOUT` | `3s` |

Шлюз запускается без доступного backend. В этом случае запрос
`curl http://127.0.0.1:8080/api/v1/items/42` получит HTTP 503 с gRPC-ошибкой
`Unavailable`. При доступном backend возвращаются его поля `id` и `name`.
Deadline и отмена запроса передаются через контекст; исходящий вызов дополнительно
ограничен `BACKEND_TIMEOUT`. Серверы останавливаются по SIGINT/SIGTERM.

В этой версии соединения работают без TLS; аутентификация, лимиты,
балансировка, метрики и служебные endpoints будут добавлены позже.
Входящий HTTP пока работает по HTTP/1.1, gRPC — по HTTP/2.

## Генерация

Сгенерированные файлы в `gen/` хранятся в репозитории: для обычной сборки
генераторы не нужны. После изменения `.proto` выполни `make generate`.
Makefile добавляет `$(go env GOPATH)/bin` в PATH.

Использованные инструменты:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.9
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.27.3
```

Также нужен `protoc` со стандартными protobuf-определениями
(`protobuf-compiler` и `libprotobuf-dev` в Debian/Ubuntu).
Google API-аннотации находятся в `third_party/google/api/`;
исходный проект: https://github.com/googleapis/googleapis.

## Пакеты

- `cmd/gateway`: точка входа и сигналы остановки.
- `internal/app`: создание компонентов и управление серверами/соединениями.
- `internal/config`: настройки из окружения.
- `internal/gateway`: прикладной сервис, внутренний тип Item и backend-интерфейс.
- `internal/transport`: входящие gRPC и HTTP.
- `internal/upstream`: исходящий gRPC-клиент.
- `api/proto`: исходный контракт; `gen`: сгенерированный код.

Интеграционные тесты пока не добавляются. `make test` зарезервирован для
последующих модульных тестов; сейчас тестовых файлов нет.
