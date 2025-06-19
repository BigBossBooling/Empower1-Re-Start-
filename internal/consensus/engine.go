package consensus

import (
	"bytes" // Added for bytes.Equal
	"crypto/sha256" // Added for prevHash placeholder in proposeBlock
	"empower1.com/empower1blockchain/internal/core" // Added for core.Block
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
	proposerService    *ProposerService
	validationService  *ValidationService
	consensusState     *ConsensusState
	blockchain         *blockchain.Blockchain
	network            *network.SimulatedNetwork // Using SimulatedNetwork for V1
	nodeValidatorAddress []byte                  // Address of this node, if it's a validator

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
	nodeValAddr []byte, // New parameter
) *ConsensusEngine {
	if ps == nil || vs == nil || cs == nil || bc == nil || net == nil {
		return nil // Or return an error
	}
	return &ConsensusEngine{
		proposerService:    ps,
		validationService:  vs,
		consensusState:     cs,
		blockchain:         bc,
		network:            net,
		nodeValidatorAddress: nodeValAddr,
		stopChan:           make(chan struct{}),
	}
}

// attemptBlockProposal checks if this node is the current proposer and initiates block creation.
func (ce *ConsensusEngine) attemptBlockProposal() {
	currentHeight := ce.blockchain.CurrentHeight()
	nextHeight := currentHeight + 1

	proposer, err := ce.consensusState.GetProposerForHeight(nextHeight)
	if err != nil {
		log.Printf("CONSENSUS_ENGINE: Error determining proposer for height %d: %v", nextHeight, err)
		return
	}

	if bytes.Equal(ce.nodeValidatorAddress, proposer.Address) {
		log.Printf("CONSENSUS_ENGINE: Node %x is the proposer for height %d.", ce.nodeValidatorAddress, nextHeight)
		ce.proposeBlock(nextHeight)
	} else {
		// log.Printf("CONSENSUS_ENGINE: Node %x is NOT the proposer for height %d. Expected proposer: %x", ce.nodeValidatorAddress, nextHeight, proposer.Address)
	}
}

// Start begins the consensus engine's main operational loop.
func (ce *ConsensusEngine) Start() {
	log.Println("CONSENSUS_ENGINE: Starting...")
	ce.wg.Add(1)

	go func() { // Main engine loop
		defer ce.wg.Done()
		// ce.attemptBlockProposal() // Initial check - might propose before genesis if not careful with height logic

		// TODO: Replace ticker with more sophisticated round/slot timing mechanism.
		slotTicker := time.NewTicker(10 * time.Second) // Example block/slot time
		defer slotTicker.Stop()

		blockChan := ce.network.GetBlockReceptionChannel() // Get the channel

		// Perform an initial proposal check if this node might be the genesis proposer
		// This depends on whether genesis block is pre-set or proposed.
		// Assuming genesis is added before engine starts, first proposal is for height 1.
		if ce.blockchain.CurrentHeight() >= 0 { // Ensure genesis exists before first proposal attempt
			ce.attemptBlockProposal()
		}


		for {
			select {
			case <-ce.stopChan:
				log.Println("CONSENSUS_ENGINE: Received stop signal, shutting down engine loop...")
				return
			case <-slotTicker.C:
				log.Println("CONSENSUS_ENGINE: Slot tick...")
				ce.attemptBlockProposal()
			case serializedBlock, ok := <-blockChan: // Listen for new blocks
				if !ok {
					log.Println("CONSENSUS_ENGINE: Block reception channel closed. Exiting loop.")
					return // Channel closed
				}
				log.Printf("CONSENSUS_ENGINE: Received potential block data from network (size: %d bytes)", len(serializedBlock))
				block, err := core.DeserializeBlock(serializedBlock)
				if err != nil {
					log.Printf("CONSENSUS_ENGINE_ERROR: Failed to deserialize received block: %v", err)
					continue // Skip this message
				}
				ce.processIncomingBlock(block)
			}
		}
	}()
	// TODO: Register network message handlers here for other message types, e.g.,
	// ce.network.RegisterMessageHandler(ce.handleGenericNetworkMessage)
	log.Println("CONSENSUS_ENGINE: Started successfully.")
}

// Stop signals the consensus engine to shut down gracefully.
func (ce *ConsensusEngine) Stop() {
	log.Println("CONSENSUS_ENGINE: Stopping...")
	close(ce.stopChan)
	ce.wg.Wait()
	log.Println("CONSENSUS_ENGINE: Stopped successfully.")
}

