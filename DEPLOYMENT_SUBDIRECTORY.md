# Деплой в подпапке (например /E/)

## Обзор изменений

Приложение теперь поддерживает работу в подпапке URL (например, `https://example.com/E/`).

### Изменения в Frontend (SvelteKit)

1. **appDir изменен с `_app` на `internal`** - избегаем проблем с Nginx и Go
2. **Добавлена поддержка BASE_PATH** - позволяет собирать приложение для работы в подпапке

### Изменения в Backend (Go)

1. **Добавлена поддержка HTTP_PATH_PREFIX** - сервер понимает, что работает в подпапке
2. **Автоматическое стриппинг префикса** - правильная обработка статических файлов
3. **Дублирование маршрутов** - API доступно как на `/api`, так и на `/E/api`

## Сборка для продакшена

### Windows (PowerShell)

```powershell
# Установить переменную окружения
$env:BASE_PATH="/E"

# Перейти в папку frontend
cd web

# Собрать frontend
npm run build

# Вернуться в корень и собрать backend
cd ..
go build -o eckwms.exe ./cmd/api
```

### Linux/Mac (Bash)

```bash
# Установить переменную окружения
export BASE_PATH="/E"

# Перейти в папку frontend
cd web

# Собрать frontend
npm run build

# Вернуться в корень и собрать backend
cd ..
go build -o eckwms ./cmd/api
```

### Использование скриптов

Если у вас есть скрипты сборки:

```bash
# Windows
$env:BASE_PATH="/E"; ./scripts/build_release.bat

# Linux/Mac
BASE_PATH="/E" ./scripts/build_release.sh
```

## Запуск на сервере

### Переменные окружения

Добавьте в файл `.env` или systemd service:

```env
HTTP_PATH_PREFIX=/E
BASE_PATH=/E
```

### Пример systemd service

```ini
[Unit]
Description=ECK WMS API Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/var/www/eckwms
Environment="HTTP_PATH_PREFIX=/E"
Environment="PORT=8080"
ExecStart=/var/www/eckwms/eckwms
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Запуск вручную

```bash
# Установить переменную окружения
export HTTP_PATH_PREFIX=/E

# Запустить сервер
./eckwms
```

## Конфигурация Nginx

### Вариант 1: Проксирование в подпапку

```nginx
location /E/ {
    proxy_pass http://localhost:8080/E/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_cache_bypass $http_upgrade;
}
```

### Вариант 2: Стриппинг префикса в Nginx

```nginx
location /E/ {
    # Убираем /E/ перед передачей в backend
    rewrite ^/E/(.*) /$1 break;

    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_cache_bypass $http_upgrade;
}
```

**Важно**: При использовании Варианта 2 НЕ устанавливайте `HTTP_PATH_PREFIX` в Go приложении.

## Структура URL

После деплоя приложение будет доступно по следующим URL:

| Ресурс | URL |
|--------|-----|
| Главная страница | `https://example.com/E/` |
| Dashboard | `https://example.com/E/dashboard` |
| API Status | `https://example.com/E/api/status` |
| Auth Login | `https://example.com/E/auth/login` |
| Static Assets | `https://example.com/E/internal/...` |
| WebSocket | `wss://example.com/E/ws` |

## Проверка работоспособности

### Health Check

```bash
curl https://example.com/E/health
```

Ожидаемый ответ:
```json
{
  "status": "ok",
  "server": "local"
}
```

### Проверка статических файлов

Откройте в браузере:
- `https://example.com/E/` - должна отобразиться главная страница
- Проверьте консоль браузера на ошибки 404

### Проверка API

```bash
curl -X POST https://example.com/E/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test123"}'
```

## Откат к корневой папке

Если нужно вернуться к работе в корневой папке (`/`):

1. **Frontend**: Удалите `BASE_PATH` при сборке
2. **Backend**: Не устанавливайте `HTTP_PATH_PREFIX`
3. **Nginx**: Используйте стандартную конфигурацию без префикса

```bash
# Сборка для корневой папки
unset BASE_PATH
cd web && npm run build && cd ..
go build -o eckwms ./cmd/api
```

## Troubleshooting

### Проблема: Страница загружается, но все ресурсы 404

**Причина**: Frontend собран без `BASE_PATH`, но сервер ожидает префикс.

**Решение**: Пересоберите frontend с правильным `BASE_PATH`:
```bash
BASE_PATH=/E npm run build
```

### Проблема: API возвращает 404

**Причина**: Backend не настроен на работу с префиксом.

**Решение**: Установите переменную окружения:
```bash
export HTTP_PATH_PREFIX=/E
```

### Проблема: CSS/JS файлы не загружаются

**Причина**: Nginx неправильно проксирует статические файлы.

**Решение**: Проверьте конфигурацию Nginx и убедитесь, что используете правильный вариант (с стриппингом или без).

### Проблема: WebSocket не подключается

**Причина**: WebSocket пытается подключиться к неправильному URL.

**Решение**: Проверьте, что frontend правильно формирует URL для WebSocket с учетом `BASE_PATH`.

## Дополнительные настройки

### Несколько экземпляров на одном домене

Вы можете запустить несколько экземпляров приложения на разных подпапках:

```nginx
# Экземпляр 1
location /E/ {
    proxy_pass http://localhost:8080/E/;
}

# Экземпляр 2
location /warehouse/ {
    proxy_pass http://localhost:8081/warehouse/;
}
```

Для каждого экземпляра нужно:
1. Собрать frontend с соответствующим `BASE_PATH`
2. Установить `HTTP_PATH_PREFIX` при запуске backend
3. Использовать разные порты для каждого экземпляра

## Заметки

- Папка `internal` (бывшая `_app`) содержит скомпилированные JS/CSS бандлы SvelteKit
- Go сервер автоматически обрабатывает префикс для всех маршрутов
- Все API endpoints доступны как с префиксом, так и без (для обратной совместимости)
- Health check endpoint доступен на обоих путях: `/health` и `/E/health`

## Ссылки

- [SvelteKit Configuration](https://kit.svelte.dev/docs/configuration)
- [Gorilla Mux Documentation](https://github.com/gorilla/mux)
- [Nginx Reverse Proxy Guide](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/)
