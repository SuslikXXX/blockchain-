# Анализатор активности блокчейна

Этот проект разделен на две логические части:

## 📁 Структура проекта

### 🔧 Backend (`/backend`)
Go-приложение для анализа и мониторинга блокчейн активности:
- **cmd/** - главное приложение
- **configs/** - конфигурация
- **internal/** - внутренние компоненты (models, repositories, services)
- **pkg/** - пакеты для работы с Ethereum (клиенты, контракты, слушатели)
- **logs/** - логи приложения

### ⛓️ Blockchain (`/blockchain`)
Инфраструктура смарт-контрактов и деплоя:
- **contracts/** - Solidity контракты
- **scripts/** - скрипты деплоя
- **ignition/** - модули деплоя
- **artifacts/** - скомпилированные контракты
- **hardhat.config.js** - конфигурация Hardhat

## 🚀 Запуск

### Backend
```bash
cd backend
go run cmd/app/main.go
```

### Blockchain (деплой контрактов)
```bash
cd blockchain
npm install
npx hardhat compile
npx hardhat run scripts/deploy.js --network localhost
```

## 🔗 Взаимодействие

Backend использует pkg/ethereum для подключения к блокчейну и мониторинга контрактов, задеплоенных через blockchain часть проекта. 