const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {
  const [sender, receiver] = await ethers.getSigners();


  console.log("\n🚀 Деплой токена...");
  const Token = await ethers.getContractFactory("AnalyzerToken");
  const token = await Token.deploy(
      "Test Token",
      "TST",
      1000000       //1M
  );
  await token.waitForDeployment();

  const tokenAddress = await token.getAddress();
  console.log("✅ Token задеплоен:", tokenAddress);

  // Получаем балансы до трансфера
  const decimals = await token.decimals();
  const balance0Before = await token.balanceOf(sender.address);
  const balance1Before = await token.balanceOf(receiver.address);

  console.log(`Отправитель: ${sender.address}`, " Баланс: ", ethers.formatUnits(balance0Before, decimals));
  console.log(`Получатель: ${receiver.address}`, " Баланс: ", ethers.formatUnits(balance1Before, decimals));

  const transferAmount = ethers.parseUnits("10000", decimals);
  console.log("\n Переводим 10000 токенов с аккаунта 0 на аккаунт 1...");
  const tx1 = await token.transfer(receiver.address, transferAmount);
  await tx1.wait();

  // Получаем обновленные балансы после первого трансфера
  const balance0AfterFirst = await token.balanceOf(sender.address);
  const balance1AfterFirst = await token.balanceOf(receiver.address);

  console.log("\n Балансы после первого трансфера:");
  console.log("Jn:", ethers.formatUnits(balance0AfterFirst, decimals));
  console.log("Account 1:", ethers.formatUnits(balance1AfterFirst, decimals));

  // Переводим 50,000 токенов обратно
  console.log("\nПереводим 5000 токенов обратно из account1 в account0...");
  const returnAmount = ethers.parseUnits("5000", decimals);
  const tx2 = await token.connect(receiver).transfer(sender.address, returnAmount);
  await tx2.wait();

  // Получаем финальные балансы
  const finalBalance0 = await token.balanceOf(sender.address);
  const finalBalance1 = await token.balanceOf(receiver.address);

  console.log(" 📍 Адрес контракта: ", tokenAddress);

  console.log("\n💰 Финальные балансы:");
  console.log("Аккаунт 0:", ethers.formatUnits(finalBalance0, decimals));
  console.log("Аккаунт 1:", ethers.formatUnits(finalBalance1, decimals));
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });