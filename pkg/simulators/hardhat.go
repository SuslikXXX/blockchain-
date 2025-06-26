package simulators

import (
	"blockchain/pkg/contracts"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

// ERC20 байт-код для деплоя (упрощенная версия OpenZeppelin)
const ERC20Bytecode = "0x608060405234801561001057600080fd5b506040516108003803806108008339818101604052810190610032919061017a565b818160039080519060200190610049929190610051565b505050610233565b82805461005d906101d2565b90600052602060002090601f01602090048101928261007f57600085556100c6565b82601f1061009857805160ff19168380011785556100c6565b828001600101855582156100c6579182015b828111156100c55782518255916020019190600101906100aa565b5b5090506100d391906100d7565b5090565b5b808211156100f05760008160009055506001016100d8565b5090565b6000610107610102846101f8565b6101cf565b90508281526020810184848401111561011f57600080fd5b61012a848285610229565b509392505050565b600082601f83011261014357600080fd5b81516101538482602086016100f4565b91505092915050565b60008151905061016b8161031c565b92915050565b6000819050919050565b60008060006060848603121561019057600080fd5b600084015167ffffffffffffffff8111156101aa57600080fd5b6101b686828701610132565b935050602084015167ffffffffffffffff8111156101d357600080fd5b6101df86828701610132565b92505060406101f08682870161015c565b9150509250925092565b600067ffffffffffffffff82111561021557610214610322565b5b61021e82610369565b9050602081019050919050565b60005b8381101561024757808201518184015260208101905061022c565b83811115610256576000848401525b50505050565b6000600282049050600182168061027457607f821691505b6020821081141561028857610287610293565b5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b61032581610171565b811461033057600080fd5b50565b6105be806103426000396000f3fe"

type HardhatDeployer struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	chainID    *big.Int
}

func NewHardhatDeployer(client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int) *HardhatDeployer {
	return &HardhatDeployer{
		client:     client,
		privateKey: privateKey,
		chainID:    chainID,
	}
}

func (h *HardhatDeployer) DeployERC20(ctx context.Context, name, symbol string, initialSupply *big.Int) (common.Address, error) {
	// Создаем ABI для конструктора
	constructorABI := `[{
		"inputs": [
			{"name": "name", "type": "string"},
			{"name": "symbol", "type": "string"},
			{"name": "initialSupply", "type": "uint256"}
		],
		"stateMutability": "nonpayable",
		"type": "constructor"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(constructorABI))
	if err != nil {
		return common.Address{}, err
	}

	// Подготавливаем параметры конструктора
	input, err := parsedABI.Pack("", name, symbol, initialSupply)
	if err != nil {
		return common.Address{}, err
	}

	// Создаем транзактор
	auth, err := bind.NewKeyedTransactorWithChainID(h.privateKey, h.chainID)
	if err != nil {
		return common.Address{}, err
	}

	auth.Context = ctx
	auth.GasLimit = uint64(3000000) // Увеличиваем лимит газа для деплоя

	// Получаем nonce
	nonce, err := h.client.PendingNonceAt(ctx, crypto.PubkeyToAddress(*h.privateKey.Public().(*ecdsa.PublicKey)))
	if err != nil {
		return common.Address{}, err
	}

	// Получаем gas price
	gasPrice, err := h.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Address{}, err
	}

	// Деплоим контракт
	bytecode := common.FromHex(ERC20Bytecode)
	fullBytecode := append(bytecode, input...)

	tx := types.NewContractCreation(
		nonce,
		big.NewInt(0),
		auth.GasLimit,
		gasPrice,
		fullBytecode,
	)

	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return common.Address{}, err
	}

	err = h.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Address{}, err
	}

	logrus.Infof("Транзакция деплоя отправлена: %s", signedTx.Hash().Hex())

	// Ждем подтверждения
	receipt, err := bind.WaitMined(ctx, h.client, signedTx)
	if err != nil {
		return common.Address{}, err
	}

	if receipt.Status == 0 {
		return common.Address{}, fmt.Errorf("деплой не удался")
	}

	logrus.Infof("ERC20 контракт задеплоен по адресу: %s", receipt.ContractAddress.Hex())
	return receipt.ContractAddress, nil
}

func (h *HardhatDeployer) TestTransfer(ctx context.Context, contractAddr common.Address, to common.Address, amount *big.Int) error {
	erc20, err := contracts.NewERC20Contract(contractAddr, h.client)
	if err != nil {
		return err
	}

	// Выполняем трансфер
	tx, err := erc20.Transfer(ctx, h.privateKey, to, amount, h.chainID)
	if err != nil {
		return err
	}

	logrus.Infof("Транзакция transfer отправлена: %s", tx.Hash().Hex())

	// Ждем подтверждения
	receipt, err := bind.WaitMined(ctx, h.client, tx)
	if err != nil {
		return err
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transfer не удался")
	}

	logrus.Infof("Transfer выполнен успешно. Gas used: %d", receipt.GasUsed)
	return nil
}
