# 🚀 Добавление нового сервера в Mesh Network

## ⚠️ КРИТИЧЕСКИ ВАЖНО: База данных

### Конвенция именования колонок

При добавлении нового сервера **ОБЯЗАТЕЛЬНО** убедиться что база данных использует **snake_case** для всех колонок:

```
Go Struct     →    PostgreSQL DB    →    JSON API
─────────────────────────────────────────────────
DeviceID           device_id             deviceId
PublicKey          public_key            publicKey
LastSeenAt         last_seen_at          lastSeenAt
CreatedAt          created_at            createdAt
UpdatedAt          updated_at            updatedAt
```

### ❌ НЕ ДОПУСКАЕТСЯ (Legacy)

```sql
-- Это НЕПРАВИЛЬНО! Не создавать такие колонки!
"deviceId"     -- camelCase в БД
"publicKey"    -- camelCase в БД
"lastSeenAt"   -- camelCase в БД
```

### ✅ ПРАВИЛЬНО

```sql
-- Это ПРАВИЛЬНО! Все колонки в snake_case
device_id      -- snake_case
public_key     -- snake_case
last_seen_at   -- snake_case
```

---

## 📋 Чеклист для нового сервера

### 1. Конфигурация (.env)

```env
# Обязательные переменные
NODE_ROLE=peer                              # peer | master | edge | blind_relay
INSTANCE_ID=warehouse_berlin_01             # Уникальный ID (snake_case)
MESH_SECRET=eckwms_mesh_secret_2026_secure  # Общий секрет для всех нод
BOOTSTRAP_NODES=https://pda.repair/E        # URL мастер-ноды
BASE_URL=http://localhost:3210              # URL этого сервера
SYNC_NETWORK_KEY=<32_byte_hex>              # Ключ шифрования (одинаковый для всех)

# PostgreSQL
PG_HOST=localhost
PG_PORT=5432
PG_USERNAME=openpg
PG_PASSWORD=your_password
PG_DATABASE=eckwms
DB_ALTER=true                               # Разрешить автомиграции
```

### 2. Проверка схемы БД перед запуском

```bash
# Запустить скрипт проверки camelCase колонок
go run scripts/migrate_all_to_snakecase.go

# Ожидаемый результат:
# "Найдено camelCase колонок: 0"
```

### 3. Если найдены camelCase колонки - МИГРИРОВАТЬ!

```sql
-- Пример миграции для registered_devices
-- 1. Копировать данные из camelCase в snake_case
UPDATE registered_devices
SET device_id = "deviceId"
WHERE device_id IS NULL AND "deviceId" IS NOT NULL;

-- 2. Удалить legacy колонки
ALTER TABLE registered_devices DROP COLUMN IF EXISTS "deviceId" CASCADE;
ALTER TABLE registered_devices DROP COLUMN IF EXISTS "publicKey" CASCADE;
ALTER TABLE registered_devices DROP COLUMN IF EXISTS "lastSeenAt" CASCADE;

-- 3. Установить NOT NULL где нужно
ALTER TABLE registered_devices ALTER COLUMN device_id SET NOT NULL;
ALTER TABLE registered_devices ADD PRIMARY KEY (device_id);
```

### 4. Запуск сервера

```bash
# Сборка
go build -o eckwmsgo ./cmd/api

# Запуск
./eckwmsgo

# Или через systemd
sudo systemctl start eckwmsgo
```

### 5. Проверка подключения к mesh

```bash
# Должно появиться в логах:
# "Mesh: Handshake success with https://pda.repair/E (master, ID: production_pda_repair)"

# Проверить статус mesh
curl http://localhost:3210/api/mesh/status
```

### 6. Триггер первой синхронизации

```bash
# Ручной триггер sync
curl -X POST http://localhost:3210/api/mesh/trigger

# Проверить логи на успешный push/pull
# "Mesh Sync: Successfully pushed to production_pda_repair"
# "Mesh Sync: Successfully pulled from production_pda_repair"
```

---

## 🔧 Скрипты для миграции

### Проверка всех таблиц на camelCase

```bash
go run scripts/migrate_all_to_snakecase.go
```

### Проверка конкретной таблицы

```bash
go run scripts/check_schema.go
```

### Очистка старых колонок

```bash
go run scripts/cleanup_old_columns.go
```

---

## ⚡ Быстрый старт (если БД чистая)

Если база данных **новая и пустая**, GORM автоматически создаст все таблицы с правильными snake_case колонками:

```bash
# 1. Настроить .env
cp .env.example .env
nano .env  # Заполнить переменные

# 2. Запустить (GORM создаст таблицы)
go run cmd/api/main.go

# 3. Проверить mesh
curl http://localhost:3210/health
curl http://localhost:3210/api/mesh/status
```

---

## 🚨 Типичные ошибки

### Ошибка: "NULL-Wert in Spalte deviceId verletzt Not-Null-Constraint"

**Причина:** БД имеет camelCase колонку `"deviceId"` как primary key, но код пишет в `device_id`.

**Решение:**
```sql
-- Миграция primary key
ALTER TABLE registered_devices DROP CONSTRAINT registered_devices_pkey;
ALTER TABLE registered_devices DROP COLUMN "deviceId" CASCADE;
ALTER TABLE registered_devices ALTER COLUMN device_id SET NOT NULL;
ALTER TABLE registered_devices ADD PRIMARY KEY (device_id);
```

### Ошибка: "commit unexpectedly resulted in rollback"

**Причина:** Транзакция помечена как failed из-за ошибки в одном из upsert.

**Решение:** Убедиться что все колонки в snake_case и nullable constraints корректны.

### Ошибка: "Spalte updated_at existiert nicht"

**Причина:** Модель использует другое поле для timestamp (например, `write_date` для ProductProduct).

**Решение:** Проверить модель и использовать правильное поле в запросах.

---

## 📊 Роли серверов

| Role | Описание | Pull | Push | Database |
|------|----------|------|------|----------|
| `master` | Центральный сервер | ✅ | ✅ | Full |
| `peer` | Warehouse node | ✅ | ✅ | Full |
| `edge` | PDA/Scanner | ✅ | Limited | Partial |
| `blind_relay` | Encrypted proxy | ❌ | ❌ | None |

---

## 📁 Важные файлы

- `.eck/DATABASE_STRATEGY.md` - Полная документация по конвенциям БД
- `.eck/CONTEXT.md` - Архитектура проекта
- `scripts/migrate_all_to_snakecase.go` - Скрипт миграции
- `scripts/check_schema.go` - Проверка схемы

---

**Последнее обновление:** 2026-01-27
**Автор:** Claude Opus 4.5
