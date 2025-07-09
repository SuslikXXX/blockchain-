package models

import (
	"math/big"
	"time"
)

type AccountStats struct {
	ID                     uint   `gorm:"primarykey"`
	Address                string `gorm:"uniqueIndex;not null"`
	TotalTransactions      uint64 `gorm:"default:0"`
	ERC20Transactions      uint64 `gorm:"default:0"`
	LastActivityTime       *time.Time
	FirstActivityTime      *time.Time
	TotalVolumeETH         string `gorm:"default:'0'"`
	UniqueTokensCount      uint32 `gorm:"default:0"`
	LastPeriodTransactions uint32 `gorm:"default:0"`   // Количество транзакций за последние 15 секунд
	LastPeriodVolumeETH    string `gorm:"default:'0'"` // Объем за последние 15 секунд
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (a *AccountStats) SetTotalVolumeETH(value *big.Int) {
	if value != nil {
		a.TotalVolumeETH = value.String()
	} else {
		a.TotalVolumeETH = "0"
	}
}

func (a *AccountStats) GetTotalVolumeETH() *big.Int {
	value, ok := big.NewInt(0).SetString(a.TotalVolumeETH, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}

// SetLastPeriodVolumeETH устанавливает объем за последние 15 секунд
func (a *AccountStats) SetLastPeriodVolumeETH(value *big.Int) {
	if value != nil {
		a.LastPeriodVolumeETH = value.String()
	} else {
		a.LastPeriodVolumeETH = "0"
	}
}

// GetLastPeriodVolumeETH возвращает объем за последние 15 секунд
func (a *AccountStats) GetLastPeriodVolumeETH() *big.Int {
	value, ok := big.NewInt(0).SetString(a.LastPeriodVolumeETH, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}
