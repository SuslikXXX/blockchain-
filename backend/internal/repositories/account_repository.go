package repositories

import (
	"backend/internal/models"
	"backend/internal/utils"
	"math/big"
	"time"

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
				Address:             address,
				TotalVolumeETH:      "0",
				LastPeriodVolumeETH: "0", // Объем за последние 15 секунд
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

func (r *AccountRepository) GetTransactionCountForPeriod(address string, period time.Time) (uint32, error) {
	nextPeriod := period.Add(15 * time.Second)
	var count int64
	err := r.db.Model(&models.Transaction{}).
		Where("(\"from\" = ? OR \"to\" = ?) AND timestamp >= ? AND timestamp < ?",
			address, address, period, nextPeriod).
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

func (r *AccountRepository) GetVolumeETHForPeriod(address string, period time.Time) (*big.Int, error) {
	nextPeriod := period.Add(15 * time.Second)
	var transactions []models.Transaction
	err := r.db.Where("\"from\" = ? AND timestamp >= ? AND timestamp < ?",
		address, period, nextPeriod).Find(&transactions).Error
	if err != nil {
		return big.NewInt(0), err
	}

	total := big.NewInt(0)
	for _, tx := range transactions {
		total = total.Add(total, tx.GetValue())
	}

	return total, nil
}

func (r *AccountRepository) GetActiveAccountsLastPeriod() ([]string, error) {
	var addresses []string

	// ИСПРАВЛЕНО: используем утилитарную функцию для расчета периода
	currentPeriodStart := utils.GetCurrentPeriodStart()

	err := r.db.Model(&models.Transaction{}).
		Select("DISTINCT \"from\"").
		Where("timestamp >= ?", currentPeriodStart).
		Pluck("\"from\"", &addresses).Error

	return addresses, err
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

func (r *AccountRepository) SaveAccountActivity(activity *models.AccountActivity) error {
	// Используем ON CONFLICT для обновления существующей записи (поле period хранит 15-секундные периоды)
	return r.db.Where("address = ? AND period = ?", activity.Address, activity.Period).
		FirstOrCreate(activity).Error
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
