package types

import (
	"crypto/sha256"
	"time"
)

// Hash represents a SHA256 hash.
type Hash [sha256.Size]byte

// Address represents a blockchain address, typically derived from a public key.
// For simplicity, using a byte slice. In a real scenario, this might be a fixed-size array or a more complex type.
type Address []byte

// TransactionType defines the type of a transaction.
// This maps to the TxType in our conceptual model.
type TransactionType uint8

const (
	// StandardTransaction for regular value transfers.
	StandardTransaction TransactionType = iota
	// StimulusTransaction for distributing stimulus payments.
	StimulusTransaction
	// TaxTransaction for collecting network taxes or fees for specific purposes.
	TaxTransaction
	// GovernanceTransaction for proposals, voting, etc.
	GovernanceTransaction
	// AIConsensusMetaTransaction for AI model updates or parameters related to consensus.
	AIConsensusMetaTransaction
)

// String returns a human-readable representation of the TransactionType.
func (tt TransactionType) String() string {
	switch tt {
	case StandardTransaction:
		return "Standard"
	case StimulusTransaction:
		return "Stimulus"
	case TaxTransaction:
		return "Tax"
	case GovernanceTransaction:
		return "Governance"
	case AIConsensusMetaTransaction:
		return "AIConsensusMeta"
	default:
		return "Unknown"
	}
}

// TransactionInput represents an input to a transaction.
// This is conceptual for UTXO-like models or for tracking provenance.
// For an account-based model, inputs might be implicitly the sender's account.
type TransactionInput struct {
	PreviousTxHash Hash    // Hash of the transaction from which these funds are spent
	OutputIndex    uint32  // Index of the output in the previous transaction
	Signature      []byte  // Signature of the spender, authorizing this input
	PublicKey      []byte  // Public key corresponding to the signature
}

// TransactionOutput represents an output from a transaction.
// This defines where value is being sent.
type TransactionOutput struct {
	RecipientAddress Address // Address of the recipient
	Amount           uint64  // Amount of EmPower1 Coin (PTCN) to transfer
}

// Transaction represents the fundamental unit of value transfer and state change.
// Aligns with the Basic Transaction Model from Phase 1.
type Transaction struct {
	TxID        Hash                `json:"txId"`        // Unique identifier for the transaction (hash of its content)
	Type        TransactionType     `json:"type"`        // Type of the transaction
	Inputs      []TransactionInput  `json:"inputs"`      // List of inputs (relevant for UTXO-like aspects)
	Outputs     []TransactionOutput `json:"outputs"`     // List of outputs
	Timestamp   time.Time           `json:"timestamp"`   // Time the transaction was created by the user
	Signature   []byte              `json:"signature"`   // Signature of the transaction sender (for account model) or combined inputs
	PublicKey   []byte              `json:"publicKey"`   // Sender's public key (for account model)
	Fee         uint64              `json:"fee"`         // Transaction fee
	Nonce       uint64              `json:"nonce"`       // Sender's nonce (for account model, to prevent replay attacks)
	Metadata    map[string][]byte   `json:"metadata"`    // Metadata for AI/ML logging, DApp data, etc.
	// Example: Metadata["ai_audit_ref"] = []byte("audit_log_id_xyz")
	// Example: Metadata["stimulus_batch_id"] = []byte("batch_abc")
}

// BlockHeader defines the structure of a block's header.
// Contains metadata about the block and links to the previous block.
type BlockHeader struct {
	Version           uint32    `json:"version"`           // Block version number
	PreviousBlockHash Hash      `json:"previousBlockHash"` // Hash of the preceding block
	MerkleRoot        Hash      `json:"merkleRoot"`        // Root hash of the Merkle tree of transactions in this block
	Timestamp         time.Time `json:"timestamp"`         // Time the block was mined/created
	Nonce             uint64    `json:"nonce"`             // Nonce used in Proof-of-Work or other consensus mechanisms
	Difficulty        uint64    `json:"difficulty"`        // Difficulty target for mining this block (relevant for PoW)
	Height            uint64    `json:"height"`            // Block height in the chain
	ProposerAddress   Address   `json:"proposerAddress"`   // Address of the block proposer/validator
}

// Block represents a single block in the blockchain.
// Contains the block header and a list of transactions.
type Block struct {
	Header       BlockHeader   `json:"header"`       // The header of the block
	Transactions []Transaction `json:"transactions"` // List of transactions included in this block
	BlockHash    Hash          `json:"blockHash"`    // Hash of the entire block (header + transactions)
}

// Account represents a user's account on the EmPower1 blockchain.
// This is a simplified representation for an account-based model.
type Account struct {
	Address         Address `json:"address"`         // Unique blockchain address
	Balance         uint64  `json:"balance"`         // Current balance of EmPower1 Coin (PTCN)
	Nonce           uint64  `json:"nonce"`           // Last transaction nonce for this account, prevents replay attacks
	ReputationScore int64   `json:"reputationScore"` // AI-assessed reputation score (can be positive or negative)
	// Other potential fields:
	// StakedAmount    uint64    `json:"stakedAmount"`
	// LastActivityBlock uint64 `json:"lastActivityBlock"`
	// DIDLink         string `json:"didLink"` // Link to a DID document
}

// TODO: Implement methods for hashing transactions and blocks.
// TODO: Implement signing and verification logic for transactions.
// TODO: Define canonical serialization for hashing (`prepareDataForHashing`).

// Note: The choice between UTXO and Account model (or a hybrid) for EmPower1
// will significantly influence the final structure of TransactionInput, TransactionOutput,
// and how balances are managed in the Account struct. For now, elements of both are present
// to allow flexibility as per Phase 1 conceptual discussions. The `Advanced_PoS_Consensus`
// will primarily interact with `Account` structures for staking and reputation.
// `EmPower1_Core_Node` will handle `Block` and `Transaction` processing.
// `Basic_Transaction_Model` is primarily embodied by the `Transaction` struct.
// `AIAuditLog` entries could be a specific `TransactionType` or stored in `Transaction.Metadata`.

// Further considerations for PoS:
// Validator struct (could be an extension of Account or separate)
// Stake struct (linking Account to staked amount and duration)
// RewardDistributionLog (could be a transaction type or metadata)
```
