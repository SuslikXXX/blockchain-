const hre = require("hardhat");

async function main() {
  // Деплоим Token контракт
  const Token = await hre.ethers.getContractFactory("Token");
  const token = await Token.deploy("TestToken", "TST", 1000000); // 1M tokens

  await token.waitForDeployment();

  const address = await token.getAddress();
  console.log("Token deployed to:", address);

  // Получаем аккаунты для трансферов
  const [owner, addr1, addr2, addr3, addr4] = await hre.ethers.getSigners();
  
  console.log("\n🚀 Выполняем серию ERC20 трансферов для демонстрации анализатора...");
  
  // Проверяем начальный баланс
  const initialBalance = await token.balanceOf(owner.address);
  console.log(`💰 Начальный баланс owner: ${hre.ethers.formatEther(initialBalance)} TST`);

  // Выполняем серию трансферов от owner к разным адресам
  const initialTransfers = [
    { to: addr1.address, amount: "500", name: "Addr1" },
    { to: addr2.address, amount: "750", name: "Addr2" },
    { to: addr3.address, amount: "300", name: "Addr3" },
    { to: addr4.address, amount: "200", name: "Addr4" },
  ];

  for (let i = 0; i < initialTransfers.length; i++) {
    const transfer = initialTransfers[i];
    const amount = hre.ethers.parseEther(transfer.amount);
    
    console.log(`\n🔄 Transfer ${i + 1}: ${transfer.amount} TST -> ${transfer.name} (${transfer.to})`);
    
    const tx = await token.transfer(transfer.to, amount);
    await tx.wait();
    
    console.log(`✅ Успешно! TxHash: ${tx.hash}`);
    
    // Пауза между трансферами для лучшей демонстрации
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  // Выполняем трансферы между пользователями
  console.log("\n🔄 Трансферы между пользователями:");
  
  const userTransfers = [
    { from: addr1, to: addr2.address, amount: "100", fromName: "Addr1", toName: "Addr2" },
    { from: addr2, to: addr3.address, amount: "200", fromName: "Addr2", toName: "Addr3" },
    { from: addr3, to: addr4.address, amount: "50", fromName: "Addr3", toName: "Addr4" },
    { from: addr4, to: addr1.address, amount: "75", fromName: "Addr4", toName: "Addr1" },
  ];

  for (let i = 0; i < userTransfers.length; i++) {
    const transfer = userTransfers[i];
    const amount = hre.ethers.parseEther(transfer.amount);
    
    console.log(`\n🔄 User Transfer ${i + 1}: ${transfer.amount} TST -> ${transfer.fromName} → ${transfer.toName}`);
    
    const tx = await token.connect(transfer.from).transfer(transfer.to, amount);
    await tx.wait();
    
    console.log(`✅ Успешно! TxHash: ${tx.hash}`);
    
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  // Показываем итоговые балансы
  console.log("\n📊 Итоговые балансы:");
  const accounts = [
    { name: "Owner", addr: owner.address },
    { name: "Addr1", addr: addr1.address },
    { name: "Addr2", addr: addr2.address },
    { name: "Addr3", addr: addr3.address },
    { name: "Addr4", addr: addr4.address },
  ];

  for (const account of accounts) {
    const balance = await token.balanceOf(account.addr);
    console.log(`${account.name}: ${hre.ethers.formatEther(balance)} TST`);
  }

  console.log("\n🎉 Все трансферы выполнены!");
  console.log("📋 Контракт:", address);
  console.log("🔍 Теперь можете проверить базу данных анализатора на наличие ERC20 трансферов.");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 