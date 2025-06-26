package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"blockchain/configs"
	"blockchain/internal/repositories"
	"blockchain/internal/services"
	"blockchain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

func init() {
	// Настройка логирования
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
		PadLevelText:  true,
	})
	logrus.SetOutput(os.Stdout)

	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logrus.Fatalf("Ошибка открытия файла логирования: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logrus.SetOutput(multiWriter)

	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	logrus.Info("🚀 Запуск анализатора активности блокчейна")

	// Загружаем конфигурацию
	cfg := configs.Load()

	// Подключаемся к базе данных
	if err := repositories.Connect(cfg); err != nil {
		logrus.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Выполняем миграции
	if err := repositories.Migrate(); err != nil {
		logrus.Fatalf("Ошибка миграции БД: %v", err)
	}

	// Создаем Ethereum клиент
	ethClient, err := ethereum.NewClient(cfg)
	if err != nil {
		logrus.Fatalf("Ошибка подключения к Ethereum: %v", err)
	}
	defer ethClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Используем уже задеплоенный контракт
	contractAddr := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	logrus.Infof("📋 Используем ERC20 контракт: %s", contractAddr.Hex())

	// Создаем и запускаем анализатор
	analyzer, err := services.NewAnalyzer(cfg)
	if err != nil {
		logrus.Fatalf("Ошибка создания анализатора: %v", err)
	}

	if err := analyzer.Start(ctx, contractAddr); err != nil {
		logrus.Fatalf("Ошибка запуска анализатора: %v", err)
	}

	// Ожидаем сигнал остановки
	waitForShutdown(cancel, analyzer)
}

func waitForShutdown(cancel context.CancelFunc, analyzer *services.Analyzer) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logrus.Info("📡 Анализатор работает. Нажмите Ctrl+C для остановки...")

	<-sigChan
	logrus.Info("🛑 Получен сигнал остановки. Завершаем работу...")

	cancel()
	analyzer.Stop()

	if err := repositories.Close(); err != nil {
		logrus.Errorf("Ошибка закрытия БД: %v", err)
	}

	logrus.Info("👋 Анализатор остановлен")
}
