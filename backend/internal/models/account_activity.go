package models

import (
	"math/big"
	"time"
)

type AccountActivity struct {
	ID               uint      `gorm:"primarykey"`
	Address          string    `gorm:"index;not null"`
	Period           time.Time `gorm:"column:period;index;not null"` // 15-секундные периоды
	TransactionCount uint32    `gorm:"default:0"`
	VolumeETH        string    `gorm:"default:'0'"`
	TokenTransfers   uint32    `gorm:"default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (a *AccountActivity) SetVolumeETH(value *big.Int) {
	if value != nil {
		a.VolumeETH = value.String()
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
