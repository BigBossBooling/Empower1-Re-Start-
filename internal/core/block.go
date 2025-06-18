package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"time" // Required for NewBlock
	// "fmt" // Not used in the provided Block methods
	// "log" // Not used
	// "os"  // Not used
)

// Block represents a block in the EmPower1 Blockchain.
// It's the primary unit of data storage and state transition.
type Block struct {
	Height          int64         `json:"height"`          // Block height (index in the chain)
	Timestamp       int64         `json:"timestamp"`       // Unix nanoseconds timestamp of block creation
	PrevBlockHash   []byte        `json:"prevBlockHash"`   // Hash of the previous block
	Transactions    []Transaction `json:"transactions"`    // List of transactions included in this block
	ProposerAddress []byte        `json:"proposerAddress"` // Public key or identifier of the validator who proposed this block
	Signature       []byte        `json:"signature"`       // Cryptographic signature of the block header by the proposer
	Hash            []byte        `json:"hash"`            // The cryptographic hash of this block (calculated from header + proposer + signature)

	// --- EmPower1 Specific Fields for AI/ML & State Commitment ---
	AIAuditLog      []byte        `json:"aiAuditLog,omitempty"` // Hash of an AI/ML audit log related to this block's redistribution/validation (conceptual)
	// StateRoot is a Merkle/Patricia Trie root hash of the entire blockchain state (e.g., UTXO set + account states)
	// This commits the state after applying this block's transactions, crucial for light clients and verification.
	StateRoot       []byte        `json:"stateRoot"`            // Merkle/Patricia Trie root hash of the entire state post-block application
}

// NewBlock creates a new block without the final Hash, ProposerAddress, Signature, or StateRoot.
// These fields are typically populated by the proposer and the state manager/consensus mechanism.
// This design aligns with an "Iterate Intelligently" approach, separating concerns.
func NewBlock(height int64, prevBlockHash []byte, transactions []Transaction) *Block {
	block := &Block{
		Height:        height,
		Timestamp:     time.Now().UnixNano(), // Set current time on creation
		PrevBlockHash: prevBlockHash,
		Transactions:  transactions,
		// Hash, ProposerAddress, Signature, AIAuditLog, and StateRoot are populated later.
	}
	return block
}

// HeaderForSigning returns the byte representation of the block's content that gets signed by the proposer.
// This ensures cryptographic integrity. It explicitly EXCLUDES the final Hash and Signature itself,
// as well as the StateRoot (which is committed *after* the block's transactions are applied to the state).
// Order matters greatly for cryptographic integrity and deterministic hashing.
func (b *Block) HeaderForSigning() []byte {
    var buf bytes.Buffer
    binary.Write(&buf, binary.BigEndian, b.Height)
    binary.Write(&buf, binary.BigEndian, b.Timestamp)
    buf.Write(b.PrevBlockHash)

    // Concatenate all transaction IDs (or a Merkle Root of them) for signing
    // Using HashTransactions() which creates a single hash from all tx IDs.
    txsHash := b.HashTransactions()
    buf.Write(txsHash)

    // ProposerAddress needs to be set *before* calling this for self-signing
    // Ensure ProposerAddress is not nil to avoid panic, though in a real scenario
    // it should always be set before signing.
    if b.ProposerAddress != nil {
        buf.Write(b.ProposerAddress)
    }

	// EmPower1 Specific: Include AI Audit Log in what's signed for transparency
	// This implies AIAuditLog is generated before signing.
    // Ensure AIAuditLog is not nil to avoid panic.
    if b.AIAuditLog != nil {
	    buf.Write(b.AIAuditLog)
    }

    return buf.Bytes()
}

// HashTransactions creates a single hash from all transaction IDs in the block.
// In a production blockchain, this would typically be a Merkle Root.
// TODO: Implement proper Merkle Root calculation for transactions.
func (b *Block) HashTransactions() []byte {
    if len(b.Transactions) == 0 {
        emptyHash := sha256.Sum256([]byte{})
        return emptyHash[:]
    }

    var txHashes [][]byte
    for _, tx := range b.Transactions {
        if len(tx.ID) == 0 {
            // This case should ideally not happen if transactions are finalized
            // and signed before being added to a block.
            // For robustness in this V1, hash a placeholder if ID is missing.
            // In a stricter system, this might be an error.
            placeholderForMissingID := sha256.Sum256([]byte("MISSING_TX_ID"))
            txHashes = append(txHashes, placeholderForMissingID[:])
        } else {
            txHashes = append(txHashes, tx.ID)
        }
    }

    // Simple concatenation and hash for V1 - not a proper Merkle root
    // TODO: Implement sorting of txHashes before concatenation for determinism.
    var combinedTxHashes []byte
    for _, txHash := range txHashes {
        combinedTxHashes = append(combinedTxHashes, txHash...)
    }
    finalHash := sha256.Sum256(combinedTxHashes)
    return finalHash[:]
}

// TODO: Implement SetHash(hash []byte) method
// TODO: Implement Sign(privateKey []byte) error method (requires crypto library for signing)
// TODO: Implement VerifySignature(publicKey []byte) (requires crypto library for verification)
