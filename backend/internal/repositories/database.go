package repositories

import (
	"backend/configs"
	"backend/internal/models"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(ctx context.Context, cfg *configs.Config) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	logrus.Info("Успешное подключение к базе данных PostgreSQL")
	return nil
}

func Migrate() error {
	if DB == nil {
		return fmt.Errorf("соединение с базой данных не установлено")
	}

	err := DB.AutoMigrate(
		&models.Transaction{},
		&models.ERC20Transfer{},
	)
	if err != nil {
		return fmt.Errorf("ошибка миграции: %w", err)
	}

	logrus.Info("Миграции выполнены успешно")
	return nil
}

func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
