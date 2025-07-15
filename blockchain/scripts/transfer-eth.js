const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {
  const [account0, account1] = await ethers.getSigners();

  console.log("\n🚀 Получаем начальные балансы...");
  const balance0Before = await ethers.provider.getBalance(account0.address);
  const balance1Before = await ethers.provider.getBalance(account1.address);

  console.log("\n💰 Начальные балансы:");
  console.log("Аккаунт 0:", ethers.formatEther(balance0Before), "ETH");
  console.log("Аккаунт 1:", ethers.formatEther(balance1Before), "ETH");

  const transferAmount = ethers.parseEther("100");
  console.log("\n Переводим 100 ETH с аккаунта 0 на аккаунт 1...");
  const tx1 = await account0.sendTransaction({
    to: account1.address,
    value: transferAmount
  });
  await tx1.wait();

  const balance0AfterFirst = await ethers.provider.getBalance(account0.address);
  const balance1AfterFirst = await ethers.provider.getBalance(account1.address);

  console.log("\n Балансы после первого трансфера:");
  console.log("Account 0:", ethers.formatEther(balance0AfterFirst), "ETH");
  console.log("Account 1:", ethers.formatEther(balance1AfterFirst), "ETH");

  console.log("\nПереводим 100 ETH обратно из account1 в account0...");
  const tx2 = await account1.sendTransaction({
    to: account0.address,
    value: transferAmount
  });
  await tx2.wait();

  const finalBalance0 = await ethers.provider.getBalance(account0.address);
  const finalBalance1 = await ethers.provider.getBalance(account1.address);

  console.log("\n💰 Финальные балансы:");
  console.log("Аккаунт 0:", ethers.formatEther(finalBalance0), "ETH");
  console.log("Аккаунт 1:", ethers.formatEther(finalBalance1), "ETH");
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });