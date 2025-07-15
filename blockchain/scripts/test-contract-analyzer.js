const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {
  console.log("Deploying test token...");
  
  // Deploy token
  const Token = await ethers.getContractFactory("AnalyzerToken");
  const token = await Token.deploy("TestToken", "TST", ethers.parseEther("1000000"));
  await token.waitForDeployment();
  
  console.log("Token deployed to:", await token.getAddress());

  // Get signers
  const [owner, addr1, addr2] = await ethers.getSigners();
  
  // Make some transfers
  console.log("\nMaking test transfers...");
  
  // Transfer 1: owner -> addr1
  await token.transfer(addr1.address, ethers.parseEther("1000"));
  console.log("Transfer 1: Owner -> Addr1 (1000 tokens)");
  
  // Transfer 2: addr1 -> addr2 
  await token.connect(addr1).transfer(addr2.address, ethers.parseEther("500"));
  console.log("Transfer 2: Addr1 -> Addr2 (500 tokens)");
  
  // Transfer 3: addr2 -> owner
  await token.connect(addr2).transfer(owner.address, ethers.parseEther("250"));
  console.log("Transfer 3: Addr2 -> Owner (250 tokens)");

  console.log("\nTest transfers completed!");
  console.log("Token contract address:", await token.getAddress());
  console.log("\nNow you can use these addresses in the analyzer:");
  console.log("Owner:", owner.address);
  console.log("Address 1:", addr1.address);
  console.log("Address 2:", addr2.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 