# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## URL
http://localhost:8080
http://localhost:8080/update/test/name/2
http://localhost:8080/update/test/name
http://localhost:8080/update/test
http://localhost:8080/update

## Запуск с параметрами по умолчанию
go run cmd/server/main.go -l debug -k "test"
go run cmd/agent/main.go -k "test"
export DATABASE_DSN="host=localhost user=admin password=*** dbname=practicum sslmode=disable"
go build -o server main.go

## Для отладки - тестовый запрос через curl
curl -X GET -H "Content-Type: text/html" -H "Accept-Encoding: gzip" "http://localhost:8080/" -I -v
curl -X GET -H "Content-Type: text/plain" "http://localhost:8080/value/gauge/TestMetric/"
curl -X POST -H "Content-Type: text/plain" "http://localhost:8080/update/gauge/TestMetric/123.456"
curl -X POST -H "Content-Type: text/plain" -w '%{http_code}\n' "http://localhost:8080/update/gauge/TestMetric/123.456"

curl -X POST -H "Content-Type: application/json" -d '{"id":"LastGC","type":"gauge","value":1744184459}' "http://localhost:8080/update/" 
curl -X POST -H "Content-Type: application/json" -H "Accept: application/json" -d '{"id":"Test","type":"counter","delta":2}' "http://localhost:8080/update/" 
curl -X POST -H "Content-Type: application/json" -H "Accept: application/json" -d '{"id":"PollCount","type":"counter"}' "http://localhost:8080/value/" 
curl -X POST -H "Content-Type: application/json" -H "Accept: application/json" -v -d '{"id":"LastGC","type":"gauge"}' "http://localhost:8080/value/"
curl -X POST -H "Content-Type: application/json" -H "Accept-Encoding: gzip" -v -d '{"id":"LastGC","type":"gauge","value":1744184459}' "http://localhost:8080/update/" 

curl -X GET "http://localhost:8080/ping"

curl -X POST -H "Content-Type: application/json" -d '[{"id":"Test3","type":"gauge","value":5.3},{"id":"Test4","type":"gauge","value":7},{"id":"Test","type":"counter","delta":2},{"id":"Test3","type":"counter","delta":4}]' "http://localhost:8080/updates/" 

metricstest -test.v -test.run=^TestIteration14$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -database-dsn='postgres://admin:***@localhost:5432/practicum?sslmode=disable' \
            -key="test" \
            -server-port=8080 \
            -source-path=.