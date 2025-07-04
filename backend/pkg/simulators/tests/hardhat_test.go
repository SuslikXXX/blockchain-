package tests

import (
	"backend/pkg/contracts"
	"backend/pkg/simulators"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

// ExecuteTransfer выполняет тестовый трансфер токенов
func ExecuteTransfer(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, contractAddr common.Address, to common.Address, amount *big.Int) error {
	erc20, err := contracts.NewERC20Contract(contractAddr, client)
	if err != nil {
		return err
	}

	// Создаем TransactOpts
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return err
	}

	// Выполняем трансфер
	tx, err := erc20.Transfer(auth)
	if err != nil {
		return err
	}

	logrus.Infof("Транзакция transfer отправлена: %s", tx.Hash().Hex())

	// Ждем подтверждения
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return err
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transfer не удался")
	}

	logrus.Infof("Transfer выполнен успешно. Gas used: %d", receipt.GasUsed)
	return nil
}

// TestDeployAndTransfer тестирует полный цикл: деплой контракта и трансфер
func TestDeployAndTransfer(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, artifactsPath string) error {
	deployer := simulators.NewHardhatDeployer(client, privateKey, chainID, artifactsPath)

	// Деплоим контракт
	initialSupply := big.NewInt(1000000) // 1 миллион токенов
	contractAddr, err := deployer.DeployERC20(ctx, "TestToken", "TTK", initialSupply)
	if err != nil {
		return fmt.Errorf("ошибка деплоя контракта: %w", err)
	}

	// Создаем тестовый адрес для трансфера
	testPrivateKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("ошибка генерации тестового ключа: %w", err)
	}
	testAddress := crypto.PubkeyToAddress(*testPrivateKey.Public().(*ecdsa.PublicKey))

	// Выполняем трансфер
	transferAmount := big.NewInt(1000) // 1000 токенов
	err = ExecuteTransfer(ctx, client, privateKey, chainID, contractAddr, testAddress, transferAmount)
	if err != nil {
		return fmt.Errorf("ошибка трансфера: %w", err)
	}

	logrus.Infof("Тест деплоя и трансфера завершен успешно")
	return nil
}
