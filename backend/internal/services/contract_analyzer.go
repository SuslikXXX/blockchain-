package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/pkg/contracts"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	eth "github.com/ethereum/go-ethereum"

	"backend/pkg/ethereum"

	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ContractAnalyzer struct {
	ethClient    *ethereum.Client
	accountRepo  *repositories.AccountRepository
	contracts    map[common.Address]*contracts.ERC20Contract
	tokenAddress common.Address // Адрес нашего токена
}

func NewContractAnalyzer(ethClient *ethereum.Client, accountRepo *repositories.AccountRepository, tokenAddress common.Address) *ContractAnalyzer {
	return &ContractAnalyzer{
		ethClient:    ethClient,
		accountRepo:  accountRepo,
		contracts:    make(map[common.Address]*contracts.ERC20Contract),
		tokenAddress: tokenAddress,
	}
}

// GetOrCreateContract получает или создает инстанс контракта по адресу
func (ca *ContractAnalyzer) GetOrCreateContract(address common.Address) (*contracts.ERC20Contract, error) {
	if contract, exists := ca.contracts[address]; exists {
		return contract, nil
	}

	contract, err := contracts.NewERC20Contract(address, ca.ethClient.GetClient())
	if err != nil {
		return nil, fmt.Errorf("ошибка создания контракта: %v", err)
	}

	ca.contracts[address] = contract
	return contract, nil
}

// AnalyzeContractTransactions анализирует транзакции конкретного контракта
func (ca *ContractAnalyzer) AnalyzeContractTransactions(ctx context.Context, fromBlock, toBlock uint64) error {
	contract, err := ca.GetOrCreateContract(ca.tokenAddress)
	if err != nil {
		return err
	}

	// Создаем фильтр для событий Transfer
	filterQuery := eth.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{ca.tokenAddress},
		Topics: [][]common.Hash{{
			// Transfer event signature
			common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
		}},
	}

	// Получаем логи
	logs, err := ca.ethClient.GetClient().FilterLogs(ctx, filterQuery)
	if err != nil {
		return fmt.Errorf("ошибка получения логов: %v", err)
	}

	logrus.Infof("Найдено %d событий для контракта %s", len(logs), ca.tokenAddress.Hex())

	// Обрабатываем каждый лог
	for _, log := range logs {
		if err := ca.processTransferLog(ctx, contract, &log); err != nil {
			logrus.Errorf("Ошибка обработки лога %s: %v", log.TxHash.Hex(), err)
			continue
		}
	}

	return nil
}

// processTransferLog обрабатывает отдельное событие Transfer
func (ca *ContractAnalyzer) processTransferLog(ctx context.Context, contract *contracts.ERC20Contract, log *types.Log) error {
	// Проверяем, что это наш токен
	if log.Address != ca.tokenAddress {
		return nil // Пропускаем другие токены
	}

	// Проверяем, что это событие Transfer
	if len(log.Topics) != 3 {
		return nil // Не Transfer событие
	}

	// Извлекаем данные из лога
	from := common.BytesToAddress(log.Topics[1].Bytes())
	to := common.BytesToAddress(log.Topics[2].Bytes())
	value := big.NewInt(0)
	if len(log.Data) > 0 {
		value.SetBytes(log.Data)
	}

	// Получаем информацию о блоке для timestamp
	block, err := ca.ethClient.GetClient().BlockByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
	if err != nil {
		return fmt.Errorf("ошибка получения блока: %v", err)
	}

	// Создаем запись о трансфере
	transfer := &models.ERC20Transfer{
		TransactionHash: log.TxHash.Hex(),
		ContractAddress: log.Address.Hex(),
		From:            from.Hex(),
		To:              to.Hex(),
		Value:           value.String(),
		BlockNumber:     log.BlockNumber,
		CreatedAt:       time.Unix(int64(block.Time()), 0),
	}

	// Сохраняем в БД
	if err := repositories.DB.Create(transfer).Error; err != nil {
		// Если ошибка дублирования, просто игнорируем
		if err.Error() == "duplicate key value violates unique constraint" {
			logrus.Debugf("ERC20 трансфер уже существует, пропускаем")
			return nil
		}
		return fmt.Errorf("ошибка сохранения трансфера: %v", err)
	}

	// Обновляем балансы токенов
	if err := ca.updateTokenBalances(transfer); err != nil {
		logrus.Errorf("Ошибка обновления балансов токенов: %v", err)
	}

	// Обновляем статистику только для отправителя
	if err := ca.updateAccountStats(transfer.From, transfer, true); err != nil {
		logrus.Errorf("Ошибка обновления статистики отправителя %s: %v", transfer.From, err)
	}

	logrus.Debugf("Сохранен трансфер: %s от %s к %s, значение %s",
		transfer.TransactionHash, transfer.From, transfer.To, transfer.Value)

	return nil
}

