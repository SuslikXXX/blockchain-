package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// ContractArtifact представляет структуру JSON файла с артефактами контракта
type ContractArtifact struct {
	Abi json.RawMessage `json:"abi"`
}

// ERC20Contract представляет контракт ERC20
type ERC20Contract struct {
	address common.Address
	abi     abi.ABI
	*bind.BoundContract
}

// NewERC20Contract создает новый экземпляр контракта ERC20
func NewERC20Contract(address common.Address, backend bind.ContractBackend) (*ERC20Contract, error) {
	// Пытаемся найти ABI файл в разных местах
	abiPaths := []string{
		"blockchain/artifacts/contracts/Token.sol/AnalyzerToken.json",
		"../blockchain/artifacts/contracts/Token.sol/AnalyzerToken.json",
		"../../blockchain/artifacts/contracts/Token.sol/AnalyzerToken.json",
	}

	var artifact ContractArtifact
	var err error

	for _, path := range abiPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := json.Unmarshal(data, &artifact); err == nil {
				break
			}
		}
	}

	if artifact.Abi == nil {
		return nil, fmt.Errorf("не удалось найти или прочитать файл ABI")
	}

	parsed, err := abi.JSON(strings.NewReader(string(artifact.Abi)))
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга ABI: %v", err)
	}

	contract := bind.NewBoundContract(address, parsed, backend, backend, backend)

	return &ERC20Contract{
		address:       address,
		abi:           parsed,
		BoundContract: contract,
	}, nil
}

// GetABI возвращает ABI контракта
func (e *ERC20Contract) GetABI() abi.ABI {
	return e.abi
}

// GetAddress возвращает адрес контракта
func (e *ERC20Contract) GetAddress() common.Address {
	return e.address
}

// BalanceOf возвращает баланс токенов для указанного адреса
func (e *ERC20Contract) BalanceOf(ctx context.Context, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := e.Call(nil, &out, "balanceOf", account)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения баланса: %v", err)
	}
	return out[0].(*big.Int), nil
}

// TotalSupply возвращает общее количество токенов
func (e *ERC20Contract) TotalSupply(ctx context.Context) (*big.Int, error) {
	var out []interface{}
	err := e.Call(nil, &out, "totalSupply")
	if err != nil {
		return nil, fmt.Errorf("ошибка получения total supply: %v", err)
	}
	return out[0].(*big.Int), nil
}

// Decimals возвращает количество десятичных знаков токена
func (e *ERC20Contract) Decimals(ctx context.Context) (uint8, error) {
	var out []interface{}
	err := e.Call(nil, &out, "decimals")
	if err != nil {
		return 0, fmt.Errorf("ошибка получения decimals: %v", err)
	}
	return out[0].(uint8), nil
}

// Symbol возвращает символ токена
func (e *ERC20Contract) Symbol(ctx context.Context) (string, error) {
	var out []interface{}
	err := e.Call(nil, &out, "symbol")
	if err != nil {
		return "", fmt.Errorf("ошибка получения symbol: %v", err)
	}
	return out[0].(string), nil
}

// Name возвращает имя токена
func (e *ERC20Contract) Name(ctx context.Context) (string, error) {
	var out []interface{}
	err := e.Call(nil, &out, "name")
	if err != nil {
		return "", fmt.Errorf("ошибка получения name: %v", err)
	}
	return out[0].(string), nil
}

// Transfer представляет событие Transfer
type Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log
}

// WatchTransfer создает подписку на события Transfer
func (e *ERC20Contract) WatchTransfer(opts *bind.WatchOpts, sink chan<- *Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := e.BoundContract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, fmt.Errorf("ошибка подписки на события: %v", err)
	}

	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				event := new(Transfer)
				if err := e.UnpackLog(event, "Transfer", log); err != nil {
					return fmt.Errorf("ошибка распаковки события: %v", err)
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return fmt.Errorf("ошибка в подписке: %v", err)
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return fmt.Errorf("ошибка в подписке: %v", err)
			case <-quit:
				return nil
			}
		}
	}), nil
}
