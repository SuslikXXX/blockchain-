const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {
  console.log("Тестирование анализа транзакций контракта...");

  // Получаем аккаунты
  const [owner, addr1, addr2] = await ethers.getSigners();

  // Деплоим тестовый токен
  const Token = await ethers.getContractFactory("Token");
  const token = await Token.deploy();
  await token.deployed();
  console.log("Токен развернут по адресу:", token.address);

  // Выполняем несколько транзакций
  console.log("Выполняем тестовые транзакции...");

  // Минтим токены
  const mintTx = await token.mint(owner.address, ethers.utils.parseEther("1000"));
  await mintTx.wait();
  console.log("Токены созданы");

  // Делаем transfer
  const transferTx = await token.transfer(addr1.address, ethers.utils.parseEther("100"));
  await transferTx.wait();
  console.log("Выполнен transfer");

  // Делаем approve и transferFrom
  const approveTx = await token.approve(addr2.address, ethers.utils.parseEther("50"));
  await approveTx.wait();
  console.log("Выполнен approve");

  // Подключаемся к контракту от имени addr2
  const tokenAsAddr2 = token.connect(addr2);
  const transferFromTx = await tokenAsAddr2.transferFrom(owner.address, addr2.address, ethers.utils.parseEther("50"));
  await transferFromTx.wait();
  console.log("Выполнен transferFrom");

  console.log("Все тестовые транзакции выполнены");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 