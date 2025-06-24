package types_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	// Assuming 'empower1/internal/core/types' is the Go module path to our types
	// Adjust if the actual module path is different.
	"empower1/internal/core/types"
	internalerrors "empower1/internal/errors"
)

// Helper function to create a zero hash for comparisons
func zeroHash() types.Hash {
	return types.Hash{}
}

// Helper function to create a sample valid hash (non-zero)
func sampleHash(val byte) types.Hash {
	var h types.Hash
	for i := range h {
		h[i] = val
	}
	return h
}

// Helper function to compare error types or messages if internal/errors is not yet implemented.
// This helper robustly checks errors using errors.Is for proper error wrapping comparisons.
func checkError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
	t.Helper()
	if wantErr {
		if gotErr == nil {
			t.Errorf("expected an error but got nil")
			return
		}
		if wantErrType == nil {
			// This case should ideally not happen if wantErr is true,
			// it implies the test setup might be expecting an error but not specifying which one.
			t.Logf("warning: wantErr is true but wantErrType is nil. Error received: %v", gotErr)
			return
		}
		if !errors.Is(gotErr, wantErrType) {
			t.Errorf("got error '%v' (type %T), want error to wrap or be of type %T (base error: '%v')", gotErr, gotErr, wantErrType, wantErrType)
		}
	} else { // wantErr is false
		if gotErr != nil {
			t.Errorf("did not expect an error but got: %v (type %T)", gotErr, gotErr)
		}
	}
}

// --- Serialization Tests (Placeholders) ---

func TestTransaction_SerializationJSON(t *testing.T) {
	validTimestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Use a fixed time for consistent marshalling
	originalTx := types.Transaction{
		TxID:      sampleHash(10),
		Type:      types.StandardTransaction,
		Inputs:    []types.TransactionInput{{PreviousTxHash: sampleHash(11), OutputIndex: 0, Signature: []byte("sig"), PublicKey: []byte("pubkey")}},
		Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}},
		Timestamp: validTimestamp,
		Signature: []byte("mainSig"),
		PublicKey: []byte("mainPubkey"),
		Fee:       10,
		Nonce:     1,
		Metadata:  map[string][]byte{"key": []byte("value")},
	}

	// Marshal to JSON
	jsonData, err := originalTx.ToJSON() // Assuming ToJSON method exists or will be implemented
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}
	if len(jsonData) == 0 {
		t.Fatalf("ToJSON() returned empty data")
	}
	t.Logf("Marshalled Transaction JSON: %s", string(jsonData))


	// Unmarshal from JSON
	var newTx types.Transaction
	err = newTx.FromJSON(jsonData) // Assuming FromJSON method exists or will be implemented
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	// Compare (basic check, deep equality would be better with a proper library or custom function)
	// This requires TxID, Timestamp, and potentially other fields to be consistently (de)serialized.
	// For this placeholder, we'll do a few key fields.
	// Note: Direct comparison of structs with slices/maps needs careful handling or deep equality.
	// For now, we'll compare a few critical, simple fields.
	if !bytes.Equal(originalTx.TxID[:], newTx.TxID[:]) {
		t.Errorf("TxID mismatch after JSON roundtrip. Got %x, want %x", newTx.TxID, originalTx.TxID)
	}
	if originalTx.Type != newTx.Type {
		t.Errorf("Type mismatch after JSON roundtrip. Got %v, want %v", newTx.Type, originalTx.Type)
	}
	if !originalTx.Timestamp.Equal(newTx.Timestamp) {
		// Timestamps can be tricky with JSON (e.g. precision, timezone).
		// Ensure the chosen JSON marshaller/unmarshaller handles time.Time consistently.
		t.Errorf("Timestamp mismatch after JSON roundtrip. Got %v, want %v", newTx.Timestamp, originalTx.Timestamp)
	}
	if len(originalTx.Outputs) != len(newTx.Outputs) || (len(originalTx.Outputs) > 0 && originalTx.Outputs[0].Amount != newTx.Outputs[0].Amount) {
		t.Errorf("Outputs mismatch after JSON roundtrip.")
	}

	// A more robust test would use reflect.DeepEqual or a custom comparison function
	// that handles all fields, including slices and maps.
	// e.g. if !reflect.DeepEqual(originalTx, newTx) { t.Errorf("Transaction changed after JSON roundtrip") }
	// This often requires all fields to be exportable and for time.Time to be handled carefully by the JSON lib.

	// More robust comparison using reflect.DeepEqual
	// Note: For DeepEqual to work reliably with time.Time, ensure consistent time zone handling (e.g., always UTC).
	// JSON marshalling of time.Time typically preserves timezone info if present or defaults to UTC.
	if !reflect.DeepEqual(originalTx, newTx) {
		t.Errorf("Transaction differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", originalTx, newTx)
	}

	// Test with nil/empty slices and map
	emptyTx := types.Transaction{
		TxID:      sampleHash(12),
		Type:      types.TaxTransaction,
		Inputs:    nil, // Test nil slice
		Outputs:   []types.TransactionOutput{}, // Test empty slice
		Timestamp: validTimestamp,
		Signature: []byte("sig"),
		PublicKey: []byte("pubkey"),
		Fee:       5,
		Nonce:     2,
		Metadata:  nil, // Test nil map
	}

	jsonDataEmpty, err := emptyTx.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() for emptyTx failed: %v", err)
	}
	var newEmptyTx types.Transaction
	err = newEmptyTx.FromJSON(jsonDataEmpty)
	if err != nil {
		t.Fatalf("FromJSON() for emptyTx failed: %v", err)
	}
	if !reflect.DeepEqual(emptyTx, newEmptyTx) {
		t.Errorf("emptyTx differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", emptyTx, newEmptyTx)
	}

	// Test with empty map for metadata
	emptyMetaTx := types.Transaction{
		TxID:      sampleHash(13),
		Type:      types.GovernanceTransaction,
		Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addrGov"), Amount: 1}},
		Timestamp: validTimestamp,
		Signature: []byte("govSig"),
		PublicKey: []byte("govPubkey"),
		Fee:       1,
		Nonce:     3,
		Metadata:  map[string][]byte{}, // Test empty map
	}
	jsonDataEmptyMeta, err := emptyMetaTx.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() for emptyMetaTx failed: %v", err)
	}
	var newEmptyMetaTx types.Transaction
	err = newEmptyMetaTx.FromJSON(jsonDataEmptyMeta)
	if err != nil {
		t.Fatalf("FromJSON() for emptyMetaTx failed: %v", err)
	}
	if !reflect.DeepEqual(emptyMetaTx, newEmptyMetaTx) {
		t.Errorf("emptyMetaTx differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", emptyMetaTx, newEmptyMetaTx)
	}
}