// updateTokenBalances обновляет балансы токенов для участников трансфера
func (ca *ContractAnalyzer) updateTokenBalances(transfer *models.ERC20Transfer) error {
	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		// Парсим значение
		value, ok := big.NewInt(0).SetString(transfer.Value, 10)
		if !ok {
			return fmt.Errorf("ошибка парсинга значения трансфера: %s", transfer.Value)
		}

		// Пропускаем нулевой адрес
		zeroAddress := "0x0000000000000000000000000000000000000000"

		// Обновляем баланс отправителя
		if transfer.From != zeroAddress {
			fromBalance, err := ca.accountRepo.GetTokenBalance(transfer.From, transfer.ContractAddress)
			if err != nil {
				return err
			}
			currentBalance := fromBalance.GetBalance()
			newBalance := new(big.Int).Sub(currentBalance, value)
			if newBalance.Sign() < 0 {
				newBalance = big.NewInt(0)
			}
			fromBalance.SetBalance(newBalance)
			if err := ca.accountRepo.UpdateTokenBalance(fromBalance); err != nil {
				return err
			}
		}

		// Обновляем баланс получателя
		if transfer.To != zeroAddress {
			toBalance, err := ca.accountRepo.GetTokenBalance(transfer.To, transfer.ContractAddress)
			if err != nil {
				return err
			}
			currentBalance := toBalance.GetBalance()
			newBalance := new(big.Int).Add(currentBalance, value)
			toBalance.SetBalance(newBalance)
			if err := ca.accountRepo.UpdateTokenBalance(toBalance); err != nil {
				return err
			}
		}

		return nil
	})
}

// updateAccountStats обновляет статистику аккаунта
func (ca *ContractAnalyzer) updateAccountStats(address string, transfer *models.ERC20Transfer, isSender bool) error {
	// Пропускаем нулевой адрес
	if address == "0x0000000000000000000000000000000000000000" {
		return nil
	}

	stats, err := ca.accountRepo.GetOrCreateAccountStats(address)
	if err != nil {
		return err
	}

	// Увеличиваем счетчик ERC20 транзакций
	stats.ERC20Transactions++

	// Обновляем время активности
	now := transfer.CreatedAt
	if stats.LastActivityTime == nil || now.After(*stats.LastActivityTime) {
		stats.LastActivityTime = &now
	}
	if stats.FirstActivityTime == nil || now.Before(*stats.FirstActivityTime) {
		stats.FirstActivityTime = &now
	}

	// Сохраняем обновленную статистику
	if err := ca.accountRepo.UpdateAccountStats(stats); err != nil {
		return err
	}

	return nil
}

// GetContractTransfers получает все трансферы контракта
func (ca *ContractAnalyzer) GetContractTransfers(contractAddress string, limit, offset int) ([]models.ERC20Transfer, error) {
	var transfers []models.ERC20Transfer

	result := repositories.DB.
		Where("contract_address = ?", contractAddress).
		Order("block_number DESC").
		Limit(limit).
		Offset(offset).
		Find(&transfers)

	if result.Error != nil {
		return nil, fmt.Errorf("ошибка получения трансферов: %v", result.Error)
	}

	return transfers, nil
}

// GetContractStats получает статистику по контракту
func (ca *ContractAnalyzer) GetContractStats(contractAddress string) (map[string]interface{}, error) {
	var stats struct {
		TotalTransfers  int64
		UniqueAddresses int64
		TotalVolume     string
	}

	// Получаем общее количество трансферов
	if err := repositories.DB.Model(&models.ERC20Transfer{}).
		Where("contract_address = ?", contractAddress).
		Count(&stats.TotalTransfers).Error; err != nil {
		return nil, err
	}

	// Используем DISTINCT ON для исключения дублей адресов
	var uniqueAddresses []string
	if err := repositories.DB.Model(&models.ERC20Transfer{}).
		Where("contract_address = ? AND \"from\" != '0x0000000000000000000000000000000000000000'", contractAddress).
		Distinct("\"from\"").
		Pluck("\"from\"", &uniqueAddresses).Error; err != nil {
		return nil, err
	}

	addressMap := make(map[string]struct{})
	for _, addr := range uniqueAddresses {
		addressMap[addr] = struct{}{}
	}

	if err := repositories.DB.Model(&models.ERC20Transfer{}).
		Where("contract_address = ? AND \"to\" != '0x0000000000000000000000000000000000000000'", contractAddress).
		Distinct("\"to\"").
		Pluck("\"to\"", &uniqueAddresses).Error; err != nil {
		return nil, err
	}

	for _, addr := range uniqueAddresses {
		addressMap[addr] = struct{}{}
	}

	stats.UniqueAddresses = int64(len(addressMap))

	// Получаем общий объем трансферов
	if err := repositories.DB.Model(&models.ERC20Transfer{}).
		Where("contract_address = ?", contractAddress).
		Select("COALESCE(SUM(CAST(value AS NUMERIC)), 0)").
		Row().Scan(&stats.TotalVolume); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_transfers":  stats.TotalTransfers,
		"unique_addresses": stats.UniqueAddresses,
		"total_volume":     stats.TotalVolume,
	}, nil
}

