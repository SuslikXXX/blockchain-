package listeners

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/pkg/contracts"
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

type EventListener struct {
	client          *ethclient.Client
	contractAddress common.Address
	erc20Contract   *contracts.ERC20Contract
}

func NewEventListener(client *ethclient.Client, contractAddress common.Address) (*EventListener, error) {
	erc20Contract, err := contracts.NewERC20Contract(contractAddress, client)
	if err != nil {
		return nil, err
	}

	return &EventListener{
		client:          client,
		contractAddress: contractAddress,
		erc20Contract:   erc20Contract,
	}, nil
}

func (el *EventListener) StartListening(ctx context.Context) error {
	logrus.Infof("Начинаем прослушивание событий контракта: %s (polling mode)", el.contractAddress.Hex())

	// Используем polling вместо WebSocket подписок
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		var lastBlock uint64 = 0

		for {
			select {
			case <-ticker.C:
				if err := el.pollForEvents(ctx, &lastBlock); err != nil {
					logrus.Errorf("Ошибка опроса событий: %v", err)
				}
			case <-ctx.Done():
				logrus.Info("Остановка прослушивания событий")
				return
			}
		}
	}()

	return nil
}

func (el *EventListener) pollForEvents(ctx context.Context, lastBlock *uint64) error {
	// Получаем текущий блок
	header, err := el.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	currentBlock := header.Number.Uint64()

	if *lastBlock == 0 {
		*lastBlock = currentBlock
		return nil
	}

	if currentBlock <= *lastBlock {
		return nil
	}

	// Создаем фильтр для новых событий
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(*lastBlock + 1)),
		ToBlock:   big.NewInt(int64(currentBlock)),
		Addresses: []common.Address{el.contractAddress},
	}

	logs, err := el.client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	// Обрабатываем найденные события
	for _, vLog := range logs {
		if err := el.processLog(ctx, vLog); err != nil {
			logrus.Errorf("Ошибка обработки события: %v", err)
		}
	}

	*lastBlock = currentBlock
	return nil
}

func (el *EventListener) processLog(ctx context.Context, vLog types.Log) error {
	// Проверяем, что это Transfer событие
	transferEventSignature := el.erc20Contract.GetABI().Events["Transfer"].ID
	if len(vLog.Topics) > 0 && vLog.Topics[0] == transferEventSignature {
		return el.processTransferEvent(ctx, vLog)
	}

	return nil
}

func (el *EventListener) processTransferEvent(ctx context.Context, vLog types.Log) error {
	if len(vLog.Topics) < 3 {
		return nil
	}

	from := common.HexToAddress(vLog.Topics[1].Hex())
	to := common.HexToAddress(vLog.Topics[2].Hex())
	value := new(big.Int).SetBytes(vLog.Data)

	transfer := &models.ERC20Transfer{
		TransactionHash: vLog.TxHash.Hex(),
		ContractAddress: el.contractAddress.Hex(),
		From:            from.Hex(),
		To:              to.Hex(),
		BlockNumber:     vLog.BlockNumber,
		LogIndex:        vLog.Index,
	}
	transfer.SetValue(value)

	// Сохраняем в базу данных
	result := repositories.DB.Create(transfer)
	if result.Error != nil {
		return result.Error
	}

	logrus.Infof("ERC20 Transfer: %s -> %s, Amount: %s, TxHash: %s",
		from.Hex(), to.Hex(), value.String(), vLog.TxHash.Hex())

	return nil
}