func TestBlock_SerializationJSON(t *testing.T) {
	validTimestamp := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC) // Use a fixed time
	originalBlock := types.Block{
		Header: types.BlockHeader{
			Version:           1,
			PreviousBlockHash: sampleHash(20),
			MerkleRoot:        sampleHash(21),
			Timestamp:         validTimestamp,
			Nonce:             123,
			Difficulty:        10,
			Height:            1,
			ProposerAddress:   types.Address("proposer"),
		},
		Transactions: []types.Transaction{{
			TxID:      sampleHash(22),
			Type:      types.StandardTransaction,
			Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr2"), Amount: 50}},
			Timestamp: validTimestamp.Add(-time.Minute),
			Fee: 5,
		}},
		BlockHash: sampleHash(23),
	}

	// Marshal to JSON
	jsonData, err := originalBlock.ToJSON() // Assuming ToJSON method exists
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}
	if len(jsonData) == 0 {
		t.Fatalf("ToJSON() returned empty data")
	}
	t.Logf("Marshalled Block JSON: %s", string(jsonData))

	// Unmarshal from JSON
	var newBlock types.Block
	err = newBlock.FromJSON(jsonData) // Assuming FromJSON method exists
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	// Compare (basic check)
	if !bytes.Equal(originalBlock.BlockHash[:], newBlock.BlockHash[:]) {
		t.Errorf("BlockHash mismatch after JSON roundtrip. Got %x, want %x", newBlock.BlockHash, originalBlock.BlockHash)
	}
	if originalBlock.Header.Height != newBlock.Header.Height {
		t.Errorf("BlockHeader Height mismatch. Got %v, want %v", newBlock.Header.Height, originalBlock.Header.Height)
	}
	if !originalBlock.Header.Timestamp.Equal(newBlock.Header.Timestamp) {
		t.Errorf("BlockHeader Timestamp mismatch. Got %v, want %v", newBlock.Header.Timestamp, originalBlock.Header.Timestamp)
	}
	if len(originalBlock.Transactions) != len(newBlock.Transactions) {
		t.Errorf("Number of transactions mismatch.")
	}
	// Add more comprehensive checks with reflect.DeepEqual once (de)serialization methods are firm.
	if !reflect.DeepEqual(originalBlock, newBlock) {
		t.Errorf("Block differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", originalBlock, newBlock)
	}

	// Test with empty transactions
	emptyTxBlock := types.Block{
		Header: types.BlockHeader{
			Version:           1,
			PreviousBlockHash: sampleHash(30),
			MerkleRoot:        sampleHash(31), // Should ideally be hash of empty list
			Timestamp:         validTimestamp,
			Height:            2,
			ProposerAddress:   types.Address("proposer2"),
		},
		Transactions: []types.Transaction{}, // Empty slice
		BlockHash:    sampleHash(32),
	}
	jsonDataEmptyTx, err := emptyTxBlock.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() for emptyTxBlock failed: %v", err)
	}
	var newEmptyTxBlock types.Block
	err = newEmptyTxBlock.FromJSON(jsonDataEmptyTx)
	if err != nil {
		t.Fatalf("FromJSON() for emptyTxBlock failed: %v", err)
	}
	if !reflect.DeepEqual(emptyTxBlock, newEmptyTxBlock) {
		t.Errorf("emptyTxBlock differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", emptyTxBlock, newEmptyTxBlock)
	}

	// Test with nil transactions (should be handled same as empty by encoding/json for slices)
	nilTxBlock := types.Block{
		Header: types.BlockHeader{
			Version:           1,
			PreviousBlockHash: sampleHash(40),
			MerkleRoot:        sampleHash(41),
			Timestamp:         validTimestamp,
			Height:            3,
			ProposerAddress:   types.Address("proposer3"),
		},
		Transactions: nil, // Nil slice
		BlockHash:    sampleHash(42),
	}
	jsonDataNilTx, err := nilTxBlock.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() for nilTxBlock failed: %v", err)
	}
	var newNilTxBlock types.Block
	err = newNilTxBlock.FromJSON(jsonDataNilTx)
	if err != nil {
		t.Fatalf("FromJSON() for nilTxBlock failed: %v", err)
	}
	if !reflect.DeepEqual(nilTxBlock, newNilTxBlock) {
		t.Errorf("nilTxBlock differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", nilTxBlock, newNilTxBlock)
	}

	// Test with multiple, varied transactions
	multiTx1 := types.Transaction{
		TxID:      sampleHash(50),
		Type:      types.StandardTransaction,
		Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addrMulti1"), Amount: 10}},
		Timestamp: validTimestamp.Add(-2 * time.Minute),
		Fee:       1,
	}
	multiTx2 := types.Transaction{
		TxID:      sampleHash(51),
		Type:      types.StimulusTransaction,
		Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addrMulti2"), Amount: 20}},
		Timestamp: validTimestamp.Add(-3 * time.Minute),
		Fee:       0,
		Metadata:  map[string][]byte{"AILogicID": []byte("stimulusAI")},
	}
	multiTxBlock := types.Block{
		Header: types.BlockHeader{
			Version:           1,
			PreviousBlockHash: sampleHash(52),
			MerkleRoot:        sampleHash(53), // Placeholder, real would be complex
			Timestamp:         validTimestamp,
			Height:            4,
			ProposerAddress:   types.Address("proposer4"),
		},
		Transactions: []types.Transaction{multiTx1, multiTx2},
		BlockHash:    sampleHash(54),
	}
	jsonDataMultiTx, err := multiTxBlock.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() for multiTxBlock failed: %v", err)
	}
	var newMultiTxBlock types.Block
	err = newMultiTxBlock.FromJSON(jsonDataMultiTx)
	if err != nil {
		t.Fatalf("FromJSON() for multiTxBlock failed: %v", err)
	}
	if !reflect.DeepEqual(multiTxBlock, newMultiTxBlock) {
		t.Errorf("multiTxBlock differs after JSON roundtrip.\nOriginal: %+v\nNew:      %+v", multiTxBlock, newMultiTxBlock)
	}
}

