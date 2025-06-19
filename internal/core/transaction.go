package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob" // Added for GOB serialization
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// TxType defines the type of a transaction.
// Enums are used for clarity and to prevent string comparison errors.
type TxType string

const (
	TxStandard        TxType = "STANDARD"         // Standard P2PKH or P2SH transaction
	TxContractDeploy  TxType = "CONTRACT_DEPLOY"  // Deploying a new smart contract
	TxContractCall    TxType = "CONTRACT_CALL"    // Calling a function in an existing smart contract
	TxDIDCreate       TxType = "DID_CREATE"       // Creating a new Decentralized Identifier
	TxDIDUpdate       TxType = "DID_UPDATE"       // Updating an existing Decentralized Identifier
	TxStimulusPayment TxType = "STIMULUS_PAYMENT" // AI/ML-driven stimulus payment
	TxWealthTax       TxType = "WEALTH_TAX"        // AI/ML-driven wealth tax collection
	TxValidatorStake  TxType = "VALIDATOR_STAKE"  // Staking PTCN for validator participation
	TxGovernanceVote  TxType = "GOVERNANCE_VOTE"  // Governance proposals or voting
	// V2+: Add more types as needed (e.g., TxOracleCommit, TxOracleReveal)
)

// Custom errors for transaction processing
var (
	ErrInvalidSignature           = errors.New("invalid signature")
	ErrInvalidTxID                = errors.New("invalid transaction ID or mismatch with payload hash")
	ErrInsufficientSignatures     = errors.New("insufficient signatures for multi-sig transaction")
	ErrUnauthorizedSigner         = errors.New("signer not authorized for multi-sig transaction")
	ErrDuplicateSignature         = errors.New("duplicate signature in multi-sig transaction")
	ErrInvalidPublicKey           = errors.New("invalid or missing public key")
	ErrHashingFailed              = errors.New("failed to hash transaction data")
	ErrSigningFailed              = errors.New("failed to sign transaction")
	ErrVerificationFailed         = errors.New("signature verification failed")
	ErrMultiSigNotConfigured      = errors.New("transaction not configured for multi-signature")
	ErrNotMultiSigTransaction     = errors.New("transaction is not a multi-signature transaction")
	ErrTransactionSerialization   = errors.New("transaction serialization error")   // For GOB
	ErrTransactionDeserialization = errors.New("transaction deserialization error") // For GOB
)

// TxInput represents a transaction input.
// In a UTXO model, it references a previous unspent transaction output (UTXO).
type TxInput struct {
	PrevTxID   []byte `json:"prevTxId"`           // ID (hash) of the transaction containing the output to be spent
	Vout       uint32 `json:"vout"`               // Index of the output in the referenced transaction (using uint32 for consistency)
	PubKey     []byte `json:"pubKey"`             // Public key of the spender; used to verify the signature in ScriptSig
	ScriptSig  []byte `json:"scriptSig"`          // Signature script (e.g., the signature itself for P2PKH)
	Sequence   uint32 `json:"sequence,omitempty"` // Optional: for features like RBF or timelocks (e.g., nSequence in Bitcoin)
	IsMultiSig bool   `json:"isMultiSig,omitempty"`// Indicates if this specific input requires multi-sig (P2SH-like)
}

// TxOutput represents a transaction output.
// It creates a new UTXO that can be spent in a future transaction.
type TxOutput struct {
	Value      uint64 `json:"value"`      // Value in the smallest unit of PTCN
	PubKeyHash []byte `json:"pubKeyHash"` // Hash of the public key of the recipient (Pay-to-PubKeyHash - P2PKH)
	// V2+: ScriptPubKey []byte `json:"scriptPubKey"` // For more complex output types like P2SH, P2WPKH, smart contract triggers
}

// SignerInfo holds information about a single signer in a multi-signature transaction.
type SignerInfo struct {
	PublicKey []byte `json:"publicKey"` // Public key of the signer (hex-encoded string is also an option for JSON)
	Signature []byte `json:"signature"` // Signature provided by this signer (hex-encoded string for JSON)
}

