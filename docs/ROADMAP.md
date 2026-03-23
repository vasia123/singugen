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

## Phase 5 — Multi-Agent (команда) ✅

**Цель:** Главный агент создаёт и управляет sub-агентами.

- [x] Agent Pool (`internal/spawner/`)
  - Создание агента: имя, описание, модель
  - Горутины (не Docker) — проще, один бинарник (ADR-0006)
  - Per-agent Session, Memory, workspace directory
- [x] Inter-agent communication (`internal/comms/`)
  - Go channels — Message Bus (Subscribe/Send/Broadcast)
  - Type-safe, zero-copy, non-blocking
- [x] Telegram команды
  - `/hire <name> <description>` — создать специалиста
  - `/fire <name>` — остановить агента
  - `/agents` — список агентов с состояниями
  - `/status` — статус всех агентов
- [x] Routing в Telegram: `@agent_name сообщение`
- [x] Главный агент ("main") — всегда существует, нельзя уволить

**Результат:** Главный агент может создать `@researcher` для поиска
информации и `@coder` для кода, делегировать через `@agent_name` prefix.

---

## Phase 6 — Kanban Board ✅

**Цель:** Визуальное управление задачами.

- [x] Выбран MD-файловый подход (ADR-0007): obsidian-like, без зависимостей
- [x] Директории-колонки: backlog/ → in-progress/ → review/ → done/
- [x] Task model: frontmatter YAML (title, assignee, priority, created, due) + MD body
- [x] Board CRUD: Add, Get, Move, List, ListAll, Delete
- [x] Prompt injection: assigned → агенту, unassigned → main, done исключены
- [x] Telegram команды: `/task`, `/tasks`, `/move`, `/done`
- [ ] GitHub Projects sync (опционально, будущая фаза)
- [ ] Telegram WebApp UI для доски (Phase 7)

**Результат:** Агенты видят свои задачи в промпте, пользователь управляет через Telegram.

---

## Phase 7 — Telegram WebApp ✅

**Цель:** Расширенный UI через Telegram Mini Apps.

- [x] WebApp сервер (`internal/webapp/`)
  - Go HTTP server (stdlib ServeMux, Go 1.22+ routing)
  - Telegram initData HMAC-SHA256 валидация
  - REST API: kanban, memory, agents endpoints
- [x] API endpoints:
  - POST /api/auth — валидация initData, session token
  - GET/POST/DELETE /api/kanban — задачи
  - GET/PUT /api/memory/{agent}/{name} — память агентов
  - GET /api/agents — список агентов со статусами
- [x] cloudflared quick tunnel — бесплатный HTTPS без конфигурации
- [ ] Frontend SPA (агент может самостоятельно построить через self-modification)
- [ ] Settings UI (конфигурация через WebApp)

**Результат:** REST API для WebApp, cloudflared туннель для HTTPS доступа.

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
| Phase 5 | 19 | ~600 | 0006 |
| Phase 6 | 12 | ~500 | 0007 |
| Phase 7 | 7 | ~500 | 0008 |
| **Итого** | **144** | **~6900** | **8** |
