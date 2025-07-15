package services

import (
	"backend/configs"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/pkg/ethereum"
	"backend/pkg/listeners"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	maxBlocksPerBatch = 1000 // Максимальное количество блоков для обработки за раз
)

type Analyzer struct {
	ethClient          *ethereum.Client
	eventListener      *listeners.EventListener
	accountAnalyzer    *AccountAnalyzer
	activityCalculator *ActivityCalculator
	contractAnalyzer   *ContractAnalyzer
	notifier           *Notifier
	config             *configs.Config
	tokenAddress       common.Address
}

func NewAnalyzer(cfg *configs.Config) (*Analyzer, error) {
	ethClient, err := ethereum.NewClient(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	// Получаем адрес токена из конфигурации
	tokenAddress := common.HexToAddress(cfg.Ethereum.ContractAddress)

	// Создаем репозиторий
	accountRepo := repositories.NewAccountRepository()

	// Создаем калькулятор активности
	activityCalculator := NewActivityCalculator(accountRepo, ethClient, tokenAddress)

	// Создаем анализатор контрактов
	contractAnalyzer := NewContractAnalyzer(ethClient, accountRepo, tokenAddress)

	// Создаем нотификатор
	notifier, err := NewNotifier()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания нотификатора: %v", err)
	}

	return &Analyzer{
		ethClient:          ethClient,
		accountAnalyzer:    NewAccountAnalyzer(accountRepo, ethClient, tokenAddress),
		activityCalculator: activityCalculator,
		contractAnalyzer:   contractAnalyzer,
		notifier:           notifier,
		config:             cfg,
		tokenAddress:       tokenAddress,
	}, nil
}

func (a *Analyzer) Start(ctx context.Context) error {
	logrus.Info("Запуск анализатора блокчейн активности...")

	// Создаем event listener для указанного контракта
	eventListener, err := listeners.NewEventListener(a.ethClient.GetClient(), a.tokenAddress)
	if err != nil {
		return err
	}
	a.eventListener = eventListener

	// Запускаем прослушивание событий
	if err := a.eventListener.StartListening(ctx); err != nil {
		return err
	}

	// Запускаем мониторинг транзакций
	go a.startTransactionMonitoring(ctx)

	// Запускаем обработку активности аккаунтов
	go a.startActivityProcessing(ctx)

	// Запускаем нотификатор
	if err := a.notifier.Start(ctx); err != nil {
		return fmt.Errorf("ошибка запуска нотификатора: %v", err)
	}

	logrus.Info("Анализатор успешно запущен")
	return nil
}

func (a *Analyzer) startTransactionMonitoring(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Обрабатываем первый раз сразу при запуске
	if err := a.processBlockRange(ctx); err != nil {
		logrus.Errorf("Ошибка обработки диапазона блоков: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Остановка мониторинга транзакций")
			return
		case <-ticker.C:
			// Обрабатываем блоки каждую секунду
			if err := a.processBlockRange(ctx); err != nil {
				logrus.Errorf("Ошибка обработки диапазона блоков: %v", err)
			}
		}
	}
}

func (a *Analyzer) startActivityProcessing(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Остановка обработки активности аккаунтов")
			return
		case <-ticker.C:
			if err := a.activityCalculator.ProcessAccountActivities(); err != nil {
				logrus.Errorf("Ошибка обработки активности аккаунтов: %v", err)
			}
		}
	}
}

func (a *Analyzer) processBlockRange(ctx context.Context) error {
	client := a.ethClient.GetClient()

	// Получаем текущий блок
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	currentBlock := header.Number.Uint64()

	// Получаем последний обработанный блок
	lastProcessedBlock, err := a.getLastProcessedBlock()
	if err != nil {
		return err
	}

	// Проверяем, сколько блоков нужно обработать
	blocksToProcess := currentBlock - lastProcessedBlock
	if blocksToProcess == 0 {
		logrus.Debugf("Нет новых блоков для обработки. Текущий блок: #%d", currentBlock)
		return nil
	}

	// Ограничиваем количество обрабатываемых блоков
	if blocksToProcess > maxBlocksPerBatch {
		blocksToProcess = maxBlocksPerBatch
		logrus.Warnf("Ограничение количества обрабатываемых блоков до %d", maxBlocksPerBatch)
	}

	logrus.Infof("Нужно обработать %d блоков (с #%d по #%d)", blocksToProcess, lastProcessedBlock+1, lastProcessedBlock+blocksToProcess)

	// Обрабатываем каждый блок в диапазоне
	for blockNum := lastProcessedBlock + 1; blockNum <= lastProcessedBlock+blocksToProcess; blockNum++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := a.processBlock(ctx, blockNum); err != nil {
				logrus.Errorf("Ошибка обработки блока #%d: %v", blockNum, err)
				return err
			}

			// Обновляем последний обработанный блок после каждого блока
			if err := a.saveLastProcessedBlock(blockNum); err != nil {
				logrus.Errorf("Ошибка сохранения состояния для блока #%d: %v", blockNum, err)
				return err
			}
		}
	}

	logrus.Infof("Успешно обработано %d блоков", blocksToProcess)
	return nil
}