// Transaction defines the structure for EmPower1 transactions.
// It's designed to be flexible for various types including standard, contract, and AI-driven.
// NOTE: For multi-sig transactions, the conceptual 'From' address (representing the multi-sig wallet)
// should be derived from RequiredSignatures and AuthorizedPublicKeys by the transaction constructor
// or a utility. The AddSignature method appends individual signer details; it does not set a 'From'
// field for the multi-sig entity itself on the Transaction struct.
type Transaction struct {
	ID        []byte     `json:"id"`        // Hash of the canonicalized transaction data (set by Sign or after adding all multi-sig)
	Timestamp int64      `json:"timestamp"` // Unix nanoseconds timestamp (block proposer might adjust this for consensus)
	TxType    TxType     `json:"txType"`    // Type of the transaction (e.g., STANDARD, CONTRACT_CALL)
	Inputs    []TxInput  `json:"inputs"`    // List of transaction inputs
	Outputs   []TxOutput `json:"outputs"`   // List of transaction outputs
	Fee       uint64     `json:"fee"`       // Transaction fee in the smallest unit of PTCN

	// Single-signer fields (populated by Sign method if not multi-sig at top level)
	PublicKey []byte `json:"publicKey,omitempty"` // Public key of the primary/single signer for the transaction context
	Signature []byte `json:"signature,omitempty"` // Signature for the transaction context if single-signer

	// Fields specific to smart contract interactions
	ContractCode          []byte `json:"contractCode,omitempty"`          // Bytecode for deploying a new smart contract (used with TxContractDeploy)
	TargetContractAddress []byte `json:"targetContractAddress,omitempty"` // Address of the contract to call (used with TxContractCall)
	FunctionName          string `json:"functionName,omitempty"`          // Name of the function to call in the contract
	Arguments             []byte `json:"arguments,omitempty"`             // Arguments for the function call, typically ABI encoded

	// Multi-Signature Fields (for transaction-level multi-sig, not input-level P2SH)
	RequiredSignatures   uint32       `json:"requiredSignatures,omitempty"` // M in M-of-N multi-sig scheme
	AuthorizedPublicKeys [][]byte     `json:"authorizedPublicKeys,omitempty"` // List of N authorized public keys for multi-sig
	Signers              []SignerInfo `json:"signers,omitempty"`              // List of signatures provided for multi-sig

	// EmPower1 AI/ML Specific Metadata (can be used by various TxTypes like STIMULUS_PAYMENT, WEALTH_TAX)
	AILogicID     string `json:"aiLogicId,omitempty"`    // Identifier for the AI model/logic instance used
	AIRuleTrigger string `json:"aiRuleTrigger,omitempty"` // Specific rule or condition in the AI logic that was met
	AIProof       []byte `json:"aiProof,omitempty"`       // Cryptographic proof or attestation from an AI oracle or system
}

// CanonicalTxPayload is a temporary struct used to create a deterministic,
// serializable representation of the transaction for hashing and signing.
type CanonicalTxPayload struct {
	Timestamp             int64    `json:"timestamp"`
	TxType                TxType   `json:"txType"`
	Inputs                []string `json:"inputs"`
	Outputs               []string `json:"outputs"`
	Fee                   uint64   `json:"fee"`
	PublicKey             string   `json:"publicKey,omitempty"`
	From                  string   `json:"from,omitempty"`
	To                    string   `json:"to,omitempty"`
	ContractCode          string   `json:"contractCode,omitempty"`
	TargetContractAddress string   `json:"targetContractAddress,omitempty"`
	FunctionName          string   `json:"functionName,omitempty"`
	Arguments             string   `json:"arguments,omitempty"`
	RequiredSignatures    uint32   `json:"requiredSignatures,omitempty"`
	AuthorizedPublicKeys  []string `json:"authorizedPublicKeys,omitempty"`
	AILogicID             string   `json:"aiLogicId,omitempty"`
	AIRuleTrigger         string   `json:"aiRuleTrigger,omitempty"`
	AIProof               string   `json:"aiProof,omitempty"`
}

