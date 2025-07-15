package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"backend/configs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CustomFormatter struct {
	logrus.TextFormatter
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("%s %s\n", entry.Time.Format(time.RFC3339), entry.Message)), nil
}

const (
	notificationInterval = 20 * time.Second
	batchSize            = 100
)

type Notifier struct {
	db       *gorm.DB
	notifLog *logrus.Logger
	bot      *tgbotapi.BotAPI
	chatID   string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewNotifier(cfg *configs.Config) (*Notifier, error) {
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

	// Создаем бота если есть конфигурация
	var bot *tgbotapi.BotAPI
	if cfg.TelegramBot.BotToken != "" {
		bot, err = tgbotapi.NewBotAPI(cfg.TelegramBot.BotToken)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания Telegram бота: %v", err)
		}
	}

	return &Notifier{
		db:       repositories.DB,
		notifLog: notifLog,
		bot:      bot,
		chatID:   cfg.TelegramBot.ChatID,
	}, nil
}

func (n *Notifier) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()

		// Ждем начала следующего 15-секундного периода
		now := time.Now()
		nextPeriod := now.Truncate(15 * time.Second).Add(15 * time.Second)
		time.Sleep(time.Until(nextPeriod))

		// Создаем тикер, который будет срабатывать точно в начале каждого периода
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		// Сразу обрабатываем активность при старте
		if err := n.processNewActivities(); err != nil {
			logrus.Errorf("Ошибка обработки активности: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				if err := n.processNewActivities(); err != nil {
					logrus.Errorf("Ошибка обработки активности: %v", err)
				}
				// Логируем, если обработка заняла слишком много времени
				if elapsed := time.Since(start); elapsed > 5*time.Second {
					logrus.Warnf("Обработка активности заняла %v", elapsed)
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
	// Получаем время начала текущего периода
	currentPeriod := time.Now().Truncate(15 * time.Second)

	// Получаем новые записи активности только за последние два периода
	var activities []models.AccountActivity
	if err := n.db.Where("period >= ? AND period < ?",
		currentPeriod.Add(-15*time.Second), // предыдущий период
		currentPeriod.Add(15*time.Second),  // следующий период
	).Order("id ASC").Find(&activities).Error; err != nil {
		return fmt.Errorf("ошибка получения активности: %v", err)
	}

	for _, activity := range activities {
		if err := n.analyzeActivity(&activity); err != nil {
			logrus.Errorf("Ошибка анализа активности %d: %v", activity.ID, err)
			continue
		}
	}

	return nil
}

func (n *Notifier) analyzeActivity(activity *models.AccountActivity) error {

	if (activity.TransactionCount + activity.TokenTransfers) > 3 {

		err := n.SendMessageTelegram(activity)
		if err != nil {
			logrus.Errorf("Ошибка отправки в Telegram: %v", err)
		}

		err = n.MakeNotificationLog(activity)
		if err != nil {
			logrus.Errorf("Ошибка создания лога: %v", err)
		}
	}
	return nil
}

func (n *Notifier) SendMessageTelegram(activity *models.AccountActivity) error {
	message := fmt.Sprintf(
		"🔍 <b>Высокая активность аккаунта</b>\n\n"+
			"📍 Адрес: <code>%s</code>\n"+
			"⏰ Период: %v\n"+
			"🔄 Транзакции: %d\n"+
			"🔁 Токен-трансферы: %d\n"+
			"💰 Объем (ETH): %s",
		activity.Address,
		activity.Period.Format("2006-01-02 15:04:05"),
		activity.TransactionCount,
		activity.TokenTransfers,
		activity.GetVolumeETH().String(),
	)

	if n.bot != nil && n.chatID != "" {
		chatID, err := strconv.ParseInt(n.chatID, 10, 64)
		if err != nil {
			logrus.Errorf("Неверный формат chat_id: %v", err)
			return nil
		}

		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = tgbotapi.ModeHTML

		if _, err := n.bot.Send(msg); err != nil {
			logrus.Errorf("Ошибка отправки в Telegram: %v", err)
		}
	}

	return nil
}

func (n *Notifier) MakeNotificationLog(activity *models.AccountActivity) error {
	n.notifLog.Printf("Address: %s, Period: %v, TransactionCount: %d, TokenTransfers: %d",
		activity.Address,
		activity.Period,
		activity.TransactionCount,
		activity.TokenTransfers)

	return nil
}
