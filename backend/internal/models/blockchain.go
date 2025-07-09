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

// GetValue возвращает значение Value как big.Int
func (t *Transaction) GetValue() *big.Int {
	value, ok := big.NewInt(0).SetString(t.Value, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}

// SetValue устанавливает значение Value из big.Int
func (t *Transaction) SetValue(value *big.Int) {
	if value != nil {
		t.Value = value.String()
	} else {
		t.Value = "0"
	}
}

// GetGasPrice возвращает цену газа как big.Int
func (t *Transaction) GetGasPrice() *big.Int {
	price, ok := big.NewInt(0).SetString(t.GasPrice, 10)
	if !ok {
		return big.NewInt(0)
	}
	return price
}

// SetGasPrice устанавливает цену газа из big.Int
func (t *Transaction) SetGasPrice(price *big.Int) {
	if price != nil {
		t.GasPrice = price.String()
	} else {
		t.GasPrice = "0"
	}
}

// GetTotalGasCost возвращает общую стоимость газа (GasUsed * GasPrice)
func (t *Transaction) GetTotalGasCost() *big.Int {
	gasPrice := t.GetGasPrice()
	gasUsed := big.NewInt(int64(t.GasUsed))
	return gasPrice.Mul(gasPrice, gasUsed)
}

// IsSuccessful проверяет успешность транзакции
func (t *Transaction) IsSuccessful() bool {
	return t.Status == 1
}

// GetTotalCost возвращает общую стоимость транзакции (Value + GasCost)
func (t *Transaction) GetTotalCost() *big.Int {
	value := t.GetValue()
	gasCost := t.GetTotalGasCost()
	return value.Add(value, gasCost)
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

// GetValue возвращает значение Value как big.Int
func (e *ERC20Transfer) GetValue() *big.Int {
	value, ok := big.NewInt(0).SetString(e.Value, 10)
	if !ok {
		return big.NewInt(0)
	}
	return value
}

// SetValue устанавливает значение Value из big.Int
func (e *ERC20Transfer) SetValue(value *big.Int) {
	if value != nil {
		e.Value = value.String()
	} else {
		e.Value = "0"
	}
}

// AnalyzerState - модель для сохранения состояния анализатора
type AnalyzerState struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	LastProcessedBlock uint64    `gorm:"not null" json:"last_processed_block"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
