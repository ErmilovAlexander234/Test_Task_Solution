# Shizo_Solution for test task

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Миграции применяются автоматически при старте (migrations/).

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

### Базовый префикс API:

```text
/api/v1
```

## Задачи одиночные:

```
POST   /tasks
GET    /tasks
GET    /tasks/{id}
PUT    /tasks/{id}
DELETE /tasks/{id}
```

## Повторяющиеся задачи

### Создание правила

```
POST /tasks/recurring
```

### JSON-Body запроса

```
{
  "title": "string",
  "description": "string (optional)",
  "status": "new",
  "recurrence": {}
}
```

### Типы повторений

#### Ежедневные

```
{
  "type": "daily",
  "every_n_days": 2,
  "start_date": "2025-05-01"
}
```

#### Ежемесячные

```
{
  "type": "monthly_days",
  "monthly_days": [1, 15]
}
```

#### Конкретные даты

```
{
  "type": "specific_dates",
  "specific_dates": ["2026-05-10", "2026-06-15"]
}
```

#### Чётные / нечётные дни

```
{
  "type": "even_odd_days",
  "even_odd": "even",
  "start_date": "2025-05-01"
}
```

Горизонт генерации: 365 дней
Если дат нет — возвращается []

### Правила повторения

```
GET    /recurrence-rules
GET    /recurrence-rules/{id}/tasks
DELETE /recurrence-rules/{id}/tasks
DELETE /recurrence-rules/{id}
```

## Архитектура
```
internal/domain/task         — сущности
internal/usecase/task        — бизнес-логика
internal/repository/postgres — PostgreSQL
internal/transport/http      — HTTP слой
cmd/api                      — точка входа
```

Основные маршруты:

- `POST /api/v1/tasks`
- `POST /api/v1/tasks/recurring`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`
- `POST /api/v1/tasks/recurring`
- `GET /api/v1/recurrence-rules`
- `GET /api/v1/recurrence-rules/{id}`

![Springtrap fire](springtrap-fire.gif)
