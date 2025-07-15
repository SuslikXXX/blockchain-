package configs

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Database    DatabaseConfig
	Ethereum    EthereumConfig
	Server      ServerConfig
	TelegramBot TelegramBotConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type EthereumConfig struct {
	RPCEndpoint     string
	ChainID         int64
	PrivateKey      string
	ContractAddress string
}

type ServerConfig struct {
	Port string
	Host string
}

type TelegramBotConfig struct {
	BotToken string
	ChatID   string // теперь просто строка для одного ID
}

func Load() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	envDir := filepath.Join(home, "Desktop", "blockchain_analyzer", "backend")
	envPath := filepath.Join(envDir, ".env")

	err = godotenv.Load(envPath)
	if err != nil {
		logrus.Warning("Файл .env не найден, используем переменные окружения")
	}

	chainID, err := strconv.ParseInt(os.Getenv("ETH_CHAIN_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("Неверный формат ChainID: %v", err)
	}

	return &Config{
		Database: DatabaseConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSL_MODE"),
		},
		Ethereum: EthereumConfig{
			RPCEndpoint:     os.Getenv("ETH_RPC_ENDPOINT"),
			ChainID:         chainID,
			PrivateKey:      os.Getenv("ETH_PRIVATE_KEY"),
			ContractAddress: os.Getenv("ETH_CONTRACT_ADDRESS"),
		},
		Server: ServerConfig{
			Port: os.Getenv("SERVER_PORT"),
			Host: os.Getenv("SERVER_HOST"),
		},
		TelegramBot: TelegramBotConfig{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
			ChatID:   os.Getenv("TELEGRAM_CHAT_ID"), // один ID
		},
	}
}
