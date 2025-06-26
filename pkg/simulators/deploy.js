const { ethers } = require("hardhat");

async function main() {
  // Получаем фабрику контракта
  const Token = await ethers.getContractFactory("Token");
  
  // Деплоим контракт
  const token = await Token.deploy(
    "TestToken",           // name
    "TST",                // symbol  
    ethers.utils.parseEther("1000000") // 1M tokens
  );

  await token.deployed();

  console.log("Token deployed to:", token.address);
  
  // Выполняем тестовый transfer
  const [owner, addr1] = await ethers.getSigners();
  
  const transferAmount = ethers.utils.parseEther("1000");
  const tx = await token.transfer(addr1.address, transferAmount);
  await tx.wait();
  
  console.log(`Transferred ${ethers.utils.formatEther(transferAmount)} tokens to ${addr1.address}`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 