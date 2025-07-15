package listeners

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/pkg/contracts"
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

const (
	maxRetries          = 3
	retryDelay          = 2 * time.Second
	confirmations       = 12 // количество блоков для подтверждения
	batchSize           = 100
	defaultPollInterval = 1 * time.Second
)

type EventListener struct {
	client          *ethclient.Client
	contractAddress common.Address
	erc20Contract   *contracts.ERC20Contract
	pollInterval    time.Duration
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type ListenerConfig struct {
	PollInterval time.Duration
}

func NewEventListener(client *ethclient.Client, contractAddress common.Address) (*EventListener, error) {
	erc20Contract, err := contracts.NewERC20Contract(contractAddress, client)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания контракта: %v", err)
	}

	return &EventListener{
		client:          client,
		contractAddress: contractAddress,
		erc20Contract:   erc20Contract,
		pollInterval:    defaultPollInterval,
	}, nil
}

func (el *EventListener) SetConfig(cfg ListenerConfig) {
	if cfg.PollInterval > 0 {
		el.pollInterval = cfg.PollInterval
	}
}

func (el *EventListener) StartListening(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	el.cancel = cancel

	logrus.Infof("Начинаем прослушивание событий контракта: %s (polling mode)", el.contractAddress.Hex())

	// Используем polling вместо WebSocket подписок
	el.wg.Add(1)
	go func() {
		defer el.wg.Done()
		ticker := time.NewTicker(el.pollInterval)
		defer ticker.Stop()

		var lastBlock uint64 = 0
		var lastProcessedEvents = make(map[string]bool)

		for {
			select {
			case <-ticker.C:
				if err := el.pollForEvents(ctx, &lastBlock, lastProcessedEvents); err != nil {
					logrus.Errorf("Ошибка опроса событий: %v", err)
					// Делаем retry с exponential backoff
					for i := 0; i < maxRetries; i++ {
						time.Sleep(retryDelay * time.Duration(i+1))
						if err := el.pollForEvents(ctx, &lastBlock, lastProcessedEvents); err == nil {
							break
						}
					}
				}
			case <-ctx.Done():
				logrus.Info("Остановка прослушивания событий")
				return
			}
		}
	}()

	return nil
}

func (el *EventListener) Stop() {
	if el.cancel != nil {
		el.cancel()
		el.wg.Wait()
	}
}

func (el *EventListener) pollForEvents(ctx context.Context, lastBlock *uint64, lastProcessedEvents map[string]bool) error {
	// Получаем текущий блок
	header, err := el.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка получения текущего блока: %v", err)
	}

	currentBlock := header.Number.Uint64()

	if *lastBlock == 0 {
		*lastBlock = currentBlock - confirmations
		return nil
	}

	// Проверяем, есть ли достаточно подтверждений
	if currentBlock < *lastBlock+confirmations {
		return nil
	}

	// Обрабатываем только подтвержденные блоки
	toBlock := currentBlock - confirmations
	if toBlock <= *lastBlock {
		return nil
	}

	// Создаем фильтр для новых событий
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(*lastBlock + 1)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{el.contractAddress},
	}

	logs, err := el.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка фильтрации логов: %v", err)
	}

	// Группируем события для батчинга
	var transfers []*models.ERC20Transfer

	// Обрабатываем найденные события
	for _, vLog := range logs {
		// Проверяем, не обработали ли мы уже это событие
		eventKey := fmt.Sprintf("%s-%d-%d", vLog.TxHash.Hex(), vLog.BlockNumber, vLog.Index)
		if lastProcessedEvents[eventKey] {
			continue
		}

		transfer, err := el.processLog(ctx, vLog)
		if err != nil {
			logrus.Errorf("Ошибка обработки события: %v", err)
			continue
		}

		if transfer != nil {
			transfers = append(transfers, transfer)
			lastProcessedEvents[eventKey] = true
		}

		// Если накопилось достаточно событий, сохраняем их
		if len(transfers) >= batchSize {
			if err := el.saveTransfers(transfers); err != nil {
				return fmt.Errorf("ошибка сохранения batch трансферов: %v", err)
			}
			transfers = transfers[:0]
		}
	}

	// Сохраняем оставшиеся события
	if len(transfers) > 0 {
		if err := el.saveTransfers(transfers); err != nil {
			return fmt.Errorf("ошибка сохранения оставшихся трансферов: %v", err)
		}
	}

	// Очищаем старые записи в кэше обработанных событий
	for key := range lastProcessedEvents {
		delete(lastProcessedEvents, key)
	}

	*lastBlock = toBlock
	return nil
}

func (el *EventListener) processLog(ctx context.Context, vLog types.Log) (*models.ERC20Transfer, error) {
	// Получаем сигнатуру события Transfer
	transferEventSignature := el.erc20Contract.GetABI().Events["Transfer"].ID
	if len(vLog.Topics) == 0 || vLog.Topics[0] != transferEventSignature {
		return nil, nil
	}

	if len(vLog.Topics) < 3 {
		return nil, fmt.Errorf("неверное количество topics в событии Transfer")
	}

	from := common.HexToAddress(vLog.Topics[1].Hex())
	to := common.HexToAddress(vLog.Topics[2].Hex())

	// Декодируем данные события
	amount := new(big.Int).SetBytes(vLog.Data)

	return &models.ERC20Transfer{
		TransactionHash: vLog.TxHash.Hex(),
		ContractAddress: el.contractAddress.Hex(),
		From:            from.Hex(),
		To:              to.Hex(),
		Value:           amount.String(),
		BlockNumber:     vLog.BlockNumber,
		CreatedAt:       time.Now(),
	}, nil
}

func (el *EventListener) saveTransfers(transfers []*models.ERC20Transfer) error {
	if len(transfers) == 0 {
		return nil
	}

	// Используем транзакцию для батчинга
	tx := repositories.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", tx.Error)
	}

	for _, transfer := range transfers {
		if err := tx.Create(transfer).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("ошибка сохранения трансфера: %v", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %v", err)
	}

	logrus.Infof("Сохранено %d ERC20 трансферов", len(transfers))
	return nil
}
