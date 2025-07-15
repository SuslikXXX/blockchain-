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
        [user2, user2, user3, user4], // 4 трансфера
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
    console.log("\nТестирование завершено!");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 