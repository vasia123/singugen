# SinguGen Roadmap

## Vision

Самоэволюционирующая обвязка вокруг Claude Code. Один главный агент помогает
пользователю в ежедневных делах, развивается сам и развивает пользователя.
По мере необходимости нанимает в свою команду специалистов (порождает sub-агентов).
Живёт в Docker, общается через Telegram, может полностью переписать свой код,
пересобраться и перезапуститься.

---

## Phase 0 — Foundation (скелет проекта) ✅

**Цель:** Go-проект собирается, запускается, есть базовая структура.

- [x] `go mod init`, структура каталогов (`cmd/`, `internal/`, `configs/`, `docs/`)
- [x] Supervisor процесс (`cmd/singugen/main.go`)
  - Запускает child-процесс (основной агент)
  - Ловит сигналы, перезапускает child после self-update
  - Healthcheck child-процесса
- [x] Конфиг (YAML): токен Telegram, пути, параметры агента
- [x] Логирование (`slog` из stdlib)
- [x] Dockerfile (multi-stage, CGO_ENABLED=0, ARM-совместимый)
- [x] docker-compose.yml (volumes для данных и воркспейсов)
- [x] CI: `go vet`, `staticcheck`, `go test` на каждый коммит

**Результат:** `docker compose up` запускает supervisor, supervisor запускает
пустой child, child завершается — supervisor перезапускает.

---

## Phase 1 — Claude Code Integration (MVP ядро) ✅

**Цель:** Получить сообщение → передать Claude Code → получить ответ.

- [x] Claude Code launcher (`internal/claude/`)
  - Запуск `claude` бинарника в `--input-format stream-json --output-format stream-json`
  - Парсинг stream-json протокола (messages, tool_use, result)
  - Управление сессией (session ID, resume)
  - Graceful shutdown
- [x] Agent runtime (`internal/agent/`)
  - Lifecycle: start → ready → processing → idle → dreaming → stopped
  - Message queue (входящие сообщения пока агент занят)
  - Системный промпт из CLAUDE.md файлов
- [x] Тесты: mock Claude Code binary для unit-тестов (ProcessLauncher + FakeLauncher)

**Результат:** Программно отправить текст агенту, получить ответ от Claude Code.

---

## Phase 2 — Telegram Bot ✅

**Цель:** Пользователь общается с агентом через Telegram.

- [x] Telegram bot (`internal/telegram/`)
  - Long-polling (не webhooks — проще для старта)
  - Получение текстовых сообщений
  - Отправка ответов (chunking для >4096 символов)
  - Статус-сообщения ("Thinking...", "Reading file...", "Writing code...")
  - [ ] Поддержка файлов (фото, документы, голос — позже)
- [x] Команды: `/start`, `/status`, `/stop`, `/reset`
- [x] Авторизация: whitelist Telegram user IDs из конфига
- [x] Sender interface для тестируемости без Telegram API

**Результат:** Полноценный диалог с агентом через Telegram бота.

---

## Phase 3 — Memory (Obsidian-like) ✅

**Цель:** Агент помнит контекст между сессиями.

- [x] Memory store (`internal/memory/`)
  - Per-agent директория с MD файлами
  - Категории: `user.md`, `projects.md`, `skills.md`, `journal.md`, `team.md`
  - Загрузка в системный промпт при старте сессии
  - API: Save/Load/Delete/LoadAll/FormatForPrompt
- [x] Dreaming phase (`internal/dreaming/`)
  - Триггер: idle timeout (StateReady N минут, НЕ во время processing)
  - Реорганизация памяти через Claude с протоколом <<<MEMORY_UPDATE>>>
  - [ ] Отключение неиспользуемых MCP/skills, перенос в архив (Phase 8)
- [x] Хук на остановку: принудительный dreaming перед shutdown
- [x] Idle detection: timer считает только Ready-время, останавливается при Processing

**Результат:** Агент помнит пользователя, свои решения, контекст проектов.

---

## Phase 4 — Self-Modification ✅

**Цель:** Агент может дорабатывать собственный код.

- [x] Self-update pipeline (`internal/selfupdate/`)
  - Claude Code редактирует код (уже имеет инструменты)
  - CommandRunner interface для тестируемости без реальных процессов
  - `go build ./cmd/agent/` + `go vet ./...` + `go test ./...` — валидация
  - Если тесты прошли → git commit, опционально push
  - SIGUSR1 → supervisor перезапускает child с новым кодом
- [x] Protected directories: cmd/singugen/ и internal/supervisor/ неизменяемы
- [x] Telegram команды: `/update` (validate→commit→restart), `/rollback`
- [ ] Approval flow (отправка diff + inline кнопки — будущая фаза)
- [x] Safety guardrails
  - Supervisor неизменяем агентом (отдельный бинарник + protected dirs)
  - Circuit breaker: 5 перезапусков за 2 минуты
  - git revert для rollback (сохраняет историю)

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
| Telegram Bot | `mymmrac/telego` v1.7.0 | Нативный Go, хорошая поддержка ✅ |
| Config | `gopkg.in/yaml.v3` | Стандарт де-факто ✅ |
| SQLite | `modernc.org/sqlite` | Pure Go, без CGO |
| Logging | `log/slog` (stdlib) | Достаточно, нет внешних зависимостей ✅ |
| Testing | `testing` (stdlib) | Без testify, достаточно stdlib ✅ |

---

*Последнее обновление: 2026-03-23*

### Статистика

| Фаза | Тесты | Строки | ADR |
|------|-------|--------|-----|
| Phase 0 | 10 | ~1000 | 0001 |
| Phase 1 | 22 | ~1700 | 0002 |
| Phase 2 | 26 | ~1100 | 0003 |
| Phase 3 | 30 | ~800 | 0004 |
| Phase 4 | 18 | ~700 | 0005 |
| **Итого** | **106** | **~5300** | **5** |
