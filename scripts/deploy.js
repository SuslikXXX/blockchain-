const hre = require("hardhat");

async function main() {
  // Деплоим Token контракт
  const Token = await hre.ethers.getContractFactory("Token");
  const token = await Token.deploy("TestToken", "TST", 1000000); // 1M tokens

  await token.waitForDeployment();

  const address = await token.getAddress();
  console.log("Token deployed to:", address);

  // Выполняем тестовый transfer
  const [owner, addr1] = await hre.ethers.getSigners();
  
  const transferAmount = hre.ethers.parseEther("1000");
  const tx = await token.transfer(addr1.address, transferAmount);
  await tx.wait();
  
  console.log(`Transferred ${hre.ethers.formatEther(transferAmount)} tokens to ${addr1.address}`);
  console.log(`Transaction hash: ${tx.hash}`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 