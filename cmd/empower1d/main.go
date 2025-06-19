package main

import (
	"log" // Using log package for output

	"empower1.com/empower1blockchain/internal/blockchain"
	"empower1.com/empower1blockchain/internal/state"
)

func main() {
	log.Println("Starting EmPower1 Blockchain Node (empower1d)...")

	// Initialize State Manager
	stateManager, err := state.NewState()
	if err != nil {
		log.Fatalf("Failed to initialize state manager: %v", err)
	}
	log.Println("State manager initialized successfully.")

	// Initialize Blockchain
	chain, err := blockchain.NewBlockchain(stateManager)
	if err != nil {
		log.Fatalf("Failed to initialize blockchain: %v", err)
	}
	log.Println("Blockchain initialized successfully.")

	// Create Genesis Block if chain is empty
    // (NewBlockchain currently always creates an empty chain, so this check is more for future persistence)
	if chain.CurrentHeight() == -1 {
		log.Println("Blockchain is empty. Creating Genesis Block...")
		genesisBlock, err := blockchain.CreateGenesisBlock()
		if err != nil {
			log.Fatalf("Failed to create genesis block: %v", err)
		}
		log.Printf("Genesis block created successfully. Hash: %x", genesisBlock.Hash)

		// Add and process Genesis Block
		// Note: The genesis block's StateRoot is a placeholder.
		// A full system would:
		// 1. Create genesis transactions (if any).
		// 2. Initialize state with these transactions (or an empty state).
		// 3. Calculate the true StateRoot from this initial state.
		// 4. Set this true StateRoot in the genesisBlock before adding it.
		// For V1, CreateGenesisBlock() sets a zero StateRoot, and UpdateStateFromBlock for an empty block is trivial.
		err = chain.AddBlock(genesisBlock)
		if err != nil {
			log.Fatalf("Failed to add genesis block to chain: %v", err)
		}
		log.Println("Genesis block added to chain and processed successfully.")
	} else {
        log.Printf("Blockchain already initialized. Current height: %d", chain.CurrentHeight())
    }

	log.Printf("EmPower1 Blockchain node initialization sequence complete. Current chain height: %d", chain.CurrentHeight())
	// TODO: Implement further node startup logic: P2P networking, RPC server, consensus engine start, etc.
	log.Println("Node running (conceptual)... further implementation needed.")
}
