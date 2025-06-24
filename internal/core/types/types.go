package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	internalerrors "empower1/internal/errors"
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

// Validate checks the structural validity of the Account.
func (a *Account) Validate() error {
	if a.Address == nil || len(a.Address) == 0 {
		return internalerrors.ErrInvalidAccountAddress
	}
	// Conceptual: Add address length check if a fixed length is decided.
	// if len(a.Address) != ExpectedAddressLength {
	//     return internalerrors.ErrInvalidAddressLength
	// }

	// Balance and Nonce are uint64, their validity is mostly inherent in the type (non-negative).
	// Specific protocol rules (e.g., max balance) are not checked here.

	// Validate ReputationScore bounds.
	const minReputation = -1000000 // Example bounds
	const maxReputation = 1000000  // Example bounds
	if a.ReputationScore < minReputation || a.ReputationScore > maxReputation {
		return fmt.Errorf("reputation score %d is out of range [%d, %d]: %w",
			a.ReputationScore, minReputation, maxReputation, internalerrors.ErrInvalidReputationScore)
	}

	return nil
}

// TODO: Implement methods for hashing transactions and blocks.
// TODO: Implement signing and verification logic for transactions.
// TODO: Define canonical serialization for hashing (`prepareDataForHashing`).

// ToJSON marshals the Block to a JSON byte slice.
func (b *Block) ToJSON() ([]byte, error) {
	return json.Marshal(b)
}

// FromJSON unmarshals a JSON byte slice into the Block.
func (b *Block) FromJSON(data []byte) error {
	return json.Unmarshal(data, b)
}

// ToJSON marshals the Transaction to a JSON byte slice.
func (tx *Transaction) ToJSON() ([]byte, error) {
	return json.Marshal(tx)
}

// FromJSON unmarshals a JSON byte slice into the Transaction.
func (tx *Transaction) FromJSON(data []byte) error {
	return json.Unmarshal(data, tx)
}

// Validate checks the structural validity of the Block.
// It orchestrates validation of its Header and Transactions.
func (b *Block) Validate() error {
	// Validate Header
	// Assuming Header is a direct struct member, not a pointer, so it cannot be nil.
	// If it were a pointer: if b.Header == nil { return fmt.Errorf("block header cannot be nil: %w", internalerrors.ErrBlockHeaderValidationFailed) }
	if err := b.Header.Validate(); err != nil {
		return fmt.Errorf("block header validation failed: %w", err) // No need to wrap with ErrBlockHeaderValidationFailed if already specific
	}

	// Validate Transactions
	// Transactions slice can be nil or empty (e.g., for blocks with no user transactions).
	if b.Transactions != nil {
		if len(b.Transactions) > 10000 { // Example: Arbitrary limit on max transactions
			return fmt.Errorf("block exceeds maximum transaction count (10000): %w", internalerrors.ErrMaxTransactionsPerBlockExceeded)
		}
		for i, tx := range b.Transactions {
			if err := tx.Validate(); err != nil {
				return fmt.Errorf("transaction %d in block failed validation: %w", i, err) // No need to wrap with ErrBlockTransactionFailed if already specific
			}
		}
	}
	// Conceptual: Add check for total block size limit if defined. This would require serializing the block.

	// Validate BlockHash
	var zeroHash Hash
	if b.BlockHash == zeroHash {
		return internalerrors.ErrInvalidBlockHash
	}
	// Conceptual: Validating that BlockHash is the correct hash of the block content
	// is a separate, more complex operation usually done during block acceptance.

	return nil
}

