const hre = require("hardhat");

async function main() {
  console.log("=== Тестирование анализатора блокчейна ===");
  
  // Получаем аккаунты для тестирования
  const [owner, addr1, addr2, addr3, addr4] = await hre.ethers.getSigners();
  
  console.log("Адреса для тестирования:");
  console.log("Owner:", owner.address);
  console.log("Addr1:", addr1.address);
  console.log("Addr2:", addr2.address);
  console.log("Addr3:", addr3.address);
  console.log("Addr4:", addr4.address);
  
  // Деплоим новый токен для тестирования
  console.log("\n🚀 Деплой токена...");
  const Token = await hre.ethers.getContractFactory("Token");
  const token = await Token.deploy("AnalyzerTestToken", "ATT", 10000000); // 10M tokens
  await token.waitForDeployment();
  
  const tokenAddress = await token.getAddress();
  console.log("✅ Token задеплоен:", tokenAddress);
  
  // Проверяем баланс owner
  const ownerBalance = await token.balanceOf(owner.address);
  console.log(`💰 Баланс owner: ${hre.ethers.formatEther(ownerBalance)} ATT`);
  
  console.log("\n📊 Начинаем создание множественных транзакций...");
  console.log("Это должно создать много блоков для тестирования анализатора\n");
  
  const transactions = [];
  const transferAmount = hre.ethers.parseEther("1000"); // 1000 tokens per transfer
  
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
  
  console.log("\n📈 Статистика транзакций:");
  console.log(`✅ Всего создано транзакций: ${transactions.length}`);
  console.log(`🔢 Диапазон блоков: ${Math.min(...transactions.map(t => t.block))} - ${Math.max(...transactions.map(t => t.block))}`);
  console.log(`📊 Количество блоков: ${Math.max(...transactions.map(t => t.block)) - Math.min(...transactions.map(t => t.block)) + 1}`);
  
  console.log("\n🎯 Информация для анализатора:");
  console.log(`📍 Адрес контракта: ${tokenAddress}`);
  console.log("🔍 Анализатор должен обработать все эти блоки и транзакции");
  
  // Показываем финальные балансы
  console.log("\n💰 Финальные балансы:");
  const finalBalances = await Promise.all([
    token.balanceOf(owner.address),
    token.balanceOf(addr1.address),
    token.balanceOf(addr2.address),
    token.balanceOf(addr3.address),
    token.balanceOf(addr4.address)
  ]);
  
  console.log(`Owner: ${hre.ethers.formatEther(finalBalances[0])} ATT`);
  console.log(`Addr1: ${hre.ethers.formatEther(finalBalances[1])} ATT`);
  console.log(`Addr2: ${hre.ethers.formatEther(finalBalances[2])} ATT`);
  console.log(`Addr3: ${hre.ethers.formatEther(finalBalances[3])} ATT`);
  console.log(`Addr4: ${hre.ethers.formatEther(finalBalances[4])} ATT`);
  
  console.log("\n🏁 Тестирование завершено!");
  console.log("💡 Теперь запустите анализатор с адресом контракта:", tokenAddress);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 