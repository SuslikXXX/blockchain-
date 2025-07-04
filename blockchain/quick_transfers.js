const hre = require("hardhat");

async function main() {
  // Подключаемся к уже задеплоенному контракту
  const contractAddress = "0x70e0bA845a1A0F2DA3359C97E0285013525FFC49";
  const Token = await hre.ethers.getContractFactory("Token");
  const token = Token.attach(contractAddress);
  
  console.log("🚀 Подключились к Token контракту:", contractAddress);
  
  const [owner, addr1, addr2, addr3, addr4] = await hre.ethers.getSigners();
  
  console.log("💸 Создаем новые трансферы для анализатора...");
  
  // Создаем серию новых трансферов
  const newTransfers = [
    { from: owner, to: addr1.address, amount: "100", name: "Owner->Addr1" },
    { from: addr1, to: addr2.address, amount: "50", name: "Addr1->Addr2" },
    { from: addr2, to: addr3.address, amount: "25", name: "Addr2->Addr3" },
    { from: addr3, to: addr4.address, amount: "10", name: "Addr3->Addr4" },
    { from: addr4, to: owner.address, amount: "5", name: "Addr4->Owner" },
  ];
  
  for (let i = 0; i < newTransfers.length; i++) {
    const transfer = newTransfers[i];
    const amount = hre.ethers.parseEther(transfer.amount);
    
    console.log(`\n🔄 Transfer ${i + 1}: ${transfer.name} - ${transfer.amount} TST`);
    
    const tx = await token.connect(transfer.from).transfer(transfer.to, amount);
    await tx.wait();
    
    console.log(`✅ TxHash: ${tx.hash}`);
    
    // Пауза между трансферами
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  
  console.log("\n🎉 Новые трансферы созданы! Backend анализатор должен их подхватить.");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 