// PrepareInputsForHashing creates a canonical string representation for each TxInput.
func PrepareInputsForHashing(inputs []TxInput) []string {
	canonicalInputs := make([]string, len(inputs))
	tempInputs := make([]core.TxInput, len(inputs))
	copy(tempInputs, inputs)
	sort.Slice(tempInputs, func(i, j int) bool {
		cmp := bytes.Compare(tempInputs[i].PrevTxID, tempInputs[j].PrevTxID)
		if cmp == 0 {
			return tempInputs[i].Vout < tempInputs[j].Vout
		}
		return cmp < 0
	})
	for i, input := range tempInputs {
		inputStr := fmt.Sprintf("%s:%d:%s:%d",
			hex.EncodeToString(input.PrevTxID),
			input.Vout,
			hex.EncodeToString(input.PubKey),
			input.Sequence)
		canonicalInputs[i] = inputStr
	}
	return canonicalInputs
}

// PrepareOutputsForHashing creates a canonical string representation for each TxOutput.
func PrepareOutputsForHashing(outputs []TxOutput) []string {
	canonicalOutputs := make([]string, len(outputs))
	tempOutputs := make([]core.TxOutput, len(outputs))
	copy(tempOutputs, outputs)
	sort.Slice(tempOutputs, func(i, j int) bool {
		if tempOutputs[i].Value == tempOutputs[j].Value {
			return bytes.Compare(tempOutputs[i].PubKeyHash, tempOutputs[j].PubKeyHash) < 0
		}
		return tempOutputs[i].Value < tempOutputs[j].Value
	})
	for i, output := range tempOutputs {
		outputStr := fmt.Sprintf("%d:%s", output.Value, hex.EncodeToString(output.PubKeyHash))
		canonicalOutputs[i] = outputStr
	}
	return canonicalOutputs
}

// NewTransaction creates a new transaction with a timestamp.
func NewTransaction(txType TxType, inputs []TxInput, outputs []TxOutput, fee uint64) *Transaction {
	return &Transaction{
		Timestamp: time.Now().UnixNano(),
		TxType:    txType,
		Inputs:    inputs,
		Outputs:   outputs,
		Fee:       fee,
		Signers:   make([]SignerInfo, 0),
	}
}

// prepareDataForHashing creates a canonical payload for hashing.
func (tx *Transaction) prepareDataForHashing() ([]byte, error) {
	canonicalInputs := PrepareInputsForHashing(tx.Inputs)
	canonicalOutputs := PrepareOutputsForHashing(tx.Outputs)
	var sortedAuthPubKeysHex []string
	if len(tx.AuthorizedPublicKeys) > 0 {
		tempHexKeys := make([]string, len(tx.AuthorizedPublicKeys))
		for i, pkBytes := range tx.AuthorizedPublicKeys {
			tempHexKeys[i] = hex.EncodeToString(pkBytes)
		}
		sort.Strings(tempHexKeys)
		sortedAuthPubKeysHex = tempHexKeys
	}
	payload := CanonicalTxPayload{
		Timestamp:             tx.Timestamp,
		TxType:                tx.TxType,
		Inputs:                canonicalInputs,
		Outputs:               canonicalOutputs,
		Fee:                   tx.Fee,
		ContractCode:          base64.StdEncoding.EncodeToString(tx.ContractCode),
		TargetContractAddress: hex.EncodeToString(tx.TargetContractAddress),
		FunctionName:          tx.FunctionName,
		Arguments:             base64.StdEncoding.EncodeToString(tx.Arguments),
		RequiredSignatures:    tx.RequiredSignatures,
		AuthorizedPublicKeys:  sortedAuthPubKeysHex,
		AILogicID:             tx.AILogicID,
		AIRuleTrigger:         tx.AIRuleTrigger,
		AIProof:               base64.StdEncoding.EncodeToString(tx.AIProof),
	}
	if tx.PublicKey != nil {
		payload.PublicKey = hex.EncodeToString(tx.PublicKey)
		payload.From = hex.EncodeToString(tx.PublicKey)
	} else {
		payload.PublicKey = ""
		payload.From = ""
	}
	payload.To = ""
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal canonical payload: %v", ErrHashingFailed, err)
	}
	return jsonData, nil
}

