package services

import (
	"backend/configs"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/pkg/ethereum"
	"backend/pkg/listeners"
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Analyzer struct {
	ethClient       *ethereum.Client
	eventListener   *listeners.EventListener
	accountAnalyzer *AccountAnalyzer
	config          *configs.Config
}

func NewAnalyzer(cfg *configs.Config) (*Analyzer, error) {
	ethClient, err := ethereum.NewClient(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	// ИСПРАВЛЕНО: создаем общий репозиторий для всех сервисов
	accountRepo := repositories.NewAccountRepository()

	return &Analyzer{
		ethClient:       ethClient,
		accountAnalyzer: NewAccountAnalyzer(accountRepo),
		config:          cfg,
	}, nil
}

func (a *Analyzer) Start(ctx context.Context, contractAddress common.Address) error {
	logrus.Info("Запуск анализатора блокчейн активности...")

	// Создаем event listener для указанного контракта
	eventListener, err := listeners.NewEventListener(a.ethClient.GetClient(), contractAddress)
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

	// Запускаем периодические задачи анализа аккаунтов
	go a.accountAnalyzer.StartPeriodicTasks(ctx)

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

	logrus.Infof("Нужно обработать %d блоков (с #%d по #%d)", blocksToProcess, lastProcessedBlock+1, currentBlock)

	// Обрабатываем каждый блок в диапазоне
	for blockNum := lastProcessedBlock + 1; blockNum <= currentBlock; blockNum++ {

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

	// Сохраняем новую транзакцию
	result = repositories.DB.Create(transaction)
	if result.Error != nil {
		return result.Error
	}

	// Обновляем статистику аккаунтов
	if err := a.accountAnalyzer.UpdateAccountStats(transaction); err != nil {
		logrus.Errorf("Ошибка обновления статистики аккаунтов для транзакции %s: %v", transaction.Hash, err)
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
	if a.ethClient != nil {
		a.ethClient.Close()
	}
	logrus.Info("Анализатор остановлен")
}
