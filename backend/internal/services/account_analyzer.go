package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/utils"
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type AccountAnalyzer struct {
	accountRepo  *repositories.AccountRepository
	calculator   *ActivityCalculator
	tokenTracker *TokenTracker
}

// ИСПРАВЛЕНО: добавлен конструктор с dependency injection
func NewAccountAnalyzer(accountRepo *repositories.AccountRepository) *AccountAnalyzer {
	return &AccountAnalyzer{
		accountRepo:  accountRepo,
		calculator:   NewActivityCalculator(accountRepo),
		tokenTracker: NewTokenTracker(accountRepo),
	}
}

// Deprecated: оставлен для совместимости, но лучше использовать конструктор с DI
func NewAccountAnalyzerDeprecated() *AccountAnalyzer {
	repo := repositories.NewAccountRepository()
	return &AccountAnalyzer{
		accountRepo:  repo,
		calculator:   NewActivityCalculator(repo),
		tokenTracker: NewTokenTracker(repo),
	}
}

// AnalyzeAccountActivity анализирует активность конкретного аккаунта
func (a *AccountAnalyzer) AnalyzeAccountActivity(address string) (*models.AccountStats, error) {
	// Получаем или создаем статистику аккаунта
	stats, err := a.accountRepo.GetOrCreateAccountStats(address)
	if err != nil {
		return nil, err
	}

	// Пересчитываем все метрики
	if err := a.calculator.CalculateAllMetrics(address, stats); err != nil {
		logrus.Errorf("Ошибка расчета метрик для аккаунта %s: %v", address, err)
		return stats, err
	}

	// Сохраняем обновленную статистику
	if err := a.accountRepo.UpdateAccountStats(stats); err != nil {
		return nil, err
	}

	logrus.Debugf("Анализ аккаунта %s завершен", address)
	return stats, nil
}

// UpdateAccountStats обновляет статистику аккаунтов при обработке транзакции
func (a *AccountAnalyzer) UpdateAccountStats(tx *models.Transaction) error {
	// Обновляем статистику отправителя
	if err := a.updateSingleAccountStats(tx.From, tx, true); err != nil {
		return err
	}

	// Обновляем статистику получателя (если есть)
	if tx.To != "" {
		if err := a.updateSingleAccountStats(tx.To, tx, false); err != nil {
			return err
		}
	}

	return nil
}

// updateSingleAccountStats обновляет статистику одного аккаунта
func (a *AccountAnalyzer) updateSingleAccountStats(address string, tx *models.Transaction, isSender bool) error {
	stats, err := a.accountRepo.GetOrCreateAccountStats(address)
	if err != nil {
		return err
	}

	// Обновляем общие метрики
	stats.TotalTransactions++

	// Обновляем время активности
	if stats.LastActivityTime == nil || tx.Timestamp.After(*stats.LastActivityTime) {
		stats.LastActivityTime = &tx.Timestamp
	}

	if stats.FirstActivityTime == nil || tx.Timestamp.Before(*stats.FirstActivityTime) {
		stats.FirstActivityTime = &tx.Timestamp
	}

	// Обновляем объем (только для отправителя)
	if isSender {
		currentVolume := stats.GetTotalVolumeETH()
		txValue := tx.GetValue()
		newVolume := currentVolume.Add(currentVolume, txValue)
		stats.SetTotalVolumeETH(newVolume)
	}

	// Обновляем метрики последних 15 секунд
	// ИСПРАВЛЕНО: используем утилитарную функцию для расчета периода
	currentPeriodStart := utils.GetCurrentPeriodStart()

	if tx.Timestamp.After(currentPeriodStart) && tx.Timestamp.Before(time.Now()) {
		stats.LastPeriodTransactions++

		if isSender {
			currentPeriodVolume := stats.GetLastPeriodVolumeETH()
			txValue := tx.GetValue()
			newPeriodVolume := currentPeriodVolume.Add(currentPeriodVolume, txValue)
			stats.SetLastPeriodVolumeETH(newPeriodVolume)
		}
	}

	// Сохраняем обновленную статистику
	if err := a.accountRepo.UpdateAccountStats(stats); err != nil {
		return err
	}

	logrus.Debugf("Обновлена статистика аккаунта %s: транзакций = %d", address, stats.TotalTransactions)
	return nil
}

// CalculatePeriodStats пересчитывает статистику каждые 15 секунд для активных аккаунтов
func (a *AccountAnalyzer) CalculatePeriodStats() error {
	// Получаем все активные аккаунты за последние 15 секунд
	accounts, err := a.accountRepo.GetActiveAccountsLastPeriod()
	if err != nil {
		return err
	}

	logrus.Infof("Пересчитываем статистику для %d аккаунтов", len(accounts))

	errorCount := 0
	for _, account := range accounts {
		if err := a.calculator.CalculatePeriodMetrics(account); err != nil {
			logrus.Errorf("Ошибка расчета метрик для %s: %v", account, err)
			errorCount++
		}
	}

	if errorCount > 0 {
		logrus.Warnf("Обработано %d аккаунтов с %d ошибками", len(accounts), errorCount)
	} else {
		logrus.Infof("Статистика успешно пересчитана для %d аккаунтов", len(accounts))
	}

	return nil
}

// GetAccountSummary возвращает сводку по аккаунту
func (a *AccountAnalyzer) GetAccountSummary(address string) (*models.AccountStats, error) {
	stats, err := a.accountRepo.GetOrCreateAccountStats(address)
	if err != nil {
		return nil, err
	}

	// Пересчитываем актуальные метрики
	if err := a.calculator.CalculateAllMetrics(address, stats); err != nil {
		logrus.Errorf("Ошибка пересчета метрик для %s: %v", address, err)
		// Возвращаем существующие данные даже при ошибке пересчета
	}

	return stats, nil
}

// GetTokenTracker возвращает трекер токенов для интеграции с другими сервисами
func (a *AccountAnalyzer) GetTokenTracker() *TokenTracker {
	return a.tokenTracker
}

// StartPeriodicTasks запускает периодические задачи (можно вызвать в отдельной горутине)
func (a *AccountAnalyzer) StartPeriodicTasks(ctx context.Context) {
	// Пересчитываем статистику каждые 15 секунд
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Остановка периодических задач анализа аккаунтов")
			return
		case <-ticker.C:
			if err := a.CalculatePeriodStats(); err != nil {
				logrus.Errorf("Ошибка пересчета статистики: %v", err)
			}
		}
	}
}
