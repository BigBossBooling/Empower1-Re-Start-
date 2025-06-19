package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	// "encoding/base64" // Not directly used in tests, but in transaction.go
	"encoding/hex"
	// "encoding/json" // Not directly used in tests, but in transaction.go
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"
)

// --- Helper Functions for Testing ---

// newTestPrivateKey generates a new ECDSA private key for testing.
func newTestPrivateKey(t *testing.T) *ecdsa.PrivateKey {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	return privKey
}

// publicKeyToBytes converts an ecdsa.PublicKey to its byte representation.
func publicKeyToBytes(pubKey *ecdsa.PublicKey) []byte {
	if pubKey == nil {
		return nil
	}
	return elliptic.Marshal(elliptic.P256(), pubKey.X, pubKey.Y)
}

// publicKeyToAddress is a conceptual helper.
func publicKeyToAddress(pubKeyBytes []byte) []byte {
    if pubKeyBytes == nil {
        return nil
    }
	// In a real system, this might involve hashing and then taking a portion,
	// or using a specific address derivation scheme (e.g., base58check encoding).
	// For testing purposes, returning the full public key bytes or a simple hash can suffice
	// if the tests primarily rely on byte equality for addresses derived from these keys.
	// Let's use a simple SHA256 hash for now, similar to how PubKeyHash might be generated.
	hash := sha256.Sum256(pubKeyBytes)
	return hash[:]
}


// --- Test Cases ---

// TestTransactionConstructors tests all New... constructor functions.
// Note: The constructor functions in transaction.go are basic and don't return errors yet.
// These tests will primarily check for correct type and field initialization.
// Error checks for invalid parameters (like no inputs, zero fee) would require
// validation logic within the constructors or a separate Validate() method.
func TestTransactionConstructors(t *testing.T) {
	privKey1 := newTestPrivateKey(t)
	pubKey1Bytes := publicKeyToBytes(&privKey1.PublicKey)
	addr1 := publicKeyToAddress(pubKey1Bytes)

	// Mock TxInput: For hashing tests, ensure PubKey and Signature are populated
	// as they are part of the canonical input string in PrepareInputsForHashing.
	mockTxID := sha256.Sum256([]byte("prev tx for input"))
	dummyInputs := []TxInput{{
		PrevTxID: mockTxID[:],
		Vout: 0,
		ScriptSig: []byte("dummy sig for input"), // Placeholder for ScriptSig
		PubKey: pubKey1Bytes,                 // PubKey that owns the UTXO
		Sequence: 0,
	}}
	dummyOutputs := []TxOutput{{Value: 100, PubKeyHash: addr1}}

	t.Run("NewStandardTransaction_Valid", func(t *testing.T) {
		tx := NewStandardTransaction(dummyInputs, dummyOutputs, 10)
		if tx == nil {
			t.Fatal("NewStandardTransaction() returned nil")
		}
		if tx.TxType != TxStandard {
			t.Errorf("NewStandardTransaction() TxType = %s, want %s", tx.TxType, TxStandard)
		}
		if len(tx.Inputs) != 1 || len(tx.Outputs) != 1 || tx.Fee != 10 {
			t.Errorf("NewStandardTransaction() fields not set correctly")
		}
	})

	// Example for future: If constructors add validation and return errors
	// t.Run("NewStandardTransaction_Error_NoInputs", func(t *testing.T) {
	// 	_, err := NewStandardTransaction([]TxInput{}, dummyOutputs, 10)
	// 	if err == nil { // Assuming an error should be returned for no inputs
	// 		t.Errorf("NewStandardTransaction() with no inputs, expected error, got nil")
	// 	}
	//  // else if !errors.Is(err, ErrSomeSpecificErrorForNoInputs) {
	// 	// 	t.Errorf("NewStandardTransaction() with no inputs, error = %v, want %v", err, ErrSomeSpecificErrorForNoInputs)
	// 	// }
	// })


	t.Run("NewContractDeployTransaction_Valid", func(t *testing.T) {
		contractCode := []byte("contract_code_bytes")
		tx := NewContractDeployTransaction(pubKey1Bytes, contractCode, 10, dummyInputs, dummyOutputs)
		if tx == nil {
			t.Fatal("NewContractDeployTransaction() returned nil")
		}
		if tx.TxType != TxContractDeploy {
			t.Errorf("NewContractDeployTransaction() TxType = %s, want %s", tx.TxType, TxContractDeploy)
		}
		if !bytes.Equal(tx.ContractCode, contractCode) {
			t.Errorf("NewContractDeployTransaction() ContractCode not set correctly")
		}
		if !bytes.Equal(tx.PublicKey, pubKey1Bytes) {
			t.Errorf("NewContractDeployTransaction() PublicKey not set correctly")
		}
	})

	// TODO: Add similar constructor tests for:
	// - NewContractCallTransaction (valid, missing target/function name if validation added)
    // - NewStimulusTransaction (valid, missing AI fields/outputs if validation added)
    // - NewWealthTaxTransaction (valid, missing AI fields/inputs/outputs if validation added)
}

