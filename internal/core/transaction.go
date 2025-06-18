package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// TxType defines the type of a transaction.
type TxType string

const (
	// Standard transaction for typical value transfers
	TxStandard TxType = "STANDARD"
	// Transaction for deploying a new smart contract
	TxContractDeploy TxType = "CONTRACT_DEPLOY"
	// Transaction for calling a function in an existing smart contract
	TxContractCall TxType = "CONTRACT_CALL"
	// Transaction for creating a Decentralized Identifier (DID)
	TxDIDCreate TxType = "DID_CREATE"
	// Transaction for updating a Decentralized Identifier (DID)
	TxDIDUpdate TxType = "DID_UPDATE"
	// Transaction initiated by AI/ML for stimulus payments
	TxStimulusPayment TxType = "STIMULUS_PAYMENT"
	// Transaction initiated by AI/ML for wealth tax collection
	TxWealthTax TxType = "WEALTH_TAX"
	// Transaction for staking PTCN for validator participation
	TxValidatorStake TxType = "VALIDATOR_STAKE"
	// Transaction for governance proposals or voting
	TxGovernanceVote TxType = "GOVERNANCE_VOTE"
)

// Custom errors for transaction processing
var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrInvalidTxID           = errors.New("invalid transaction ID")
	ErrInsufficientSignatures = errors.New("insufficient signatures for multi-sig transaction")
	ErrUnauthorizedSigner    = errors.New("signer not authorized for multi-sig transaction")
	ErrDuplicateSignature    = errors.New("duplicate signature in multi-sig transaction")
)

// TxInput represents a transaction input.
// For a UTXO model, this would reference a previous unspent transaction output.
type TxInput struct {
	PrevTxID    []byte `json:"prevTxId"`          // ID of the transaction containing the output to be spent
	Vout        int    `json:"vout"`              // Index of the output in the referenced transaction
	PubKey      []byte `json:"pubKey"`            // Public key of the spender (used for single-sig inputs)
	ScriptSig   []byte `json:"scriptSig"`         // Signature script (e.g., signature for P2PKH, or redeem script for P2SH)
	Sequence    uint32 `json:"sequence,omitempty"` // Optional, for features like RBF or timelocks
	IsMultiSig  bool   `json:"isMultiSig,omitempty"`// Indicates if this input expects multi-sig validation
}

// TxOutput represents a transaction output.
type TxOutput struct {
	Value      uint64 `json:"value"`           // Value in the smallest unit of PTCN
	PubKeyHash []byte `json:"pubKeyHash"`      // Hash of the public key of the recipient (Pay-to-PubKeyHash)
	// ScriptPubKey []byte `json:"scriptPubKey"` // More general script for various output types (P2SH, P2WPKH, etc.)
}

// SignerInfo holds information about a single signer in a multi-signature transaction.
type SignerInfo struct {
	PublicKey []byte `json:"publicKey"` // Public key of the signer
	Signature []byte `json:"signature"` // Signature provided by this signer
}

// Transaction represents a transaction in the EmPower1 Blockchain.
type Transaction struct {
	ID        []byte     `json:"id"`        // Hash of the canonicalized transaction data (excluding signatures for multi-sig, or including for single-sig if ID is post-sign)
	Timestamp int64      `json:"timestamp"` // Unix nanoseconds timestamp
	TxType    TxType     `json:"txType"`    // Type of the transaction
	Inputs    []TxInput  `json:"inputs"`    // List of inputs
	Outputs   []TxOutput `json:"outputs"`   // List of outputs
	Fee       uint64     `json:"fee"`       // Transaction fee in the smallest unit of PTCN

	// Single-signer fields (populated by Sign method if not multi-sig)
	// For UTXO inputs, individual inputs carry PubKey and ScriptSig (which includes the signature).
	// These top-level fields are more for account-based models or a simplified single signature representation.
	// Let's keep them for now as per the struct, but note their interaction with TxInput.
	PublicKey []byte `json:"publicKey,omitempty"` // Public key of the single signer (if applicable)
	Signature []byte `json:"signature,omitempty"` // Signature of the single signer (if applicable)

	// Fields specific to contract interactions
	ContractCode          []byte `json:"contractCode,omitempty"`          // Bytecode for deploying a new smart contract (TxContractDeploy)
	TargetContractAddress []byte `json:"targetContractAddress,omitempty"` // Address of the contract to call (TxContractCall)
	FunctionName          string `json:"functionName,omitempty"`          // Name of the function to call in the contract
	Arguments             []byte `json:"arguments,omitempty"`             // Arguments for the function call, typically ABI encoded

	// Multi-Signature Fields
	// These define the multi-sig policy if the transaction itself is multi-sig controlled at the top level.
	// Individual UTXO inputs can also be multi-sig controlled via P2SH-like mechanisms.
	RequiredSignatures   uint32       `json:"requiredSignatures,omitempty"` // M in M-of-N
	AuthorizedPublicKeys [][]byte     `json:"authorizedPublicKeys,omitempty"` // List of N authorized public keys
	Signers              []SignerInfo `json:"signers,omitempty"`              // List of signatures provided for multi-sig

	// EmPower1 AI/ML Specific Metadata (can be used by various TxTypes)
	AILogicID     string `json:"aiLogicId,omitempty"`    // Identifier for the AI model/logic used (e.g., for Stimulus/Tax)
	AIRuleTrigger string `json:"aiRuleTrigger,omitempty"` // Specific rule or condition in the AI logic that was met
	AIProof       []byte `json:"aiProof,omitempty"`       // Cryptographic proof or attestation from an AI oracle
}

