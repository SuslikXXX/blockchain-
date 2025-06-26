package services

import (
	"blockchain/configs"
	"blockchain/internal/models"
	"blockchain/internal/repositories"
	"blockchain/pkg/ethereum"
	"blockchain/pkg/listeners"
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
)

type Analyzer struct {
	ethClient     *ethereum.Client
	eventListener *listeners.EventListener
	config        *configs.Config
}

func NewAnalyzer(cfg *configs.Config) (*Analyzer, error) {
	ethClient, err := ethereum.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Analyzer{
		ethClient: ethClient,
		config:    cfg,
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

	logrus.Info("Анализатор успешно запущен")
	return nil
}

func (a *Analyzer) startTransactionMonitoring(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.processLatestBlock(ctx); err != nil {
				logrus.Errorf("Ошибка обработки последнего блока: %v", err)
			}
		case <-ctx.Done():
			logrus.Info("Остановка мониторинга транзакций")
			return
		}
	}
}

func (a *Analyzer) processLatestBlock(ctx context.Context) error {
	client := a.ethClient.GetClient()

	// Получаем последний блок
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	block, err := client.BlockByNumber(ctx, header.Number)
	if err != nil {
		return err
	}

	logrus.Debugf("Обработка блока #%d с %d транзакциями", block.Number().Uint64(), len(block.Transactions()))

	// Обрабатываем каждую транзакцию в блоке
	for _, tx := range block.Transactions() {
		if err := a.processTransaction(ctx, tx, block.Time()); err != nil {
			logrus.Errorf("Ошибка обработки транзакции %s: %v", tx.Hash().Hex(), err)
		}
	}

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
		GasUsed:     receipt.GasUsed,
		Status:      uint64(receipt.Status),
		Timestamp:   time.Unix(int64(blockTime), 0),
	}
	transaction.SetValue(tx.Value())

	// Обрабатываем GasPrice для разных типов транзакций
	if tx.GasPrice() != nil {
		transaction.GasPrice = tx.GasPrice().String()
	} else {
		// Для EIP-1559 транзакций используем EffectiveGasPrice из receipt
		if receipt.EffectiveGasPrice != nil {
			transaction.GasPrice = receipt.EffectiveGasPrice.String()
		} else {
			transaction.GasPrice = "0"
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

	logrus.Debugf("Сохранена транзакция: %s", transaction.Hash)
	return nil
}

func (a *Analyzer) Stop() {
	if a.ethClient != nil {
		a.ethClient.Close()
	}
	logrus.Info("Анализатор остановлен")
}
