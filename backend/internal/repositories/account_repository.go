package repositories

import (
	"backend/internal/models"
	"math/big"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{db: DB}
}

func (r *AccountRepository) GetOrCreateAccountStats(address string) (*models.AccountStats, error) {
	var stats models.AccountStats

	result := r.db.Where("address = ?", address).First(&stats)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Создаем новую запись
			stats = models.AccountStats{
				Address:        address,
				TotalVolumeETH: "0",
			}
			if err := r.db.Create(&stats).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}

	return &stats, nil
}

func (r *AccountRepository) UpdateAccountStats(stats *models.AccountStats) error {
	return r.db.Save(stats).Error
}

func (r *AccountRepository) GetTransactionCountSince(address string, since time.Time) (uint32, error) {
	var count int64
	err := r.db.Model(&models.Transaction{}).
		Where("(\"from\" = ? OR \"to\" = ?) AND timestamp >= ?", address, address, since).
		Count(&count).Error

	return uint32(count), err
}

func (r *AccountRepository) GetVolumeETHSince(address string, since time.Time) (*big.Int, error) {
	var transactions []models.Transaction
	err := r.db.Where("\"from\" = ? AND timestamp >= ?", address, since).Find(&transactions).Error
	if err != nil {
		return big.NewInt(0), err
	}

	total := big.NewInt(0)
	for _, tx := range transactions {
		total = total.Add(total, tx.GetValue())
	}

	return total, nil
}

func (r *AccountRepository) GetTokenBalance(address, tokenAddress string) (*models.TokenBalance, error) {
	var balance models.TokenBalance

	result := r.db.Where("address = ? AND token_address = ?", address, tokenAddress).First(&balance)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			balance = models.TokenBalance{
				Address:      address,
				TokenAddress: tokenAddress,
				Balance:      "0",
				LastUpdate:   time.Now(),
			}
			if err := r.db.Create(&balance).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}

	return &balance, nil
}

func (r *AccountRepository) UpdateTokenBalance(balance *models.TokenBalance) error {
	balance.LastUpdate = time.Now()
	return r.db.Save(balance).Error
}

func (r *AccountRepository) GetUniqueTokensCount(address string) (uint32, error) {
	var count int64
	err := r.db.Model(&models.TokenBalance{}).
		Where("address = ? AND balance != '0'", address).
		Count(&count).Error

	return uint32(count), err
}

func (r *AccountRepository) GetAccountTokens(address string) ([]models.TokenBalance, error) {
	var balances []models.TokenBalance

	err := r.db.Where("address = ? AND balance != '0'", address).
		Find(&balances).Error

	return balances, err
}

func (r *AccountRepository) GetTransactionsSince(since time.Time) ([]models.Transaction, error) {
	var transactions []models.Transaction
	// ИСПРАВЛЕНО: используем r.db вместо глобального DB
	nextPeriod := since.Add(15 * time.Second)
	result := r.db.Where("timestamp >= ? AND timestamp < ?", since, nextPeriod).Find(&transactions)
	return transactions, result.Error
}

func (r *AccountRepository) SaveAccountActivity(activity *models.AccountActivity) error {
	// Пытаемся найти существующую запись
	var existing models.AccountActivity
	// ИСПРАВЛЕНО: используем r.db вместо глобального DB
	result := r.db.Where("address = ? AND period = ?", activity.Address, activity.Period).First(&existing)

	if result.Error == nil {
		// Обновляем существующую запись
		existing.TransactionCount = activity.TransactionCount
		existing.VolumeETH = activity.VolumeETH
		existing.TokenTransfers = activity.TokenTransfers
		return r.db.Save(&existing).Error
	}

	// Создаем новую запись
	// ИСПРАВЛЕНО: используем r.db вместо глобального DB
	err := r.db.Create(activity).Error
	if err != nil {
		// Проверяем тип ошибки
		if err.Error() == "duplicate key value violates unique constraint" {
			logrus.Debugf("Активность для %s за период %v уже существует, пропускаем",
				activity.Address, activity.Period)
			return nil
		}
		if err.Error() == "check constraint violation" {
			logrus.Warnf("Попытка сохранить активность с невалидным адресом: %s",
				activity.Address)
			return nil
		}
		return err
	}

	return nil
}