// CanonicalTxPayload is used to create a deterministic, serializable representation
// of the transaction for hashing and signing, excluding signatures themselves where appropriate.
// Fields must be ordered consistently.
type CanonicalTxPayload struct {
	Timestamp int64    `json:"timestamp"`
	TxType    TxType   `json:"txType"`
	Inputs    []string `json:"inputs"`  // Canonical representation of inputs
	Outputs   []string `json:"outputs"` // Canonical representation of outputs
	Fee       uint64   `json:"fee"`

	// Single-signer related fields (consistent representation)
	PublicKey string `json:"publicKey"` // Hex encoded
	From      string `json:"from"`      // Hex encoded (derived from PublicKey for consistency)
	To        string `json:"to"`        // Hex encoded (Note: 'To' is not a direct field in Tx, typically derived from outputs)

	// Contract-specific fields
	ContractCode          string `json:"contractCode,omitempty"`          // Base64 encoded
	TargetContractAddress string `json:"targetContractAddress,omitempty"` // Hex encoded
	FunctionName          string `json:"functionName,omitempty"`
	Arguments             string `json:"arguments,omitempty"`             // Base64 encoded

	// Multi-Signature configuration (if applicable to the transaction itself)
	RequiredSignatures   uint32   `json:"requiredSignatures,omitempty"`
	AuthorizedPublicKeys []string `json:"authorizedPublicKeys,omitempty"` // List of hex encoded public keys, sorted

	// EmPower1 AI/ML Specific Metadata
	AILogicID     string `json:"aiLogicId,omitempty"`
	AIRuleTrigger string `json:"aiRuleTrigger,omitempty"`
	AIProof       string `json:"aiProof,omitempty"` // Base64 encoded
}

// PrepareInputsForHashing creates a canonical string representation for each TxInput.
// This ensures that logically identical inputs produce the same hash component.
func PrepareInputsForHashing(inputs []TxInput) []string {
	canonicalInputs := make([]string, len(inputs))
	for i, input := range inputs {
		// For UTXO, PrevTxID+Vout is a unique identifier. ScriptSig is excluded from ID hash.
		// PubKey might be part of what's signed if it's not in ScriptSig.
		// For simplicity here, hash PrevTxID and Vout.
		// In a real UTXO model, you'd hash the PubKey script being unlocked.
		// Sorting inputs by PrevTxID and Vout before this step is also crucial.
		inputStr := fmt.Sprintf("%s:%d:%s:%d",
			hex.EncodeToString(input.PrevTxID),
			input.Vout,
			hex.EncodeToString(input.PubKey), // Include PubKey as it identifies the UTXO owner
			input.Sequence)
		canonicalInputs[i] = inputStr
	}
	// IMPORTANT: The list of canonicalInputs strings should be sorted alphabetically
	// before being included in the CanonicalTxPayload to ensure deterministic hashing
	// if the order of inputs in the original Transaction.Inputs slice is not guaranteed.
	sort.Strings(canonicalInputs)
	return canonicalInputs
}

// PrepareOutputsForHashing creates a canonical string representation for each TxOutput.
func PrepareOutputsForHashing(outputs []TxOutput) []string {
	canonicalOutputs := make([]string, len(outputs))
	for i, output := range outputs {
		outputStr := fmt.Sprintf("%d:%s", output.Value, hex.EncodeToString(output.PubKeyHash))
		canonicalOutputs[i] = outputStr
	}
	// IMPORTANT: The list of canonicalOutputs strings should be sorted alphabetically
	// if their order in Transaction.Outputs is not semantically important or guaranteed.
	sort.Strings(canonicalOutputs)
	return canonicalOutputs
}

