package services

import (
	"backend/internal/models"
	"backend/internal/repositories"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
)

type TokenTracker struct {
	accountRepo *repositories.AccountRepository
}

// ИСПРАВЛЕНО: добавлен конструктор с dependency injection
func NewTokenTracker(accountRepo *repositories.AccountRepository) *TokenTracker {
	return &TokenTracker{
		accountRepo: accountRepo,
	}
}

// Deprecated: оставлен для совместимости, но лучше использовать конструктор с DI
func NewTokenTrackerDeprecated() *TokenTracker {
	return &TokenTracker{
		accountRepo: repositories.NewAccountRepository(),
	}
}

// TrackERC20Transfer обрабатывает Transfer событие токена
func (t *TokenTracker) TrackERC20Transfer(log types.Log) error {
	// Проверяем, что это Transfer событие (должно быть 3 топика)
	if len(log.Topics) < 3 {
		return nil // Не Transfer событие
	}

	// Парсим адреса из топиков
	from := common.BytesToAddress(log.Topics[1].Bytes())
	to := common.BytesToAddress(log.Topics[2].Bytes())

	// Парсим значение из данных
	value := big.NewInt(0)
	if len(log.Data) > 0 {
		value.SetBytes(log.Data)
	}

	// Пропускаем mint/burn операции (от/к нулевому адресу)
	zeroAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")

	// Обновляем баланс отправителя (если не mint)
	if from != zeroAddress {
		if err := t.UpdateTokenBalance(from.Hex(), log.Address.Hex(), value, false); err != nil {
			logrus.Errorf("Ошибка обновления баланса отправителя %s: %v", from.Hex(), err)
			return err
		}
	}

	// Обновляем баланс получателя (если не burn)
	if to != zeroAddress {
		if err := t.UpdateTokenBalance(to.Hex(), log.Address.Hex(), value, true); err != nil {
			logrus.Errorf("Ошибка обновления баланса получателя %s: %v", to.Hex(), err)
			return err
		}
	}

	logrus.Debugf("Обработан Transfer: от %s к %s токен %s значение %s",
		from.Hex(), to.Hex(), log.Address.Hex(), value.String())

	return nil
}

// UpdateTokenBalance обновляет баланс токена у аккаунта
func (t *TokenTracker) UpdateTokenBalance(address, tokenAddress string, amount *big.Int, isIncoming bool) error {
	// Получаем текущий баланс
	balance, err := t.accountRepo.GetTokenBalance(address, tokenAddress)
	if err != nil {
		return err
	}

	// Получаем текущий баланс как big.Int
	currentBalance := balance.GetBalance()

	// Обновляем баланс
	if isIncoming {
		currentBalance = currentBalance.Add(currentBalance, amount)
	} else {
		currentBalance = currentBalance.Sub(currentBalance, amount)
		// Защита от отрицательного баланса
		if currentBalance.Sign() < 0 {
			logrus.Debugf("Предотвращение отрицательного баланса для %s токен %s: было бы %s, устанавливаем 0",
				address, tokenAddress, currentBalance.String())
			currentBalance = big.NewInt(0)
		}
	}

	// Сохраняем обновленный баланс
	balance.SetBalance(currentBalance)

	if err := t.accountRepo.UpdateTokenBalance(balance); err != nil {
		return err
	}

	logrus.Debugf("Обновлен баланс %s токен %s: %s",
		address, tokenAddress, currentBalance.String())

	return nil
}

// CalculateTokenVolume рассчитывает объем токена с момента времени
func (t *TokenTracker) CalculateTokenVolume(address, tokenAddress string, since time.Time) (*big.Int, error) {
	// Это сложная операция, требующая анализа всех Transfer событий
	// Пока возвращаем 0, можно реализовать позже
	return big.NewInt(0), nil
}

// GetTokenBalance возвращает текущий баланс токена у аккаунта
func (t *TokenTracker) GetTokenBalance(address, tokenAddress string) (*big.Int, error) {
	balance, err := t.accountRepo.GetTokenBalance(address, tokenAddress)
	if err != nil {
		return big.NewInt(0), err
	}

	return balance.GetBalance(), nil
}

// GetAccountTokens возвращает все токены с ненулевым балансом у аккаунта
func (t *TokenTracker) GetAccountTokens(address string) ([]models.TokenBalance, error) {
	return t.accountRepo.GetAccountTokens(address)
}
