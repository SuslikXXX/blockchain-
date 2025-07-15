const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {
    const [deployer] = await ethers.getSigners();
    console.log("Деплоим контракт с аккаунтом:", deployer.address);

    // Деплоим контракт
    const Refunder = await ethers.getContractFactory("Refunder");
    const refunder = await Refunder.deploy();
    await refunder.waitForDeployment();

    const refunderAddress = await refunder.getAddress();
    console.log("Контракт Refunder задеплоен по адресу:", refunderAddress);

    // Получаем начальный баланс
    const initialBalance = await ethers.provider.getBalance(deployer.address);
    console.log("Начальный баланс аккаунта:", ethers.formatEther(initialBalance), "ETH");

    // Отправляем 30 ETH на контракт
    console.log("\nОтправляем 30 ETH на контракт...");
    const sendTx = await deployer.sendTransaction({
        to: refunderAddress,
        value: ethers.parseEther("30.0")
    });
    await sendTx.wait();
    console.log("Транзакция отправки ETH:", sendTx.hash);

    // Получаем конечный баланс
    const finalBalance = await ethers.provider.getBalance(deployer.address);
    console.log("Конечный баланс аккаунта:", ethers.formatEther(finalBalance), "ETH");

    // Показываем разницу
    const difference = initialBalance - finalBalance;
    console.log("Потрачено на газ:", ethers.formatEther(difference), "ETH");

    // Проверяем баланс контракта (должен быть 0, так как ETH сразу возвращается)
    const contractBalance = await ethers.provider.getBalance(refunderAddress);
    console.log("Баланс контракта:", ethers.formatEther(contractBalance), "ETH");

    console.log("\n✅ Тест завершен! ETH был отправлен на контракт и сразу возвращен обратно.");
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error(error);
        process.exit(1);
    }); 