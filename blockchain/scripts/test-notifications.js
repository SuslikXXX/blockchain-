const hre = require("hardhat");
const { ethers } = require("hardhat");
require('dotenv').config({ path: '../../backend/.env' });

async function main() {
    console.log("Начинаем тестирование контракта с множественными трансферами...");
  
    // Получаем аккаунты
    const [owner, user1, user2, user3, user4] = await ethers.getSigners();
    console.log("Owner address:", owner.address);
    console.log("User1 address:", user1.address);
    console.log("User2 address:", user2.address);
    console.log("User3 address:", user3.address);
    console.log("User4 address:", user4.address);
  
    // Подключаемся к существующему контракту
    const contractAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
    const Token = await ethers.getContractFactory("AnalyzerToken");
    const token = await Token.attach(contractAddress);
    console.log("Connected to Token at:", await token.getAddress());

    // Распределяем токены
    const initialAmount = ethers.parseEther("10000");
    await token.transfer(user1.address, initialAmount);
    await token.transfer(user2.address, initialAmount);
    await token.transfer(user3.address, initialAmount);
    await token.transfer(user4.address, initialAmount);
    console.log("Токены распределены между пользователями");
  
    // Функция для паузы
    const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

    // Функция для выполнения серии трансферов от одного пользователя
    async function performTransferSeries(from, recipients, amount) {
        const tokenWithSigner = token.connect(from);
        console.log(`\nНачинаем серию трансферов от ${from.address}:`);
    
        for (const recipient of recipients) {
            const tx = await tokenWithSigner.transfer(recipient.address, amount);
    await tx.wait();
            console.log(`- Трансфер к ${recipient.address}: ${ethers.formatEther(amount)} токенов`);
            // Небольшая пауза между транзакциями
            await sleep(1000);
        }
    }

    // Сценарий 1: Множественные трансферы от user1 (должен вызвать уведомление)
    console.log("\nСценарий 1: Быстрые трансферы от user1");
    const transferAmount = ethers.parseEther("100");
    await performTransferSeries(
        user1, 
        [user2, user3, user4, user2, user3], // 5 трансферов
        transferAmount
    );

    // Ждем 2 секунды
    console.log("\nЖдем 2 секунды...");
    await sleep(2000);

    // Сценарий 2: Множественные трансферы от user2 (должен вызвать уведомление)
    console.log("\nСценарий 2: Быстрые трансферы от user2");
    await performTransferSeries(
        user2,
        [user1, user3, user4, user1], // 4 трансфера
        transferAmount
    );

    // Сценарий 3: Параллельные трансферы от разных пользователей
    console.log("\nСценарий 3: Параллельные трансферы");
    await Promise.all([
        token.connect(user1).transfer(user4.address, transferAmount),
        token.connect(user2).transfer(user3.address, transferAmount),
        token.connect(user3).transfer(user2.address, transferAmount),
        token.connect(user4).transfer(user1.address, transferAmount)
    ]);

    // Проверяем балансы
    async function checkBalance(address, name) {
        const balance = await token.balanceOf(address);
        console.log(`Баланс ${name}: ${ethers.formatEther(balance)} токенов`);
    }

    console.log("\nФинальные балансы:");
    await checkBalance(user1.address, "User1");
    await checkBalance(user2.address, "User2");
    await checkBalance(user3.address, "User3");
    await checkBalance(user4.address, "User4");

    // Получаем статистику аккаунтов
    async function checkAccountStats(address, name) {
        const [totalTx, totalSent, totalReceived, lastActivity, firstActivity] = await token.getAccountStats(address);
        console.log(`\nСтатистика ${name}:`);
        console.log(`- Всего транзакций: ${totalTx.toString()}`);
        console.log(`- Отправлено: ${ethers.formatEther(totalSent)} токенов`);
        console.log(`- Получено: ${ethers.formatEther(totalReceived)} токенов`);
        console.log(`- Последняя активность: ${new Date(Number(lastActivity) * 1000).toISOString()}`);
        console.log(`- Первая активность: ${new Date(Number(firstActivity) * 1000).toISOString()}`);
    }

    console.log("\nСтатистика аккаунтов:");
    await checkAccountStats(user1.address, "User1");
    await checkAccountStats(user2.address, "User2");
    await checkAccountStats(user3.address, "User3");
    await checkAccountStats(user4.address, "User4");

    console.log("\nТестирование завершено!");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 