func (a *Analyzer) processTransaction(ctx context.Context, tx *types.Transaction, blockTime uint64) error {
	client := a.ethClient.GetClient()

	// Получаем receipt для статуса транзакции
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}

	// Проверяем статус транзакции
	if receipt.Status == 0 {
		logrus.Debugf("Транзакция %s была revert", tx.Hash().Hex())
		return nil // Пропускаем revert транзакции
	}

	// Получаем информацию об отправителе
	var signer types.Signer
	if tx.Type() == types.DynamicFeeTxType {
		// EIP-1559 транзакция
		signer = types.NewLondonSigner(a.ethClient.GetChainID())
	} else {
		// Обычная транзакция
		signer = types.NewEIP155Signer(a.ethClient.GetChainID())
	}

	from, err := types.Sender(signer, tx)
	if err != nil {
		return err
	}

	var to string
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	// Создаем запись транзакции
	transaction := &models.Transaction{
		Hash:        tx.Hash().Hex(),
		BlockNumber: receipt.BlockNumber.Uint64(),
		From:        from.Hex(),
		To:          to,
		Value:       tx.Value().String(),
		GasUsed:     receipt.GasUsed,
		Status:      uint64(receipt.Status),
		Timestamp:   time.Unix(int64(blockTime), 0),
	}

	// Обрабатываем GasPrice для разных типов транзакций
	if tx.GasPrice() != nil {
		transaction.SetGasPrice(tx.GasPrice())
	} else {
		// Для EIP-1559 транзакций используем EffectiveGasPrice из receipt
		if receipt.EffectiveGasPrice != nil {
			transaction.SetGasPrice(receipt.EffectiveGasPrice)
		} else {
			transaction.SetGasPrice(big.NewInt(0))
		}
	}

	// Проверяем, существует ли уже такая транзакция
	var existingTx models.Transaction
	result := repositories.DB.Where("hash = ?", transaction.Hash).First(&existingTx)
	if result.Error == nil {
		// Транзакция уже существует
		return nil
	}

	// Сохраняем новую транзакцию с обработкой дублирования
	result = repositories.DB.Create(transaction)
	if result.Error != nil {
		// Если ошибка дублирования, просто игнорируем
		if result.Error.Error() == "duplicate key value violates unique constraint \"idx_transactions_hash\"" {
			logrus.Debugf("Транзакция %s уже существует, пропускаем", transaction.Hash)
			return nil
		}
		return result.Error
	}

	// Обновляем статистику аккаунтов только если это не транзакция деплоя контракта
	if to != "" {
		if err := a.accountAnalyzer.UpdateAccountStats(transaction); err != nil {
			logrus.Errorf("Ошибка обновления статистики аккаунтов для транзакции %s: %v", transaction.Hash, err)
			// Не возвращаем ошибку, чтобы не прерывать обработку транзакций
		}
	} else {
		logrus.Debugf("Пропускаем обновление статистики для транзакции деплоя контракта: %s", transaction.Hash)
	}

	// Обрабатываем ERC20 события в логах транзакции
	if err := a.processTransactionLogs(ctx, receipt); err != nil {
		logrus.Errorf("Ошибка обработки логов транзакции %s: %v", transaction.Hash, err)
		// Не возвращаем ошибку, чтобы не прерывать обработку транзакций
	}

	logrus.Debugf("Сохранена транзакция: %s", transaction.Hash)
	return nil
}

