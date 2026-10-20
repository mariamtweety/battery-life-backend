# battery-life-backend

## Лабораторная 3 — веб-сервис REST API

### Домен «Услуга» (сценарий использования аккумулятора)

- `GET /api/scenarios` — список опубликованных услуг; необязательные query-фильтры `search`, `max_hours`, `min_drain`, `max_drain` (диапазоны полей по теме). Ответ 200: массив объектов с признаками `is_creator` (1, если создатель — текущий пользователь) и `is_liked`.
- `GET /api/scenarios/feed` — лента: первая опубликованная услуга, id не указывается.
- `GET /api/scenarios/feed/:id` — опубликованная услуга по id; 404, если не существует или удалена.
- `GET /api/scenarios/feed/:id?next=true` — следующая опубликованная после id, в конце ленты заворот на первую.
- `GET /api/scenarios/draft` — единственный черновик текущего пользователя, id с клиента не передаётся; 404, если черновика нет.
- `POST /api/scenarios` — создание черновика; принимает `scenario_title` (текст) и файлы `image`, `video` (multipart/form-data, именно файлы, а не URL). Файлы сохраняются в MinIO под сгенерированными латинскими именами, имена/ссылки пишутся в поля БД. Ответ 201 + объект черновика.
- `PUT /api/scenarios/:id/publish` — публикация своего черновика (смена статуса); JSON-тело: `scenario_description`, `battery_drain_mah`, `usage_duration_hours`. Ответ 200 + обновлённый объект; 404, если черновик чужой или уже опубликован.
- `DELETE /api/scenarios/:id` — логическое удаление своей услуги (scenario_status = 'deleted', formation_date = NOW(), SQL UPDATE без каскадного удаления). Ответ 200; чужая услуга — 403.
- `POST /api/scenarios/:id/like` — лайк текущего пользователя; JSON `{"like": 1}` ставит, `{"like": 0}` отменяет. Ответ 200.

### Домен «Пользователь»

- `POST /api/users/register` — регистрация; JSON `{username, password}`. Ответ 201 + `{id, username}` (пароль не сериализуется); 409, если логин занят.
- `POST /api/users/login` — заглушка аутентификации для ЛР4. Ответ 200.
- `POST /api/users/logout` — заглушка деавторизации для ЛР4. Ответ 200.

### Соглашения ответов

- Набор полей JSON всегда одинаковый; незаполненные значения приходят как `null` / `""`.
- Статус услуги в ответах не передаётся (хранится только в БД); коды состояния в тело не дублируются.
- Успех: 200 / 201; ошибки: 400, 403, 404, 409 без текстовых сообщений.

## Таблицы базы данных

### battery_app_users — зарегистрированные пользователи
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| username | VARCHAR(50), UNIQUE, NOT NULL | логин |
| password_hash | VARCHAR(100), NOT NULL | пароль (хэш, скрыт из JSON) |
| is_moderator | BOOLEAN, NOT NULL, DEFAULT FALSE | флаг прав модератора |
| created_at | TIMESTAMPTZ, NOT NULL | дата регистрации |

### battery_scenarios — черновики, опубликованные и удалённые услуги
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| scenario_status | VARCHAR(20), NOT NULL, DEFAULT 'draft' | draft / published / deleted |
| scenario_title | VARCHAR(24), NOT NULL | название сценария |
| scenario_description | VARCHAR(1000), DEFAULT '' | описание сценария |
| image_url | VARCHAR(500), DEFAULT '' | имя/ссылка изображения в MinIO |
| video_url | VARCHAR(500), DEFAULT '' | имя/ссылка видео в MinIO |
| battery_drain_mah | INT, NOT NULL | расход аккумулятора (поле по теме) |
| usage_duration_hours | NUMERIC(4,1), NOT NULL | длительность сессии (поле по теме) |
| created_by_user_id | INTEGER, NOT NULL, FK → battery_app_users.id | создатель (ON DELETE RESTRICT) |
| created_at | TIMESTAMPTZ, NOT NULL | дата создания |
| formation_date | TIMESTAMPTZ, NULL | дата формирования / изменения |

### battery_scenario_likes — связь многие-ко-многим пользователи ↔ услуги
| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| user_id | INTEGER, FK → battery_app_users.id | кто лайкнул (ON DELETE RESTRICT) |
| scenario_id | INTEGER, FK → battery_scenarios.id | что лайкнули (ON DELETE RESTRICT) |
| — | UNIQUE(user_id, scenario_id) | один лайк на пару |