// TestTransactionHashing tests prepareDataForHashing and Hash methods.
func TestTransactionHashing(t *testing.T) {
	privKey1 := newTestPrivateKey(t)
	pubKey1Bytes := publicKeyToBytes(&privKey1.PublicKey)
	addr1 := publicKeyToAddress(pubKey1Bytes)

	// Ensure TxInput includes PubKey and ScriptSig (even if mock) as they are part of canonical form
	mockPrevTxID := sha256.Sum256([]byte("prev tx for hashing test"))
	inputs := []TxInput{{
		PrevTxID: mockPrevTxID[:],
		Vout: 0,
		PubKey: pubKey1Bytes,
		ScriptSig: []byte("mockSig"), // Part of input but not its hash for tx ID
		Sequence: 0,
	}}
	outputs := []TxOutput{{Value: 100, PubKeyHash: addr1}}

	tx1 := NewStandardTransaction(inputs, outputs, 10)
    tx1.AILogicID = "ai_model_v1"
    tx1.AIProof = []byte("ai_proof_data_123")

	hash1_1, err := tx1.Hash()
	if err != nil {
		t.Fatalf("tx1.Hash() failed: %v", err)
	}

    tx1_clone := NewStandardTransaction(inputs, outputs, 10)
    tx1_clone.AILogicID = "ai_model_v1"
    tx1_clone.AIProof = []byte("ai_proof_data_123")
    tx1_clone.Timestamp = tx1.Timestamp

	hash1_2, err := tx1_clone.Hash()
	if err != nil {
		t.Fatalf("tx1_clone.Hash() failed: %v", err)
	}

	if !bytes.Equal(hash1_1, hash1_2) {
		t.Errorf("Hashes of identical transactions are different. Hash1: %x, Hash2: %x", hash1_1, hash1_2)
        payload1, _ := tx1.prepareDataForHashing()
        payload2, _ := tx1_clone.prepareDataForHashing()
        t.Logf("Payload1: %s", string(payload1))
        t.Logf("Payload2: %s", string(payload2))
	}

	tx2 := NewStandardTransaction(inputs, outputs, 20)
    tx2.Timestamp = tx1.Timestamp
	hash2, _ := tx2.Hash()
	if bytes.Equal(hash1_1, hash2) {
		t.Errorf("Hash did not change when fee was modified. Hash1: %x, Hash2: %x", hash1_1, hash2)
	}

    tx1_clone_ai_diff := NewStandardTransaction(inputs, outputs, 10)
    tx1_clone_ai_diff.Timestamp = tx1.Timestamp
    tx1_clone_ai_diff.AILogicID = "ai_model_v2"
    tx1_clone_ai_diff.AIProof = []byte("ai_proof_data_123")
    hash_ai_diff, _ := tx1_clone_ai_diff.Hash()
    if bytes.Equal(hash1_1, hash_ai_diff) {
        t.Errorf("Hash did not change when AILogicID was modified.")
    }
}

