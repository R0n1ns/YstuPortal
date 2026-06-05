# YstuPortal

## Описание
Пет-проект: backend на Go для портала с авторизацией, парсингом данных, кешем оценок и ролью пользователя.

### Стек
- Go, Fiber
- PostgreSQL + миграции
- Redis (кеш оценок)
- Swagger/OpenAPI

## Быстрый старт (Docker)

### Требования
- Docker Desktop

### Подготовка
Скопируйте переменные окружения:

```cmd
copy .env.example .env
```

### Запуск
Запустите сервисы:

```cmd
docker compose up -d
```

Примените миграции:

```cmd
set DATABASE_URL=postgres://user:user@localhost:5432/ystu_db?sslmode=disable
go run ./cmd/migrate/main.go -direction up
```

Откройте в браузере:
- http://127.0.0.1:8081/web_version.html
- http://127.0.0.1:8080/swagger
- http://127.0.0.1:8080/metrics

### Остановка
```cmd
docker compose down
```

## Локальный запуск (без Docker)

1) Поднимите PostgreSQL и Redis локально.
2) Создайте файл `.env` на основе `.env.example`.
3) Примените миграции и запустите API:

```cmd
go run ./cmd/migrate/main.go -direction up
go run ./cmd/api/main.go
```

## Проверки качества

```cmd
go test ./...
golangci-lint run
```

## Миграции

```cmd
go run ./cmd/migrate/main.go -direction up
go run ./cmd/migrate/main.go -direction down -steps 1
```

