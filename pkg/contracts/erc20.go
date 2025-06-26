package contracts

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ERC20ABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [{"name": "", "type": "string"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "symbol",
		"outputs": [{"name": "", "type": "string"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [{"name": "", "type": "uint8"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "totalSupply",
		"outputs": [{"name": "", "type": "uint256"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "from", "type": "address"},
			{"indexed": true, "name": "to", "type": "address"},
			{"indexed": false, "name": "value", "type": "uint256"}
		],
		"name": "Transfer",
		"type": "event"
	}
]`

type ERC20Contract struct {
	contract *bind.BoundContract
	abi      abi.ABI
	address  common.Address
	client   *ethclient.Client
}

func NewERC20Contract(address common.Address, client *ethclient.Client) (*ERC20Contract, error) {
	parsedABI, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return nil, err
	}

	contract := bind.NewBoundContract(address, parsedABI, client, client, client)

	return &ERC20Contract{
		contract: contract,
		abi:      parsedABI,
		address:  address,
		client:   client,
	}, nil
}

func (e *ERC20Contract) Name(ctx context.Context) (string, error) {
	var result []interface{}
	err := e.contract.Call(&bind.CallOpts{Context: ctx}, &result, "name")
	if err != nil {
		return "", err
	}
	return result[0].(string), nil
}

func (e *ERC20Contract) Symbol(ctx context.Context) (string, error) {
	var result []interface{}
	err := e.contract.Call(&bind.CallOpts{Context: ctx}, &result, "symbol")
	if err != nil {
		return "", err
	}
	return result[0].(string), nil
}

func (e *ERC20Contract) BalanceOf(ctx context.Context, address common.Address) (*big.Int, error) {
	var result []interface{}
	err := e.contract.Call(&bind.CallOpts{Context: ctx}, &result, "balanceOf", address)
	if err != nil {
		return nil, err
	}
	return result[0].(*big.Int), nil
}

func (e *ERC20Contract) Transfer(ctx context.Context, privateKey *ecdsa.PrivateKey, to common.Address, amount *big.Int, chainID *big.Int) (*types.Transaction, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, err
	}

	auth.Context = ctx

	tx, err := e.contract.Transact(auth, "transfer", to, amount)
	return tx, err
}

func (e *ERC20Contract) GetAddress() common.Address {
	return e.address
}

func (e *ERC20Contract) GetABI() abi.ABI {
	return e.abi
}