// NewTransaction creates a new transaction.
// For multi-sig, requiredSignatures and authorizedPublicKeys should be set.
func NewTransaction(txType TxType, inputs []TxInput, outputs []TxOutput, fee uint64) *Transaction {
	return &Transaction{
		Timestamp: time.Now().UnixNano(),
		TxType:    txType,
		Inputs:    inputs,
		Outputs:   outputs,
		Fee:       fee,
		Signers:   make([]SignerInfo, 0), // Initialize for multi-sig
	}
}

// prepareDataForHashing creates a canonical payload for hashing and signing.
// This method ensures that the data used for generating the transaction ID (hash)
// and for signing is consistent and deterministic.
// For single-signature transactions, this data is hashed to become tx.ID, then signed.
// For multi-signature transactions, this data is what each authorized party signs.
func (tx *Transaction) prepareDataForHashing() ([]byte, error) {
	// Sort inputs and outputs to ensure deterministic hashing if their order isn't fixed
	// For UTXO model, inputs are typically sorted by PrevTxID and Vout.
	// Outputs are often sorted by value then PubKeyHash.
	// This sorting should ideally happen *before* calling this, or be part of PrepareInputs/OutputsForHashing.
	// For now, assuming PrepareInputsForHashing and PrepareOutputsForHashing handle internal sorting.

	canonicalInputs := PrepareInputsForHashing(tx.Inputs)
	canonicalOutputs := PrepareOutputsForHashing(tx.Outputs)

	// Sort authorized public keys for multi-sig configuration
	sortedAuthPubKeys := make([]string, len(tx.AuthorizedPublicKeys))
	tempAuthPubKeys := make([][]byte, len(tx.AuthorizedPublicKeys))
	copy(tempAuthPubKeys, tx.AuthorizedPublicKeys) // Avoid modifying the original slice

	sort.Slice(tempAuthPubKeys, func(i, j int) bool {
		return bytes.Compare(tempAuthPubKeys[i], tempAuthPubKeys[j]) < 0
	})
	for i, pk := range tempAuthPubKeys {
		sortedAuthPubKeys[i] = hex.EncodeToString(pk)
	}

	// Populate the canonical struct
	payload := CanonicalTxPayload{
		Timestamp:            tx.Timestamp,
		TxType:               tx.TxType,
		Inputs:               canonicalInputs,
		Outputs:              canonicalOutputs,
		Fee:                  tx.Fee,
		ContractCode:         base64.StdEncoding.EncodeToString(tx.ContractCode),
		TargetContractAddress: hex.EncodeToString(tx.TargetContractAddress),
		FunctionName:         tx.FunctionName,
		Arguments:            base64.StdEncoding.EncodeToString(tx.Arguments),
		RequiredSignatures:   tx.RequiredSignatures,
		AuthorizedPublicKeys: sortedAuthPubKeys,
		AILogicID:            tx.AILogicID,
		AIRuleTrigger:        tx.AIRuleTrigger,
		AIProof:              base64.StdEncoding.EncodeToString(tx.AIProof),
	}

	// Handle single-signer PublicKey and From fields
	if tx.PublicKey != nil {
		payload.PublicKey = hex.EncodeToString(tx.PublicKey)
		payload.From = hex.EncodeToString(tx.PublicKey) // Assuming 'From' is derived from the single PublicKey
	} else {
		payload.PublicKey = ""
		payload.From = ""
	}
    // 'To' is not a direct field in the Transaction struct. It's implicitly defined by TxOutputs.
    // For canonical representation, if a single 'To' address was meaningful, it would be derived
    // from outputs or set based on TxType. Here, we'll leave it empty as per instruction.
    payload.To = ""


	// Marshal the canonical payload to JSON. Using JSON for canonical representation
	// requires careful handling of field order and encoding, but it's human-readable.
	// For stricter canonicalization, a binary format like protobuf or custom serialization is often preferred.
	return json.Marshal(payload)
}

// Hash calculates the SHA256 hash of the transaction's canonical payload.
// This hash is used as the transaction ID.
func (tx *Transaction) Hash() ([]byte, error) {
	data, err := tx.prepareDataForHashing()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare data for hashing: %w", err)
	}
	hash := sha256.Sum256(data)
	return hash[:], nil
}

