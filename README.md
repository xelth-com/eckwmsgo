# ECKWMS Go Backend

Go backend для ECKWMS (E-Commerce Warehouse Management System). Полнофункциональное приложение с встроенным SvelteKit фронтендом.

## Implemented Features
- **Database**: Hybrid mode (Embedded/External PostgreSQL) with Auto-migrations
- **Authentication**: JWT-based auth with Bcrypt password hashing
- **Testing**: Unit tests for authentication logic
- **WebSocket**: Real-time device communication with Hub pattern
- **Device Pairing**: Ed25519 cryptographic registration with QR codes
- **API**: RESTful endpoints for RMA and Warehouse management
- **Frontend**: Modern SvelteKit SPA with QR code scanning
- **Subdirectory Deployment**: Full support for deployment in URL subdirectories

## Структура проекта

```
eckwmsgo/
├── cmd/
│   └── api/
│       └── main.go              # Точка входа приложения
├── internal/
│   ├── config/
│   │   └── config.go            # Конфигурация из .env
│   ├── database/
│   │   └── database.go          # Подключение к PostgreSQL (GORM)
│   ├── models/
│   │   ├── user.go              # Модели пользователей
│   │   ├── warehouse.go         # Модели складов
│   │   ├── item.go              # Модели товаров/инвентаря
│   │   └── rma.go               # Модели RMA
│   ├── handlers/
│   │   ├── router.go            # HTTP роутер
│   │   ├── auth.go              # Аутентификация
│   │   ├── rma.go               # RMA endpoints
│   │   └── warehouse.go         # Склад endpoints
│   ├── middleware/              # Middleware
│   ├── services/                # Бизнес-логика
│   └── utils/                   # Утилиты
├── web/                         # SvelteKit Frontend
│   ├── src/                     # Исходники фронтенда
│   ├── build/                   # Собранный статический фронтенд
│   └── package.json             # NPM зависимости
├── pkg/                         # Публичные пакеты
├── .env                         # Конфигурация
├── go.mod                       # Go модуль
└── eckwmsgo.exe                 # Скомпилированный бинарник
```

## Требования

- **Go 1.21+**
- **PostgreSQL** (опционально, можно использовать Embedded PostgreSQL)
- **Node.js 18+** (для сборки фронтенда)

## Установка

1. Клонируй проект:
```bash
git clone <eckwmsgo-repo>
cd eckwmsgo
```

2. Настрой `.env`:
```bash
# Создай .env файл с настройками (см. секцию "Конфигурация" ниже)
```

3. Собери фронтенд:
```bash
cd web
npm install
npm run build
cd ..
```

4. Собери и запусти backend:
```bash
go build -o eckwms ./cmd/api
./eckwms
```

## Конфигурация

### Переменные окружения (.env)

```env
# Server Ports
PORT=3210                                    # Главный порт приложения
LOCAL_SERVER_PORT=3000                       # Локальный сервер (для совместимости)
GLOBAL_SERVER_PORT=8080                      # Глобальный сервер

# Database
# Zero-config: Оставь PG_PASSWORD пустым для использования Embedded PostgreSQL
# Для внешней БД: Установи PG_HOST, PG_USERNAME, PG_PASSWORD
PG_DATABASE=eckwmsgo_local
PG_USERNAME=postgres
PG_PASSWORD=                                 # Пусто = Embedded PostgreSQL
PG_HOST=localhost
PG_PORT=5432
DB_ALTER=true                                # Auto-migrations

# Security
JWT_SECRET=your_jwt_secret_here
ENC_KEY=your_encryption_key_here

# Frontend (опционально)
FRONTEND_DIR=web/build                       # Путь к собранному фронтенду (по умолчанию)

# Server Keys (Ed25519 для device pairing)
SERVER_PUBLIC_KEY=...
SERVER_PRIVATE_KEY=...
INSTANCE_ID=...

# Global Server Sync (опционально)
GLOBAL_SERVER_URL=https://your-domain.com
GLOBAL_SERVER_API_ENDPOINT=https://your-domain.com/api/internal/sync
GLOBAL_SERVER_API_KEY=your_api_key
```

**Zero-config режим**: Если оставить `PG_PASSWORD` пустым, приложение автоматически загрузит и использует Embedded PostgreSQL - никакой дополнительной настройки БД не требуется!

**Важно**: По умолчанию Go сервер ищет фронтенд в `web/build`. После сборки фронтенда (`npm run build` в папке `web/`) приложение готово к работе.

## Деплой в подпапке (Subdirectory Deployment)

**Новая функция (2026-01-13)**: Приложение теперь поддерживает работу в подпапке URL (например, `https://example.com/E/`).

### Быстрая настройка

1. **Сборка frontend с BASE_PATH**:
```bash
cd web
BASE_PATH=/E npm run build
```

2. **Сборка backend**:
```bash
cd ..
go build -o eckwms ./cmd/api
```

3. **Запуск с префиксом**:
```bash
HTTP_PATH_PREFIX=/E ./eckwms
```

### Переменные окружения для подпапки

```env
# В .env или systemd service
HTTP_PATH_PREFIX=/E    # Префикс для всех URL
```

### Конфигурация Nginx

```nginx
location /E/ {
    proxy_pass http://localhost:3001/E/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_cache_bypass $http_upgrade;
}
```

