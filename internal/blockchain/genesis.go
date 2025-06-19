package blockchain

import (
	"bytes"
	"crypto/sha256"
	"errors" // Added for CalculateBlockHash error
	"fmt"    // Added for CreateGenesisBlock error wrapping
	"time"
	// "log" // For potential logging during genesis creation

	"empower1.com/empower1blockchain/internal/core"
	// "empower1.com/empower1blockchain/internal/state" // Not strictly needed if StateRoot is placeholder
)

// CalculateBlockHash computes a hash for a block.
// For V1, this is a simple hash of concatenated significant fields.
// In a real blockchain, this would be more rigorous and likely involve
// serializing the block header (or header + other relevant fields) into a canonical format.
// This helper is defined here for genesis block creation; a more robust version
// might live in internal/core/block.go as a method on *Block.
func CalculateBlockHash(block *core.Block) ([]byte, error) {
	// Hash of (HeaderForSigning() + ProposerAddress + Signature)
	// HeaderForSigning already includes: Height, Timestamp, PrevBlockHash, HashTransactions(), AIAuditLog
	// Note: User's Block struct comment says Hash is "calculated from header + proposer + signature"
	// and HeaderForSigning() already includes ProposerAddress and AIAuditLog if they are set before calling it.
	// Let's assume ProposerAddress and AIAuditLog are set on the block before this calculation.
	// And Signature is also set.

	// The current core.Block.HeaderForSigning() includes:
	// Height, Timestamp, PrevBlockHash, HashTransactions(Transactions), ProposerAddress (if set), AIAuditLog (if set)
	// So, we just need to append the Signature to what HeaderForSigning() returns.

	// To be very precise based on user's comment for Block.Hash:
    // "(calculated from header + proposer + signature)"
    // And HeaderForSigning includes ProposerAddress.
    // So, effectively: Hash(HeaderForSigningOutput + Signature)

	if block == nil {
		return nil, errors.New("cannot calculate hash of nil block")
	}

	headerBytes := block.HeaderForSigning() // This will use block.ProposerAddress and block.AIAuditLog

	var dataToHash []byte
	dataToHash = append(dataToHash, headerBytes...)
	if block.Signature != nil { // Signature should be included in the block's own hash
		dataToHash = append(dataToHash, block.Signature...)
	}

	hash := sha256.Sum256(dataToHash)
	return hash[:], nil
}


// CreateGenesisBlock creates and returns the genesis block for the EmPower1 Blockchain.
func CreateGenesisBlock() (*core.Block, error) {
	// log.Println("Creating Genesis Block...") // Optional logging

	genesisTimestamp := time.Date(2024, time.July, 8, 0, 0, 0, 0, time.UTC).UnixNano() // Example timestamp

	// For V1, genesis block has an empty transaction list.
	transactions := []core.Transaction{}

	// Placeholder for StateRoot (hash of initial empty state)
	// In a real system, this would be calculated from an initialized empty state.State manager.
	var emptyStateRoot [sha256.Size]byte // All zeros
	// Placeholder for AIAuditLog hash (hash of empty/initial AI audit log)
	var emptyAIAuditLogHash [sha256.Size]byte // All zeros


	// Conceptual Genesis Proposer and Signature
	// In a real PoS system, the genesis proposer might be defined by initial configuration
	// or be a special "protocol" address.
	genesisProposerAddress := []byte("EMPOWER1_GENESIS_PROPOSER_PUBKEY_PLACEHOLDER")
	genesisSignature := []byte("EMPOWER1_GENESIS_SIGNATURE_PLACEHOLDER_SIGNED_OVER_HEADER")


	genesisBlock := &core.Block{
		Height:          0,
		Timestamp:       genesisTimestamp,
		PrevBlockHash:   make([]byte, sha256.Size), // Zero hash for genesis PrevBlockHash
		Transactions:    transactions,
		ProposerAddress: genesisProposerAddress,
		// Signature and Hash are set after all other fields are populated.
		AIAuditLog:      emptyAIAuditLogHash[:],
		StateRoot:       emptyStateRoot[:],
	}

    // The block's HeaderForSigning() method uses ProposerAddress and AIAuditLog, so they must be set.
    // Now set the conceptual signature.
    genesisBlock.Signature = genesisSignature

	// Calculate and set the block's hash
	hash, err := CalculateBlockHash(genesisBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate genesis block hash: %w", err)
	}
	genesisBlock.Hash = hash

	// log.Printf("Genesis Block created. Hash: %x, Height: %d", genesisBlock.Hash, genesisBlock.Height)
	return genesisBlock, nil
}
