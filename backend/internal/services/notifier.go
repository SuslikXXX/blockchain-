package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CustomFormatter struct {
	logrus.TextFormatter
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("%s %s\n", entry.Time.Format(time.RFC3339), entry.Message)), nil
}

const (
	notificationInterval = 15 * time.Second
	batchSize            = 100
)

type Notifier struct {
	db       *gorm.DB
	notifLog *logrus.Logger
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewNotifier() (*Notifier, error) {
	// Создаем отдельный логгер для нотификаций
	notifLog := logrus.New()
	notifLog.SetFormatter(&CustomFormatter{})

	// Создаем директорию для логов, если её нет
	logDir := filepath.Join("logs", "notifications")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории для логов: %v", err)
	}

	// Открываем файл для логирования нотификаций
	logFile, err := os.OpenFile(
		filepath.Join(logDir, "notifications.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла логов: %v", err)
	}

	notifLog.SetOutput(logFile)

	return &Notifier{
		db:       repositories.DB,
		notifLog: notifLog,
	}, nil
}

func (n *Notifier) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		ticker := time.NewTicker(notificationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := n.processNewActivities(); err != nil {
					logrus.Errorf("Ошибка обработки активности: %v", err)
				}
			}
		}
	}()

	return nil
}

func (n *Notifier) Stop() {
	if n.cancel != nil {
		n.cancel()
		n.wg.Wait()
	}
}

func (n *Notifier) processNewActivities() error {
	var state models.AnalyzerState
	if err := n.db.First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			state = models.AnalyzerState{
				LastProcessedActivityID: 0,
			}
			if err := n.db.Create(&state).Error; err != nil {
				return fmt.Errorf("ошибка создания начального состояния: %v", err)
			}
		} else {
			return fmt.Errorf("ошибка получения состояния: %v", err)
		}
	}

	// Получаем новые записи активности
	var activities []models.AccountActivity
	if err := n.db.Where("id > ?", state.LastProcessedActivityID).
		Order("id ASC").
		Limit(batchSize).
		Find(&activities).Error; err != nil {
		return fmt.Errorf("ошибка получения активности: %v", err)
	}

	if len(activities) == 0 {
		return nil
	}

	// Обрабатываем каждую запись
	for _, activity := range activities {
		if err := n.analyzeActivity(&activity); err != nil {
			logrus.Errorf("Ошибка анализа активности %d: %v", activity.ID, err)
			continue
		}

		// Обновляем последний обработанный ID
		state.LastProcessedActivityID = uint64(activity.ID)
	}

	// Сохраняем новое состояние
	if err := n.db.Save(&state).Error; err != nil {
		return fmt.Errorf("ошибка сохранения состояния: %v", err)
	}

	return nil
}

func (n *Notifier) analyzeActivity(activity *models.AccountActivity) error {
	// Проверяем подозрительную активность
	if (activity.TransactionCount + activity.TokenTransfers) > 3 {

		n.notifLog.Printf("Address: %s, Period: %v, TransactionCount: %d, TokenTransfers: %d",
			activity.Address,
			activity.Period,
			activity.TransactionCount,
			activity.TokenTransfers)
	}

	return nil
}