// GetAccountActivityForPeriod получает активность аккаунта за конкретный период (с кешированием)
func (r *AccountRepository) GetAccountActivityForPeriod(address string, period time.Time) (*models.AccountActivity, error) {
	// Сначала проверяем кеш в account_activities
	var cachedActivity models.AccountActivity
	err := r.db.Where("address = ? AND period = ?", address, period).First(&cachedActivity).Error

	// Если данные найдены в кеше и они свежие (не старше 15 секунд), возвращаем их
	if err == nil {
		cacheAge := time.Since(cachedActivity.UpdatedAt)
		if cacheAge < 15*time.Second {
			logrus.Debugf("Данные из кеша для %s за %v (возраст: %v)", address, period, cacheAge)
			return &cachedActivity, nil
		}
		logrus.Debugf("Кеш устарел для %s за %v (возраст: %v), пересчитываем", address, period, cacheAge)
	} else if err != gorm.ErrRecordNotFound {
		// Если ошибка не "запись не найдена", возвращаем её
		return nil, err
	}

	// Кеша нет или он устарел - вычисляем из transactions
	nextPeriod := period.Add(15 * time.Second)

	var result struct {
		TransactionCount uint32
		VolumeETH        string
	}

	// Получаем количество исходящих транзакций и суммарный объем за период
	err = r.db.Model(&models.Transaction{}).
		Select(`
			COUNT(*) as transaction_count,
			COALESCE(SUM(CAST(value AS NUMERIC)), '0')::text as volume_eth
		`).
		Where("\"from\" = ? AND timestamp >= ? AND timestamp < ?",
			address, period, nextPeriod).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	// Если нет исходящих транзакций, возвращаем nil
	if result.TransactionCount == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// Получаем количество исходящих ERC20 трансферов
	var count int64
	err = r.db.Model(&models.ERC20Transfer{}).
		Where("\"from\" = ? AND created_at >= ? AND created_at < ?",
			address, period, nextPeriod).
		Count(&count).Error

	if err != nil {
		return nil, err
	}

	// Создаем объект активности только если есть исходящие транзакции
	activity := &models.AccountActivity{
		Address:          address,
		Period:           period,
		TransactionCount: result.TransactionCount + uint32(count),
		VolumeETH:        result.VolumeETH,
		TokenTransfers:   uint32(count),
	}

	// Сохраняем в кеш (асинхронно, чтобы не блокировать ответ)
	go func() {
		if saveErr := r.SaveAccountActivity(activity); saveErr != nil {
			logrus.Errorf("Не удалось сохранить в кеш: %v", saveErr)
		} else {
			logrus.Debugf("Сохранено в кеш для %s за %v", address, period)
		}
	}()

	return activity, nil
}

