# SinguGen Roadmap

## Vision

Самоэволюционирующая обвязка вокруг Claude Code. Один главный агент помогает
пользователю в ежедневных делах, развивается сам и развивает пользователя.
По мере необходимости нанимает в свою команду специалистов (порождает sub-агентов).
Живёт в Docker, общается через Telegram, может полностью переписать свой код,
пересобраться и перезапуститься.

---

## Phase 0 — Foundation (скелет проекта)

**Цель:** Go-проект собирается, запускается, есть базовая структура.

- [ ] `go mod init`, структура каталогов (`cmd/`, `internal/`, `configs/`, `docs/`)
- [ ] Supervisor процесс (`cmd/singugen/main.go`)
  - Запускает child-процесс (основной агент)
  - Ловит сигналы, перезапускает child после self-update
  - Healthcheck child-процесса
- [ ] Конфиг (YAML): токен Telegram, пути, параметры агента
- [ ] Логирование (`slog` из stdlib)
- [ ] Dockerfile (multi-stage, CGO_ENABLED=0, ARM-совместимый)
- [ ] docker-compose.yml (volumes для данных и воркспейсов)
- [ ] CI: `go vet`, `staticcheck`, `go test` на каждый коммит

**Результат:** `docker compose up` запускает supervisor, supervisor запускает
пустой child, child завершается — supervisor перезапускает.

---

## Phase 1 — Claude Code Integration (MVP ядро)

**Цель:** Получить сообщение → передать Claude Code → получить ответ.

- [ ] Claude Code launcher (`internal/claude/`)
  - Запуск `claude` бинарника в `--input-format stream-json --output-format stream-json`
  - Парсинг stream-json протокола (messages, tool_use, result)
  - Управление сессией (session ID, resume)
  - Graceful shutdown
- [ ] Agent runtime (`internal/agent/`)
  - Lifecycle: start → ready → processing → idle → dreaming → stopped
  - Message queue (входящие сообщения пока агент занят)
  - Системный промпт из CLAUDE.md файлов
- [ ] Тесты: mock Claude Code binary для unit-тестов

**Результат:** Программно отправить текст агенту, получить ответ от Claude Code.

---

## Phase 2 — Telegram Bot

**Цель:** Пользователь общается с агентом через Telegram.

- [ ] Telegram bot (`internal/telegram/`)
  - Long-polling (не webhooks — проще для старта)
  - Получение текстовых сообщений
  - Отправка ответов (chunking для >4096 символов)
  - Статус-сообщения ("думаю...", "пишу код...")
  - Поддержка файлов (фото, документы, голос — позже)
- [ ] Команды: `/start`, `/status`, `/stop`, `/reset`
- [ ] Авторизация: whitelist Telegram user IDs из конфига
- [ ] Graceful: при остановке агента — уведомление в чат

**Результат:** Полноценный диалог с агентом через Telegram бота.

---

## Phase 3 — Memory (Obsidian-like)

**Цель:** Агент помнит контекст между сессиями.

- [ ] Memory store (`internal/memory/`)
  - Per-agent директория с MD файлами
  - Категории: `user.md`, `projects.md`, `skills.md`, `journal.md`, `team.md`
  - Загрузка в системный промпт при старте сессии
  - API для агента: save/update/delete memory entries
- [ ] Dreaming phase (`internal/dreaming/`)
  - Триггер: idle timeout (агент не получал сообщений N минут)
  - Реорганизация памяти: удаление неактуального, структурирование
  - Отключение неиспользуемых MCP/skills, перенос в архив
  - Лог dreaming-сессии для прозрачности
- [ ] Хук на остановку: принудительный dreaming перед shutdown

**Результат:** Агент помнит пользователя, свои решения, контекст проектов.

---

## Phase 4 — Self-Modification

**Цель:** Агент может дорабатывать собственный код.

- [ ] Self-update pipeline (`internal/selfupdate/`)
  - Агент вносит изменения в собственный код (workspace = свой репозиторий)
  - `go build` + `go test` + `go vet` — валидация перед применением
  - Если тесты прошли → сигнал supervisor на перезапуск
  - Если тесты упали → rollback, уведомление пользователя
  - Git: commit в feature branch, push в fork
- [ ] Approval flow (опционально)
  - Отправка diff в Telegram перед применением
  - Inline кнопки: Approve / Reject / Discuss
- [ ] Safety guardrails
  - Supervisor неизменяем агентом (отдельный бинарник)
  - Максимум N перезапусков за период (circuit breaker)
  - Rollback к последнему рабочему состоянию при crash loop

**Результат:** Агент дорабатывает себя, тестирует, перезапускается.
Supervisor гарантирует восстановление при ошибках.

---

## Phase 5 — Multi-Agent (команда)

**Цель:** Главный агент создаёт и управляет sub-агентами.

- [ ] Agent spawner (`internal/spawner/`)
  - Создание нового агента: имя, описание, специализация, CLAUDE.md
  - Запуск в отдельном Docker контейнере
  - Выделенный workspace и memory directory
