const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {

  console.log("\n🚀 Деплой токена...");
  const Token = await ethers.getContractFactory("AnalyzerToken");
  const token = await Token.deploy(
      "Test Token",
      "TST",
      1000000       // 1M
  );
  await token.waitForDeployment();

  const tokenAddress = await token.getAddress();
  console.log("✅ Token задеплоен:", tokenAddress);

}

main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });