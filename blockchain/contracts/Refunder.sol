// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Refunder {
    receive() external payable {
        (bool success, ) = msg.sender.call{value: msg.value}("");
        require(success, "Failed to send ETH back");
    }
} 