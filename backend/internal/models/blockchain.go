package models

import (
	"math/big"
	"time"
)

// Transaction - оптимизированная модель для транзакций Ethereum
type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Hash        string    `gorm:"uniqueIndex;not null;type:char(66)" json:"hash"` // 0x + 64 hex chars
	BlockNumber uint64    `gorm:"not null;index:idx_transactions_block_time" json:"block_number"`
	From        string    `gorm:"not null;index;type:char(42)" json:"from"` // 0x + 40 hex chars
	To          string    `gorm:"not null;index;type:char(42)" json:"to"`   // 0x + 40 hex chars
	Value       string    `gorm:"not null;type:numeric" json:"value"`       // Используем numeric для больших чисел
	GasUsed     uint64    `gorm:"not null" json:"gas_used"`
	GasPrice    string    `gorm:"not null;type:numeric" json:"gas_price"`                      // numeric для точности
	Status      uint64    `gorm:"not null;index" json:"status"`                                // Индекс для фильтрации успешных/неуспешных
	Timestamp   time.Time `gorm:"not null;index:idx_transactions_block_time" json:"timestamp"` // Составной индекс с BlockNumber
	CreatedAt   time.Time `gorm:"index" json:"created_at"`                                     // Индекс для сортировки по времени создания
	UpdatedAt   time.Time `json:"updated_at"`
}

// ERC20Transfer - оптимизированная модель для ERC20 трансферов
type ERC20Transfer struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TransactionHash string    `gorm:"not null;index;type:char(66)" json:"transaction_hash"`
	ContractAddress string    `gorm:"not null;index;type:char(42)" json:"contract_address"`
	From            string    `gorm:"not null;index;type:char(42)" json:"from"`
	To              string    `gorm:"not null;index;type:char(42)" json:"to"`
	Value           string    `gorm:"not null;type:numeric" json:"value"` // numeric для точности
	BlockNumber     uint64    `gorm:"not null;index:idx_erc20_block_time" json:"block_number"`
	LogIndex        uint      `gorm:"not null" json:"log_index"`
	CreatedAt       time.Time `gorm:"index:idx_erc20_block_time" json:"created_at"` // Составной индекс с BlockNumber
}

// Методы для работы с big.Int
func (t *Transaction) SetValue(value *big.Int) {
	t.Value = value.String()
}

func (t *Transaction) GetValue() *big.Int {
	value := new(big.Int)
	value.SetString(t.Value, 10)
	return value
}

func (t *Transaction) SetGasPrice(value *big.Int) {
	t.GasPrice = value.String()
}

func (t *Transaction) GetGasPrice() *big.Int {
	value := new(big.Int)
	value.SetString(t.GasPrice, 10)
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