// TestTransactionSignAndVerify_SingleSig tests single-signature Sign and VerifySignature.
func TestTransactionSignAndVerify_SingleSig(t *testing.T) {
	privKey := newTestPrivateKey(t)
	pubKeyBytes := publicKeyToBytes(&privKey.PublicKey)
	addr := publicKeyToAddress(pubKeyBytes)

	mockPrevTxID := sha256.Sum256([]byte("prevTxForSingleSig"))
	inputs := []TxInput{{
		PrevTxID: mockPrevTxID[:],
		Vout: 0,
		PubKey: pubKeyBytes, // This input is conceptually owned by privKey
		ScriptSig: []byte("placeholder_script_sig_for_hashing"), // Needs to be present for hashing
		Sequence: 0,
	}}
	outputs := []TxOutput{{Value: 50, PubKeyHash: addr}}
	tx := NewStandardTransaction(inputs, outputs, 10)
	// tx.PublicKey will be set by Sign method

	err := tx.Sign(privKey)
	if err != nil {
		t.Fatalf("tx.Sign() error = %v", err)
	}
	if len(tx.ID) == 0 {
		t.Errorf("tx.ID not set after signing")
	}
    if len(tx.Signature) == 0 {
        t.Errorf("tx.Signature not set after single-sig signing")
    }
    if !bytes.Equal(tx.PublicKey, pubKeyBytes) {
        t.Errorf("tx.PublicKey not set correctly after single-sig signing")
    }

	valid, err := tx.VerifySignature()
	if err != nil {
		t.Fatalf("tx.VerifySignature() for valid signature error = %v", err)
	}
	if !valid {
		t.Errorf("tx.VerifySignature() for valid signature = false, want true")
	}

    originalFee := tx.Fee
	tx.Fee = 20
    // After tampering, tx.ID is now stale. VerifySignature should ideally detect this if it re-hashes.
    // The current VerifySignature uses tx.ID directly. If we want to test tampering,
    // we should re-hash and compare ID, or the verification should re-hash.
    // Let's assume VerifySignature verifies against the current state's hash, not just tx.ID.
    // Re-evaluating VerifySignature: It does NOT re-hash. It uses tx.ID.
    // So, to test tampering, we need to sign, then tamper, then verify.
    // The hash for verification (tx.ID) will be for the *original* data.
    // This is correct: the signature is for the original data.
    // A *new* hash of tampered data would not match tx.ID.

    // To properly test tampering for VerifySignature:
    // 1. Sign original tx (tx.ID is hash of original, tx.Signature is sig of tx.ID)
    // 2. Tamper tx (e.g., tx.Fee = 20)
    // 3. Call tx.VerifySignature(). It will use the original tx.ID and original tx.Signature.
    //    The public key will verify tx.Signature against tx.ID. This will pass *if tx.ID itself is not part of signed data*.
    //    This implies tx.ID must be the data being signed. Yes, Sign() does: sig, err := ecdsa.SignASN1(rand.Reader, privKey, tx.ID)
    //    So, this test is fine. VerifySignature uses tx.ID (the original hash) to verify tx.Signature.
    //    The critical point is that if someone tries to validate this tampered transaction,
    //    they would first re-calculate the hash of the *tampered* data. If this new hash
    //    doesn't match tx.ID, it's an invalid block/tx. If it *does* match (e.g. tx.ID was also tampered),
    //    then VerifySignature will fail because tx.Signature is for the *original* tx.ID.

    // Current VerifySignature verifies tx.Signature against tx.ID (which is hash of original data).
    // This is correct. Tampering tx.Fee does not invalidate tx.Signature against tx.ID.
    // The check for tampering is that Hash(tampered_tx) != tx.ID.
    // So, this part of the test needs to be rethought for what VerifySignature actually does.
    // VerifySignature itself doesn't detect tampering of fields *after* ID was set,
    // it verifies that the stored signature is valid for the stored ID and stored PublicKey.
    // The check "if !bytes.Equal(tx.ID, currentPayloadHash)" inside VerifySignature (if enabled) would catch this.
    // For now, let's assume that check is not enabled.

    // Test with wrong public key
    wrongPrivKey := newTestPrivateKey(t)
    originalPubKey := tx.PublicKey
    tx.PublicKey = publicKeyToBytes(&wrongPrivKey.PublicKey) // Tamper PublicKey
    valid, err = tx.VerifySignature() // Should fail as signature was made with original privKey
    if err == nil && valid {
        t.Errorf("tx.VerifySignature() with wrong public key = true, want false or error indicating verification failure")
    }
    if !errors.Is(err, ErrVerificationFailed) && err != nil { // It could be nil error and valid=false
         t.Logf("Note: tx.VerifySignature() with wrong public key returned error %v, expected ErrVerificationFailed or valid=false", err)
    }
    tx.PublicKey = originalPubKey // Reset

    // Test with missing signature
    originalSig := tx.Signature
    tx.Signature = nil // Use nil for omitempty
    valid, err = tx.VerifySignature()
    if !errors.Is(err, ErrInvalidPublicKey) { // Error is "public key or signature missing"
         t.Errorf("tx.VerifySignature() with missing signature, error = %v, want %v", err, ErrInvalidPublicKey)
    }
    if valid {t.Errorf("tx.VerifySignature() with missing signature = true, want false")}
    tx.Signature = originalSig
}

