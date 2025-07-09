package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
)

type ActivityCalculator struct {
	accountRepo *repositories.AccountRepository
}

// ИСПРАВЛЕНО: добавлен конструктор с dependency injection
func NewActivityCalculator(accountRepo *repositories.AccountRepository) *ActivityCalculator {
	return &ActivityCalculator{
		accountRepo: accountRepo,
	}
}

// Deprecated: оставлен для совместимости, но лучше использовать конструктор с DI
func NewActivityCalculatorDeprecated() *ActivityCalculator {
	return &ActivityCalculator{
		accountRepo: repositories.NewAccountRepository(),
	}
}

func (c *ActivityCalculator) CalculateTransactionFrequency(address string, stats *models.AccountStats) error {

	// ИСПРАВЛЕНО: используем утилитарную функцию для расчета периода
	currentPeriodStart := utils.GetCurrentPeriodStart()

	periodTxCount, err := c.accountRepo.GetTransactionCountSince(address, currentPeriodStart)
	if err != nil {
		return err
	}

	stats.LastPeriodTransactions = periodTxCount

	// Получаем транзакции за последние 24 часа
	since := time.Now().Add(-24 * time.Hour)
	txCount, err := c.accountRepo.GetTransactionCountSince(address, since)
	if err != nil {
		return err
	}

	logrus.Debugf("Аккаунт %s: транзакций за 24ч = %d, за 15 сек = %d", address, txCount, periodTxCount)
	return nil
}

func (c *ActivityCalculator) CalculateVolumeMetrics(address string, stats *models.AccountStats) error {
	// ИСПРАВЛЕНО: используем утилитарную функцию для расчета периода
	currentPeriodStart := utils.GetCurrentPeriodStart()

	volume, err := c.accountRepo.GetVolumeETHSince(address, currentPeriodStart)
	if err != nil {
		return err
	}

	stats.SetLastPeriodVolumeETH(volume)

	logrus.Debugf("Аккаунт %s: объем за 15 секунд = %s ETH", address, volume.String())
	return nil
}

func (c *ActivityCalculator) CalculateTokenInteractions(address string, stats *models.AccountStats) error {
	// Подсчитываем уникальные токены, с которыми взаимодействовал аккаунт
	uniqueTokens, err := c.accountRepo.GetUniqueTokensCount(address)
	if err != nil {
		return err
	}

	stats.UniqueTokensCount = uniqueTokens

	logrus.Debugf("Аккаунт %s: уникальных токенов = %d", address, uniqueTokens)
	return nil
}

func (c *ActivityCalculator) CalculatePeriodMetrics(address string) error {
	// ИСПРАВЛЕНО: используем утилитарную функцию для расчета периода
	currentPeriod := utils.GetCurrentPeriodStart()

	// Получаем транзакции за текущий период 15 секунд
	txCount, err := c.accountRepo.GetTransactionCountForPeriod(address, currentPeriod)
	if err != nil {
		return err
	}

	// Получаем объем за текущий период 15 секунд
	volume, err := c.accountRepo.GetVolumeETHForPeriod(address, currentPeriod)
	if err != nil {
		return err
	}

	// Создаем или обновляем запись активности
	activity := &models.AccountActivity{
		Address:          address,
		Period:           currentPeriod,
		TransactionCount: txCount,
		VolumeETH:        volume.String(),
		TokenTransfers:   0, // Пока не реализовано
	}

	if err := c.accountRepo.SaveAccountActivity(activity); err != nil {
		return err
	}

	logrus.Debugf("Сохранены метрики для %s за %v: транзакций = %d, объем = %s ETH",
		address, currentPeriod, txCount, volume.String())

	return nil
}

func (c *ActivityCalculator) CalculateAllMetrics(address string, stats *models.AccountStats) error {
	// Рассчитываем все метрики для аккаунта
	if err := c.CalculateTransactionFrequency(address, stats); err != nil {
		logrus.Errorf("Ошибка расчета частоты транзакций для %s: %v", address, err)
		return err
	}

	if err := c.CalculateVolumeMetrics(address, stats); err != nil {
		logrus.Errorf("Ошибка расчета объемов для %s: %v", address, err)
		return err
	}

	if err := c.CalculateTokenInteractions(address, stats); err != nil {
		logrus.Errorf("Ошибка расчета взаимодействий с токенами для %s: %v", address, err)
		return err
	}

	return nil
}
