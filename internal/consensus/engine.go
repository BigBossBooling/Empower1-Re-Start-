package consensus

import (
	"log"
	"sync" // For WaitGroup in Stop()
	"time"

	"empower1.com/empower1blockchain/internal/blockchain"
	"empower1.com/empower1blockchain/internal/network"
	// ProposerService, ValidationService, ConsensusState are in the same 'consensus' package.
	// Mempool is not directly used by engine, but by ProposerService.
)

// ConsensusEngine orchestrates the consensus process, including block proposal,
// validation, and interaction with the network and blockchain.
type ConsensusEngine struct {
	proposerService   *ProposerService
	validationService *ValidationService
	consensusState    *ConsensusState
	blockchain        *blockchain.Blockchain
	network           *network.SimulatedNetwork // Using SimulatedNetwork for V1

	stopChan chan struct{}    // Signals the engine's main loop to stop
	wg       sync.WaitGroup // Waits for goroutines to finish
	// TODO: Add more fields like current validator status, timers for rounds/epochs
	// TODO: Add a proper logger instance.
}

// NewConsensusEngine creates a new ConsensusEngine instance.
func NewConsensusEngine(
	ps *ProposerService,
	vs *ValidationService,
	cs *ConsensusState,
	bc *blockchain.Blockchain,
	net *network.SimulatedNetwork,
) *ConsensusEngine {
	if ps == nil || vs == nil || cs == nil || bc == nil || net == nil {
		return nil // Or return an error
	}
	return &ConsensusEngine{
		proposerService:   ps,
		validationService: vs,
		consensusState:    cs,
		blockchain:        bc,
		network:           net,
		stopChan:          make(chan struct{}),
	}
}

// Start begins the consensus engine's main operational loop.
// For V1, this is a placeholder loop that logs periodically.
// TODO: Implement actual PoS consensus logic (rounds, proposer selection, voting, block finalization).
func (ce *ConsensusEngine) Start() {
	log.Println("CONSENSUS_ENGINE: Starting...")
	ce.wg.Add(1) // Add to WaitGroup for the main loop goroutine

	go func() {
		defer ce.wg.Done() // Decrement counter when goroutine finishes
		ticker := time.NewTicker(10 * time.Second) // Example: log every 10 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ce.stopChan:
				log.Println("CONSENSUS_ENGINE: Received stop signal, shutting down main loop...")
				return
			case <-ticker.C:
				log.Printf("CONSENSUS_ENGINE: Running... Current chain height: %d. (Placeholder tick)", ce.blockchain.CurrentHeight())
				// Conceptual:
				// 1. Check if it's our turn to propose (based on ConsensusState, current height/slot)
				// 2. If so, call proposerService.CreateProposalBlock(...)
				// 3. Broadcast the new block via network.Broadcast(...)
				// 4. Handle incoming blocks/messages via network message handler (registered by engine)
				//    - Validate block using validationService.ValidateBlock(...)
				//    - If valid, attempt blockchain.AddBlock(...)
			}
		}
	}()
	// TODO: Register network message handlers here, e.g.,
	// ce.network.RegisterMessageHandler(ce.handleNetworkMessage)
	log.Println("CONSENSUS_ENGINE: Started successfully.")
}

// Stop signals the consensus engine to shut down gracefully.
func (ce *ConsensusEngine) Stop() {
	log.Println("CONSENSUS_ENGINE: Stopping...")
	close(ce.stopChan) // Signal the main loop to stop
	ce.wg.Wait()       // Wait for all goroutines (e.g., main loop) to complete
	log.Println("CONSENSUS_ENGINE: Stopped successfully.")
}

// handleNetworkMessage would be a method on ConsensusEngine to process messages from the network.
// func (ce *ConsensusEngine) handleNetworkMessage(peerID string, messageType string, data []byte) {
//     log.Printf("CONSENSUS_ENGINE: Received message from %s - Type: %s", peerID, messageType)
//     // Process different message types (e.g., new block, vote, proposal)
// }
