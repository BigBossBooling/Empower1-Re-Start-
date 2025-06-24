package internalerrors

import "errors"

// Validation Errors
var (
	ErrInvalidTransactionID         = errors.New("transaction ID cannot be zero hash")
	ErrUnknownTransactionType       = errors.New("unknown transaction type")
	ErrNoTransactionOutputs         = errors.New("transaction outputs cannot be nil or empty")
	ErrZeroTimestamp                = errors.New("timestamp cannot be zero")
	ErrInvalidTimestampRange        = errors.New("timestamp is out of acceptable range")
	ErrEmptyInputsForSpendingTx     = errors.New("inputs cannot be empty for a spending transaction type")
	ErrUnexpectedInputsForTxType    = errors.New("inputs are not expected for this transaction type")
	ErrTransactionInputValidationFailed = errors.New("transaction input validation failed")
	ErrTransactionOutputValidationFailed = errors.New("transaction output validation failed")
	ErrMissingSignature             = errors.New("missing signature")
	ErrMissingPublicKey             = errors.New("missing public key")
	ErrInvalidSignatureLength       = errors.New("signature has an invalid length")
	ErrInvalidPublicKeyLength       = errors.New("public key has an invalid length")
	ErrInvalidFee                   = errors.New("invalid transaction fee")

	ErrInvalidMetadataKey           = errors.New("invalid metadata key (e.g., empty or too long)")
	ErrInvalidMetadataValueSize     = errors.New("metadata value size exceeds limit")
	ErrMissingRequiredMetadata      = errors.New("missing required metadata field for transaction type")

	ErrInvalidPreviousTxHash        = errors.New("previous transaction hash cannot be zero hash")
	ErrInvalidOutputIndex           = errors.New("invalid output index")
	ErrMissingInputSignature        = errors.New("input signature cannot be empty")
	ErrMissingInputPublicKey        = errors.New("input public key cannot be empty")

	ErrInvalidRecipientAddress      = errors.New("recipient address cannot be empty")
	ErrInvalidAddressLength         = errors.New("address has an invalid length")
	ErrInvalidOutputAmount          = errors.New("output amount is invalid (e.g., zero or negative)")

	ErrInvalidBlockVersion          = errors.New("unsupported block version")
	ErrInvalidPreviousBlockHash     = errors.New("previous block hash cannot be zero for non-genesis block")
	ErrInvalidMerkleRoot            = errors.New("merkle root cannot be zero hash")
	ErrInvalidBlockTimestamp        = errors.New("block timestamp is out of acceptable range") // More specific than ErrZeroTimestamp for blocks
	ErrMissingProposerAddress       = errors.New("block proposer address cannot be empty")
	ErrInvalidDifficulty            = errors.New("block difficulty is invalid")
	ErrInvalidBlockHeight           = errors.New("block height is invalid")


	ErrBlockHeaderValidationFailed    = errors.New("block header validation failed")
	ErrBlockTransactionValidationFailed = errors.New("block transaction validation failed")
	ErrInvalidBlockHash             = errors.New("block hash cannot be zero hash")
	ErrMaxTransactionsPerBlockExceeded = errors.New("maximum number of transactions per block exceeded")
	ErrBlockSizeExceeded            = errors.New("block size exceeded maximum limit")

	ErrInvalidAccountAddress        = errors.New("account address cannot be empty")
	ErrInvalidBalance               = errors.New("account balance is invalid") // e.g. if specific rules apply
	ErrInvalidNonce                 = errors.New("account nonce is invalid")   // e.g. if specific rules apply
	ErrInvalidReputationScore       = errors.New("reputation score is outside the valid range")
)

// General Errors
var (
	ErrNotImplemented         = errors.New("feature or method not implemented yet")
	ErrInvalidOperation       = errors.New("operation is invalid in the current context")
	ErrSignatureFailed        = errors.New("cryptographic signature operation failed")
	ErrTxIDAlreadySet         = errors.New("transaction ID is already set")
	ErrUTXONotFound           = errors.New("UTXO not found")
	ErrCriticalStateCorruption = errors.New("critical state corruption detected")
	ErrAccountNotFound        = errors.New("account not found")      // For GetBalance/FindSpendableOutputs if account map is used
	ErrInsufficientBalance    = errors.New("insufficient balance") // For FindSpendableOutputs
)

// TODO: Consider creating a custom error type that can include more context,
// e.g.,  type ValidationError struct { Code ErrorCode; Message string; Field string; Cause error }
// For now, standard error variables are used for simplicity and direct use with errors.Is().
```