func (a *Analyzer) processBlock(ctx context.Context, blockNum uint64) error {
	client := a.ethClient.GetClient()

	// Получаем блок по номеру
	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		return err
	}

	logrus.Debugf("Обработка блока #%d с %d транзакциями", blockNum, len(block.Transactions()))

	// Счетчики для статистики
	processedCount := 0
	errorCount := 0

	// Обрабатываем каждую транзакцию в блоке
	for _, tx := range block.Transactions() {
		if err := a.processTransaction(ctx, tx, block.Time()); err != nil {
			logrus.Errorf("Ошибка обработки транзакции %s: %v", tx.Hash().Hex(), err)
			errorCount++
		} else {
			processedCount++
		}
	}

	logrus.Debugf("Блок #%d обработан: %d транзакций успешно, %d ошибок",
		blockNum, processedCount, errorCount)

	return nil
}

func (a *Analyzer) getLastProcessedBlock() (uint64, error) {
	var state models.AnalyzerState

	// Попытка получить состояние из БД
	result := repositories.DB.First(&state)
	if result.Error != nil {
		// Если запись не найдена, возвращаем 0 (первый запуск)
		if result.Error == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, result.Error
	}

	return state.LastProcessedBlock, nil
}

func (a *Analyzer) saveLastProcessedBlock(blockNumber uint64) error {
	// Используем транзакцию для атомарности операции
	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		var state models.AnalyzerState

		// Попытка найти существующую запись с блокировкой
		result := tx.Set("gorm:query_option", "FOR UPDATE").First(&state)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// Создаем новую запись
				state = models.AnalyzerState{
					LastProcessedBlock: blockNumber,
				}
				return tx.Create(&state).Error
			}
			return result.Error
		}

		// Обновляем существующую запись только если новый блок больше
		if blockNumber > state.LastProcessedBlock {
			state.LastProcessedBlock = blockNumber
			return tx.Save(&state).Error
		}

		return nil // Блок уже обработан или более старый
	})
}

func (a *Analyzer) Stop() {
	logrus.Info("Остановка анализатора...")

	if a.eventListener != nil {
		a.eventListener.Stop()
	}

	if a.notifier != nil {
		a.notifier.Stop()
	}

	logrus.Info("Анализатор остановлен")
}

func (a *Analyzer) processTransactionLogs(ctx context.Context, receipt *types.Receipt) error {
	// Обрабатываем каждый лог в транзакции
	for _, log := range receipt.Logs {
		// Проверяем, является ли лог ERC20 Transfer событием
		if err := a.processERC20TransferLog(log); err != nil {
			logrus.Errorf("Ошибка обработки ERC20 лога: %v", err)
			continue
		}
	}
	return nil
}

func (a *Analyzer) processERC20TransferLog(log *types.Log) error {
	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		// Проверяем, что это Transfer событие (должно быть 3 топика)
		if len(log.Topics) < 3 {
			return nil // Не Transfer событие
		}

		// Проверяем сигнатуру Transfer события
		// keccak256("Transfer(address,address,uint256)") = 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
		transferSignature := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
		if log.Topics[0] != transferSignature {
			return nil // Не Transfer событие
		}

		// Парсим адреса из топиков
		from := common.BytesToAddress(log.Topics[1].Bytes())
		to := common.BytesToAddress(log.Topics[2].Bytes())

		// Проверяем на нулевой адрес
		zeroAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")
		if from == zeroAddress || to == zeroAddress {

			return nil
		}

		// Парсим значение из данных
		value := big.NewInt(0)
		if len(log.Data) > 0 {
			value.SetBytes(log.Data)
		}

		// Создаем запись ERC20 трансфера
		erc20Transfer := &models.ERC20Transfer{
			TransactionHash: log.TxHash.Hex(),
			ContractAddress: log.Address.Hex(),
			From:            from.Hex(),
			To:              to.Hex(),
			Value:           value.String(),
			BlockNumber:     log.BlockNumber,
		}

		// Проверяем, существует ли уже такой трансфер
		var existingTransfer models.ERC20Transfer
		result := tx.Where("transaction_hash = ? AND contract_address = ? AND \"from\" = ? AND \"to\" = ?",
			erc20Transfer.TransactionHash, erc20Transfer.ContractAddress, erc20Transfer.From, erc20Transfer.To).First(&existingTransfer)
		if result.Error == nil {
			// Трансфер уже существует
			return nil
		}

		// Сохраняем новый трансфер с обработкой дублирования
		result = tx.Create(erc20Transfer)
		if result.Error != nil {
			// Если ошибка дублирования, просто игнорируем
			if result.Error.Error() == "duplicate key value violates unique constraint" {
				logrus.Debugf("ERC20 трансфер уже существует, пропускаем")
				return nil
			}
			return result.Error
		}

		// ИСПРАВЛЕНО: Обновляем статистику ERC20 транзакций только для новых уникальных транзакций
		if err := a.updateERC20StatsForUniqueTransaction(erc20Transfer); err != nil {
			logrus.Errorf("Ошибка обновления ERC20 статистики: %v", err)
		}

		// Обновляем балансы токенов
		if err := a.updateTokenBalances(erc20Transfer); err != nil {
			logrus.Errorf("Ошибка обновления балансов токенов: %v", err)
		}

		logrus.Debugf("Сохранен ERC20 трансфер: %s от %s к %s значение %s",
			erc20Transfer.ContractAddress, erc20Transfer.From, erc20Transfer.To, erc20Transfer.Value)

		return nil
	})
}