// Validate checks the structural validity of the BlockHeader.
func (bh *BlockHeader) Validate() error {
	// Assuming CurrentBlockVersion is a defined constant, e.g., const CurrentBlockVersion uint32 = 1
	// This constant would ideally be defined globally or passed in if versions change often.
	// For now, let's assume a conceptual CurrentBlockVersion for the check.
	const CurrentBlockVersion uint32 = 1 // Example, should be a global constant

	if bh.Version != CurrentBlockVersion && bh.Version != 0 { // Allow 0 for potential initial/unspecified version
		// A more robust check might involve a list of supported versions.
		// return fmt.Errorf("unsupported block version %d: %w", bh.Version, internalerrors.ErrInvalidBlockVersion)
		// For simplicity with current error structure:
		if bh.Version != CurrentBlockVersion { // If we only support one version strictly other than an initial 0
			return internalerrors.ErrInvalidBlockVersion
		}
	}

	var zeroHash Hash
	if bh.Height > 0 && bh.PreviousBlockHash == zeroHash {
		return internalerrors.ErrInvalidPreviousBlockHash
	}
	// For genesis block (Height 0), PreviousBlockHash must be zero.
	if bh.Height == 0 && bh.PreviousBlockHash != zeroHash {
		return fmt.Errorf("genesis block (height 0) must have zero PreviousBlockHash: %w", internalerrors.ErrInvalidPreviousBlockHash)
	}


	if bh.MerkleRoot == zeroHash {
		return internalerrors.ErrInvalidMerkleRoot
	}

	if bh.Timestamp.IsZero() {
		return internalerrors.ErrZeroTimestamp // More generic zero time error
	}
	// Conceptual: Add range check for timestamp (e.g., not too far in future/past from network time)
	// This often requires access to current network time or median of past blocks, making it
	// not purely stateless. A basic sanity check:
	// const EmPower1EpochStartUnix = 1609459200 // Example: Jan 1, 2021 UTC
	// if bh.Timestamp.Unix() < EmPower1EpochStartUnix {
	//    return fmt.Errorf("block timestamp precedes EmPower1 epoch: %w", internalerrors.ErrInvalidBlockTimestamp)
	// }
	// if bh.Timestamp.After(time.Now().Add(2 * time.Hour)) { // Max 2 hours in future
	//    return fmt.Errorf("block timestamp too far in future: %w", internalerrors.ErrInvalidBlockTimestamp)
	// }


	if bh.ProposerAddress == nil || len(bh.ProposerAddress) == 0 {
		return internalerrors.ErrMissingProposerAddress
	}
	// Conceptual: Add address length/format check for ProposerAddress
	// if len(bh.ProposerAddress) != ExpectedAddressLength {
	//    return internalerrors.ErrInvalidAddressLength
	// }

	// Nonce and Difficulty validation depends heavily on the consensus mechanism (PoW vs PoS).
	// For PoW, Difficulty > 0 would be a rule.
	// For PoS, these might have different meanings or default values.
	// Example for PoW:
	// if isPoWConsensus && bh.Difficulty == 0 {
	//    return internalerrors.ErrInvalidDifficulty
	// }

	// Height is uint64, so always >= 0. Specific check for Height == 0 is tied to PreviousBlockHash.

	return nil
}

// Validate checks the structural and semantic validity of the Transaction.
// This is a stateless validation, meaning it does not consult the blockchain state
// (e.g., to check if inputs are unspent or if an account has sufficient balance).
func (tx *Transaction) Validate() error {
	// A. Basic Structure & Required Fields
	var zeroHash Hash
	if tx.TxID == zeroHash {
		return internalerrors.ErrInvalidTransactionID
	}

	// Check transaction type validity
	if tx.Type > AIConsensusMetaTransaction { // Assuming AIConsensusMetaTransaction is the max valid enum value
		return internalerrors.ErrUnknownTransactionType
	}

	if tx.Timestamp.IsZero() {
		return internalerrors.ErrZeroTimestamp
	}
	// Conceptual: Add timestamp range check (e.g., not too far in future/past)
	// if tx.Timestamp.After(time.Now().Add(2 * time.Hour)) {
	// 	return internalerrors.ErrInvalidTimestampRange
	// }

	// Outputs must not be nil or empty for most transactions.
	// Specific types might allow no outputs (e.g. pure data transaction - to be defined)
	if tx.Outputs == nil || len(tx.Outputs) == 0 {
		// Exception: A specific TxType could allow no outputs.
		// Example: if tx.Type == SomeDataOnlyTxType { /* allow */ } else { return error }
		// For now, assume most common transactions require outputs.
		return internalerrors.ErrNoTransactionOutputs
	}

	// B. Inputs Validation (if applicable to Type)
	// Most standard transactions require inputs. Some system transactions (issuances) might not.
	isSpendingTx := tx.Type == StandardTransaction // Add other types that spend existing state

	if isSpendingTx {
		if tx.Inputs == nil || len(tx.Inputs) == 0 {
			return internalerrors.ErrEmptyInputsForSpendingTx
		}
		for i, input := range tx.Inputs {
			if err := input.Validate(); err != nil {
				// Wrap the error for more context
				return fmt.Errorf("input %d validation failed: %w", i, internalerrors.ErrTransactionInputValidationFailed)
			}
		}
	} else { // Not a typical spending transaction (e.g., stimulus, coinbase-like)
		if tx.Inputs != nil && len(tx.Inputs) > 0 {
			// Inputs are not expected for this type
			return internalerrors.ErrUnexpectedInputsForTxType
		}
	}

	// C. Outputs Validation
	for i, output := range tx.Outputs {
		if err := output.Validate(); err != nil {
			// Wrap the error
			return fmt.Errorf("output %d validation failed: %w", i, internalerrors.ErrTransactionOutputValidationFailed)
		}
	}

	// D. Signature & PublicKey (if applicable to account-based Type)
	// Most transactions will be account-based and require these.
	// Exceptions could be system-generated txs or specific UTXO-only internal ops.
	requiresSignature := true // Default, adjust based on type if needed
	if tx.Type == AIConsensusMetaTransaction { // Example: system tx might not need user sig
		// requiresSignature = false
	}

	if requiresSignature {
		if tx.Signature == nil || len(tx.Signature) == 0 {
			return internalerrors.ErrMissingSignature
		}
		if tx.PublicKey == nil || len(tx.PublicKey) == 0 {
			return internalerrors.ErrMissingPublicKey
		}
		// Conceptual: Add length checks for signature and public key if defined
		// if len(tx.Signature) > MaxSigLen { return internalerrors.ErrInvalidSignatureLength }
		// if len(tx.PublicKey) != ExpectedPubKeyLen { return internalerrors.ErrInvalidPublicKeyLength }
	}

	// E. Metadata Validation
	if tx.Metadata != nil {
		if len(tx.Metadata) > 50 { // Arbitrary limit on number of metadata entries
			return fmt.Errorf("too many metadata entries (max 50): %w", internalerrors.ErrInvalidMetadataKey)
		}
		for key, value := range tx.Metadata {
			if key == "" {
				return fmt.Errorf("metadata key cannot be empty: %w", internalerrors.ErrInvalidMetadataKey)
			}
			if len(key) > 64 { // Max key length
				return fmt.Errorf("metadata key '%s' exceeds max length (64): %w", key, internalerrors.ErrInvalidMetadataKey)
			}
			if len(value) > 256 { // Max value size
				return fmt.Errorf("metadata value for key '%s' exceeds max size (256): %w", key, internalerrors.ErrInvalidMetadataValueSize)
			}
		}

		// Type-Specific Mandatory Metadata
		if tx.Type == StimulusTransaction || tx.Type == TaxTransaction {
			requiredKeys := []string{"AILogicID", "AIProof"} // Example
			for _, reqKey := range requiredKeys {
				if val, ok := tx.Metadata[reqKey]; !ok || len(val) == 0 {
					return fmt.Errorf("missing required metadata field '%s' for transaction type %s: %w", reqKey, tx.Type.String(), internalerrors.ErrMissingRequiredMetadata)
				}
			}
		}
	}

	// Fee and Nonce are uint64, so non-negativity is guaranteed by type.
	// Specific rules (e.g. minimum fee) would be consensus-level.

	return nil
}

