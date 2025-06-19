package blockchain

import (
	"bytes"
	"crypto/sha256" // For zero hash comparison if needed
	"errors"
	"fmt"
	"sync"

	"empower1.com/empower1blockchain/internal/core"
	"empower1.com/empower1blockchain/internal/state"
)

var (
	ErrBlockNotFound      = errors.New("block not found")
	ErrInvalidBlockHeight = errors.New("invalid block height")
	ErrInvalidPrevHash    = errors.New("invalid previous block hash")
	ErrStateUpdateFailed  = errors.New("failed to update state from block")
	ErrBlockchainInit     = errors.New("blockchain initialization error")
)

// Blockchain manages the chain of blocks and interacts with the state manager.
// For V1, this is an in-memory blockchain.
type Blockchain struct {
	mu             sync.RWMutex
	blocks         []*core.Block // In-memory list of blocks
	blockByHashMap map[string]*core.Block // For quick hash-based lookups
	stateManager   *state.State
	// TODO: Add a proper logger instance, like in the State manager.
}

// NewBlockchain creates and returns a new Blockchain instance.
// It takes an initialized State manager.
func NewBlockchain(initialState *state.State) (*Blockchain, error) {
	if initialState == nil {
		return nil, fmt.Errorf("%w: state manager cannot be nil", ErrBlockchainInit)
	}
	bc := &Blockchain{
		blocks:         make([]*core.Block, 0),
		blockByHashMap: make(map[string]*core.Block),
		stateManager:   initialState,
	}
	// In the future, this is where we might load an existing chain from disk.
	// For now, it starts as an empty chain, ready for the genesis block.
	return bc, nil
}

// AddBlock validates a block and, if valid, adds it to the blockchain
// and updates the state through the stateManager.
func (bc *Blockchain) AddBlock(block *core.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Basic Validation
	if block == nil {
		return errors.New("cannot add nil block")
	}

	currentHeight := bc.currentHeightInternal() // Internal helper to avoid re-lock
	latestBlock := bc.getLatestBlockInternal()  // Internal helper

	if block.Height != currentHeight+1 {
		return fmt.Errorf("%w: expected height %d, got %d", ErrInvalidBlockHeight, currentHeight+1, block.Height)
	}

	// Note: The genesis block (block.Height == 0) specific PrevBlockHash check (e.g. must be all zeros or nil)
	// is implicitly handled. If currentHeight == -1 (meaning chain is empty), then latestBlock will be nil.
	// The PrevBlockHash check below only applies if latestBlock is not nil.
	// A robust system might have an explicit CreateGenesisBlock method that bypasses some of these checks.
	if latestBlock != nil { // Not the genesis block
		if !bytes.Equal(block.PrevBlockHash, latestBlock.Hash) {
			return fmt.Errorf("%w: expected prevHash %x, got %x", ErrInvalidPrevHash, latestBlock.Hash, block.PrevBlockHash)
		}
	} else { // This is the genesis block (currentHeight was -1)
		// Optional: Add specific checks for genesis block's PrevBlockHash if desired (e.g., must be nil or zero hash)
		// For example:
		 isZeroHash := true
		 if block.PrevBlockHash != nil && len(block.PrevBlockHash) > 0 {
			 for _, b := range block.PrevBlockHash {
				 if b != 0 {
					 isZeroHash = false
					 break
				 }
			 }
		 } else if block.PrevBlockHash != nil { // It's not nil, but it's empty (len 0), which is also fine for genesis
            // isZeroHash remains true
         }


		 if len(block.PrevBlockHash) > 0 && !isZeroHash { // Allow nil or empty or all-zero PrevBlockHash for genesis
			// This specific check using sha256.Size was commented out in the prompt, using a more general nil/empty/zero check
			// if !bytes.Equal(block.PrevBlockHash, make([]byte, sha256.Size)) { ... }
			// For now, we are more lenient for genesis: if there's a previous block, its hash must match.
            // If there's NO previous block, PrevBlockHash should be empty/nil/zero.
            // The existing logic (if latestBlock != nil) covers the non-genesis case.
            // This 'else' branch handles the genesis case. If PrevBlockHash is non-empty AND not all zeros, it's an error.
            // return fmt.Errorf("%w: genesis block must have zero or nil PrevBlockHash, got %x", ErrInvalidPrevHash, block.PrevBlockHash)
		 }
	}

	// Validate transactions and update state *before* adding block to the chain.
	// This ensures the block contains valid state transitions.
	err := bc.stateManager.UpdateStateFromBlock(block)
	if err != nil {
		return fmt.Errorf("%w: block %x (height %d): %v", ErrStateUpdateFailed, block.Hash, block.Height, err)
	}

	// If all validations pass and state update is successful:
	bc.blocks = append(bc.blocks, block)
	// Ensure block.Hash is not nil and has a value before using it as a map key.
	// The block should have its hash computed *before* being passed to AddBlock.
	if len(block.Hash) == 0 {
		// This indicates an issue with how the block was prepared before calling AddBlock.
		// For robustness, we could try to compute it here, but ideally, it's pre-computed.
		// For now, let's assume it's an error condition if block.Hash is not set.
		// However, the prompt implies block.Hash is already computed.
		return errors.New("block hash not computed before adding to blockchain")
	}
	bc.blockByHashMap[string(block.Hash)] = block

	// fmt.Printf("Blockchain: Added block Height %d, Hash %x\n", block.Height, block.Hash) // For debugging
	return nil
}

// GetBlockByHeight retrieves a block by its height.
func (bc *Blockchain) GetBlockByHeight(height int64) (*core.Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if height < 0 || height >= int64(len(bc.blocks)) {
		return nil, ErrBlockNotFound
	}
	return bc.blocks[height], nil
}

// GetBlockByHash retrieves a block by its hash.
func (bc *Blockchain) GetBlockByHash(hash []byte) (*core.Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	block, exists := bc.blockByHashMap[string(hash)]
	if !exists {
		return nil, ErrBlockNotFound
	}
	return block, nil
}

// currentHeightInternal is a non-locking helper for internal use.
func (bc *Blockchain) currentHeightInternal() int64 {
	if len(bc.blocks) == 0 {
		return -1 // Representing an empty chain, next block is height 0 (genesis)
	}
	return bc.blocks[len(bc.blocks)-1].Height
}

// CurrentHeight returns the height of the latest block in the blockchain.
// Height is 0-indexed, so a chain with one (genesis) block has height 0.
// Returns -1 if the chain is empty.
func (bc *Blockchain) CurrentHeight() int64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.currentHeightInternal()
}

// getLatestBlockInternal is a non-locking helper for internal use.
func (bc *Blockchain) getLatestBlockInternal() *core.Block {
	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

// GetLatestBlock returns the latest block in the blockchain.
// Returns nil if the chain is empty.
func (bc *Blockchain) GetLatestBlock() *core.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.getLatestBlockInternal()
}
