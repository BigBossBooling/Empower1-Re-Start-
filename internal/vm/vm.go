package vm

// Package vm implements the WebAssembly (WASM) execution environment for
// EmPower1 smart contracts. It includes the WASM runtime, host function
// implementations bridging WASM to blockchain state (UTXO and Contract State Trie),
// and gas accounting for contract execution.
