# Анализатор активности блокчейна

Многослойный модульный анализатор для мониторинга активности Ethereum блокчейна с использованием go-ethereum библиотеки.

## Структура проекта

```
blockchain/
├── cmd/                    # Точки входа приложения
│   └── app/
│       └── main.go
├── internal/               # Внутренние пакеты (private)
│   ├── models/            # Модели данных
│   │   └── blockchain.go
│   ├── repositories/      # Слой доступа к данным
│   │   └── database.go
│   └── services/          # Бизнес-логика
│       └── analyzer.go
├── pkg/                   # Публичные пакеты (reusable)
│   ├── ethereum/          # Клиент для работы с Ethereum
│   │   └── client.go
│   ├── contracts/         # Интерфейсы для смарт-контрактов
│   │   └── erc20.go
│   ├── listeners/         # Event listening сервисы
│   │   └── event_listener.go
│   └── simulators/        # Утилиты для тестирования (Hardhat)
│       └── hardhat.go
├── configs/               # Конфигурация приложения
│   └── config.go
├── hardhat-scripts/       # Скрипты для Hardhat
│   └── deploy.js
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Функциональность

### Основные возможности
- 🔗 Подключение к Ethereum RPC (поддержка Hardhat node)
- 📊 Мониторинг транзакций в реальном времени
- 🎯 Event listening для ERC20 событий Transfer
- 💾 Сохранение данных в PostgreSQL
- 🧪 Автоматическое тестирование с деплоем ERC20 контракта

### Архитектурные слои
1. **Конфигурационный слой** - управление настройками
2. **Слой данных** - модели и подключение к БД
3. **Blockchain слой** - взаимодействие с Ethereum
4. **Сервисный слой** - бизнес-логика анализа
5. **Event слой** - обработка событий блокчейна

## Требования

- Go 1.23+
- PostgreSQL
- Hardhat node (для тестирования)

## Установка и запуск

### 1. Клонирование и установка зависимостей
```bash
git clone <your-repo>
cd blockchain
go mod tidy
```

### 2. Настройка PostgreSQL
```bash
# Создание базы данных
createdb blockchain_analyzer
```

### 3. Настройка Hardhat (для тестирования)
```bash
# В отдельном терминале
npx hardhat node
```

### 4. Конфигурация
```bash
cp .env.example .env
# Отредактируйте .env с вашими настройками
```

### 5. Запуск
```bash
go run main.go
```

## Конфигурация

Основные переменные окружения:

- `ETH_RPC_ENDPOINT` - RPC endpoint (по умолчанию: http://localhost:8545)
- `ETH_CHAIN_ID` - Chain ID (31337 для Hardhat)
- `ETH_PRIVATE_KEY` - Приватный ключ для деплоя и транзакций
- `DB_*` - Настройки PostgreSQL

## Использование

При запуске приложение:

1. 🚀 Инициализирует подключения к БД и Ethereum
2. 🔧 Деплоит тестовый ERC20 контракт
3. 🔄 Выполняет тестовый transfer
4. 📡 Начинает мониторинг событий и транзакций
5. 💾 Сохраняет данные в PostgreSQL

### Мониторинг

- **Транзакции**: сохраняются в таблице `transactions`
- **ERC20 Transfers**: сохраняются в таблице `erc20_transfers`
- **Логи**: цветные логи в консоли

## Схема базы данных

### Таблица transactions
- id, hash, block_number, from, to
- value, gas_used, gas_price, status
- timestamp, created_at, updated_at

### Таблица erc20_transfers  
- id, transaction_hash, contract_address
- from, to, value, block_number, log_index
- created_at

## Расширение

Проект легко расширяется для:
- Мониторинга других типов контрактов
- Добавления веб API
- Интеграции с другими блокчейнами
- Добавления аналитики и метрик 