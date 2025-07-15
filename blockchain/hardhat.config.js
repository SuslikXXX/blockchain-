require("@nomicfoundation/hardhat-toolbox");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.20",
  networks: {
    hardhat: {
      mining: {
        auto: true,
        interval: 1000
      },
      // Включаем поддержку фильтрации логов
      allowUnlimitedContractSize: true,
      loggingEnabled: true,
      chainId: 31337
    },
    localhost: {
      url: "http://127.0.0.1:8545",
      mining: {
        auto: true,
        interval: 1000
      },
      // Включаем поддержку фильтрации логов
      allowUnlimitedContractSize: true,
      loggingEnabled: true,
      chainId: 31337
    }
  }
};