// Hash calculates the SHA256 hash of the transaction's canonical payload.
func (tx *Transaction) Hash() ([]byte, error) {
	// TODO: Performance consideration: tx.Hash() is called here and involves JSON marshaling.
	// For high-throughput validation, caching tx.Hash() result after first computation,
	// or using a more performant canonical serialization than JSON (e.g., protobuf, custom binary)
	// for hashing could be future optimizations.
	data, err := tx.prepareDataForHashing()
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	return hash[:], nil
}

// Sign populates the transaction with a signature from a single private key.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	if tx.RequiredSignatures > 0 && len(tx.AuthorizedPublicKeys) > 0 {
		return fmt.Errorf("%w: use AddSignature for transactions configured with AuthorizedPublicKeys", ErrMultiSigNotConfigured)
	}
	txID, err := tx.Hash()
	if err != nil {
		return err
	}
	tx.ID = txID
	// NOTE: In a production environment, ensuring crypto/rand.Reader provides
	// non-blocking, high-quality entropy is paramount for secure nonces.
	sig, err := ecdsa.SignASN1(rand.Reader, privKey, tx.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSigningFailed, err)
	}
	tx.Signature = sig
	tx.PublicKey = elliptic.Marshal(privKey.Curve, privKey.X, privKey.Y)
	return nil
}

// AddSignature adds a signature from one of the authorized signers for a multi-signature transaction.
func (tx *Transaction) AddSignature(privKey *ecdsa.PrivateKey) error {
	if tx.RequiredSignatures == 0 || len(tx.AuthorizedPublicKeys) == 0 {
		return ErrMultiSigNotConfigured
	}
	if len(tx.ID) == 0 {
		currentPayloadHash, err := tx.Hash()
		if err != nil {
			return err
		}
		tx.ID = currentPayloadHash
	}
	// NOTE: In a production environment, ensuring crypto/rand.Reader provides
	// non-blocking, high-quality entropy is paramount for secure nonces.
	sig, err := ecdsa.SignASN1(rand.Reader, privKey, tx.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSigningFailed, err)
	}
	signerPubKeyBytes := elliptic.Marshal(privKey.Curve, privKey.X, privKey.Y)
	isAuthorized := false
	for _, authKey := range tx.AuthorizedPublicKeys {
		if bytes.Equal(signerPubKeyBytes, authKey) {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized {
		return fmt.Errorf("signer %s: %w", hex.EncodeToString(signerPubKeyBytes), ErrUnauthorizedSigner)
	}
	// Check for duplicate public keys in existing signers
	for _, existingSigner := range tx.Signers {
		if bytes.Equal(existingSigner.PublicKey, signerPubKeyBytes) {
			return fmt.Errorf("%w: signature from public key %s already exists", ErrDuplicateSignature, hex.EncodeToString(signerPubKeyBytes))
		}
	}
	tx.Signers = append(tx.Signers, SignerInfo{PublicKey: signerPubKeyBytes, Signature: sig})
	return nil
}

// VerifySignature checks the transaction's signature(s) based on its configuration.
func (tx *Transaction) VerifySignature() (bool, error) {
	// TODO: Performance consideration: tx.Hash() (called implicitly if ID is not set or re-verified) involves JSON marshaling.
	// For high-throughput validation, caching tx.Hash() result or using performant serialization is advised.
	if len(tx.ID) == 0 {
		return false, errors.New("transaction ID is not set, cannot verify signature")
	}
	if tx.RequiredSignatures > 0 && len(tx.AuthorizedPublicKeys) > 0 {
		return tx.verifyMultiSignature(tx.ID)
	}
	return tx.verifySingleSignature(tx.ID)
}

// verifySingleSignature verifies the top-level signature for a single-signer transaction.
func (tx *Transaction) verifySingleSignature(dataHash []byte) (bool, error) {
	if tx.PublicKey == nil || tx.Signature == nil {
		return false, fmt.Errorf("%w: public key or signature missing", ErrInvalidPublicKey)
	}
	// NOTE: Robust parsing of public key bytes is critical. elliptic.Unmarshal is a basic step.
	// Production systems might require more stringent validation of public key format/source.
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, tx.PublicKey)
	if x == nil {
		return false, fmt.Errorf("%w: failed to unmarshal public key", ErrInvalidPublicKey)
	}
	pubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	if !ecdsa.VerifyASN1(pubKey, dataHash, tx.Signature) {
		return false, ErrVerificationFailed
	}
	return true, nil
}