- [ ] Inter-agent communication (`internal/comms/`)
  - Механизм обмена сообщениями между агентами
  - Вариант 1: embedded NATS (как в Praktor)
  - Вариант 2: Unix sockets / named pipes (проще, меньше зависимостей)
  - Вариант 3: файловый обмен через shared volume (самый простой)
  - **Решение примем на этапе реализации (ADR)**
- [ ] Команды главного агента
  - "Наняты" (hire): создать специалиста с описанием роли
  - "Делегировать" (delegate): отправить задачу конкретному агенту
  - "Отчёт" (report): получить статус от sub-агента
  - "Уволить" (fire): остановить и архивировать агента
- [ ] Routing в Telegram: `@agent_name сообщение`

**Результат:** Главный агент может создать, например, `@researcher` для
поиска информации и `@coder` для написания кода, делегировать им задачи.

---

## Phase 6 — Kanban Board

**Цель:** Визуальное управление задачами.

- [ ] Оценить: GitHub Projects API vs собственная реализация (ADR)
- [ ] **Вариант A: GitHub Projects**
  - Интеграция через GitHub API (`gh` CLI или REST)
  - Агент создаёт/двигает карточки
  - Пользователь видит доску на GitHub
  - Плюс: готовая UI, минус: зависимость от GitHub
- [ ] **Вариант B: Telegram WebApp**
  - Kanban board как Mini App в Telegram
  - Backend: Go HTTP сервер
  - Frontend: lightweight SPA (можно Preact или vanilla)
  - Drag & drop колонки: Backlog → In Progress → Review → Done
  - Агент управляет через API, пользователь — через WebApp
- [ ] Task model: title, description, assignee (agent), status, priority, due date
- [ ] Хранение: SQLite или MD файлы (в стиле памяти)

**Результат:** Пользователь и агенты видят общую доску задач.

---

## Phase 7 — Telegram WebApp

**Цель:** Расширенный UI через Telegram Mini Apps.

- [ ] WebApp сервер (`internal/webapp/`)
  - Go HTTP server для SPA
  - Telegram WebApp SDK интеграция (валидация initData)
  - API endpoints для WebApp
- [ ] Функционал:
  - Kanban board (если выбран вариант B)
  - Memory browser (просмотр/редактирование MD файлов агента)
  - Agent dashboard (список агентов, статусы, логи)
  - Settings (конфигурация агента через UI)
- [ ] Self-evolving: агент может модифицировать frontend-код WebApp

**Результат:** Богатый UI внутри Telegram для сложных взаимодействий.

---

## Phase 8 — Skills & MCP Ecosystem

**Цель:** Агент расширяет свои возможности через внешние инструменты.

- [ ] Skill manager (`internal/skills/`)
  - Установка из реестра (claude skill marketplace, npm, GitHub)
  - Анализ безопасности перед установкой:
    - Статический анализ кода
    - Проверка permissions/scopes
    - Sandboxing при первом запуске
    - Approval пользователя для dangerous permissions
  - Архивирование неиспользуемых (dreaming phase)
- [ ] MCP manager (`internal/mcp/`)
  - Динамическое подключение/отключение MCP серверов
  - Мониторинг контекстного загрязнения (context bloat)
  - Auto-disable при превышении порога
- [ ] Security pipeline
  - Каждый внешний скилл/MCP проходит через анализ
  - Whitelist/blacklist
  - Изоляция через отдельный container/namespace

**Результат:** Агент самостоятельно находит и устанавливает нужные инструменты,
но с контролем безопасности.

---

## Phase 9 — Production Hardening

- [ ] Мониторинг и метрики (Prometheus-совместимые)
- [ ] Rate limiting и квоты
- [ ] Backup/restore памяти и данных
- [ ] Multi-user support (несколько Telegram пользователей)
- [ ] ARM-оптимизация для Orange Pi
- [ ] Auto-update из GitHub releases
- [ ] Документация: deployment guide, architecture overview

---

## Принципы на протяжении всей разработки

1. **TDD** — тесты первыми, всегда
2. **ADR** — документируем архитектурные решения в `docs/adr/`
3. **Безопасность** — каждый внешний вход валидируется
4. **Supervisor неприкосновенен** — агент не может его модифицировать
5. **Rollback всегда возможен** — git как safety net
6. **Минимальный стартовый footprint** — агент начинает голым и растёт
7. **Go-first** — TypeScript только если Go не справляется

---

## Зависимости (предварительно)

| Компонент | Библиотека | Обоснование |
|-----------|-----------|-------------|
| Telegram Bot | `go-telegram/bot` или `telego` | Нативный Go, хорошая поддержка |
| Config | `gopkg.in/yaml.v3` | Стандарт де-факто |
| SQLite | `modernc.org/sqlite` | Pure Go, без CGO |
| Logging | `log/slog` (stdlib) | Достаточно, нет внешних зависимостей |
| Testing | `testing` + `testify` | Стандарт |

---

*Последнее обновление: 2026-03-23*
