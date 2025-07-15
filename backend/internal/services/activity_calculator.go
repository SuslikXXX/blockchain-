package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/utils"
	"backend/pkg/ethereum"
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

var (
	contractCache    = make(map[string]bool)
	contractCacheMux sync.RWMutex
)

type ActivityCalculator struct {
	accountRepo  *repositories.AccountRepository
	ethClient    *ethereum.Client
	tokenAddress common.Address // Адрес нашего токена
}

func NewActivityCalculator(accountRepo *repositories.AccountRepository, ethClient *ethereum.Client, tokenAddress common.Address) *ActivityCalculator {
	return &ActivityCalculator{
		accountRepo:  accountRepo,
		ethClient:    ethClient,
		tokenAddress: tokenAddress,
	}
}

// Deprecated: оставлен для совместимости, но лучше использовать конструктор с DI
func NewActivityCalculatorDeprecated() *ActivityCalculator {
	return &ActivityCalculator{
		accountRepo: repositories.NewAccountRepository(),
	}
}

// isContract проверяет, является ли адрес контрактом
func (c *ActivityCalculator) isContract(address string) bool {
	if c.ethClient == nil {
		return false // Если клиент не инициализирован, считаем что это не контракт
	}

	// Проверяем кэш
	contractCacheMux.RLock()
	if isContract, exists := contractCache[address]; exists {
		contractCacheMux.RUnlock()
		return isContract
	}
	contractCacheMux.RUnlock()

	code, err := c.ethClient.GetClient().CodeAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		logrus.Errorf("Ошибка получения кода для адреса %s: %v", address, err)
		return false
	}

	// Если есть байткод (длина > 0), значит это контракт
	isContract := len(code) > 0

	// Сохраняем в кэш
	contractCacheMux.Lock()
	contractCache[address] = isContract
	contractCacheMux.Unlock()

	return isContract
}

func (c *ActivityCalculator) CalculateTransactionFrequency(address string, stats *models.AccountStats) error {

	// ИСПРАВЛЕНО: используем утилитарную функцию для расчета периода
	currentPeriodStart := utils.GetCurrentPeriodStart()

	periodTxCount, err := c.accountRepo.GetTransactionCountSince(address, currentPeriodStart)
	if err != nil {
		return err
	}

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

func (c *ActivityCalculator) CalculateETHBalance(address string, stats *models.AccountStats) error {
	if c.ethClient == nil {
		return fmt.Errorf("eth client not initialized")
	}

	balance, err := c.ethClient.GetBalance(context.Background(), common.HexToAddress(address))
	if err != nil {
		return fmt.Errorf("ошибка получения баланса ETH: %v", err)
	}

	stats.SetETHBalance(balance)
	logrus.Debugf("Аккаунт %s: баланс ETH = %s wei", address, balance.String())
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

	// Добавляем расчет баланса ETH
	if err := c.CalculateETHBalance(address, stats); err != nil {
		logrus.Errorf("Ошибка получения баланса ETH для %s: %v", address, err)
		return err
	}

	return nil
}

func (c *ActivityCalculator) ProcessAccountActivities() error {
	// Получаем начало текущего периода
	currentPeriod := time.Now().Truncate(15 * time.Second)

	// Получаем все транзакции за текущий период
	transactions, err := c.accountRepo.GetTransactionsSince(currentPeriod)
	if err != nil {
		return err
	}

	// Создаем map для хранения активности по адресам
	activityMap := make(map[string]*models.AccountActivity)

	// Создаем map для кэширования результатов проверки на контракт
	contractCache := make(map[string]bool)

	// Обрабатываем каждую транзакцию
	for _, tx := range transactions {

		// Обрабатываем отправителя, проверяем, не является ли он контрактом, ведь нас интересуют только активность обычных аккаунтов
		// В будущем можно будет легко переделать еще и на обработку контрактов, но сейчас это не нужно
		isContract, exists := contractCache[tx.From]
		if !exists {
			isContract = c.isContract(tx.From)
			contractCache[tx.From] = isContract
		}

		// Проверяем, является ли это обычной ETH транзакцией
		if tx.To != c.tokenAddress.Hex() && !isContract {
			// Это обычная ETH транзакция
			fromActivity := getOrCreateActivity(activityMap, tx.From, currentPeriod)
			fromActivity.TransactionCount++

			if value, ok := big.NewInt(0).SetString(tx.Value, 10); ok {
				currentVolume := fromActivity.GetVolumeETH()
				fromActivity.SetVolumeETH(currentVolume.Add(currentVolume, value))
			}
		}

		// Если транзакция к токен-контракту, она будет обработана как ERC20 трансфер позже

	}

	// Получаем все ERC20 трансферы за текущий период
	erc20Transfers, err := c.getERC20TransfersSince(currentPeriod)
	if err != nil {
		logrus.Errorf("Ошибка получения ERC20 трансферов: %v", err)
	} else {
		// Обрабатываем каждый ERC20 трансфер
		for _, transfer := range erc20Transfers {

			// Проверяем отправителя на контракт
			// Если в будущем я реализую автомавызов транзакций контрактом, то это проверку нужно будет убрать
			isContract, exists := contractCache[transfer.From]
			if !exists {
				isContract = c.isContract(transfer.From)
				contractCache[transfer.From] = isContract
			}

			// Обрабатываем только отправителя и только если это не контракт
			if !isContract {
				fromActivity := getOrCreateActivity(activityMap, transfer.From, currentPeriod)
				fromActivity.TokenTransfers++
			}
		}
	}

	// Сохраняем или обновляем все активности
	for _, activity := range activityMap {
		if err := c.accountRepo.SaveAccountActivity(activity); err != nil {
			logrus.Errorf("Ошибка сохранения активности для %s: %v", activity.Address, err)
		}
	}

	return nil
}

func (c *ActivityCalculator) getERC20TransfersSince(since time.Time) ([]models.ERC20Transfer, error) {
	nextPeriod := since.Add(15 * time.Second)
	var transfers []models.ERC20Transfer
	// Для наносекундной точности
	err := repositories.DB.Where("created_at >= ? AND created_at < ?",
		since.Truncate(time.Microsecond),
		nextPeriod.Truncate(time.Microsecond)).Find(&transfers).Error
	return transfers, err
}

func getOrCreateActivity(activityMap map[string]*models.AccountActivity, address string, period time.Time) *models.AccountActivity {
	activity, exists := activityMap[address]
	if !exists {
		activity = &models.AccountActivity{
			Address: address,
			Period:  period,
		}
		activityMap[address] = activity
	}
	return activity
}