// verifyMultiSignature verifies all collected signatures for a multi-signature transaction.
func (tx *Transaction) verifyMultiSignature(dataHash []byte) (bool, error) {
	// CRITICAL TODO (Post-V1 Security Hardening): Implement verification that the transaction's
	// effective sender (e.g., a derived multi-sig address from AuthorizedPublicKeys and RequiredSignatures,
	// which would need to be part of the CanonicalTxPayload and thus tx.Hash())
	// actually corresponds to the multi-signature configuration.
	// This requires a deterministic multi-sig address derivation function:
	// e.g., conceptualDerivedAddr, err := DeriveMultiSigAddress(tx.RequiredSignatures, tx.AuthorizedPublicKeys)
	// and then checking it against the transaction's claimed sender identity.
	// This prevents misattribution of multi-sig transactions.
	if tx.RequiredSignatures == 0 {
		return false, ErrMultiSigNotConfigured
	}
	if uint32(len(tx.Signers)) < tx.RequiredSignatures {
		return false, fmt.Errorf("%w: have %d, need %d", ErrInsufficientSignatures, len(tx.Signers), tx.RequiredSignatures)
	}
	// TODO: Canonical Ordering of Signers: While this verification logic correctly handles
	// unordered signers using a map, some off-chain protocols or debugging tools might benefit
	// from tx.Signers array being canonically ordered (e.g., sorted by PublicKey).
	// This is not enforced here for on-chain validation but could be a client-side convention.
	verifiedSignersMap := make(map[string]struct{})
	for _, signerInfo := range tx.Signers {
		isAuthorized := false
		for _, authKey := range tx.AuthorizedPublicKeys {
			if bytes.Equal(signerInfo.PublicKey, authKey) {
				isAuthorized = true
				break
			}
		}
		if !isAuthorized {
			return false, fmt.Errorf("signer %s: %w", hex.EncodeToString(signerInfo.PublicKey), ErrUnauthorizedSigner)
		}
		if _, exists := verifiedSignersMap[string(signerInfo.PublicKey)]; exists {
			return false, fmt.Errorf("duplicate valid signature from signer %s: %w", hex.EncodeToString(signerInfo.PublicKey), ErrDuplicateSignature)
		}
		// NOTE: Robust parsing of public key bytes is critical. elliptic.Unmarshal is a basic step.
		// Production systems might require more stringent validation of public key format/source.
		curve := elliptic.P256()
		x, y := elliptic.Unmarshal(curve, signerInfo.PublicKey)
		if x == nil {
			return false, fmt.Errorf("%w: failed to unmarshal public key for signer %s", ErrInvalidPublicKey, hex.EncodeToString(signerInfo.PublicKey))
		}
		pubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		if !ecdsa.VerifyASN1(pubKey, dataHash, signerInfo.Signature) {
			return false, fmt.Errorf("for signer %s: %w", hex.EncodeToString(signerInfo.PublicKey), ErrVerificationFailed)
		}
		verifiedSignersMap[string(signerInfo.PublicKey)] = struct{}{}
	}
	if uint32(len(verifiedSignersMap)) < tx.RequiredSignatures {
		return false, fmt.Errorf("%w: have %d unique valid, need %d", ErrInsufficientSignatures, len(verifiedSignersMap), tx.RequiredSignatures)
	}
	return true, nil
}