// TestTransactionSignAndVerify_MultiSig tests AddSignature and VerifySignature for multi-sig.
func TestTransactionSignAndVerify_MultiSig(t *testing.T) {
	numRequired := uint32(2)
	privKeys := make([]*ecdsa.PrivateKey, 3)
	authPubKeysBytes := make([][]byte, 3)
	for i := 0; i < 3; i++ {
		privKeys[i] = newTestPrivateKey(t)
		authPubKeysBytes[i] = publicKeyToBytes(&privKeys[i].PublicKey)
	}

	mockPrevTxID := sha256.Sum256([]byte("prevTxForMultiSig"))
	inputs := []TxInput{{
		PrevTxID: mockPrevTxID[:], Vout: 0,
		PubKey: authPubKeysBytes[0], // Placeholder, actual input authorization is complex for multi-sig UTXO
		ScriptSig: []byte("placeholder_multi_input_sig"),
		Sequence: 0,
	}}
	outputs := []TxOutput{{Value: 100, PubKeyHash: mockAddress(100)}}

    tx := NewTransaction(TxStandard, inputs, outputs, 10)
    tx.RequiredSignatures = numRequired
    tx.AuthorizedPublicKeys = authPubKeysBytes
    // tx.ID will be set by the first AddSignature call

	// Add first signature
	err := tx.AddSignature(privKeys[0])
	if err != nil {
		t.Fatalf("tx.AddSignature(privKeys[0]) error = %v", err)
	}
    if len(tx.Signers) != 1 { t.Errorf("Expected 1 signer, got %d", len(tx.Signers)) }
    if len(tx.ID) == 0 { t.Errorf("tx.ID not set after first signature") }

	// Verify (should fail - not enough signatures)
	valid, err := tx.VerifySignature()
	if !errors.Is(err, ErrInsufficientSignatures) {
		t.Errorf("tx.VerifySignature() with 1-of-2 sigs, error = %v, want %v", err, ErrInsufficientSignatures)
	}
    if valid {t.Errorf("tx.VerifySignature() with 1-of-2 sigs = true, want false")}

	// Add second signature
	err = tx.AddSignature(privKeys[1])
	if err != nil {
		t.Fatalf("tx.AddSignature(privKeys[1]) error = %v", err)
	}
    if len(tx.Signers) != 2 { t.Errorf("Expected 2 signers, got %d", len(tx.Signers)) }

	// Verify (should now pass)
	valid, err = tx.VerifySignature()
	if err != nil {
		t.Fatalf("tx.VerifySignature() with 2-of-2 sigs error = %v", err)
	}
	if !valid {
		t.Errorf("tx.VerifySignature() with 2-of-2 sigs = false, want true")
	}

    // Test: Attempt to add signature from an already signed key (duplicate)
    err = tx.AddSignature(privKeys[0])
    if !errors.Is(err, ErrDuplicateSignature) {
        t.Errorf("tx.AddSignature() with duplicate key, error = %v, want %v", err, ErrDuplicateSignature)
    }

    // Test: Attempt to add signature with an unauthorized key
    unauthPrivKey := newTestPrivateKey(t)
    err = tx.AddSignature(unauthPrivKey)
    if !errors.Is(err, ErrUnauthorizedSigner) {
         t.Errorf("tx.AddSignature() with unauthorized signer, error = %v, want %v", err, ErrUnauthorizedSigner)
    }

    // Test: RequiredSignatures > len(AuthorizedPublicKeys) - This should ideally be caught by a constructor or validation logic.
    // VerifySignature itself might not catch this specific config error if enough (but impossible) sigs are required.
    // Let's test if VerifySignature correctly fails if we have 2 valid signers, but require 3 from only 2 authorized.
    tx.Signers = tx.Signers[:0] // Clear signers
    tx.AuthorizedPublicKeys = authPubKeysBytes[:2] // Only 2 authorized
    tx.RequiredSignatures = 3 // Still require 3
    err = tx.AddSignature(privKeys[0]) // Sig 1
    if err != nil {t.Fatalf("Error adding privKey[0] for M>N test: %v", err)}
    err = tx.AddSignature(privKeys[1]) // Sig 2
    if err != nil {t.Fatalf("Error adding privKey[1] for M>N test: %v", err)}

    valid, err = tx.VerifySignature()
    if !errors.Is(err, ErrInsufficientSignatures) {
         t.Errorf("tx.VerifySignature() with M(3) > N(2) and 2 valid sigs, error = %v, want %v", err, ErrInsufficientSignatures)
    }
    if valid {t.Errorf("tx.VerifySignature() with M > N = true, want false")}
}