// Sign populates the transaction with a signature from a single private key.
// It calculates the transaction ID (hash) first, then signs this hash.
// The public key and signature are stored directly in the transaction.
// This method is typically used for non-multi-sig transactions or when a single
// TxInput needs signing by its owner.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	if tx.RequiredSignatures > 0 {
		return errors.New("use AddSignature for multi-signature transactions")
	}

	// Calculate the transaction hash (which will become its ID)
	txID, err := tx.Hash()
	if err != nil {
		return fmt.Errorf("failed to calculate transaction hash: %w", err)
	}
	tx.ID = txID // Set the transaction ID

	// Sign the transaction ID
	sig, err := ecdsa.SignASN1(rand.Reader, privKey, tx.ID)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}
	tx.Signature = sig
	tx.PublicKey = elliptic.Marshal(privKey.Curve, privKey.X, privKey.Y)

	return nil
}

// AddSignature adds a signature from one of the authorized signers for a multi-signature transaction.
func (tx *Transaction) AddSignature(privKey *ecdsa.PrivateKey) error {
	if tx.RequiredSignatures == 0 {
		return errors.New("transaction is not configured for multi-signature")
	}

	// Calculate the hash of the data to be signed (this is the transaction's core payload)
	// This hash will also be used as tx.ID if not already set.
	dataHash, err := tx.Hash()
	if err != nil {
		return fmt.Errorf("failed to calculate transaction hash for signing: %w", err)
	}
	if len(tx.ID) == 0 {
		tx.ID = dataHash
	} else if !bytes.Equal(tx.ID, dataHash) {
        // This might happen if Hash() was called, then fields changed, then AddSignature was called.
        // Or if ID was set externally. For consistency, the ID should be the hash of what's signed.
        return errors.New("transaction ID does not match current payload hash; re-hash or ensure ID is set after all modifications")
    }


	// Sign the data hash
	sig, err := ecdsa.SignASN1(rand.Reader, privKey, dataHash)
	if err != nil {
		return fmt.Errorf("failed to sign transaction data: %w", err)
	}

	signerPubKey := elliptic.Marshal(privKey.Curve, privKey.X, privKey.Y)

	// Check if the signer is authorized
	isAuthorized := false
	for _, authKey := range tx.AuthorizedPublicKeys {
		if bytes.Equal(signerPubKey, authKey) {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized {
		return ErrUnauthorizedSigner
	}

	// Check for duplicate signatures
	for _, signerInfo := range tx.Signers {
		if bytes.Equal(signerInfo.PublicKey, signerPubKey) {
			return ErrDuplicateSignature
		}
	}

	tx.Signers = append(tx.Signers, SignerInfo{PublicKey: signerPubKey, Signature: sig})
	return nil
}

// VerifySignature checks the transaction's signature(s).
// It handles both single-signature and multi-signature transactions.
func (tx *Transaction) VerifySignature() (bool, error) {
	// Calculate the hash of the data that should have been signed
	// This ensures we are verifying against the transaction's current state.
	dataHash, err := tx.Hash()
	if err != nil {
		return false, fmt.Errorf("failed to calculate transaction hash for verification: %w", err)
	}

    // Important: Verify that tx.ID matches the current dataHash if tx.ID is set.
    // This prevents verifying signatures against a payload that doesn't match the transaction's claimed ID.
    if len(tx.ID) > 0 && !bytes.Equal(tx.ID, dataHash) {
        return false, ErrInvalidTxID // Or a more specific error like "ID mismatch with payload hash"
    }


	if tx.RequiredSignatures > 0 { // Multi-signature transaction
		return tx.verifyMultiSignature(dataHash)
	}
	// Single-signature transaction (or input)
	return tx.verifySingleSignature(dataHash)
}

// verifySingleSignature verifies a single signature against the transaction data hash.
// This is used for transactions not requiring multi-sig, or for individual TxInputs.
func (tx *Transaction) verifySingleSignature(dataHash []byte) (bool, error) {
	if tx.PublicKey == nil || tx.Signature == nil {
		return false, errors.New("public key or signature is missing for single-signature verification")
	}

	// Unmarshal the public key
	curve := elliptic.P256() // Assuming P256, make this configurable or part of tx.PublicKey type
	x, y := elliptic.Unmarshal(curve, tx.PublicKey)
	if x == nil {
		return false, errors.New("failed to unmarshal public key")
	}
	pubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	// Verify the signature
	return ecdsa.VerifyASN1(pubKey, dataHash, tx.Signature), nil
}

// verifyMultiSignature verifies all collected signatures for a multi-signature transaction.
func (tx *Transaction) verifyMultiSignature(dataHash []byte) (bool, error) {
	if uint32(len(tx.Signers)) < tx.RequiredSignatures {
		return false, ErrInsufficientSignatures
	}

	// Keep track of unique public keys that have provided valid signatures
	verifiedSigners := make(map[string]bool)

	for _, signerInfo := range tx.Signers {
		// Check if this signer is one of the authorized public keys
		isAuthorized := false
		for _, authKey := range tx.AuthorizedPublicKeys {
			if bytes.Equal(signerInfo.PublicKey, authKey) {
				isAuthorized = true
				break
			}
		}
		if !isAuthorized {
			return false, fmt.Errorf("signer %s is not in the authorized list: %w", hex.EncodeToString(signerInfo.PublicKey), ErrUnauthorizedSigner)
		}

		// Unmarshal the public key
		curve := elliptic.P256() // Assuming P256
		x, y := elliptic.Unmarshal(curve, signerInfo.PublicKey)
		if x == nil {
			return false, fmt.Errorf("failed to unmarshal public key for signer %s", hex.EncodeToString(signerInfo.PublicKey))
		}
		pubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}

		// Verify the signature
		if !ecdsa.VerifyASN1(pubKey, dataHash, signerInfo.Signature) {
			return false, fmt.Errorf("invalid signature for signer %s: %w", hex.EncodeToString(signerInfo.PublicKey), ErrInvalidSignature)
		}
		verifiedSigners[string(signerInfo.PublicKey)] = true
	}

	if uint32(len(verifiedSigners)) < tx.RequiredSignatures {
		return false, ErrInsufficientSignatures // Not enough *unique valid* signatures
	}

	return true, nil
}

// NewContractDeployTransaction creates a transaction for deploying a smart contract.
func NewContractDeployTransaction(creatorPubKey []byte, contractCode []byte, fee uint64, initialUTXOs []TxInput, changeOutput *TxOutput) *Transaction {
    outputs := []TxOutput{}
    if changeOutput != nil && changeOutput.Value > 0 {
        outputs = append(outputs, *changeOutput)
    }
	tx := NewTransaction(TxContractDeploy, initialUTXOs, outputs, fee)
	tx.ContractCode = contractCode
	tx.PublicKey = creatorPubKey // Assuming single signer deploys contract for now
	return tx
}

// NewContractCallTransaction creates a transaction for calling a smart contract function.
func NewContractCallTransaction(callerPubKey []byte, contractAddress []byte, functionName string, arguments []byte, fee uint64, inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := NewTransaction(TxContractCall, inputs, outputs, fee)
	tx.TargetContractAddress = contractAddress
	tx.FunctionName = functionName
	tx.Arguments = arguments
	tx.PublicKey = callerPubKey // Assuming single signer calls contract for now
	return tx
}

// NewStimulusTransaction creates a special transaction for stimulus payments.
// Inputs would typically be from a treasury or protocol-controlled UTXO.
// Outputs are to the stimulus recipients.
func NewStimulusTransaction(inputs []TxInput, outputs []TxOutput, aiLogicID string, aiRuleTrigger string, aiProof []byte) *Transaction {
    // Fee for stimulus might be 0 or covered by protocol
	tx := NewTransaction(TxStimulusPayment, inputs, outputs, 0)
	tx.AILogicID = aiLogicID
	tx.AIRuleTrigger = aiRuleTrigger
	tx.AIProof = aiProof
    // Stimulus transactions are typically signed by a protocol key or multi-sig authority,
    // not a standard user. Signature mechanism needs to be defined for protocol transactions.
	return tx
}
// TODO: Add more constructor functions for other TxTypes as needed.
// TODO: Add helper functions for TxInput and TxOutput, e.g., to create standard P2PKH outputs.
// TODO: Consider how ScriptSig in TxInput interacts with top-level PublicKey/Signature vs. Signers for UTXO model.
//       For UTXOs, each input is typically signed independently. The top-level signatures might be for
//       cases where the transaction itself (as a whole) needs authorization beyond input spending.
//       Or, for a simpler model, if all inputs are owned by the same key, top-level sig could cover all.
//       The current Sign/AddSignature primarily focuses on signing the *transaction hash*, implying
//       authorization for the transaction as a whole. Individual input scriptSigs would still be needed
//       for spending UTXOs in a strict model. This area needs refinement based on chosen UTXO logic.
//       For now, we assume top-level signatures authorize the overall transaction intent, and
//       input-specific scriptSigs would be populated and validated separately during UTXO spending checks.File `internal/core/transaction.go` created successfully.
