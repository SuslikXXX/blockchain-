# Blockchain Analyzer

Анализатор активности в блокчейне Ethereum с отслеживанием ERC20 токенов и уведомлениями о подозрительной активности.

## Возможности

- Отслеживание ETH транзакций и ERC20 трансферов
- Анализ активности аккаунтов
- Подсчет объема транзакций
- Уведомления о подозрительной активности
- Отслеживание балансов токенов
- Статистика использования контрактов

## Структура проекта

```
blockchain_analyzer/
├── backend/              # Go backend
│   ├── cmd/             # Точки входа приложения
│   ├── configs/         # Конфигурация
│   ├── internal/        # Внутренняя логика
│   │   ├── models/      # Модели данных
│   │   ├── repositories/# Работа с БД
│   │   ├── services/    # Бизнес-логика
│   │   └── utils/       # Утилиты
│   └── pkg/             # Публичные пакеты
│       ├── contracts/   # Интерфейсы контрактов
│       ├── ethereum/    # Клиент Ethereum
│       └── listeners/   # Слушатели событий
└── blockchain/          # Смарт-контракты и тесты
    ├── contracts/       # Solidity контракты
    └── scripts/         # Скрипты деплоя и тестов
```

## Требования

- Go 1.20+
- Node.js 18+
- PostgreSQL 14+
- Hardhat
- Доступ к Ethereum ноде (локальной или удаленной)

## Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/yourusername/blockchain_analyzer.git
cd blockchain_analyzer
```

2. Установите зависимости для смарт-контрактов:
```bash
cd blockchain
npm install
```

3. Установите зависимости для бэкенда:
```bash
cd ../backend
go mod download
```

4. Создайте файл конфигурации `.env` в директории `backend`:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=blockchain_analyzer
DB_SSLMODE=disable

ETH_RPC_URL=http://localhost:8545
CONTRACT_ADDRESS=0x...
```

## Запуск

1. Запустите локальную Ethereum ноду (например, через Hardhat):
```bash
cd blockchain
npx hardhat node
```

2. В отдельном терминале деплойте контракты:
```bash
npx hardhat run scripts/deploy.js --network localhost
```

3. Запустите бэкенд:
```bash
cd ../backend
go run cmd/app/main.go
```

## Тестирование

1. Тестирование смарт-контрактов:
```bash
cd blockchain
npx hardhat test
```

2. Тестирование бэкенда:
```bash
cd backend
go test ./...
```

## Лицензия

MIT 