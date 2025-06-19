package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt" // For error wrapping
	"log"
	"os"
	"os/signal"
	"syscall"

	"empower1.com/empower1blockchain/internal/blockchain"
	"empower1.com/empower1blockchain/internal/consensus"
	"empower1.com/empower1blockchain/internal/mempool"
	"empower1.com/empower1blockchain/internal/network"
	"empower1.com/empower1blockchain/internal/state"
)

func runNode() (*consensus.ConsensusEngine, error) {
	log.Println("Initializing EmPower1 Node components...")

	// Initialize State Manager
	stateManager, err := state.NewState()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}
	log.Println("State manager initialized successfully.")

	// Initialize Blockchain
	chain, err := blockchain.NewBlockchain(stateManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blockchain: %w", err)
	}
	log.Println("Blockchain initialized successfully.")

	// Create Genesis Block if chain is empty
	if chain.CurrentHeight() == -1 {
		log.Println("Blockchain is empty. Creating Genesis Block...")
		genesisBlock, err := blockchain.CreateGenesisBlock()
		if err != nil {
			return nil, fmt.Errorf("failed to create genesis block: %w", err)
		}
		log.Printf("Genesis block created successfully. Hash: %x", genesisBlock.Hash)
		err = chain.AddBlock(genesisBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to add genesis block to chain: %w", err)
		}
		log.Println("Genesis block added to chain and processed successfully.")
	} else {
		log.Printf("Blockchain already initialized. Current height: %d", chain.CurrentHeight())
	}

	// Initialize ConsensusState
	log.Println("Initializing Consensus State...")
	consensusState := consensus.NewConsensusState()
	if consensusState == nil { // Added nil check for robustness, though NewConsensusState doesn't return error
		return nil, fmt.Errorf("NewConsensusState returned nil")
	}
	dummyValidators := []*consensus.Validator{
		{Address: []byte("validator_address_1_placeholder"), Stake: 10000, Reputation: 1.0},
		{Address: []byte("validator_address_2_placeholder"), Stake: 15000, Reputation: 0.9},
	}
	consensusState.LoadInitialValidators(dummyValidators)
	log.Printf("Consensus State initialized and %d dummy validators loaded.", len(dummyValidators))

	// Initialize SimulatedNetwork
	log.Println("Initializing Simulated Network...")
	simNetNodeID := "empower1d_node_001"
	simNet := network.NewSimulatedNetwork(simNetNodeID)
	if simNet == nil { // Added nil check
		return nil, fmt.Errorf("NewSimulatedNetwork returned nil")
	}
	simNet.RegisterMessageHandler(func(peerID string, messageType string, data []byte) {
		log.Printf("MAIN_HANDLER (via SimNet): Received msg from %s - Type: %s, Data: %x", peerID, messageType, data)
	})
	log.Printf("Simulated Network initialized for Node ID: %s.", simNetNodeID)

	// Initialize Mempool
	log.Println("Initializing Mempool...")
	txMempool := mempool.NewMempool()
	if txMempool == nil { // Added nil check
		return nil, fmt.Errorf("NewMempool returned nil")
	}
	log.Printf("Mempool initialized successfully. Current count: %d", txMempool.Count())

	// Initialize ProposerService
	log.Println("Initializing Proposer Service...")
	dummyProposerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dummy proposer key: %w", err)
	}
	proposerService := consensus.NewProposerService(dummyProposerKey, txMempool, chain, consensusState)
	if proposerService == nil {
		return nil, fmt.Errorf("failed to initialize proposer service (NewProposerService returned nil)")
	}
	log.Println("Proposer Service initialized successfully.")

	// Initialize ValidationService
	log.Println("Initializing Validation Service...")
	validationService := consensus.NewValidationService(consensusState, chain)
	if validationService == nil {
		return nil, fmt.Errorf("failed to initialize validation service (NewValidationService returned nil)")
	}
	log.Println("Validation Service initialized successfully.")

	// Initialize ConsensusEngine
	log.Println("Initializing Consensus Engine...")
	// Use one of the dummy validator addresses for this node's identity for proposing
	thisNodeValidatorAddress := dummyValidators[0].Address
	consensusEngine := consensus.NewConsensusEngine(
		proposerService,
		validationService,
		consensusState,
		chain,
		simNet,
		thisNodeValidatorAddress, // Add this new parameter
	)
	if consensusEngine == nil {
		return nil, fmt.Errorf("failed to initialize consensus engine (NewConsensusEngine returned nil)")
	}
	log.Println("Consensus Engine initialized successfully.")

	// Start the Consensus Engine
	go consensusEngine.Start()
	log.Println("Consensus Engine started.")
	log.Printf("EmPower1 Blockchain node components initialized and engine started. Current chain height: %d", chain.CurrentHeight())

	return consensusEngine, nil
}

func main() {
	log.Println("Starting EmPower1 Blockchain Node (empower1d)...")

	consensusEngine, err := runNode()
	if err != nil {
		log.Fatalf("Node initialization failed: %v", err)
	}

	log.Println("Node running... Press Ctrl+C to stop.")
	// Setup graceful shutdown
	// Create a channel to receive OS signals
	shutdownChannel := make(chan os.Signal, 1)
	// Notify this channel for os.Interrupt (Ctrl+C) and syscall.SIGTERM (kill command)
	signal.Notify(shutdownChannel, os.Interrupt, syscall.SIGTERM)

	// Block main goroutine until a signal is received
	sig := <-shutdownChannel
	log.Printf("Caught signal: %v. Starting graceful shutdown...", sig)

	// Stop the consensus engine
	// The consensusEngine.Stop() method should block until its goroutines are done.
	if consensusEngine != nil {
		log.Println("Attempting to stop Consensus Engine...")
		consensusEngine.Stop()
		log.Println("Consensus Engine stopped.")
	}

	// TODO: Add shutdown for other services if they have explicit stop methods (e.g., network connections)

	log.Println("EmPower1 Blockchain node shut down gracefully.")
}
