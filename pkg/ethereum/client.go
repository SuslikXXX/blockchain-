package ethereum

import (
	"blockchain/configs"
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

type Client struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	address    common.Address
	chainID    *big.Int
}

func NewClient(cfg *configs.Config) (*Client, error) {
	client, err := ethclient.Dial(cfg.Ethereum.RPCEndpoint)
	if err != nil {
		return nil, err
	}

	var privateKey *ecdsa.PrivateKey
	var publicKey *ecdsa.PublicKey
	var address common.Address

	if cfg.Ethereum.PrivateKey != "" {
		privateKey, err = crypto.HexToECDSA(cfg.Ethereum.PrivateKey)
		if err != nil {
			return nil, err
		}

		publicKey = privateKey.Public().(*ecdsa.PublicKey)
		address = crypto.PubkeyToAddress(*publicKey)
	}

	chainID := big.NewInt(cfg.Ethereum.ChainID)

	logrus.Infof("Подключение к Ethereum RPC: %s", cfg.Ethereum.RPCEndpoint)
	logrus.Infof("Chain ID: %d", cfg.Ethereum.ChainID)
	if address != (common.Address{}) {
		logrus.Infof("Адрес кошелька: %s", address.Hex())
	}

	return &Client{
		client:     client,
		privateKey: privateKey,
		publicKey:  publicKey,
		address:    address,
		chainID:    chainID,
	}, nil
}

func (c *Client) GetClient() *ethclient.Client {
	return c.client
}

func (c *Client) GetAddress() common.Address {
	return c.address
}

func (c *Client) GetChainID() *big.Int {
	return c.chainID
}

func (c *Client) GetPrivateKey() *ecdsa.PrivateKey {
	return c.privateKey
}

func (c *Client) GetBalance(ctx context.Context, address common.Address) (*big.Int, error) {
	return c.client.BalanceAt(ctx, address, nil)
}

func (c *Client) Close() {
	c.client.Close()
}
