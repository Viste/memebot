# Meme Bot

## Возможности

- 🤖 Обработка фото и видео мемов
- 🧠 Интеграция с OpenAI для генерации комментариев (персона Сталина)
- 🎨 Генерация мемов: по запросу «а как бы ты сделал этот мем» (реплаем на комментарий бота) и изредка спонтанно вместо текстового комментария (`MEME_IMAGE_PROBABILITY`)
- 📝 История взаимодействий с мемами
- 👥 Групповая обработка медиа
- 🗄️ PostgreSQL база данных с GORM
- 🌐 Web сервер для health checks
- 🐳 Docker

- **Language**: Go 1.26+
- **Database**: PostgreSQL 15+
- **ORM**: GORM
- **Telegram API**: [OvyFlash/telegram-bot-api](https://github.com/OvyFlash/telegram-bot-api) — поддерживаемый drop-in форк go-telegram-bot-api с актуальным Bot API
- **OpenAI API**: go-openai (комментарии + генерация картинок `gpt-image-2`)
- **Web Framework**: Gin
- **Containerization**: Docker

## Быстрый старт

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd memebot
```

### 2. Настройка окружения

Скопируйте файл с переменными окружения:
```bash
cp .env.example .env
```

Отредактируйте `.env` файл с вашими данными:
- `TELEGRAM_BOT_TOKEN` - токен вашего Telegram бота
- `TELEGRAM_CHANNEL` - канал для отправки мемов
- `OPENAI_API_KEY` - ключ OpenAI API
- `DATABASE_URL` - строка подключения к PostgreSQL

## Структура проекта

```
meme-bot/
├── config/           # Конфигурация приложения
├── database/         # Подключение к БД и миграции
├── handlers/         # Обработчики Telegram событий
├── models/           # Модели базы данных (GORM)
├── services/         # OpenAI, etc.
├── utils/            # Утилиты и хелперы
├── web/              # Web сервер
├── main.go           # Точка входа
├── Dockerfile        # Docker образ
├── docker-compose.yml # Docker Compose конфигурация
└── README.md         # Документация
```

## API Endpoints

- `GET /healthz` - Health check endpoint
- `GET /health` - Альтернативный health check
- `GET /metrics` - Метрики приложения (расширяемо)

## Команды бота

### Приватные сообщения
- `/start` - Приветствие и инструкции
- Отправка фото/видео - Пересылка в канал

### Групповые чаты
- `/memes` - Список последних мемов в чате
- `/forget` - Очистка истории мемов (только админы)
- Отправка фото - Генерация комментария от AI
- Ответ на комментарий бота - Контекстный диалог

## Конфигурация

### Переменные окружения

| Переменная | Описание | Обязательная |
|------------|----------|--------------|
| `TELEGRAM_BOT_TOKEN` | Токен Telegram бота | ✅ |
| `TELEGRAM_CHANNEL` | ID или username канала | ✅ |
| `DATABASE_URL` | Строка подключения к PostgreSQL | ✅ |
| `OPENAI_API_KEY` | Ключ OpenAI API | ✅ |
| `OPENAI_BASE_URL` | Base URL для OpenAI API | ❌ |
| `BANNED_USER_IDS` | Список заблокированных пользователей (через запятую) | ❌ |
| `ADMIN_USER_IDS` | Список администраторов (через запятую) | ❌ |
| `PORT` | Порт веб-сервера | ❌ |

### Файл config.json (опционально)

```json
{
  "token": "your_bot_token",
  "channel": "@your_channel",
  "banned_user_ids": [123456789],
  "admin_ids": [987654321]
}
```

## База данных

Приложение автоматически создает и мигрирует следующие таблицы:

- `admins` - Администраторы
- `banned_users` - Заблокированные пользователи  
- `memes` - Мемы
- `meme_comments` - Комментарии к мемам
- `meme_history` - История мемов
- `user_dialogs` - Диалоги с пользователями
- `meme_interactions` - Взаимодействия с мемами
- `comment_meme_mappings` - Связь комментариев с мемами

## Мониторинг

### Health Checks

```bash
# Проверка статуса приложения
curl http://localhost:8080/healthz

# Ответ:
{
  "status": "ok",
  "message": "Bot is up",
  "service": "memebot"
}
```

## Разработка

### Добавление новых функций

1. **Модели**: Добавьте новые модели в `models/models.go`
2. **Миграции**: GORM автоматически применяет миграции при запуске
3. **Обработчики**: Добавьте новые обработчики в `handlers/handlers.go`
4. **Сервисы**: Расширьте функциональность в `services/`

### Тестирование

```bash
# Запуск тестов
go test ./...

# Тесты с покрытием
go test -cover ./...
```

### Сборка

```bash
# Локальная сборка
go build -o memebot .

# Сборка для Linux (если разрабатываете на другой ОС)
GOOS=linux GOARCH=amd64 go build -o memebot .

# Docker сборка
docker build -t memebot .
```

## Лицензия

MIT License

## Поддержка

Если у вас есть вопросы или предложения, создайте issue в репозитории.