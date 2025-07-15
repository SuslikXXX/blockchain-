// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/math/Math.sol";

contract AnalyzerToken is ERC20, Ownable {
    // Структуры данных для хранения информации о транзакциях
    struct Transaction {
        address from;
        address to;
        uint256 amount;
        uint256 timestamp;
        TransactionType txType;
        bytes32 description;
    }

    // Структура для внешних транзакций
    struct ExternalTransaction {
        address from;
        address to;
        uint256 value;
        uint256 timestamp;
        bytes4 methodId;
        bool success;
        uint256 gasUsed;
    }

    // Тип транзакции
    enum TransactionType {
        TRANSFER,      // Обычный перевод
        DEPOSIT,       // Пополнение
        WITHDRAWAL,    // Вывод
        REWARD,        // Награда/бонус
        FEE           // Комиссия
    }

    // Структура для агрегированной статистики
    struct AccountStats {
        uint256 totalTransactions;
        uint256 totalSent;
        uint256 totalReceived;
        uint256 lastActivityTime;
        uint256 firstActivityTime;
        uint256 externalTransactions; // Добавлено: количество внешних транзакций
    }

    // Маппинги для хранения данных
    mapping(address => Transaction[]) private userTransactions;
    mapping(address => AccountStats) private accountStats;
    mapping(address => mapping(uint256 => uint256)) private historicalBalances; // address => blockNumber => balance
    mapping(address => ExternalTransaction[]) private externalTransactions; // Добавлено: хранение внешних транзакций
    
    // События
    event TransactionRecorded(
        address indexed from,
        address indexed to,
        uint256 amount,
        uint256 timestamp,
        TransactionType txType,
        bytes32 description
    );

    event ExternalTransactionRecorded(
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 timestamp,
        bytes4 methodId,
        bool success,
        uint256 gasUsed
    );

    event StatsUpdated(
        address indexed user,
        uint256 totalTransactions,
        uint256 totalSent,
        uint256 totalReceived,
        uint256 externalTransactions
    );

    // Константы
    uint256 public constant MAX_HISTORY_DAYS = 365; // Максимальный период хранения истории
    uint256 public constant MAX_TRANSACTIONS_PER_REQUEST = 100; // Ограничение на количество транзакций в одном запросе

    constructor(string memory name, string memory symbol, uint256 initialSupply) 
        ERC20(name, symbol)
        Ownable(msg.sender)
    {
        _mint(msg.sender, initialSupply * 10 ** decimals());
    }

    // Переопределяем transfer для сохранения информации
    function transfer(address to, uint256 amount) public virtual override returns (bool) {
        return _processTransfer(msg.sender, to, amount, TransactionType.TRANSFER, "");
    }

    // Расширенный transfer с дополнительной информацией
    function transferWithInfo(
        address to, 
        uint256 amount, 
        TransactionType txType, 
        bytes32 description
    ) public returns (bool) {
        return _processTransfer(msg.sender, to, amount, txType, description);
    }

    // Внутренняя функция для обработки перевода
    function _processTransfer(
        address from,
        address to,
        uint256 amount,
        TransactionType txType,
        bytes32 description
    ) private returns (bool) {
        bool success = super.transfer(to, amount);
        if (success) {
            _recordTransaction(from, to, amount, txType, description);
            _updateHistoricalBalance(from);
            _updateHistoricalBalance(to);
            _updateAccountStats(from, to, amount);
        }
        return success;
    }

    // Запись транзакции
    function _recordTransaction(
        address from,
        address to,
        uint256 amount,
        TransactionType txType,
        bytes32 description
    ) private {
        // Создаем новую транзакцию
        Transaction memory newTx = Transaction(
            from,
            to,
            amount,
            block.timestamp,
            txType,
            description
        );

        // Сохраняем транзакцию для обоих участников
        userTransactions[from].push(newTx);
        userTransactions[to].push(newTx);

        // Очищаем старые транзакции
        _cleanOldTransactions(from);
        _cleanOldTransactions(to);

        // Генерируем событие
        emit TransactionRecorded(
            from,
            to,
            amount,
            block.timestamp,
            txType,
            description
        );
    }

    // Обновление исторического баланса
    function _updateHistoricalBalance(address account) private {
        historicalBalances[account][block.number] = balanceOf(account);
    }

    // Обновление статистики аккаунтов
    function _updateAccountStats(address from, address to, uint256 amount) private {
        // Обновляем статистику отправителя
        AccountStats storage fromStats = accountStats[from];
        if (fromStats.firstActivityTime == 0) {
            fromStats.firstActivityTime = block.timestamp;
        }
        fromStats.totalTransactions++;
        fromStats.totalSent += amount;
        fromStats.lastActivityTime = block.timestamp;

        // Генерируем событие для отправителя
        emit StatsUpdated(
            from,
            fromStats.totalTransactions,
            fromStats.totalSent,
            fromStats.totalReceived,
            fromStats.externalTransactions
        );

        // Обновляем статистику получателя только если это другой адрес
        if (from != to) {
        AccountStats storage toStats = accountStats[to];
        if (toStats.firstActivityTime == 0) {
            toStats.firstActivityTime = block.timestamp;
        }
        toStats.totalTransactions++;
        toStats.totalReceived += amount;
        toStats.lastActivityTime = block.timestamp;

            // Генерируем событие для получателя
        emit StatsUpdated(
            to,
            toStats.totalTransactions,
            toStats.totalSent,
                toStats.totalReceived,
                toStats.externalTransactions
        );
        }
    }

    // Очистка старых транзакций
    function _cleanOldTransactions(address user) private {
        Transaction[] storage txs = userTransactions[user];
        uint256 cutoffTime = block.timestamp - (MAX_HISTORY_DAYS * 1 days);
        
        uint256 i = 0;
        while (i < txs.length && txs[i].timestamp < cutoffTime) {
            i++;
        }
        
        if (i > 0) {
            uint256 j = 0;
            while (i < txs.length) {
                txs[j] = txs[i];
                j++;
                i++;
            }
            while (txs.length > j) {
                txs.pop();
            }
        }
    }

    // Публичные функции для анализа

    // Получение количества транзакций пользователя
    function getTransactionCount(address user) public view returns (uint256) {
        return userTransactions[user].length;
    }

    // Получение списка транзакций с пагинацией
    function getTransactions(
        address user,
        uint256 offset,
        uint256 limit
    ) public view returns (
        address[] memory froms,
        address[] memory tos,
        uint256[] memory amounts,
        uint256[] memory timestamps,
        TransactionType[] memory types,
        bytes32[] memory descriptions
    ) {
        // Ограничиваем количество транзакций
        uint256 actualLimit = Math.min(
            Math.min(limit, MAX_TRANSACTIONS_PER_REQUEST),
            userTransactions[user].length - offset
        );

        // Инициализируем массивы
        froms = new address[](actualLimit);
        tos = new address[](actualLimit);
        amounts = new uint256[](actualLimit);
        timestamps = new uint256[](actualLimit);
        types = new TransactionType[](actualLimit);
        descriptions = new bytes32[](actualLimit);

        // Заполняем массивы данными
        for (uint256 i = 0; i < actualLimit; i++) {
            Transaction storage txData = userTransactions[user][offset + i];
            froms[i] = txData.from;
            tos[i] = txData.to;
            amounts[i] = txData.amount;
            timestamps[i] = txData.timestamp;
            types[i] = txData.txType;
            descriptions[i] = txData.description;
        }

        return (froms, tos, amounts, timestamps, types, descriptions);
    }

    // Получение статистики аккаунта
    function getAccountStats(address user) public view returns (
        uint256 totalTx,
        uint256 totalSent,
        uint256 totalReceived,
        uint256 lastActivity,
        uint256 firstActivity
    ) {
        AccountStats memory stats = accountStats[user];
        return (
            stats.totalTransactions,
            stats.totalSent,
            stats.totalReceived,
            stats.lastActivityTime,
            stats.firstActivityTime
        );
    }

    // Получение баланса на определенный блок
    function getBalanceAtBlock(
        address user,
        uint256 blockNumber
    ) public view returns (uint256) {
        require(blockNumber <= block.number, "Block number is in the future");
        
        // Если запрашиваем текущий блок
        if (blockNumber == block.number) {
            return balanceOf(user);
        }

        // Ищем ближайший сохраненный баланс
        while (blockNumber > 0 && historicalBalances[user][blockNumber] == 0) {
            blockNumber--;
        }

        return historicalBalances[user][blockNumber];
    }

    // Получение объема транзакций за период
    function getVolumeForPeriod(
        address user,
        uint256 fromTimestamp,
        uint256 toTimestamp
    ) public view returns (uint256 sent, uint256 received) {
        sent = 0;
        received = 0;

        Transaction[] storage txs = userTransactions[user];
        for (uint256 i = 0; i < txs.length; i++) {
            if (txs[i].timestamp >= fromTimestamp && txs[i].timestamp <= toTimestamp) {
                if (txs[i].from == user) {
                    sent += txs[i].amount;
                }
                if (txs[i].to == user) {
                    received += txs[i].amount;
                }
            }
        }

        return (sent, received);
    }

    // Запись внешней транзакции
    function recordExternalTransaction(
        address from,
        address to,
        uint256 value,
        bytes4 methodId,
        bool success,
        uint256 gasUsed
    ) public onlyOwner {
        ExternalTransaction memory newTx = ExternalTransaction(
            from,
            to,
            value,
            block.timestamp,
            methodId,
            success,
            gasUsed
        );

        // Сохраняем транзакцию
        externalTransactions[from].push(newTx);
        if (from != to) {
            externalTransactions[to].push(newTx);
        }

        // Обновляем статистику отправителя
        AccountStats storage fromStats = accountStats[from];
        fromStats.externalTransactions++;
        fromStats.lastActivityTime = block.timestamp;
        if (fromStats.firstActivityTime == 0) {
            fromStats.firstActivityTime = block.timestamp;
        }

        // Генерируем событие для отправителя
        emit StatsUpdated(
            from,
            fromStats.totalTransactions,
            fromStats.totalSent,
            fromStats.totalReceived,
            fromStats.externalTransactions
        );

        // Обновляем статистику получателя только если это другой адрес
        if (from != to) {
            AccountStats storage toStats = accountStats[to];
            toStats.externalTransactions++;
            toStats.lastActivityTime = block.timestamp;
            if (toStats.firstActivityTime == 0) {
                toStats.firstActivityTime = block.timestamp;
            }

            // Генерируем событие для получателя
            emit StatsUpdated(
                to,
                toStats.totalTransactions,
                toStats.totalSent,
                toStats.totalReceived,
                toStats.externalTransactions
            );
        }
    }

    // Получение внешних транзакций с пагинацией
    function getExternalTransactions(
        address user,
        uint256 offset,
        uint256 limit
    ) public view returns (
        address[] memory froms,
        address[] memory tos,
        uint256[] memory values,
        uint256[] memory timestamps,
        bytes4[] memory methodIds,
        bool[] memory successes,
        uint256[] memory gasUsed
    ) {
        // Ограничиваем количество транзакций
        uint256 actualLimit = Math.min(
            Math.min(limit, MAX_TRANSACTIONS_PER_REQUEST),
            externalTransactions[user].length - offset
        );

        // Инициализируем массивы
        froms = new address[](actualLimit);
        tos = new address[](actualLimit);
        values = new uint256[](actualLimit);
        timestamps = new uint256[](actualLimit);
        methodIds = new bytes4[](actualLimit);
        successes = new bool[](actualLimit);
        gasUsed = new uint256[](actualLimit);

        // Заполняем массивы данными
        for (uint256 i = 0; i < actualLimit; i++) {
            ExternalTransaction storage txData = externalTransactions[user][offset + i];
            froms[i] = txData.from;
            tos[i] = txData.to;
            values[i] = txData.value;
            timestamps[i] = txData.timestamp;
            methodIds[i] = txData.methodId;
            successes[i] = txData.success;
            gasUsed[i] = txData.gasUsed;
        }

        return (froms, tos, values, timestamps, methodIds, successes, gasUsed);
    }

    // Получение количества внешних транзакций
    function getExternalTransactionCount(address user) public view returns (uint256) {
        return externalTransactions[user].length;
    }

    // Получение расширенной статистики аккаунта
    function getExtendedAccountStats(address user) public view returns (
        uint256 totalTx,
        uint256 totalSent,
        uint256 totalReceived,
        uint256 lastActivity,
        uint256 firstActivity,
        uint256 externalTx
    ) {
        AccountStats memory stats = accountStats[user];
        return (
            stats.totalTransactions,
            stats.totalSent,
            stats.totalReceived,
            stats.lastActivityTime,
            stats.firstActivityTime,
            stats.externalTransactions
        );
    }

}