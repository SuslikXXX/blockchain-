const { ethers } = require("hardhat");

async function deployToken() {
  const Token = await ethers.getContractFactory("Token");
  const token = await Token.deploy(
    "TestToken",           
    "TST",                
    ethers.utils.parseEther("1000000")
  );
  await token.deployed();
  console.log("Token deployed to:", token.address);
  return token;
}

async function simpleTransfer(token, from, to, amount) {
  console.log(`\n=== Simple Transfer ===`);
  console.log(`From: ${from.address}`);
  console.log(`To: ${to.address}`);
  console.log(`Amount: ${ethers.utils.formatEther(amount)} TST`);
  
  const tx = await token.connect(from).transfer(to.address, amount);
  await tx.wait();
  
  const balance = await token.balanceOf(to.address);
  console.log(`New balance of ${to.address}: ${ethers.utils.formatEther(balance)} TST`);
  
  return tx;
}

async function batchTransfers(token, from, recipients, amounts) {
  console.log(`\n=== Batch Transfers ===`);
  console.log(`From: ${from.address}`);
  console.log(`Recipients: ${recipients.length}`);
  
  const txs = [];
  for (let i = 0; i < recipients.length; i++) {
    console.log(`Transfer ${i + 1}: ${ethers.utils.formatEther(amounts[i])} TST to ${recipients[i].address}`);
    const tx = await token.connect(from).transfer(recipients[i].address, amounts[i]);
    await tx.wait();
    txs.push(tx);
  }
  
  return txs;
}

async function approveAndTransferFrom(token, owner, spender, recipient, amount) {
  console.log(`\n=== Approve and TransferFrom ===`);
  console.log(`Owner: ${owner.address}`);
  console.log(`Spender: ${spender.address}`);
  console.log(`Recipient: ${recipient.address}`);
  console.log(`Amount: ${ethers.utils.formatEther(amount)} TST`);
  
  // Approve
  const approveTx = await token.connect(owner).approve(spender.address, amount);
  await approveTx.wait();
  console.log(`Approved ${ethers.utils.formatEther(amount)} TST for spender`);
  
  // Check allowance
  const allowance = await token.allowance(owner.address, spender.address);
  console.log(`Allowance: ${ethers.utils.formatEther(allowance)} TST`);
  
  // TransferFrom
  const transferTx = await token.connect(spender).transferFrom(owner.address, recipient.address, amount);
  await transferTx.wait();
  console.log(`Transferred ${ethers.utils.formatEther(amount)} TST from ${owner.address} to ${recipient.address}`);
  
  const balance = await token.balanceOf(recipient.address);
  console.log(`New balance of ${recipient.address}: ${ethers.utils.formatEther(balance)} TST`);
  
  return { approveTx, transferTx };
}

async function multipleApprovals(token, owner, spenders, amounts) {
  console.log(`\n=== Multiple Approvals ===`);
  console.log(`Owner: ${owner.address}`);
  
  const txs = [];
  for (let i = 0; i < spenders.length; i++) {
    console.log(`Approving ${ethers.utils.formatEther(amounts[i])} TST for ${spenders[i].address}`);
    const tx = await token.connect(owner).approve(spenders[i].address, amounts[i]);
    await tx.wait();
    txs.push(tx);
  }
  
  return txs;
}

async function checkBalances(token, addresses) {
  console.log(`\n=== Balance Check ===`);
  for (const address of addresses) {
    const balance = await token.balanceOf(address);
    console.log(`${address}: ${ethers.utils.formatEther(balance)} TST`);
  }
}

async function main() {
  const [owner, addr1, addr2, addr3, addr4] = await ethers.getSigners();
  
  // Deploy token
  const token = await deployToken();
  
  // Check initial balances
  await checkBalances(token, [owner.address, addr1.address, addr2.address, addr3.address]);
  
  // Simple transfer
  await simpleTransfer(token, owner, addr1, ethers.utils.parseEther("1000"));
  
  // Batch transfers
  await batchTransfers(
    token, 
    owner, 
    [addr2, addr3, addr4], 
    [
      ethers.utils.parseEther("500"),
      ethers.utils.parseEther("750"),
      ethers.utils.parseEther("250")
    ]
  );
  
  // Approve and transferFrom
  await approveAndTransferFrom(
    token, 
    owner, 
    addr1, 
    addr2, 
    ethers.utils.parseEther("100")
  );
  
  // Multiple approvals
  await multipleApprovals(
    token,
    owner,
    [addr1, addr2, addr3],
    [
      ethers.utils.parseEther("200"),
      ethers.utils.parseEther("300"),
      ethers.utils.parseEther("400")
    ]
  );
  
  // Final balance check
  await checkBalances(token, [owner.address, addr1.address, addr2.address, addr3.address, addr4.address]);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  }); 