package simulators

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

// ContractArtifact представляет структуру JSON файла артефакта Hardhat
type ContractArtifact struct {
	ContractName string          `json:"contractName"`
	SourceName   string          `json:"sourceName"`
	ABI          json.RawMessage `json:"abi"`
	Bytecode     string          `json:"bytecode"`
}

// ContractLoader загружает артефакты контрактов из файлов
type ContractLoader struct {
	artifactsPath string
}

// NewContractLoader создает новый загрузчик контрактов
func NewContractLoader(artifactsPath string) *ContractLoader {
	return &ContractLoader{
		artifactsPath: artifactsPath,
	}
}

// LoadContract загружает артефакт контракта по имени
func (cl *ContractLoader) LoadContract(contractName string) (*ContractArtifact, error) {
	// Путь к JSON файлу артефакта
	artifactPath := filepath.Join(cl.artifactsPath, "contracts", contractName+".sol", contractName+".json")

	logrus.Infof("Загружаем артефакт контракта: %s", artifactPath)

	// Читаем файл
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл артефакта %s: %w", artifactPath, err)
	}

	// Парсим JSON
	var artifact ContractArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("не удалось распарсить JSON артефакта: %w", err)
	}

	logrus.Infof("Артефакт загружен: %s (байткод: %d байт)", artifact.ContractName, len(artifact.Bytecode))

	return &artifact, nil
}

// GetBytecode возвращает байткод контракта как []byte
func (ca *ContractArtifact) GetBytecode() ([]byte, error) {
	if ca.Bytecode == "" {
		return nil, fmt.Errorf("байткод пустой")
	}

	// Убираем префикс "0x" если есть
	bytecode := ca.Bytecode
	if len(bytecode) > 2 && bytecode[:2] == "0x" {
		bytecode = bytecode[2:]
	}

	return common.FromHex(bytecode), nil
}

// GetABI возвращает ABI контракта
func (ca *ContractArtifact) GetABI() (abi.ABI, error) {
	if ca.ABI == nil {
		return abi.ABI{}, fmt.Errorf("ABI пустой")
	}

	return abi.JSON(strings.NewReader(string(ca.ABI)))
}

// GetConstructorABI возвращает ABI конструктора
func (ca *ContractArtifact) GetConstructorABI() (abi.ABI, error) {
	fullABI, err := ca.GetABI()
	if err != nil {
		return abi.ABI{}, err
	}

	// Ищем конструктор в ABI
	for _, method := range fullABI.Methods {
		if method.Type == abi.Constructor {
			// Создаем новый ABI только с конструктором
			constructorABI := abi.ABI{
				Methods: map[string]abi.Method{
					"": method,
				},
			}
			return constructorABI, nil
		}
	}

	return abi.ABI{}, fmt.Errorf("конструктор не найден в ABI")
}