// proposeBlock is called when it's this node's turn to propose.
func (ce *ConsensusEngine) proposeBlock(height int64) {
    log.Printf("CONSENSUS_ENGINE: Node %x attempting to propose block for height %d.", ce.nodeValidatorAddress, height)

    var prevHash []byte
    latestBlock := ce.blockchain.GetLatestBlock()
    // This check is crucial: only propose for the *next* height.
    // currentHeight in attemptBlockProposal already does this, so height should be correct.
    if latestBlock == nil && height == 0 { // Proposing genesis
        prevHash = make([]byte, sha256.Size)
    } else if latestBlock != nil && height == latestBlock.Height+1 {
        prevHash = latestBlock.Hash
    } else {
        log.Printf("CONSENSUS_ENGINE: ProposeBlock height mismatch or no valid predecessor. Expected height %d, latest block height %d. Aborting proposal.",
            height, func() int64 { if latestBlock != nil { return latestBlock.Height } return -1 }())
        return
    }

    proposedBlock, err := ce.proposerService.CreateProposalBlock(height, prevHash, ce.nodeValidatorAddress)
    if err != nil {
        log.Printf("CONSENSUS_ENGINE: Failed to create proposal block for height %d: %v", height, err)
        return
    }
    if proposedBlock == nil {
        log.Printf("CONSENSUS_ENGINE: ProposerService returned nil block for height %d.", height)
        return
    }
    // Ensure the block's proposer address is correctly set to this node's address
    // CreateProposalBlock should handle this, but a check or re-set could be here if needed.
    // proposedBlock.ProposerAddress = ce.nodeValidatorAddress
    // And then re-sign and re-hash if ProposerAddress influences hashing/signing directly in Block methods.
    // For now, assume ProposerService does this correctly.

    log.Printf("CONSENSUS_ENGINE: Successfully created proposal block %x for height %d with %d transactions.", proposedBlock.Hash, proposedBlock.Height, len(proposedBlock.Transactions))

    serializedBlock, err := proposedBlock.Serialize()
    if err != nil {
        log.Printf("CONSENSUS_ENGINE: Failed to serialize proposed block %x: %v", proposedBlock.Hash, err)
        return
    }

    ce.network.Broadcast("NEW_BLOCK", serializedBlock)
    log.Printf("CONSENSUS_ENGINE: Broadcasted proposed block %x (size: %d bytes) for height %d.", proposedBlock.Hash, len(serializedBlock), proposedBlock.Height)

    // For V1 simplicity, we add our own block to our chain immediately after broadcasting.
    // In more complex consensus (e.g., with voting), this would happen after confirmation.
    log.Printf("CONSENSUS_ENGINE: Adding locally proposed block %x to own chain (V1 behavior).", proposedBlock.Hash)
    err = ce.blockchain.AddBlock(proposedBlock) // This calls stateManager.UpdateStateFromBlock
    if err != nil {
        log.Printf("CONSENSUS_ENGINE_CRITICAL: Failed to add own validly proposed block %x to chain: %v", proposedBlock.Hash, err)
        // This should ideally not happen if proposerService and validationService (if called prior) are correct.
    } else {
        log.Printf("CONSENSUS_ENGINE: Successfully added locally proposed block %x to own chain. New height: %d", proposedBlock.Hash, ce.blockchain.CurrentHeight())
        ce.consensusState.UpdateHeight(proposedBlock.Height) // Notify ConsensusState of the new height
    }
}

// processIncomingBlock handles blocks received from the network.
func (ce *ConsensusEngine) processIncomingBlock(block *core.Block) {
    if block == nil {
        log.Println("CONSENSUS_ENGINE: Received nil block to process.")
        return
    }
    log.Printf("CONSENSUS_ENGINE: Processing incoming block %x for height %d from proposer %x", block.Hash, block.Height, block.ProposerAddress)

    // Avoid processing our own block again if we added it directly in proposeBlock
    // This check might be more nuanced in a real system (e.g., based on broadcast source or if block is already known)
    if bytes.Equal(block.ProposerAddress, ce.nodeValidatorAddress) && ce.blockchain.GetLatestBlock() != nil && bytes.Equal(ce.blockchain.GetLatestBlock().Hash, block.Hash) {
        log.Printf("CONSENSUS_ENGINE: Ignoring processing of own block %x already added to chain.", block.Hash)
        return
    }


    // 1. Validate block (ValidationService.ValidateBlock is still a placeholder for deep consensus checks)
    // Basic structural validation and PrevHash/Height checks are done by blockchain.AddBlock.
    // ValidationService would perform more consensus-specific checks (e.g., proposer eligibility, signature).
    err := ce.validationService.ValidateBlock(block) // Placeholder for V1 - currently does minimal checks
    if err != nil {
        log.Printf("CONSENSUS_ENGINE_ERROR: Block %x validation failed via ValidationService: %v", block.Hash, err)
        return
    }
    log.Printf("CONSENSUS_ENGINE: Block %x passed ValidationService checks (placeholder).", block.Hash)

    // 2. If valid, attempt to add to blockchain
    // blockchain.AddBlock itself contains further validation (height, prevHash)
    // and calls stateManager.UpdateStateFromBlock.
    err = ce.blockchain.AddBlock(block)
    if err != nil {
        log.Printf("CONSENSUS_ENGINE_ERROR: Failed to add block %x to blockchain: %v", block.Hash, err)
        // TODO: Handle fork situations or other reasons for AddBlock failure more gracefully.
        //       This could involve storing orphaned blocks, requesting missing parents, etc.
        return
    }
    log.Printf("CONSENSUS_ENGINE: Successfully added block %x to blockchain. New height: %d", block.Hash, ce.blockchain.CurrentHeight())

    // 3. Update consensus state with new height
    ce.consensusState.UpdateHeight(block.Height)

    // TODO: If this node was the proposer of this block (and it wasn't self-added in proposeBlock),
    // it might confirm its proposal.
    // TODO: Trigger next round of consensus or new proposer selection based on new chain state.
    // TODO: Clean mempool of transactions included in this block.
}

// handleNetworkMessage would be a method on ConsensusEngine to process messages from the network.
// func (ce *ConsensusEngine) handleNetworkMessage(peerID string, messageType string, data []byte) {
//     log.Printf("CONSENSUS_ENGINE: Received message from %s - Type: %s", peerID, messageType)
//     // Process different message types (e.g., new block, vote, proposal)
//     // If messageType == "NEW_BLOCK", deserialize data to core.Block and call processIncomingBlock
// }
