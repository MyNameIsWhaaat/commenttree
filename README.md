# CommentTree

HTTP-сервис для древовидных комментариев с неограниченной вложенностью, постраничной навигацией, сортировкой, полнотекстовым поиском и простым Web UI.

## Возможности

- **CRUD**
  - `POST /comments` — создание комментария (корень: `parent_id = 0`)
  - `GET /comments?parent={id}` — получение комментариев (дети указанного `parent`) **с полным поддеревом**
  - `DELETE /comments/{id}` — удаление комментария и всего поддерева
- **Пагинация и сортировка** для выдачи детей `parent` (`page`, `limit`, `sort`)
- **Полнотекстовый поиск** (PostgreSQL FTS) + подсветка фрагментов (`snippet`)
- **Навигация из поиска**
  - `GET /comments/path?id={id}` — путь от корня до комментария
  - `GET /comments/subtree?id={id}` — поддерево указанного узла (используется UI)
- **Web UI** (без фреймворков): просмотр дерева, ответы, удаление, поиск и переход к найденному комментарию
- **Redis cache (опционально)** для дерева/поддерева (ускоряет повторные запросы)
- **Docker Compose**: `postgres + migrate + api + redis` в одной связке
- **Graceful shutdown** HTTP-сервера

## Стек

- Go (роутинг на `net/http` с method-based patterns, Go **1.22+**)
- PostgreSQL 16
- Redis 7 (опционально)
- Migrations: `migrate/migrate`
- Docker / Docker Compose

---

## Быстрый старт (Docker)

docker compose up -d --build

Проверка:

- API: http://localhost:8080/healthz
- UI: http://localhost:8080/

Логи:

```
docker compose logs -f api
docker compose logs -f migrate
docker compose logs -f postgres
docker compose logs -f redis
```

Остановка:

```
docker compose down
```

Сброс данных (вместе с volume БД):

```
docker compose down -v
```

## API
### Healthcheck

#### GET /healthz

Ответ:

```
{"result":"ok"}
```

### Создать комментарий

#### POST /comments

Body:

```
{
  "parent_id": 0,
  "text": "Привет!"
}
```

Ответ 201:

```
{
  "id": 1,
  "parent_id": 0,
  "text": "Привет!",
  "created_at": "2026-02-24T15:12:02Z"
}
```

### Получить дерево детей parent (с поддеревом)

#### GET /comments?parent=0&page=1&limit=30&sort=created_at_desc

Параметры:

- parent (default 0) — id родителя
- page (default 1)
- limit (default 20, max 100)
- sort: created_at_desc (default) | created_at_asc

Ответ 200:

```
{
  "items": [
    {
      "id": 1,
      "parent_id": 0,
      "text": "Привет!",
      "created_at": "...",
      "children": [
        { "id": 2, "parent_id": 1, "text": "Ответ", "created_at": "...", "children": [] }
      ]
    }
  ],
  "page": 1,
  "limit": 30,
  "total": 1
}
```

### Удалить поддерево

#### DELETE /comments/{id}

Ответ:

```
{ "deleted": 7 }
```

### Поиск (FTS)

#### GET /comments/search?q=привет&page=1&limit=20&sort=rank_desc

Параметры:

- q — запрос
- sort: rank_desc (default) | created_at_desc | created_at_asc

Ответ:

```
{
  "items": [
    {
      "id": 1,
      "parent_id": 0,
      "snippet": "…<mark>привет</mark>…",
      "rank": 0.12,
      "created_at": "..."
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 1
}
```

### Навигация для UI

- GET /comments/path?id={id} — путь от корня до id
- GET /comments/subtree?id={id}&sort=created_at_desc — поддерево одного корня/узла

## Web UI

UI доступен по адресу: http://localhost:8080/

Доступные действия:

- просмотр дерева комментариев с вложенностью
- добавление корневых комментариев и ответов
- удаление комментариев (вместе с поддеревом)
- поиск по комментариям + переход к найденному месту в дереве

Структура проекта

```
commenttree/
  cmd/commenttree/            # entrypoint
  internal/comment/
    model/                    # доменные типы/DTO
    storage/                  # репозитории (postgres/inmemory)
    service/                  # бизнес-логика + кеш
    handler/http/             # HTTP слой
  migrations/                 # SQL миграции (migrate)
  web/                        # UI (html/css/js)
  Dockerfile
  docker-compose.yml
  ```

## Проверки качества

Базовые:
```
go test ./...
go vet ./...
gofmt -w .
```
Race:
```
go test -race ./...
```
Lint (golangci-lint):
```
golangci-lint run ./...
```
