package models

import (
	"math/big"
	"time"
)

type TokenBalance struct {
	ID           uint   `gorm:"primarykey"`
	Address      string `gorm:"index;not null"`
	TokenAddress string `gorm:"index;not null"`
	Balance      string `gorm:"default:'0'"`
	LastUpdate   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (t *TokenBalance) SetBalance(value *big.Int) {
	if value != nil {
		t.Balance = value.String()
	} else {
		t.Balance = "0"
	}
}

func (t *TokenBalance) GetBalance() *big.Int {
	value, ok := big.NewInt(0).SetString(t.Balance, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}

// Составной индекс для быстрого поиска по адресу и токену
func (TokenBalance) TableName() string {
	return "token_balances"
}