// GetAllAccountsActivityForPeriod получает активность всех аккаунтов за период (с кешированием)
func (r *AccountRepository) GetAllAccountsActivityForPeriod(period time.Time) ([]models.AccountActivity, error) {
	// Сначала получаем все записи из кеша для данного периода
	var cachedActivities []models.AccountActivity
	cacheErr := r.db.Where("period = ?", period).Find(&cachedActivities).Error
	if cacheErr != nil {
		logrus.Errorf("Ошибка чтения кеша: %v", cacheErr)
	}

	// Проверяем какие записи из кеша еще актуальны (не старше 15 сек)
	validCached := make(map[string]models.AccountActivity)
	for _, activity := range cachedActivities {
		cacheAge := time.Since(activity.UpdatedAt)
		if cacheAge < 15*time.Second {
			validCached[activity.Address] = activity
		}
	}

	logrus.Debugf("Найдено %d актуальных записей в кеше для периода %v", len(validCached), period)

	// Получаем всех активных отправителей за период из transactions
	nextPeriod := period.Add(15 * time.Second)

	var allActiveAddresses []string
	err := r.db.Model(&models.Transaction{}).
		Select("DISTINCT \"from\"").
		Where("timestamp >= ? AND timestamp < ?", period, nextPeriod).
		Pluck("\"from\"", &allActiveAddresses).Error

	if err != nil {
		return nil, err
	}

	logrus.Debugf("Всего активных отправителей в период: %d", len(allActiveAddresses))

	// Определяем какие аккаунты нужно пересчитать
	var addressesToCalculate []string
	for _, address := range allActiveAddresses {
		if _, exists := validCached[address]; !exists {
			addressesToCalculate = append(addressesToCalculate, address)
		}
	}

	logrus.Debugf("Нужно пересчитать %d аккаунтов", len(addressesToCalculate))

	// Пересчитываем недостающие аккаунты
	if len(addressesToCalculate) > 0 {
		var results []struct {
			Address          string
			TransactionCount uint32
			VolumeETH        string
			TokenTransfers   uint32
		}

		// Получаем статистику по обычным транзакциям
		err = r.db.Model(&models.Transaction{}).
			Select(`
				"from" as address,
				COUNT(*) as transaction_count,
				COALESCE(SUM(CAST(value AS NUMERIC)), '0')::text as volume_eth
			`).
			Where("\"from\" IN ? AND timestamp >= ? AND timestamp < ?",
				addressesToCalculate, period, nextPeriod).
			Group("\"from\"").
			Scan(&results).Error

		if err != nil {
			return nil, err
		}

		// Для каждого адреса получаем количество ERC20 трансферов
		for i := range results {
			var tokenCount int64
			err = r.db.Model(&models.ERC20Transfer{}).
				Where("\"from\" = ? AND created_at >= ? AND created_at < ?",
					results[i].Address, period, nextPeriod).
				Count(&tokenCount).Error

			if err != nil {
				logrus.Errorf("Ошибка подсчета ERC20 трансферов для %s: %v", results[i].Address, err)
				continue
			}

			results[i].TokenTransfers = uint32(tokenCount)
			results[i].TransactionCount += uint32(tokenCount)
		}

		// Добавляем пересчитанные данные в validCached и сохраняем в кеш
		for _, result := range results {
			activity := models.AccountActivity{
				Address:          result.Address,
				Period:           period,
				TransactionCount: result.TransactionCount,
				VolumeETH:        result.VolumeETH,
				TokenTransfers:   result.TokenTransfers,
			}

			validCached[result.Address] = activity

			// Сохраняем в кеш асинхронно
			go func(act models.AccountActivity) {
				if saveErr := r.SaveAccountActivity(&act); saveErr != nil {
					logrus.Errorf("Не удалось сохранить в кеш %s: %v", act.Address, saveErr)
				}
			}(activity)
		}
	}

	// Преобразуем в slice
	activities := make([]models.AccountActivity, 0, len(validCached))
	for _, activity := range validCached {
		activities = append(activities, activity)
	}

	logrus.Debugf("Возвращаем активность для %d аккаунтов за период %v", len(activities), period)
	return activities, nil
}

// GetAccountActivityHistory получает историю активности аккаунта за несколько периодов
func (r *AccountRepository) GetAccountActivityHistory(address string, fromPeriod, toPeriod time.Time) ([]models.AccountActivity, error) {
	var results []struct {
		Period           time.Time
		TransactionCount uint32
		VolumeETH        string
	}

	// Группируем по 15-секундным периодам
	err := r.db.Model(&models.Transaction{}).
		Select(`
			date_trunc('minute', timestamp) + 
			INTERVAL '15 seconds' * FLOOR(EXTRACT(SECOND FROM timestamp) / 15) as period,
			COUNT(*) as transaction_count,
			COALESCE(SUM(CAST(value AS NUMERIC)), 0)::text as volume_eth
		`).
		Where("\"from\" = ? AND timestamp >= ? AND timestamp < ?", address, fromPeriod, toPeriod).
		Group("period").
		Order("period").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Преобразуем в slice моделей активности
	activities := make([]models.AccountActivity, len(results))
	for i, result := range results {
		activities[i] = models.AccountActivity{
			Address:          address,
			Period:           result.Period,
			TransactionCount: result.TransactionCount,
			VolumeETH:        result.VolumeETH,
			TokenTransfers:   0,
		}
	}

	return activities, nil
}

func (r *AccountRepository) GetActivitiesAfterID(lastID uint) ([]models.AccountActivity, error) {
	var activities []models.AccountActivity
	err := r.db.Where("id > ?", lastID).Order("id ASC").Find(&activities).Error
	return activities, err
}

func (r *AccountRepository) GetActivitiesAfterIDWithLimit(lastID uint, limit int) ([]models.AccountActivity, error) {
	var activities []models.AccountActivity
	err := r.db.Where("id > ?", lastID).Order("id ASC").Limit(limit).Find(&activities).Error
	return activities, err
}
