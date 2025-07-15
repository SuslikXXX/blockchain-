const hre = require("hardhat");

async function main() {
  console.log("=== Тестирование анализатора блокчейна ===");
  
  // Получаем аккаунты для тестирования
  const [owner, addr1, addr2, addr3, addr4] = await hre.ethers.getSigners();
  
  console.log("Адреса для тестирования:");
  console.log("Адрес владельца:", owner.address);
  console.log("Адрес аккаунта 1:", addr1.address);
  console.log("Адрес аккаунта 2:", addr2.address);
  console.log("Адрес аккаунта 3:", addr3.address);
  console.log("Адрес аккаунта 4:", addr4.address);

  console.log("\n🚀 Деплой токена...");
  const Token = await hre.ethers.getContractFactory("AnalyzerToken");
  const token = await Token.deploy(
    "Test Token", 
    "TST", 
    10000000 //10M
  );
  await token.waitForDeployment();
  
  const tokenAddress = await token.getAddress();
  console.log("✅ Token задеплоен:", tokenAddress);
 
  const decimals = await token.decimals();
  ownerBalance = await token.balanceOf(owner.address);
  console.log(`💰 Баланс owner: ${hre.ethers.formatEther(ownerBalance)} TST`);
  
  const transactions = [];
  const transferAmount = hre.ethers.parseEther("1000");
  
  try {
    // Транзакция 1: Owner -> Addr1
    console.log("🔄 Транзакция 1: Owner -> Addr1");
    let tx = await token.transfer(addr1.address, transferAmount);
    let receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 2: Owner -> Addr2
    console.log("🔄 Транзакция 2: Owner -> Addr2");
    tx = await token.transfer(addr2.address, transferAmount);
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 3: Owner -> Addr3
    console.log("🔄 Транзакция 3: Owner -> Addr3");
    tx = await token.transfer(addr3.address, transferAmount);
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 4: Owner -> Addr4
    console.log("🔄 Транзакция 4: Owner -> Addr4");
    tx = await token.transfer(addr4.address, transferAmount);
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 5: Addr1 -> Addr2
    console.log("🔄 Транзакция 5: Addr1 -> Addr2");
    tx = await token.connect(addr1).transfer(addr2.address, hre.ethers.parseEther("100"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 6: Addr2 -> Addr3
    console.log("🔄 Транзакция 6: Addr2 -> Addr3");
    tx = await token.connect(addr2).transfer(addr3.address, hre.ethers.parseEther("200"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 7: Addr3 -> Addr4
    console.log("🔄 Транзакция 7: Addr3 -> Addr4");
    tx = await token.connect(addr3).transfer(addr4.address, hre.ethers.parseEther("300"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 8: Addr4 -> Addr1
    console.log("🔄 Транзакция 8: Addr4 -> Addr1");
    tx = await token.connect(addr4).transfer(addr1.address, hre.ethers.parseEther("150"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 9: Owner -> Addr1 (большая сумма)
    console.log("🔄 Транзакция 9: Owner -> Addr1 (большая сумма)");
    tx = await token.transfer(addr1.address, hre.ethers.parseEther("5000"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 10: Addr1 -> Owner (возврат части)
    console.log("🔄 Транзакция 10: Addr1 -> Owner (возврат части)");
    tx = await token.connect(addr1).transfer(owner.address, hre.ethers.parseEther("1000"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 11: Массовый transfer Owner -> Addr2
    console.log("🔄 Транзакция 11: Массовый transfer Owner -> Addr2");
    tx = await token.transfer(addr2.address, hre.ethers.parseEther("2000"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 12: Циклический transfer Addr2 -> Addr3
    console.log("🔄 Транзакция 12: Циклический transfer Addr2 -> Addr3");
    tx = await token.connect(addr2).transfer(addr3.address, hre.ethers.parseEther("500"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
    // Транзакция 13: Финальный transfer Addr3 -> Owner
    console.log("🔄 Транзакция 13: Финальный transfer Addr3 -> Owner");
    tx = await token.connect(addr3).transfer(owner.address, hre.ethers.parseEther("800"));
    receipt = await tx.wait();
    transactions.push({ hash: tx.hash, block: receipt.blockNumber });
    console.log(`   ✅ Hash: ${tx.hash}, Block: ${receipt.blockNumber}`);
    
  } catch (error) {
    console.error("❌ Ошибка при выполнении транзакции:", error);
    throw error;
  }
  
  console.log(`📍 Адрес контракта: ${tokenAddress}`);
  
  // Показываем финальные балансы
  console.log("\n💰 Финальные балансы:");

// Get all balances first (await each one)
  ownerBalance = await token.balanceOf(owner.address);
  const addr1Balance = await token.balanceOf(addr1.address);
  const addr2Balance = await token.balanceOf(addr2.address);
  const addr3Balance = await token.balanceOf(addr3.address);
  const addr4Balance = await token.balanceOf(addr4.address);

  console.log("Аккаунта владельца:", ethers.formatUnits(ownerBalance, decimals));
  console.log("Аккаунт 1:", ethers.formatUnits(addr1Balance, decimals));
  console.log("Аккаунт 2:", ethers.formatUnits(addr2Balance, decimals));
  console.log("Аккаунт 3:", ethers.formatUnits(addr3Balance, decimals));
  console.log("Аккаунт 4:", ethers.formatUnits(addr4Balance, decimals));

}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 