// TODO: Add similar placeholder tests for other serialization formats if planned (e.g., binary).
// TODO: These tests will need actual ToJSON/FromJSON (or Marshal/Unmarshal) methods on the types
//       or a separate serialization package. For now, they serve as placeholders for the intent.

func TestUTXO_Validate(t *testing.T) {
	validTxID := sampleHash(100)
	validAddress := types.Address("validUTXOAddress")
	// Assuming expected address length for UTXO recipient is 20, similar to TransactionOutput
	validAddressFixedLength := make(types.Address, 20)
	copy(validAddressFixedLength, []byte("fixedLengthRecipient"))


	tests := []struct {
		name            string
		utxo            types.UTXO
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid UTXO",
			utxo: types.UTXO{
				TxID:             validTxID,
				OutputIndex:      0,
				Amount:           100,
				RecipientAddress: validAddressFixedLength, // Use address with expected length
			},
			wantErr: false,
		},
		{
			name: "invalid UTXO - zero TxID",
			utxo: types.UTXO{
				TxID:             zeroHash(),
				OutputIndex:      0,
				Amount:           100,
				RecipientAddress: validAddressFixedLength,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidTransactionID,
		},
		{
			name: "invalid UTXO - zero Amount",
			utxo: types.UTXO{
				TxID:             validTxID,
				OutputIndex:      1,
				Amount:           0,
				RecipientAddress: validAddressFixedLength,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidOutputAmount,
		},
		{
			name: "invalid UTXO - nil RecipientAddress",
			utxo: types.UTXO{
				TxID:             validTxID,
				OutputIndex:      2,
				Amount:           50,
				RecipientAddress: nil,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidRecipientAddress,
		},
		{
			name: "invalid UTXO - empty RecipientAddress",
			utxo: types.UTXO{
				TxID:             validTxID,
				OutputIndex:      3,
				Amount:           50,
				RecipientAddress: types.Address{},
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidRecipientAddress,
		},
		// Note: UTXO.Validate() currently doesn't enforce address length,
		// so a test for wrong address length would pass if not for other errors.
		// If UTXO.Validate() is updated to check address length like TransactionOutput.Validate(),
		// then a specific test for that should be added.
		// For now, the "valid UTXO" uses an address of a specific length for good practice.
		{
			name: "invalid UTXO - wrong RecipientAddress length",
			utxo: types.UTXO{
				TxID:             validTxID,
				OutputIndex:      4,
				Amount:           100,
				RecipientAddress: types.Address("shortAddress"), // Wrong length
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidAddressLength,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.utxo.Validate()
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}

func TestAccount_Validate(t *testing.T) {
	validAddress := types.Address("validAccountAddress")

	tests := []struct {
		name            string
		account         types.Account
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid account",
			account: types.Account{
				Address:         validAddress,
				Balance:         1000,
				Nonce:           5,
				ReputationScore: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid Address (nil)",
			account: types.Account{
				Address:         nil, // Invalid
				Balance:         1000,
				Nonce:           5,
				ReputationScore: 100,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidAccountAddress,
		},
		{
			name: "invalid Address (empty)",
			account: types.Account{
				Address:         types.Address{}, // Invalid
				Balance:         1000,
				Nonce:           5,
				ReputationScore: 100,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidAccountAddress,
		},
		// Balance and Nonce are uint64, so they cannot be negative.
		// Zero is a valid state for both.
		// No specific invalid cases for Balance/Nonce unless a max value is imposed by protocol.
		{
			name: "invalid ReputationScore (below min bound if defined)",
			account: types.Account{
				Address:         validAddress,
				Balance:         1000,
				Nonce:           5,
				ReputationScore: -2000000, // Assuming a bound like -1M to +1M
			},
			// This test case depends on ReputationScore having defined bounds in Validate()
			// For now, this might pass if Validate() doesn't check bounds.
			// Marking as wantErr: true, assuming bounds check will be part of Account.Validate()
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidReputationScore,
		},
		{
			name: "invalid ReputationScore (above max bound if defined)",
			account: types.Account{
				Address:         validAddress,
				Balance:         1000,
				Nonce:           5,
				ReputationScore: 2000000, // Assuming a bound like -1M to +1M
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidReputationScore,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// For ReputationScore tests to be meaningful, Account.Validate() must implement range checks.
			// If it doesn't, those tests might not behave as "invalid".
			err := tc.account.Validate()
			// A more refined checkError might be needed if ErrTestInvalidReputationScore is generic
			// and the actual error message contains the bounds.
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}

func TestBlock_Validate(t *testing.T) {
	// --- Reusable valid components ---
	validTxID := sampleHash(1)
	validPubKey := []byte("validPublicKey")
	validSignature := []byte("validSignature")
	validAddress := types.Address("validAddress")
	validTimestamp := time.Now()
	currentBlockVersion := uint32(1)

	validOutput := types.TransactionOutput{
		RecipientAddress: validAddress,
		Amount:           100,
	}
	validInput := types.TransactionInput{
		PreviousTxHash: sampleHash(2),
		OutputIndex:    0,
		Signature:      []byte("inputSignature"),
		PublicKey:      []byte("inputPublicKey"),
	}
	validTransaction := types.Transaction{
		TxID:      validTxID,
		Type:      types.StandardTransaction,
		Inputs:    []types.TransactionInput{validInput},
		Outputs:   []types.TransactionOutput{validOutput},
		Timestamp: validTimestamp,
		Signature: validSignature,
		PublicKey: validPubKey,
		Fee:       10,
		Nonce:     1,
	}
	validBlockHeader := types.BlockHeader{
		Version:           currentBlockVersion,
		PreviousBlockHash: sampleHash(4),
		MerkleRoot:        sampleHash(5),
		Timestamp:         validTimestamp,
		Nonce:             12345,
		Difficulty:        100,
		Height:            1,
		ProposerAddress:   types.Address("validProposer"),
	}
	// --- End Reusable components ---

	tests := []struct {
		name            string
		block           types.Block
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid block with transactions",
			block: types.Block{
				Header:       validBlockHeader,
				Transactions: []types.Transaction{validTransaction},
				BlockHash:    sampleHash(6),
			},
			wantErr: false,
		},
		{
			name: "valid block with empty transactions",
			block: types.Block{
				Header:       validBlockHeader,
				Transactions: []types.Transaction{}, // Empty but valid slice
				BlockHash:    sampleHash(7),
			},
			wantErr: false,
		},
		{
			name: "invalid BlockHeader",
			block: types.Block{
				Header: types.BlockHeader{ // Invalid header (e.g., zero timestamp)
					Version:           currentBlockVersion,
					PreviousBlockHash: sampleHash(4),
					MerkleRoot:        sampleHash(5),
					Timestamp:         time.Time{}, // zero timestamp
					Height:            1,
					ProposerAddress:   types.Address("proposer"),
				},
				Transactions: []types.Transaction{validTransaction},
				BlockHash:    sampleHash(8),
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrBlockHeaderValidationFailed,
		},
		{
			name: "block with an invalid transaction",
			block: types.Block{
				Header: validBlockHeader,
				Transactions: []types.Transaction{
					validTransaction, // A valid one
					{ // An invalid one
						TxID:    zeroHash(), // Invalid TxID
						Type:    types.StandardTransaction,
						Outputs: []types.TransactionOutput{validOutput},
					},
				},
				BlockHash: sampleHash(9),
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrBlockTransactionValidationFailed,
		},
		{
			name: "invalid BlockHash (all-zero)",
			block: types.Block{
				Header:       validBlockHeader,
				Transactions: []types.Transaction{validTransaction},
				BlockHash:    zeroHash(), // Invalid
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidBlockHash,
		},
		// TODO: Add tests for max transactions per block if check added to Block.Validate()
		// TODO: Add tests for max block size if check added to Block.Validate()
		{
			name: "too many transactions",
			block: types.Block{
				Header:       validBlockHeader,
				Transactions: make([]types.Transaction, 10001), // 10001 transactions
				BlockHash:    sampleHash(10),
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMaxTransactionsPerBlockExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.block.Validate()
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}

func TestBlockHeader_Validate(t *testing.T) {
	validPrevBlockHash := sampleHash(4)
	validMerkleRoot := sampleHash(5)
	validProposerAddress := types.Address("validProposerAddress")
	now := time.Now()
	// Assuming CurrentBlockVersion = 1 for tests
	const currentBlockVersion uint32 = 1

	tests := []struct {
		name            string
		header          types.BlockHeader
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid block header (non-genesis)",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now,
				Nonce:             12345,
				Difficulty:        100,
				Height:            1,
				ProposerAddress:   validProposerAddress,
			},
			wantErr: false,
		},
		{
			name: "valid genesis block header",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: zeroHash(), // Allowed for genesis
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now.Add(-time.Hour), // Genesis time
				Nonce:             0,
				Difficulty:        1, // Genesis difficulty
				Height:            0, // Genesis height
				ProposerAddress:   validProposerAddress, // Or a system address
			},
			wantErr: false,
		},
		{
			name: "invalid Version",
			header: types.BlockHeader{
				Version:           99, // Invalid
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now,
				Height:            1,
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidBlockVersion,
		},
		{
			name: "PreviousBlockHash is all-zero for non-genesis block",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: zeroHash(), // Invalid for Height > 0
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now,
				Height:            1, // Non-genesis
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidPreviousBlockHash,
		},
		{
			name: "MerkleRoot is all-zero",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        zeroHash(), // Invalid
				Timestamp:         now,
				Height:            1,
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidMerkleRoot,
		},
		{
			name: "Timestamp is zero",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         time.Time{}, // Invalid
				Height:            1,
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrZeroTimestamp,
		},
		// Note: Timestamp range checks (too far future, before epoch) are now tested.
		{
			name: "Timestamp too far in future",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         time.Now().Add(3 * time.Hour), // Too far
				Height:            1,
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidBlockTimestamp,
		},
		{
			name: "Timestamp before epoch",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         time.Unix(1609459199, 0), // Before Jan 1, 2021 UTC example epoch
				Height:            1,
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidBlockTimestamp,
		},
		{
			name: "ProposerAddress is empty",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now,
				Height:            1,
				ProposerAddress:   types.Address{}, // Invalid
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingProposerAddress,
		},
		// Note: PoW Difficulty zero for non-genesis is now tested.
		{
			name: "PoW Difficulty is zero for non-genesis",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: validPrevBlockHash,
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now,
				Difficulty:        0, // Invalid for PoW non-genesis
				Height:            1, // Non-genesis
				ProposerAddress:   validProposerAddress,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidDifficulty,
		},
		{
			name: "PoW Difficulty is non-zero for genesis (valid)",
			header: types.BlockHeader{
				Version:           currentBlockVersion,
				PreviousBlockHash: zeroHash(),
				MerkleRoot:        validMerkleRoot,
				Timestamp:         now.Add(-time.Hour),
				Difficulty:        1, // Valid for genesis
				Height:            0, // Genesis
				ProposerAddress:   validProposerAddress,
			},
			wantErr: false, // This specific field is fine, overall validity depends on other fields
		},
		// TODO: Add tests for Nonce if specific validation rules apply
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Need to pass current block height for some checks if Validate takes it.
			// For now, assuming Validate can infer genesis from Height field directly.
			err := tc.header.Validate()
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}

func TestTransactionOutput_Validate(t *testing.T) {
	validAddress := types.Address("validRecipientAddress")

	tests := []struct {
		name            string
		output          types.TransactionOutput
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid transaction output",
			output: types.TransactionOutput{
				RecipientAddress: validAddress,
				Amount:           100,
			},
			wantErr: false,
		},
		{
			name: "missing RecipientAddress (nil)",
			output: types.TransactionOutput{
				RecipientAddress: nil, // Invalid
				Amount:           100,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidRecipientAddress,
		},
		{
			name: "empty RecipientAddress",
			output: types.TransactionOutput{
				RecipientAddress: types.Address{}, // Invalid
				Amount:           100,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidRecipientAddress,
		},
		{
			name: "zero Amount for standard transfer",
			output: types.TransactionOutput{
				RecipientAddress: validAddress,
				Amount:           0, // Invalid for standard transfers
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidOutputAmount,
		},
		// Note: RecipientAddress invalid length is now tested.
		// TODO: Add test for Amount exceeding max supply if that check is added (consensus/stateful rule).
		{
			name: "invalid RecipientAddress (wrong length)",
			output: types.TransactionOutput{
				RecipientAddress: types.Address("shortAddress"), // Invalid length
				Amount:           100,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidAddressLength,
		},
		{
			name: "valid RecipientAddress (correct length)",
			output: types.TransactionOutput{
				RecipientAddress: make(types.Address, 20), // Correct length
				Amount:           100,
			},
			wantErr: false, // Should pass if amount is also valid
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.output.Validate()
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}

func TestTransactionInput_Validate(t *testing.T) {
	validPrevTxHash := sampleHash(3)
	validSignature := []byte("validInputSignature")
	validPublicKey := []byte("validInputPublicKey")

	tests := []struct {
		name            string
		input           types.TransactionInput
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid transaction input",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      validSignature,
				PublicKey:      validPublicKey,
			},
			wantErr: false,
		},
		{
			name: "invalid PreviousTxHash (all-zero)",
			input: types.TransactionInput{
				PreviousTxHash: zeroHash(), // Invalid
				OutputIndex:    0,
				Signature:      validSignature,
				PublicKey:      validPublicKey,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidPreviousTxHash,
		},
		{
			name: "missing Signature",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      nil, // Invalid
				PublicKey:      validPublicKey,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingInputSignature,
		},
		{
			name: "empty Signature",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      []byte{}, // Invalid
				PublicKey:      validPublicKey,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingInputSignature,
		},
		{
			name: "missing PublicKey",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      validSignature,
				PublicKey:      nil, // Invalid
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingInputPublicKey,
		},
		{
			name: "empty PublicKey",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      validSignature,
				PublicKey:      []byte{}, // Invalid
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingInputPublicKey,
		},
		// TODO: Add tests for invalid signature/public key lengths if those checks are added to Validate()
		{
			name: "invalid signature length (too short)",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      make([]byte, 59), // Too short
				PublicKey:      make([]byte, 33), // Valid length
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidSignatureLength,
		},
		{
			name: "invalid signature length (too long)",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      make([]byte, 76), // Too long
				PublicKey:      make([]byte, 33), // Valid length
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidSignatureLength,
		},
		{
			name: "invalid public key length",
			input: types.TransactionInput{
				PreviousTxHash: validPrevTxHash,
				OutputIndex:    0,
				Signature:      make([]byte, 64), // Valid length
				PublicKey:      make([]byte, 32), // Invalid length
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidPublicKeyLength,
		},
		// Note: Invalid signature/public key lengths are now tested.
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
/*
Note on module path "empower1/internal/core/types":
This is a placeholder. In a real Go project, this would be your actual module path
defined in go.mod, e.g., "github.com/YourOrg/EmPower1/internal/core/types".
The `create_file_with_block` tool doesn't know about go.mod, so we use a generic path.
When running `go test`, Go's build system resolves these paths based on the go.mod file
in the project root.
*/

// Conceptual placeholder for shared error definitions.
// In a real setup, this would be in a separate `internal/errors/errors.go`
// and imported here. For now, test-specific error variables are defined above.
// We will use these local test error variables for `errors.Is` checks.
//
// Example structure for internal/errors:
// package internalerrors
//
// import "errors"
//
// var (
//      ErrInvalidTransactionID = errors.New("invalid transaction ID")
//      // ... other error definitions
// )
//
// func New(baseErr error, msg string) error { /* wraps baseErr with msg */ }

// --- Tests will be added in subsequent steps ---

func TestTransaction_Validate(t *testing.T) {
	validTxID := sampleHash(1)
	validPubKey := []byte("validPublicKey")
	validSignature := []byte("validSignature")
	validAddress := types.Address("validAddress")
	validTimestamp := time.Now()

	// Define a basic valid TransactionOutput for re-use
	validOutput := types.TransactionOutput{
		RecipientAddress: validAddress,
		Amount:           100,
	}

	// Define a basic valid TransactionInput for re-use
	validInput := types.TransactionInput{
		PreviousTxHash: sampleHash(2),
		OutputIndex:    0,
		Signature:      []byte("inputSignature"),
		PublicKey:      []byte("inputPublicKey"),
	}

	tests := []struct {
		name            string
		tx              types.Transaction
		wantErr         bool
		expectedErrType error // Using locally defined test error vars for now
	}{
		{
			name: "valid standard transaction",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Inputs:    []types.TransactionInput{validInput},
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
				Nonce:     1,
				Metadata:  map[string][]byte{"note": []byte("payment")},
			},
			wantErr: false,
		},
		{
			name: "invalid TxID (all-zero)",
			tx: types.Transaction{
				TxID:      zeroHash(), // Invalid
				Type:      types.StandardTransaction,
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidTransactionID,
		},
		{
			name: "invalid Type (unknown numeric value)",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.TransactionType(99), // Invalid
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrUnknownTransactionType,
		},
		{
			name: "nil Outputs slice",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Outputs:   nil, // Invalid
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrNoTransactionOutputs,
		},
		{
			name: "empty Outputs slice",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Outputs:   []types.TransactionOutput{}, // Invalid for most types
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrNoTransactionOutputs,
		},
		{
			name: "zero Timestamp",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: time.Time{}, // Invalid
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrZeroTimestamp,
		},
		{
			name: "empty Inputs slice for spending type",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction, // Assumes StandardTransaction is a spending type
				Inputs:    []types.TransactionInput{}, // Invalid
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrEmptyInputsForSpendingTx,
		},
		{
			name: "invalid TransactionInput in Inputs",
			tx: types.Transaction{
				TxID: validTxID,
				Type: types.StandardTransaction,
				Inputs: []types.TransactionInput{
					{PreviousTxHash: zeroHash()}, // Invalid input
				},
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrTransactionInputValidationFailed,
		},
		{
			name: "invalid TransactionOutput in Outputs",
			tx: types.Transaction{
				TxID:   validTxID,
				Type:   types.StandardTransaction,
				Inputs: []types.TransactionInput{validInput},
				Outputs: []types.TransactionOutput{
					{RecipientAddress: nil, Amount: 100}, // Invalid output
				},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrTransactionOutputValidationFailed,
		},
		{
			name: "missing Signature for account-based type",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction, // Assuming this needs sig
				Inputs:    []types.TransactionInput{validInput},
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: nil, // Invalid
				PublicKey: validPubKey,
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingSignature,
		},
		{
			name: "missing PublicKey for account-based type",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction, // Assuming this needs pubkey
				Inputs:    []types.TransactionInput{validInput},
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: nil, // Invalid
				Fee:       10,
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingPublicKey,
		},
		{
			name: "invalid Metadata key (empty)",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
				Metadata:  map[string][]byte{"": []byte("value")}, // Invalid
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidMetadataKey,
		},
		// Note: Metadata key too long and value too large are now tested.
		{
			name: "missing required AI metadata for StimulusTransaction",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StimulusTransaction, // Requires specific AI metadata
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       0, // Stimulus might have no fee or specific fee rules
				Metadata:  map[string][]byte{"note": []byte("stimulus")}, // Missing AILogicID, AIProof
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrMissingRequiredMetadata,
		},
		{
			name: "valid StimulusTransaction with AI metadata",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StimulusTransaction,
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       0,
				Metadata: map[string][]byte{
					"AILogicID": []byte("logic123"),
					"AIProof":   []byte("proofabc"),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid Metadata key (too long)",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
				Metadata:  map[string][]byte{string(make([]byte, 65)): []byte("value")}, // Key too long
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidMetadataKey,
		},
		{
			name: "invalid Metadata value (too large)",
			tx: types.Transaction{
				TxID:      validTxID,
				Type:      types.StandardTransaction,
				Outputs:   []types.TransactionOutput{validOutput},
				Timestamp: validTimestamp,
				Signature: validSignature,
				PublicKey: validPubKey,
				Fee:       10,
				Metadata:  map[string][]byte{"key": make([]byte, 257)}, // Value too large
			},
			wantErr:         true,
			expectedErrType: internalerrors.ErrInvalidMetadataValueSize,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// This is where we'd mock or ensure the Validate methods of sub-structs (Input/Output)
			// behave as expected for the specific test case, if they are not tested in isolation first.
			// For now, we assume their Validate methods are correctly implemented and will be called.
			// If tc.tx.Inputs[0].Validate() is to return an error, that input needs to be invalid.

			err := tc.tx.Validate()
			checkError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
