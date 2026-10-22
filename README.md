# battery-life-backend

Бэкенд SPA «Battery Life» на Go + Gin. Домен — сценарии использования аккумулятора
(«услуги»): пользователь создаёт черновик, публикует его, ставит лайки и удаляет.

- **ЛР3** — REST API, PostgreSQL (GORM), MinIO.
- **ЛР4** — JWT-аутентификация, ролевая модель, blacklist токенов в Redis, Swagger.

Техническое задание и критерии приёмки ЛР4: [TZ.md](TZ.md).

## Быстрый старт

```bash
cp .env.example .env          # при необходимости поправьте доступы
docker compose up -d          # PostgreSQL, Adminer, MinIO, Redis
go run ./cmd/migrate          # миграции и тестовые данные
go run ./cmd/battery-life     # сервер на http://localhost:8080
```

Swagger UI: **http://localhost:8080/swagger/index.html**

Полезные адреса:

| Сервис | Адрес |
|---|---|
| API | http://localhost:8080/api |
| Swagger UI | http://localhost:8080/swagger/index.html |
| Adminer | http://localhost:8081 |
| MinIO Console | http://localhost:9001 |
| Redis | `127.0.0.1:6379`, пароль `battery_redis` |

## Переменные окружения

`.env` (см. `.env.example`):

| Переменная | Назначение |
|---|---|
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME` | подключение к PostgreSQL |
| `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET_NAME`, `MINIO_USE_SSL`, `MINIO_PUBLIC_URL` | подключение к MinIO |
| `JWT_TOKEN` | секрет подписи JWT (переопределяет `config.toml`) |
| `JWT_EXPIRES_IN` | время жизни access-токена, например `24h` |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_USER`, `REDIS_PASSWORD` | подключение к Redis |

Значения из окружения имеют приоритет над `config/config.toml`.

## Аутентификация и авторизация (ЛР4)

Используется **JWT (HS256) + blacklist в Redis** — вариант для Golang по методичке.

1. `POST /api/users/register` — регистрация. Пароль хэшируется bcrypt, роль всегда `user`
   (`is_moderator` с клиента не принимается).
2. `POST /api/users/login` — проверка логина/пароля, выдача JWT.
   Токен возвращается в теле (`access_token`) и дополнительно кладётся в HttpOnly-cookie
   `access_token` (удобно для Swagger UI и SPA).
3. Клиент передаёт токен в `Authorization: Bearer <JWT>` **или** в cookie.
4. `POST /api/users/logout` — токен попадает в blacklist Redis до истечения срока,
   запись сессии удаляется, cookie очищается. Повторные запросы с этим токеном → `401`.
5. `GET /api/users/me` — данные текущего пользователя.

### Роли

`battery_app_users.is_moderator = false` → роль `user` (`0` в JWT),
`true` → `moderator` (`1`).

| Действие | Гость | `user` | `moderator` |
|---|---|---|---|
| Просмотр списка/ленты | ✅ | ✅ | ✅ |
| Создание/публикация/удаление **своей** услуги | ❌ | ✅ | ✅ |
| Удаление **чужой** услуги | ❌ | ❌ | ✅ |
| Лайк | ❌ | ✅ | ✅ |

Создатель услуги всегда вычисляется на бэкенде из JWT и не принимается из тела запроса.

### Ключи Redis

Префикс сервиса — `battery_life.`:

- `battery_life.session.<jti>` — сессия выданного токена (JSON: `user_id`, `username`, `role`, `role_name`, `issued_at`);
- `battery_life.jwt.blacklist.<token>` — отозванный токен (TTL = остаток жизни JWT).

Посмотреть содержимое:

```bash
docker exec -it battery-life-redis-1 redis-cli -a battery_redis
127.0.0.1:6379> keys battery_life.*
127.0.0.1:6379> get battery_life.session.<jti>
127.0.0.1:6379> ttl battery_life.jwt.blacklist.<token>
```

## Swagger

