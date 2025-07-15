package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"backend/configs"
	"backend/internal/repositories"
	"backend/internal/services"

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

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	logDir := filepath.Join(home, "Desktop", "blockchain_analyzer", "backend", "logs")
	logName := filepath.Join(logDir, "app.log")
	logFile, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Подключаемся к базе данных
	if err := repositories.Connect(ctx, cfg); err != nil {
		logrus.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Выполняем миграции
	if err := repositories.Migrate(); err != nil {
		logrus.Fatalf("Ошибка миграции БД: %v", err)
	}

	// Проверяем адрес контракта
	contractAddr := common.HexToAddress(cfg.Ethereum.ContractAddress)
	if contractAddr == (common.Address{}) {
		logrus.Fatalf("Неверный адрес контракта: %s", cfg.Ethereum.ContractAddress)
	}

	logrus.Infof("📋 Используем ERC20 контракт: %s", contractAddr.Hex())

	// Создаем и запускаем анализатор
	analyzer, err := services.NewAnalyzer(cfg)
	if err != nil {
		logrus.Fatalf("Ошибка создания анализатора: %v", err)
	}

	if err := analyzer.Start(ctx); err != nil {
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
