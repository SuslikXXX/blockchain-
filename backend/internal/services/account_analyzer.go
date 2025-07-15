package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/utils"
	"backend/pkg/ethereum"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

type AccountAnalyzer struct {
	accountRepo  *repositories.AccountRepository
	calculator   *ActivityCalculator
	tokenTracker *TokenTracker
	tokenAddress common.Address
}

// ИСПРАВЛЕНО: добавлен tokenAddress в конструктор
func NewAccountAnalyzer(accountRepo *repositories.AccountRepository, ethClient *ethereum.Client, tokenAddress common.Address) *AccountAnalyzer {
	return &AccountAnalyzer{
		accountRepo:  accountRepo,
		calculator:   NewActivityCalculator(accountRepo, ethClient, tokenAddress),
		tokenTracker: NewTokenTracker(accountRepo),
		tokenAddress: tokenAddress,
	}
}

// Deprecated: оставлен для совместимости, но лучше использовать конструктор с DI
func NewAccountAnalyzerDeprecated() *AccountAnalyzer {
	repo := repositories.NewAccountRepository()
	zeroAddress := common.Address{} // Используем нулевой адрес для обратной совместимости
	return &AccountAnalyzer{
		accountRepo:  repo,
		calculator:   NewActivityCalculator(repo, nil, zeroAddress),
		tokenTracker: NewTokenTracker(repo),
		tokenAddress: zeroAddress,
	}
}

// AnalyzeAccountActivity анализирует активность конкретного аккаунта
func (a *AccountAnalyzer) AnalyzeAccountActivity(address string) (*models.AccountStats, error) {
	// Проверяем, не является ли адрес контрактом
	if a.calculator.isContract(address) {
		logrus.Debugf("Пропускаем анализ контракта %s", address)
		return nil, nil
	}

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
	// Проверяем на нулевой адрес
	zeroAddress := "0x0000000000000000000000000000000000000000"
	if tx.To == zeroAddress {
		logrus.Debugf("Пропускаем обновление статистики для транзакции с нулевым адресом получателя: %s", tx.Hash)
		return nil
	}

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
	// Пропускаем нулевой адрес
	if address == "0x0000000000000000000000000000000000000000" {
		return nil
	}

	// Пропускаем контракты
	if !isSender { // отправители не могут быть контрактами
		if a.calculator.isContract(address) {
			logrus.Debugf("Пропускаем обновление статистики для контракта %s", address)
			return nil
		}
	}

	stats, err := a.accountRepo.GetOrCreateAccountStats(address)
	if err != nil {
		return err
	}

	// Обновляем общие метрики только для отправителя
	if isSender {
		stats.TotalTransactions++
	}

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

	// Обновляем баланс ETH
	if err := a.calculator.CalculateETHBalance(address, stats); err != nil {
		logrus.Errorf("Ошибка обновления баланса ETH для %s: %v", address, err)
		// Не возвращаем ошибку, чтобы не прерывать обновление других метрик
	}

	// Сохраняем обновленную статистику
	if err := a.accountRepo.UpdateAccountStats(stats); err != nil {
		return err
	}

	logrus.Debugf("Обновлена статистика аккаунта %s: транзакций = %d", address, stats.TotalTransactions)
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

// GetAccountActivityForPeriod получает активность аккаунта за конкретный период
func (a *AccountAnalyzer) GetAccountActivityForPeriod(address string, period time.Time) (*models.AccountActivity, error) {
	activity, err := a.accountRepo.GetAccountActivityForPeriod(address, period)
	if err != nil {
		logrus.Errorf("Ошибка получения активности для %s за %v: %v", address, period, err)
		return nil, err
	}

	logrus.Debugf("Получена активность для %s за %v: транзакций = %d, объем = %s ETH",
		address, period, activity.TransactionCount, activity.VolumeETH)

	return activity, nil
}

// GetCurrentPeriodActivity получает активность аккаунта за текущий 15-секундный период
func (a *AccountAnalyzer) GetCurrentPeriodActivity(address string) (*models.AccountActivity, error) {
	currentPeriod := utils.GetCurrentPeriodStart()
	return a.GetAccountActivityForPeriod(address, currentPeriod)
}

// GetActiveAccountsForPeriod получает активность всех аккаунтов за конкретный период
func (a *AccountAnalyzer) GetActiveAccountsForPeriod(period time.Time) ([]models.AccountActivity, error) {
	activities, err := a.accountRepo.GetAllAccountsActivityForPeriod(period)
	if err != nil {
		logrus.Errorf("Ошибка получения активности всех аккаунтов за %v: %v", period, err)
		return nil, err
	}

	logrus.Infof("Получена активность %d аккаунтов за %v", len(activities), period)
	return activities, nil
}

// GetCurrentPeriodActiveAccounts получает активность всех аккаунтов за текущий период
func (a *AccountAnalyzer) GetCurrentPeriodActiveAccounts() ([]models.AccountActivity, error) {
	currentPeriod := utils.GetCurrentPeriodStart()
	return a.GetActiveAccountsForPeriod(currentPeriod)
}

// GetAccountActivityHistory получает историю активности аккаунта за несколько периодов
func (a *AccountAnalyzer) GetAccountActivityHistory(address string, fromPeriod, toPeriod time.Time) ([]models.AccountActivity, error) {
	activities, err := a.accountRepo.GetAccountActivityHistory(address, fromPeriod, toPeriod)
	if err != nil {
		logrus.Errorf("Ошибка получения истории активности для %s: %v", address, err)
		return nil, err
	}

	logrus.Debugf("Получена история активности для %s: %d периодов", address, len(activities))
	return activities, nil
}