// --- Constructor Functions for Specific Transaction Types ---
func NewStandardTransaction(inputs []core.TxInput, outputs []core.TxOutput, fee uint64) *Transaction {
	return NewTransaction(TxStandard, inputs, outputs, fee)
}
func NewContractDeployTransaction(creatorPubKey []byte, contractCode []byte, fee uint64, utxoInputs []TxInput, changeOutputs []TxOutput) *Transaction {
	tx := NewTransaction(TxContractDeploy, utxoInputs, changeOutputs, fee)
	tx.ContractCode = contractCode
	tx.PublicKey = creatorPubKey
	return tx
}
func NewContractCallTransaction(callerPubKey []byte, targetContractAddress []byte, functionName string, arguments []byte, fee uint64, utxoInputs []TxInput, valueOutputs []TxOutput) *Transaction {
	tx := NewTransaction(TxContractCall, utxoInputs, valueOutputs, fee)
	tx.TargetContractAddress = targetContractAddress
	tx.FunctionName = functionName
	tx.Arguments = arguments
	tx.PublicKey = callerPubKey
	return tx
}
func NewStimulusTransaction(inputs []core.TxInput, outputs []core.TxOutput, aiLogicID string, aiRuleTrigger string, aiProof []byte, protocolSignerPubKey []byte) *Transaction {
	tx := NewTransaction(TxStimulusPayment, inputs, outputs, 0)
	tx.AILogicID = aiLogicID
	tx.AIRuleTrigger = aiRuleTrigger
	tx.AIProof = aiProof
	tx.PublicKey = protocolSignerPubKey
	return tx
}
func NewWealthTaxTransaction(inputs []core.TxInput, outputs []core.TxOutput, fee uint64, aiLogicID string, aiRuleTrigger string, aiProof []byte, protocolSignerPubKey []byte) *Transaction {
	tx := NewTransaction(TxWealthTax, inputs, outputs, fee)
	tx.AILogicID = aiLogicID
	tx.AIRuleTrigger = aiRuleTrigger
	tx.AIProof = aiProof
	tx.PublicKey = protocolSignerPubKey
	return tx
}

// Serialize uses gob encoding to convert the Transaction struct into a byte slice.
// This is suitable for efficient storage or network transmission within Go-based components.
func (tx *Transaction) Serialize() ([]byte, error) {
    var result bytes.Buffer
    encoder := gob.NewEncoder(&result)

    // It's crucial that all types used within Transaction (and its nested structs)
    // are registered with gob if they are interface types or if an explicit registration
    // is needed for some other reason. For structs of concrete types, gob usually handles it.
    // For now, assuming direct encoding works for the current Transaction struct.
    // gob.Register(TxInput{}) // Example if TxInput was an interface or needed registration
    // gob.Register(TxOutput{})
    // gob.Register(SignerInfo{})
    // gob.Register(TxType("")) // For custom types like TxType, though as string alias it might be fine.

    err := encoder.Encode(tx)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrTransactionSerialization, err)
    }
    return result.Bytes(), nil
}

// DeserializeTransaction converts a byte slice (previously gob encoded) back into a Transaction struct.
func DeserializeTransaction(data []byte) (*Transaction, error) {
    var tx Transaction
    decoder := gob.NewDecoder(bytes.NewBuffer(data))

    // Similar to Serialize, ensure types are registered if necessary.
    // gob.Register(TxInput{})
    // gob.Register(TxOutput{})
    // gob.Register(SignerInfo{})
    // gob.Register(TxType(""))

    err := decoder.Decode(&tx)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrTransactionDeserialization, err)
    }
    return &tx, nil
}

// TODO: Add more constructor functions for other TxTypes (DIDCreate, DIDUpdate, ValidatorStake, GovernanceVote) as needed.
// TODO: Further refine how TxInput.PubKey and TxInput.ScriptSig are handled in conjunction with
//       Transaction.PublicKey/Signature and Transaction.Signers, especially for UTXO model where each input
//       is typically authorized independently. The current model leans towards a primary transaction-level signature
//       for intent, with input scriptSigs being crucial for UTXO spending validation.
//       This might involve a more complex Verify() method that iterates inputs and checks their ScriptSigs
//       against their respective UTXO's PubKeyHash, in addition to verifying the overall transaction signature(s).
//       For now, the Sign/Verify methods focus on the overall transaction hash.

[end of internal/core/transaction.go]
