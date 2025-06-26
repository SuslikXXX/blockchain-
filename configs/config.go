package configs

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Database DatabaseConfig
	Ethereum EthereumConfig
	Server   ServerConfig
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
	RPCEndpoint string
	ChainID     int64
	PrivateKey  string
}

type ServerConfig struct {
	Port string
	Host string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		logrus.Warning("Файл .env не найден, используем переменные окружения")
	}

	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "blockchain_password_pass"),
			DBName:   getEnv("DB_NAME", "blockchain_analyzer"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Ethereum: EthereumConfig{
			RPCEndpoint: getEnv("ETH_RPC_ENDPOINT", "http://localhost:8545"),
			ChainID:     parseInt64(getEnv("ETH_CHAIN_ID", "31337")),
			PrivateKey:  getEnv("ETH_PRIVATE_KEY", ""),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "localhost"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt64(s string) int64 {
	if s == "31337" {
		return 31337 // Hardhat default chain ID
	}
	return 1 // Ethereum mainnet default
}
