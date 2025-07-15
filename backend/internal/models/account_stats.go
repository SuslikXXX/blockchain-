package models

import (
	"math/big"
	"time"
)

type AccountStats struct {
	ID                uint   `gorm:"primarykey"`
	Address           string `gorm:"uniqueIndex;not null"`
	TotalTransactions uint64 `gorm:"default:0"`
	ERC20Transactions uint64 `gorm:"default:0"`
	LastActivityTime  *time.Time
	FirstActivityTime *time.Time
	TotalVolumeETH    string `gorm:"default:'0'"`
	ETHBalance        string `gorm:"default:'0'"` // Баланс ETH
	UniqueTokensCount uint32 `gorm:"default:0"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
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

// Методы для работы с балансом ETH
func (a *AccountStats) SetETHBalance(value *big.Int) {
	if value != nil {
		a.ETHBalance = value.String()
	} else {
		a.ETHBalance = "0"
	}
}

func (a *AccountStats) GetETHBalance() *big.Int {
	value, ok := big.NewInt(0).SetString(a.ETHBalance, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}