// Validate checks the structural validity of the TransactionInput.
// Note: Signature and PublicKey format/curve validation is a deeper cryptographic concern.
func (ti *TransactionInput) Validate() error {
	// Check PreviousTxHash (must not be all zeros)
	var zeroHash Hash
	if ti.PreviousTxHash == zeroHash {
		return internalerrors.ErrInvalidPreviousTxHash
	}

	// Check Signature (must not be empty)
	if ti.Signature == nil || len(ti.Signature) == 0 {
		return internalerrors.ErrMissingInputSignature
	}
	// Conceptual: Add signature length check if a specific scheme imposes it.
	// if len(ti.Signature) > MaxSignatureLength || len(ti.Signature) < MinSignatureLength {
	//     return internalerrors.ErrInvalidSignatureLength
	// }

	// Check PublicKey (must not be empty)
	if ti.PublicKey == nil || len(ti.PublicKey) == 0 {
		return internalerrors.ErrMissingInputPublicKey
	}
	// Conceptual: Add public key length check based on chosen curve.
	// e.g., for SECP256k1 compressed:
	// const expectedPubKeyLength = 33
	// if len(ti.PublicKey) != expectedPubKeyLength {
	//     return internalerrors.ErrInvalidPublicKeyLength
	// }
	return nil
}

// Validate checks the structural validity of the TransactionOutput.
func (to *TransactionOutput) Validate() error {
	if to.RecipientAddress == nil || len(to.RecipientAddress) == 0 {
		return internalerrors.ErrInvalidRecipientAddress // Use error from internalerrors
	}
	// Conceptual: Add address length check if a fixed length is decided.
	// if len(to.RecipientAddress) != ExpectedAddressLength {
	//     return internalerrors.ErrInvalidAddressLength
	// }

	// For standard transfers, amount must be positive.
	// Specific transaction types might allow zero amount for other purposes (e.g., data, contract interaction).
	// This basic validation assumes standard transfer intent. More nuanced checks might depend on Transaction.Type.
	if to.Amount == 0 { // Assuming 0 is invalid for most outputs.
		return internalerrors.ErrInvalidOutputAmount // Use error from internalerrors
	}
	// Conceptual: Check if Amount exceeds a protocol-defined maximum, if any.
	return nil
}

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
