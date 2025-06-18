package state

// Package state manages the global state of the EmPower1 Blockchain.
// This includes the UTXO set, smart contract state (Contract State Trie),
// account balances (derived from UTXOs), and the logic for applying
// transactions to update the state, ultimately producing the Block.StateRoot.