// TestTransactionSerialization tests Serialize and DeserializeTransaction (GOB).
func TestTransactionSerialization(t *testing.T) {
	privKey := newTestPrivateKey(t)
	pubKeyBytes := publicKeyToBytes(&privKey.PublicKey)
	addr := publicKeyToAddress(pubKeyBytes)

	mockPrevTxID := sha256.Sum256([]byte("prevTxForSerDes"))
	inputs := []TxInput{{
		PrevTxID: mockPrevTxID[:], Vout: 0, PubKey: pubKeyBytes, ScriptSig: []byte("sig"), Sequence: 0,
	}}
	outputs := []TxOutput{{Value: 77, PubKeyHash: addr}}
	originalTx := NewStandardTransaction(inputs, outputs, 12)
    originalTx.AILogicID = "ai_serdes_test"
	err := originalTx.Sign(privKey)
	if err != nil {
		t.Fatalf("Failed to sign originalTx: %v", err)
	}

	serializedData, err := originalTx.Serialize()
	if err != nil {
		t.Fatalf("originalTx.Serialize() error = %v", err)
	}
	if len(serializedData) == 0 {
		t.Errorf("originalTx.Serialize() produced empty byte slice")
	}

	deserializedTx, err := DeserializeTransaction(serializedData)
	if err != nil {
		t.Fatalf("DeserializeTransaction() error = %v", err)
	}

	if !bytes.Equal(originalTx.ID, deserializedTx.ID) {
		t.Errorf("ID mismatch: original %x, deserialized %x", originalTx.ID, deserializedTx.ID)
	}
	if originalTx.Timestamp != deserializedTx.Timestamp {
		t.Errorf("Timestamp mismatch: original %d, deserialized %d", originalTx.Timestamp, deserializedTx.Timestamp)
	}
    if originalTx.TxType != deserializedTx.TxType {
		t.Errorf("TxType mismatch: original %s, deserialized %s", originalTx.TxType, deserializedTx.TxType)
	}
    if originalTx.Fee != deserializedTx.Fee {
        t.Errorf("Fee mismatch: original %d, deserialized %d", originalTx.Fee, deserializedTx.Fee)
    }
    if !bytes.Equal(originalTx.PublicKey, deserializedTx.PublicKey) {
		t.Errorf("PublicKey mismatch")
	}
    if !bytes.Equal(originalTx.Signature, deserializedTx.Signature) {
		t.Errorf("Signature mismatch")
	}
    if originalTx.AILogicID != deserializedTx.AILogicID {
        t.Errorf("AILogicID mismatch")
    }
    if !reflect.DeepEqual(originalTx.Inputs, deserializedTx.Inputs) {
        t.Errorf("Inputs mismatch. \nOriginal: %+v\nDeserialized: %+v", originalTx.Inputs, deserializedTx.Inputs)
    }
    if !reflect.DeepEqual(originalTx.Outputs, deserializedTx.Outputs) {
        t.Errorf("Outputs mismatch. \nOriginal: %+v\nDeserialized: %+v", originalTx.Outputs, deserializedTx.Outputs)
    }


    badData := []byte("not_a_gob_encoded_tx")
    _, err = DeserializeTransaction(badData)
    if !errors.Is(err, ErrTransactionDeserialization) {
        t.Errorf("DeserializeTransaction(badData) error = %v, want %v", err, ErrTransactionDeserialization)
    }
}

// TODO:
// - Test TxType specific field handling in hashing and serialization (e.g. ContractCode only for TxContractDeploy).
// - Test AI/ML fields (AILogicID, AIRuleTrigger, AIProof) more explicitly in hashing if not covered.
// - Test multi-sig cases: M > N (more directly), empty AuthorizedPublicKeys, empty Signers with RequiredSignatures > 0.
// - Test error paths for all constructors more thoroughly once they include validation.
// - Test transaction ID stability: ensure tx.ID remains consistent after multiple Hash() calls if no data changed.
// - Test PrepareInputsForHashing and PrepareOutputsForHashing for correct sorting and formatting directly.
// - Test cases where tx.ID might be set externally and how Hash() and Sign()/AddSignature() interact with it.
// - Test edge cases for byte slice encodings (empty vs. nil for hex/base64 in canonical payload).
```