Документация генерируется из аннотаций в коде утилитой [swag](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.6
swag init -g cmd/battery-life/main.go -o docs --parseInternal
go run ./cmd/battery-life
```

Схема безопасности `BearerAuth` описана в `cmd/battery-life/main.go`; у защищённых методов
в Swagger отображается замок. Публичное чтение доступно без авторизации — это же поведение
проверяется тестами.

## API

Базовый путь — `/api`. Успех: `200`/`201`. Ошибки: `400`, `401`, `403`, `404`, `409`, `500`;
тело ошибки — `{"error": "..."}`. Статус услуги в ответах не передаётся (кроме черновика).

### Пользователи и аутентификация

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `POST` | `/users/register` | публично | `{username, password}` → `201` + `{id, username, is_moderator, created_at}`; `409`, если логин занят |
| `POST` | `/users/login` | публично | `{username, password}` → `200` + `{access_token, token_type, expires_in, user}`; `401` при неверных данных |
| `POST` | `/users/logout` | JWT | blacklist токена и удаление сессии → `200` |
| `GET` | `/users/me` | JWT | текущий пользователь |

### Услуги (сценарии)

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `GET` | `/scenarios` | публично / опционально JWT | опубликованные услуги; фильтры `search`, `max_hours`; поля `is_liked`, `is_creator` |
| `GET` | `/scenarios/feed` | публично / опционально JWT | первая опубликованная услуга |
| `GET` | `/scenarios/feed/:id` | публично / опционально JWT | услуга по id; `?next=true` — следующая, с заворотом в начало; `404`, если удалена |
| `GET` | `/scenarios/draft` | JWT | единственный черновик текущего пользователя; `404`, если его нет |
| `POST` | `/scenarios` | JWT | создание/обновление черновика, `multipart/form-data`: `scenario_title`, файлы `image`, `video` → `201` |
| `PUT` | `/scenarios/:id/publish` | JWT, создатель | публикация; JSON: `scenario_description`, `battery_drain_mah`, `usage_duration_hours` → `200` |
| `DELETE` | `/scenarios/:id` | JWT, создатель/модератор | логическое удаление (`scenario_status = 'deleted'`) → `200` |
| `POST` | `/scenarios/:id/like` | JWT | `{"like": 1}` поставить, `{"like": 0}` отменить → `200` |

Файлы сохраняются в MinIO под сгенерированными латинскими именами, в БД пишутся публичные URL.
Незаполненные значения возвращаются как `null`/`""`, набор полей JSON стабилен.

## Демонстрация ЛР4 (чек-лист)

1. Открыть Swagger в режиме инкогнито, выполнить `POST /users/login` (например,
   `admin_battery` / `admin123`), скопировать `access_token`, нажать **Authorize** и вставить
   `Bearer <токен>`.
2. Без токена выполнить `GET /scenarios` — `200`; выполнить `PUT /scenarios/{id}/publish` — `401`.
3. С токеном выполнить `GET /scenarios` — `200`; `PUT /scenarios/{id}/publish` своего черновика —
   `200` и обновлённые поля.
4. Чужим токеном опубликовать чужой черновик — `403`.
5. В Postman/Insomnia: `GET /scenarios` без токена — `200`; с токеном — `200`.
6. `POST /users/logout`, затем повторный запрос с тем же токеном — `401`.
7. Показать в Redis сессию и blacklist:
   `docker exec -it battery-life-redis-1 redis-cli -a battery_redis keys 'battery_life.*'`.
8. В StarUML открыть `battery_life_diagram.mdj` → диаграмма `Lab4AuthScenarioSequence`:
   lifeline-домены, сообщения с `signature`, переиспользованной из диаграммы классов.

Тестовые пользователи после миграции: `tester_alex` / `pass123`, `admin_battery` / `admin123`,
`tester_elena` / `pass456`. Пароли, сохранённые в ЛР3 открыто, автоматически перехэшируются
bcrypt при первом успешном входе.

## Структура проекта

```
cmd/battery-life        — точка входа, аннотации Swagger
cmd/migrate             — миграции и seed
config/config.toml      — host/port, JWT, Redis
docs                    — сгенерированная OpenAPI-документация
internal/app/auth       — выпуск/проверка JWT, bcrypt, сессии и logout
internal/app/config     — конфигурация + env-overrides
internal/app/ds         — модели GORM, JWT-claims, сериализаторы
internal/app/handler    — HTTP-обработчики, middleware авторизации
internal/app/redisclient— клиент Redis, сессии и blacklist
internal/app/repository — работа с PostgreSQL и MinIO
internal/app/role       — ролевая модель
internal/pkg            — приложение и запуск сервера
```

## Таблицы базы данных

### battery_app_users — зарегистрированные пользователи

| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| username | VARCHAR(50), UNIQUE, NOT NULL | логин |
| password_hash | VARCHAR(100), NOT NULL | bcrypt-хэш (скрыт из JSON) |
| is_moderator | BOOLEAN, NOT NULL, DEFAULT FALSE | роль: `false` — user, `true` — moderator |
| created_at | TIMESTAMPTZ, NOT NULL | дата регистрации |

### battery_scenarios — черновики, опубликованные и удалённые услуги

| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| scenario_status | VARCHAR(20), NOT NULL, DEFAULT 'draft' | draft / published / deleted |
| scenario_title | VARCHAR(24), NOT NULL | название сценария |
| scenario_description | VARCHAR(1000), DEFAULT '' | описание сценария |
| image_url | VARCHAR(500), DEFAULT '' | ссылка на изображение в MinIO |
| video_url | VARCHAR(500), DEFAULT '' | ссылка на видео в MinIO |
| battery_drain_mah | INT, NOT NULL | расход аккумулятора, мА·ч |
| usage_duration_hours | NUMERIC(4,1), NOT NULL | длительность сессии, ч |
| created_by_user_id | INTEGER, NOT NULL, FK → battery_app_users.id | создатель (ON DELETE RESTRICT) |
| created_at | TIMESTAMPTZ, NOT NULL | дата создания |
| formation_date | TIMESTAMPTZ, NULL | дата формирования / изменения |

### battery_scenario_likes — связь многие-ко-многим

| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| user_id | INTEGER, FK → battery_app_users.id | кто лайкнул (ON DELETE RESTRICT) |
| scenario_id | INTEGER, FK → battery_scenarios.id | что лайкнули (ON DELETE RESTRICT) |
| — | UNIQUE(user_id, scenario_id) | один лайк на пару |