// GetContractTransactions получает все транзакции контракта
func (ca *ContractAnalyzer) GetContractTransactions(ctx context.Context, contractAddress common.Address, fromBlock, toBlock uint64) ([]models.ContractTransaction, error) {
	// Создаем фильтр для транзакций
	filterQuery := eth.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{contractAddress},
	}

	// Получаем логи
	logs, err := ca.ethClient.GetClient().FilterLogs(ctx, filterQuery)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения логов: %v", err)
	}

	var transactions []models.ContractTransaction
	processedTxs := make(map[string]bool)

	// Обрабатываем каждый лог
	for _, log := range logs {
		// Пропускаем уже обработанные транзакции
		if processedTxs[log.TxHash.Hex()] {
			continue
		}

		// Получаем транзакцию
		tx, isPending, err := ca.ethClient.GetClient().TransactionByHash(ctx, log.TxHash)
		if err != nil {
			logrus.Errorf("Ошибка получения транзакции %s: %v", log.TxHash.Hex(), err)
			continue
		}
		if isPending {
			continue
		}

		// Получаем receipt для статуса и использованного газа
		receipt, err := ca.ethClient.GetClient().TransactionReceipt(ctx, log.TxHash)
		if err != nil {
			logrus.Errorf("Ошибка получения receipt для %s: %v", log.TxHash.Hex(), err)
			continue
		}

		// Получаем блок для timestamp
		block, err := ca.ethClient.GetClient().BlockByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
		if err != nil {
			logrus.Errorf("Ошибка получения блока для %s: %v", log.TxHash.Hex(), err)
			continue
		}

		// Получаем отправителя
		from, err := types.Sender(types.NewLondonSigner(ca.ethClient.GetChainID()), tx)
		if err != nil {
			logrus.Errorf("Ошибка получения отправителя для %s: %v", log.TxHash.Hex(), err)
			continue
		}

		// Определяем метод из input данных
		method := "unknown"
		if len(tx.Data()) >= 4 {
			methodID := hex.EncodeToString(tx.Data()[:4])
			// Здесь можно добавить маппинг известных методов
			switch methodID {
			case "a9059cbb":
				method = "transfer"
			case "23b872dd":
				method = "transferFrom"
			case "095ea7b3":
				method = "approve"
			}
		}

		// Создаем запись транзакции
		transaction := models.ContractTransaction{
			ContractAddress: contractAddress.Hex(),
			TransactionHash: log.TxHash.Hex(),
			From:            from.Hex(),
			To:              tx.To().Hex(),
			Value:           tx.Value().String(),
			Method:          method,
			BlockNumber:     log.BlockNumber,
			Timestamp:       time.Unix(int64(block.Time()), 0),
			GasUsed:         receipt.GasUsed,
			Status:          receipt.Status,
		}

		transactions = append(transactions, transaction)
		processedTxs[log.TxHash.Hex()] = true
	}

	// Сохраняем транзакции в БД
	for _, tx := range transactions {
		if err := repositories.DB.Create(&tx).Error; err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				logrus.Errorf("Ошибка сохранения транзакции %s: %v", tx.TransactionHash, err)
			}
		}
	}

	return transactions, nil
}

// GetContractTransactionsByMethod получает транзакции контракта по конкретному методу
func (ca *ContractAnalyzer) GetContractTransactionsByMethod(contractAddress string, method string, limit, offset int) ([]models.ContractTransaction, error) {
	var transactions []models.ContractTransaction

	result := repositories.DB.
		Where("contract_address = ? AND method = ?", contractAddress, method).
		Order("block_number DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions)

	if result.Error != nil {
		return nil, fmt.Errorf("ошибка получения транзакций: %v", result.Error)
	}

	return transactions, nil
}

// GetContractTransactionCount получает количество транзакций контракта
func (ca *ContractAnalyzer) GetContractTransactionCount(contractAddress string) (int64, error) {
	var count int64
	result := repositories.DB.Model(&models.ContractTransaction{}).
		Where("contract_address = ?", contractAddress).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("ошибка получения количества транзакций: %v", result.Error)
	}

	return count, nil
}
