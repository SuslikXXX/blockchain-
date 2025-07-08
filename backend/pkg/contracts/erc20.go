package contracts

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// ERC20Contract представляет контракт ERC20
type ERC20Contract struct {
	address common.Address
	abi     abi.ABI
	*bind.BoundContract
}

// NewERC20Contract создает новый экземпляр контракта ERC20
func NewERC20Contract(address common.Address, backend bind.ContractBackend) (*ERC20Contract, error) {
	// Читаем ABI из файла
	abiPath := filepath.Join("..", "blockchain", "artifacts", "Token.abi")
	abiData, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, err
	}

	parsed, err := abi.JSON(strings.NewReader(string(abiData)))
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				event := new(Transfer)
				if err := e.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
