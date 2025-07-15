package models

import (
	"math/big"
	"time"
)

type AccountActivity struct {
	ID               uint      `gorm:"primarykey"`
	Address          string    `gorm:"uniqueIndex:idx_address_period;not null;type:char(42);check:address != '0x0000000000000000000000000000000000000000' AND address != ''" json:"address"`
	Period           time.Time `gorm:"column:period;uniqueIndex:idx_address_period;not null"` // 15-секундные периоды
	TransactionCount uint32    `gorm:"default:0"`
	VolumeETH        string    `gorm:"default:'0'"`
	TokenTransfers   uint32    `gorm:"default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Добавляем составной уникальный индекс на (address, period)
func (AccountActivity) TableName() string {
	return "account_activities"
}

func (a *AccountActivity) SetVolumeETH(value *big.Int) {
	if value != nil && value.Cmp(big.NewInt(0)) > 0 {
		a.VolumeETH = value.String()
		if a.VolumeETH == "" {
			a.VolumeETH = "0"
		}
	} else {
		a.VolumeETH = "0"
	}
}

func (a *AccountActivity) GetVolumeETH() *big.Int {
	value, ok := big.NewInt(0).SetString(a.VolumeETH, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}

func GetPeriodStart(t time.Time) time.Time {
	return t.Truncate(15 * time.Second)
}
