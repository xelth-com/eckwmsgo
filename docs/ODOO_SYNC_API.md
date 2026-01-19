# Odoo Sync API

## Обзор

Система автоматически синхронизирует данные из Odoo каждые 15 минут (настраивается через `ODOO_SYNC_INTERVAL`).

Синхронизируются следующие данные:
- **Локации** (stock.location) - иерархия складов
- **Продукты** (product.product) - товары с ценами и параметрами
- **Лоты** (stock.lot) - серийные номера
- **Коробки** (stock.quant.package) - упаковки
- **Остатки** (stock.quant) - текущие остатки на складах
- **Пикинги** (stock.picking) - заказы на отгрузку/перемещение ⭐ **НОВОЕ**
- **Move Lines** (stock.move.line) - детальные операции по заказам ⭐ **НОВОЕ**

## API Эндпоинты

### 1. Запустить синхронизацию вручную

```bash
POST /api/odoo/sync/trigger

curl -X POST http://localhost:3210/api/odoo/sync/trigger \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Ответ:**
```json
{
  "success": true,
  "message": "Odoo sync started in background"
}
```

### 2. Получить статус синхронизации

```bash
GET /api/odoo/sync/status

curl http://localhost:3210/api/odoo/sync/status \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Ответ:**
```json
{
  "products": {
    "count": 156,
    "last_synced": "2026-01-16T10:30:00Z"
  },
  "locations": {
    "count": 23,
    "last_synced": "2026-01-16T10:30:00Z"
  },
  "pickings": {
    "count": 45,
    "last_received": "2026-01-16T09:15:00Z"
  },
  "quants": {
    "count": 892
  }
}
```

### 3. Получить список заказов на отгрузку (пикингов)

```bash
GET /api/odoo/pickings
GET /api/odoo/pickings?state=assigned

curl "http://localhost:3210/api/odoo/pickings?state=assigned" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Параметры:**
- `state` (опционально) - фильтр по статусу: `draft`, `waiting`, `confirmed`, `assigned`, `done`, `cancel`

**Ответ:**
```json
[
  {
    "id": 123,
    "name": "WH/OUT/00042",
    "state": "assigned",
    "location_id": 8,
    "location_dest_id": 5,
    "scheduled_date": "2026-01-17T10:00:00Z",
    "origin": "SO042",
    "priority": "1",
    "partner_id": 15
  },
  ...
]
```

### 4. Получить детали конкретного пикинга

```bash
GET /api/odoo/pickings/{id}

curl http://localhost:3210/api/odoo/pickings/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Ответ:**
```json
{
  "picking": {
    "id": 123,
    "name": "WH/OUT/00042",
    "state": "assigned",
    "location_id": 8,
    "location_dest_id": 5,
    "scheduled_date": "2026-01-17T10:00:00Z",
    "origin": "SO042",
    "priority": "1",
    "partner_id": 15
  },
  "move_lines": [
    {
      "id": 456,
      "picking_id": 123,
      "product_id": 789,
      "qty_done": 0,
      "location_id": 8,
      "location_dest_id": 5,
      "lot_id": 101,
      "state": "assigned"
    },
    ...
  ]
}
```

## Статусы заказов (state)

- `draft` - Черновик
- `waiting` - Ожидает других операций
- `confirmed` - Подтвержден, ожидает резервирования
- `assigned` - Зарезервирован, готов к выполнению
- `done` - Выполнен
- `cancel` - Отменен

## Интеграция с OPAL Delivery

После получения пикинга со статусом `assigned`, вы можете создать доставку:

```bash
# 1. Получить готовые к отправке заказы
GET /api/odoo/pickings?state=assigned

# 2. Создать доставку OPAL для пикинга
POST /api/delivery/shipments
{
  "picking_id": 123,
  "provider_code": "opal"
}

# 3. Worker обработает заказ в фоне
# 4. Проверить статус
GET /api/delivery/shipments/123
```

## Примеры использования

### JavaScript/TypeScript

```typescript
// Получить список новых заказов
async function getNewOrders() {
  const response = await fetch('/api/odoo/pickings?state=assigned', {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  return await response.json();
}

// Запустить синхронизацию вручную
async function triggerSync() {
  const response = await fetch('/api/odoo/sync/trigger', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  return await response.json();
}

// Получить детали заказа с товарами
async function getOrderDetails(pickingId: number) {
  const response = await fetch(`/api/odoo/pickings/${pickingId}`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  return await response.json();
}
```

### curl

```bash
# Запустить синхронизацию
curl -X POST http://localhost:3210/api/odoo/sync/trigger \
  -H "Authorization: Bearer $TOKEN"

# Получить статус
curl http://localhost:3210/api/odoo/sync/status \
  -H "Authorization: Bearer $TOKEN"

# Получить готовые к отгрузке заказы
curl "http://localhost:3210/api/odoo/pickings?state=assigned" \
  -H "Authorization: Bearer $TOKEN"

# Получить детали заказа
curl http://localhost:3210/api/odoo/pickings/123 \
  -H "Authorization: Bearer $TOKEN"
```

## Конфигурация

В `.env` файле:

```env
# Odoo Configuration
ODOO_URL=https://your-odoo-instance.com
ODOO_DB=your_database
ODOO_USER=your_username
ODOO_PASSWORD=your_api_key_or_password
ODOO_SYNC_INTERVAL=15  # Minutes between auto-sync
```

## Логи

Синхронизация пишет подробные логи в stdout:

```
📡 Odoo Sync Service started
🔄 Odoo: Starting full sync...
📍 Odoo: Syncing Locations...
✅ Odoo: Updated 23 locations
📦 Odoo: Syncing Products...
✅ Odoo: Updated 5 products
🏷️ Odoo: Syncing Lots...
✅ Odoo: Updated 12 lots
📦 Odoo: Syncing Packages...
✅ Odoo: Updated 3 packages
📊 Odoo: Syncing Quants...
✅ Odoo: Updated 45 quants
📋 Odoo: Syncing Pickings (Transfer Orders)...
✅ Odoo: Updated 8 pickings
📝 Odoo: Syncing Move Lines...
✅ Odoo: Updated 24 move lines
✅ Odoo: Full sync completed
```

## Troubleshooting

### Синхронизация не работает

1. Проверьте логи: `journalctl -u eckwmsgo -f`
2. Проверьте настройки в `.env`
3. Проверьте подключение к Odoo:
   ```bash
   curl https://your-odoo-instance.com
   ```

### Нет новых заказов

1. Проверьте статус синхронизации:
   ```bash
   curl http://localhost:3210/api/odoo/sync/status
   ```
2. Запустите синхронизацию вручную:
   ```bash
   curl -X POST http://localhost:3210/api/odoo/sync/trigger
   ```
3. Проверьте фильтр в Odoo - синхронизируются только заказы с датой > последней синхронизации

### Ошибки аутентификации

- Убедитесь, что `ODOO_USER` и `ODOO_PASSWORD` правильные
- Проверьте права доступа пользователя в Odoo (нужен доступ к моделям stock.*)
