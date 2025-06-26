package models

import (
	"math/big"
	"time"
)

type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Hash        string    `gorm:"uniqueIndex;not null" json:"hash"`
	BlockNumber uint64    `gorm:"not null" json:"block_number"`
	From        string    `gorm:"not null" json:"from"`
	To          string    `gorm:"not null" json:"to"`
	Value       string    `gorm:"not null" json:"value"` // Храним как строку для больших чисел
	GasUsed     uint64    `gorm:"not null" json:"gas_used"`
	GasPrice    string    `gorm:"not null" json:"gas_price"`
	Status      uint64    `gorm:"not null" json:"status"`
	Timestamp   time.Time `gorm:"not null" json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ERC20Transfer struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TransactionHash string    `gorm:"not null;index" json:"transaction_hash"`
	ContractAddress string    `gorm:"not null;index" json:"contract_address"`
	From            string    `gorm:"not null;index" json:"from"`
	To              string    `gorm:"not null;index" json:"to"`
	Value           string    `gorm:"not null" json:"value"`
	BlockNumber     uint64    `gorm:"not null;index" json:"block_number"`
	LogIndex        uint      `gorm:"not null" json:"log_index"`
	CreatedAt       time.Time `json:"created_at"`
}

func (t *Transaction) SetValue(value *big.Int) {
	t.Value = value.String()
}

func (t *Transaction) GetValue() *big.Int {
	value := new(big.Int)
	value.SetString(t.Value, 10)
	return value
}

func (e *ERC20Transfer) SetValue(value *big.Int) {
	e.Value = value.String()
}

func (e *ERC20Transfer) GetValue() *big.Int {
	value := new(big.Int)
	value.SetString(e.Value, 10)
	return value
}
