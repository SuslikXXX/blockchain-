package simulators

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

type HardhatDeployer struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	chainID    *big.Int
	loader     *ContractLoader
}

func NewHardhatDeployer(client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, artifactsPath string) *HardhatDeployer {
	return &HardhatDeployer{
		client:     client,
		privateKey: privateKey,
		chainID:    chainID,
		loader:     NewContractLoader(artifactsPath),
	}
}

func (h *HardhatDeployer) DeployERC20(ctx context.Context, name, symbol string, initialSupply *big.Int) (common.Address, error) {
	// Загружаем артефакт контракта Token
	artifact, err := h.loader.LoadContract("Token")
	if err != nil {
		return common.Address{}, fmt.Errorf("не удалось загрузить артефакт Token: %w", err)
	}

	// Получаем ABI конструктора
	constructorABI, err := artifact.GetConstructorABI()
	if err != nil {
		return common.Address{}, fmt.Errorf("не удалось получить ABI конструктора: %w", err)
	}

	// Подготавливаем параметры конструктора
	input, err := constructorABI.Pack("", name, symbol, initialSupply)
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

	// Получаем байткод контракта
	bytecode, err := artifact.GetBytecode()
	if err != nil {
		return common.Address{}, fmt.Errorf("не удалось получить байткод: %w", err)
	}

	// Деплоим контракт
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
