# Zero-Knowledge Relay Architecture

## Обзор

Эта система позволяет использовать ненадежный сервер в интернете как прокси для синхронизации данных между доверенными узлами. **Relay Server** видит только метаданные (EntityID, Version, VectorClock), но **не может расшифровать** содержимое (Payload).

## Архитектура

```
┌──────────────┐      Encrypted     ┌─────────────────┐      Encrypted     ┌──────────────┐
│  Office Node │ ──────────────────> │  Blind Relay    │ ──────────────────> │ Remote Node  │
│  (Trusted)   │                     │  (Untrusted)    │                     │  (Trusted)   │
│  + SYNC_KEY  │ <────────────────── │  - SYNC_KEY     │ <────────────────── │  + SYNC_KEY  │
└──────────────┘                     └─────────────────┘                     └──────────────┘
```

### Роли узлов

| Роль | Описание | Доступ к данным | Ключ |
|------|----------|----------------|------|
| **master** | Главный сервер (авторитетная БД) | Полный Read/Write | ✅ Обязателен |
| **peer** | Равноправный узел (полная БД) | Полный Read/Write | ✅ Обязателен |
| **edge** | Граничный узел (частичная БД) | Частичный Read/Write | ✅ Обязателен |
| **blind_relay** | Слепой прокси (без доступа к БД) | Только метаданные | ❌ Не нужен |

## Настройка

### 1. Генерация общего ключа

Все доверенные узлы должны использовать **один и тот же** ключ длиной 32 байта:

```bash
# Генерация случайного ключа (Linux/Mac)
openssl rand -hex 32

# Или (Windows PowerShell)
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))
```

**Важно:** Этот ключ должен храниться в секрете и передаваться только по защищенным каналам.

### 2. Конфигурация Blind Relay (Ненадежный сервер)

На публичном сервере в интернете:

**`.env`:**
```env
SYNC_ROLE=blind_relay
# SYNC_NETWORK_KEY не указывается!
DATABASE_URL=postgres://user:password@localhost:5432/relay_db
PORT=8080
```

**Что он делает:**
- Принимает зашифрованные пакеты от узлов
- Видит `EntityType`, `EntityID`, `Version`
- Сохраняет зашифрованные BLOB-ы в таблице `encrypted_sync_packets`
- Рассылает пакеты другим узлам по запросу
- **НЕ МОЖЕТ** прочитать содержимое

### 3. Конфигурация Office Node (Доверенный узел)

На локальном сервере в офисе:

**`.env`:**
```env
SYNC_ROLE=peer
SYNC_NETWORK_KEY=your_32_byte_hex_key_here
SYNC_RELAY_URL=https://relay.example.com
DATABASE_URL=postgres://user:password@localhost:5432/office_db
PORT=3001
```

**Что он делает:**
- Шифрует данные своим ключом
- Отправляет зашифрованные пакеты на Relay
- Получает пакеты от Relay и расшифровывает локально
- Работает с полной БД

### 4. Конфигурация Remote Node (Доверенный узел)

На ноутбуке/планшете удаленного работника:

**`.env`:**
```env
SYNC_ROLE=peer
SYNC_NETWORK_KEY=your_32_byte_hex_key_here  # ⚠️ Тот же ключ!
SYNC_RELAY_URL=https://relay.example.com
DATABASE_URL=postgres://user:password@localhost:5432/remote_db
PORT=3001
```

## Безопасность

### Что видит Blind Relay?

✅ **Видимо:**
- `EntityType` (например, "item", "order")
- `EntityID` (например, "order-123")
- `Version` (например, 42)
- `VectorClock` (временные метки)
- `SourceInstance` (ID источника)

❌ **НЕ видимо:**
- Содержимое Item (название, количество, цена)
- Содержимое Order (клиент, адрес, товары)
- Любые бизнес-данные

### Алгоритм шифрования

- **Алгоритм:** AES-256-GCM
- **Режим:** Authenticated Encryption (защита от подделки)
- **Ключ:** 256 бит (32 байта)
- **Nonce:** Уникальный для каждого пакета

## Примеры использования

### Пример 1: Синхронизация заказа

1. **Office создает заказ:**
```go
order := &Order{ID: "order-123", Customer: "John Doe"}
metadata := NewEntityMetadata(EntityTypeOrder, "order-123", "office-01")

// SecurityLayer шифрует
encryptedPacket, err := security.EncryptPacket(metadata, order)
// Отправляет на Relay
```

2. **Relay хранит пакет:**
```sql
INSERT INTO encrypted_sync_packets (
  entity_type, entity_id, version,
  encrypted_payload, nonce
) VALUES (
  'order', 'order-123', 1,
  '\x8f3a2b...', '\x7d4e1c...'  -- Бинарные данные
)
```

3. **Remote запрашивает:**
```go
// Получает зашифрованный пакет от Relay
encryptedPacket := relay.Pull("order", "order-123")

// Расшифровывает локально
var order Order
err := security.DecryptPacket(encryptedPacket, &order)
// order.Customer теперь доступен!
```

## Миграция БД

Новая таблица создается автоматически при первом запуске:

```sql
CREATE TABLE encrypted_sync_packets (
  id SERIAL PRIMARY KEY,
  entity_type VARCHAR(100),
  entity_id VARCHAR(255),
  version BIGINT,
  source_instance VARCHAR(255),
  vector_clock JSONB,
  key_id VARCHAR(50),
  algorithm VARCHAR(50),
  encrypted_payload BYTEA,
  nonce BYTEA,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_relay_lookup ON encrypted_sync_packets(entity_type, entity_id);
```

## Тестирование

### Проверка шифрования

```go
package main

import (
    "fmt"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/sync"
)

func main() {
    // Создаем SecurityLayer
    security := sync.NewSecurityLayer(sync.RolePeer)

    // Проверяем возможности
    fmt.Println("Can Encrypt:", security.CanEncrypt())
    fmt.Println("Can Decrypt:", security.CanDecrypt())
    fmt.Println("Role:", security.GetRole())
}
```

### Проверка Relay

```bash
# На Blind Relay должна быть пустая БД без ключа
export SYNC_ROLE=blind_relay
go run cmd/api/main.go

# Relay запускается без ошибок
# ✅ Таблица encrypted_sync_packets создается
# ✅ API /sync/push принимает зашифрованные пакеты
# ❌ Попытка расшифровать данные приведет к ошибке
```

## Ротация ключей

Для безопасности рекомендуется периодически менять ключ:

1. Сгенерируйте новый ключ `KEY_V2`
2. Обновите `SYNC_NETWORK_KEY` на всех доверенных узлах
3. Старые пакеты останутся с `key_id: "v1"`
4. Новые пакеты будут с `key_id: "v2"`

SecurityLayer автоматически проверяет `key_id` при расшифровке.

## FAQ

**Q: Может ли Relay подделать данные?**
A: Нет. AES-GCM включает проверку подлинности. Любая модификация будет обнаружена при расшифровке.

**Q: Что если Relay удалит пакеты?**
A: Узлы обнаружат отсутствие версий через VectorClock и запросят повторную синхронизацию.

**Q: Можно ли использовать несколько Relay?**
A: Да! Укажите несколько URL в `SYNC_RELAY_URL` через запятую.

**Q: Нужен ли HTTPS?**
A: Рекомендуется, но не критично. Данные уже зашифрованы на уровне приложения.

## Дальнейшее развитие

- [ ] Поддержка нескольких ключей (key rotation)
- [ ] Сжатие перед шифрованием (zstd)
- [ ] Поддержка DHT для P2P-синхронизации
- [ ] WebSocket для real-time push