**Подробная документация**: См. `DEPLOYMENT_SUBDIRECTORY.md`

## Запуск

### Вариант 1: Скомпилированный бинарник
```bash
./eckwms
```

### Вариант 2: Через go run
```bash
go run ./cmd/api/main.go
```

### Вариант 3: Пересобрать и запустить
```bash
go build -o eckwms ./cmd/api
./eckwms
```

Сервер стартует на порту `3210` (или указанном в `PORT`).

Откройте в браузере: `http://localhost:3210`

## API Endpoints

### Health Check
- `GET /health` - Проверка работы сервера

### Authentication
- `POST /auth/login` - Вход пользователя
- `POST /auth/register` - Регистрация
- `POST /auth/logout` - Выход

### WebSocket & Device Pairing
- `GET /ws` - WebSocket connection
- `GET /api/internal/pairing-qr` - Pairing QR code image (protected)
- `POST /api/internal/register-device` - Register device with Ed25519 signature

### RMA Management
- `GET /rma` - Список всех RMA
- `POST /rma` - Создать RMA
- `GET /rma/{id}` - Получить RMA по ID
- `PUT /rma/{id}` - Обновить RMA
- `DELETE /rma/{id}` - Удалить RMA

### Warehouse Management
- `GET /api/warehouse` - Список складов
- `POST /api/warehouse` - Создать склад
- `GET /api/warehouse/{id}` - Получить склад по ID

### Inventory Management
- `GET /api/items` - Список товаров
- `POST /api/items` - Создать товар
- `GET /api/items/{id}` - Получить товар по ID

### Static Files
- `GET /*` - SvelteKit SPA из `web/build/` (с поддержкой SPA fallback)

## Разработка

### Добавить зависимость
```bash
go get -u package-name
go mod tidy
```

### Обновить зависимости
```bash
go get -u ./...
go mod tidy
```

### Запустить тесты
```bash
go test ./...
```

### Компиляция для продакшна
```bash
# Windows
go build -ldflags="-s -w" -o eckwmsgo.exe ./cmd/api

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o eckwmsgo ./cmd/api
```

## Реализованные возможности

### Полностью реализовано ✅
- ✅ Database models (GORM)
- ✅ HTTP server (Gorilla Mux)
- ✅ Конфигурация (.env)
- ✅ CRUD endpoints для RMA, Warehouse, Items
- ✅ Статический файловый сервер (SvelteKit SPA)
- ✅ JWT аутентификация (генерация/валидация токенов)
- ✅ Bcrypt для паролей
- ✅ Authorization middleware (JWT Bearer)
- ✅ WebSocket для real-time коммуникации
- ✅ Device pairing с Ed25519 криптографией
- ✅ QR code генерация и сканирование
- ✅ Embedded PostgreSQL (zero-config режим)
- ✅ Subdirectory deployment
- ✅ Universal Smart Code Scanner (QR codes, EAN-13, ITF-14)
- ✅ Современный SvelteKit фронтенд

### В разработке 🚧
- [ ] i18n/переводы (частично)
- [ ] PDF генерация
- [ ] Google OAuth
- [ ] AI/LLM сервисы
- [ ] Расширенные интеграции с логистикой

## Преимущества Go версии

| Аспект | Node.js | Go |
|--------|---------|-----|
| Производительность | Хорошо | Отлично |
| Память | Больше | Меньше |
| Типизация | Динамическая | Статическая |
| Компиляция | JIT | AOT (native) |
| Конкурентность | Event loop | Goroutines |
| Деплой | Node.js + deps | Один бинарник |

## Зависимости

- `gorm.io/gorm` - ORM для PostgreSQL
- `gorm.io/driver/postgres` - PostgreSQL драйвер
- `github.com/gorilla/mux` - HTTP роутер
- `github.com/gorilla/websocket` - WebSocket
- `github.com/joho/godotenv` - .env loader
- `gorm.io/datatypes` - JSON и другие типы данных
- `github.com/golang-jwt/jwt/v5` - JWT токены
- `golang.org/x/crypto/bcrypt` - Хеширование паролей
- `github.com/fergusstrange/embedded-postgres` - Embedded PostgreSQL для dev
- `github.com/skip2/go-qrcode` - QR коды

## Troubleshooting

### Фронтенд не загружается
Проверь что:
1. Фронтенд собран: `cd web && npm run build`
2. Папка `web/build/` существует и содержит `index.html`
3. Если используешь кастомный путь, установи `FRONTEND_DIR` в `.env`:
   ```
   FRONTEND_DIR=/absolute/path/to/build
   ```

### Ошибка подключения к БД
Проверь что:
1. Если используешь внешний PostgreSQL:
   - PostgreSQL запущен
   - Настройки в `.env` правильные (PG_PASSWORD должен быть заполнен)
   - База данных существует
   - Пользователь имеет права доступа
2. Если используешь Embedded PostgreSQL:
   - PG_PASSWORD пустой в `.env`
   - Достаточно места на диске (скачает ~50MB)
   - Порт 5432 свободен

### Порт занят
Измени `PORT` в `.env`:
```
PORT=3210
```

## Contributing

1. Fork проект
2. Создай feature branch (`git checkout -b feature/amazing`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing`)
5. Открой Pull Request

## License

MIT