// ИСПРАВЛЕНО: Новая функция для обновления статистики по уникальным транзакциям
func (a *Analyzer) updateERC20StatsForUniqueTransaction(transfer *models.ERC20Transfer) error {
	// Обновляем статистику только для отправителя
	if err := a.incrementERC20TransactionCountIfNotProcessed(transfer.From, transfer.TransactionHash); err != nil {
		return err
	}

	return nil
}

// Увеличиваем счетчик ERC20 транзакций только если еще не обрабатывали эту транзакцию для данного адреса
func (a *Analyzer) incrementERC20TransactionCountIfNotProcessed(address, txHash string) error {
	// Пропускаем нулевой адрес
	if address == "0x0000000000000000000000000000000000000000" {
		return nil
	}

	// Проверяем, есть ли уже трансферы для этого адреса как отправителя
	var count int64
	result := repositories.DB.Model(&models.ERC20Transfer{}).
		Where("transaction_hash = ? AND \"from\" = ?", txHash, address).
		Count(&count)

	if result.Error != nil {
		return result.Error
	}

	// Если это первый трансфер от этого адреса в этой транзакции, увеличиваем счетчик
	if count > 0 {
		if err := a.incrementERC20TransactionCount(address); err != nil {
			return err
		}
		logrus.Debugf("Увеличен счетчик ERC20 для %s в транзакции %s", address, txHash)
	}

	return nil
}

func (a *Analyzer) incrementERC20TransactionCount(address string) error {
	// Получаем или создаем статистику аккаунта
	stats, err := a.accountAnalyzer.accountRepo.GetOrCreateAccountStats(address)
	if err != nil {
		return err
	}

	// Увеличиваем счетчик ERC20 транзакций
	stats.ERC20Transactions++

	// Сохраняем обновленную статистику
	if err := a.accountAnalyzer.accountRepo.UpdateAccountStats(stats); err != nil {
		return err
	}

	return nil
}

func (a *Analyzer) updateTokenBalances(transfer *models.ERC20Transfer) error {
	// Парсим значение
	value, ok := big.NewInt(0).SetString(transfer.Value, 10)
	if !ok {
		return nil // Пропускаем если не удалось распарсить значение
	}

	// Пропускаем mint/burn операции (от/к нулевому адресу)
	zeroAddress := "0x0000000000000000000000000000000000000000"

	// Обновляем баланс отправителя (если не mint)
	if transfer.From != zeroAddress {
		if err := a.accountAnalyzer.GetTokenTracker().UpdateTokenBalance(transfer.From, transfer.ContractAddress, value, false); err != nil {
			logrus.Errorf("Ошибка обновления баланса отправителя %s: %v", transfer.From, err)
		}
	}

	// Обновляем баланс получателя (если не burn)
	if transfer.To != zeroAddress {
		if err := a.accountAnalyzer.GetTokenTracker().UpdateTokenBalance(transfer.To, transfer.ContractAddress, value, true); err != nil {
			logrus.Errorf("Ошибка обновления баланса получателя %s: %v", transfer.To, err)
		}
	}

	return nil
}

// Добавляем новый метод для анализа контракта
func (a *Analyzer) AnalyzeContract(ctx context.Context, contractAddress common.Address, fromBlock, toBlock uint64) error {
	logrus.Infof("Начинаем анализ контракта %s", contractAddress.Hex())

	if err := a.contractAnalyzer.AnalyzeContractTransactions(ctx, fromBlock, toBlock); err != nil {
		return fmt.Errorf("ошибка анализа контракта: %v", err)
	}

	logrus.Infof("Анализ контракта %s завершен", contractAddress.Hex())
	return nil
}
