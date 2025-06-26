package core_test

import (
	"bytes"
	"crypto/sha256"
	"empower1/internal/core"
	"empower1/internal/core/types"
	internalerrors "empower1/internal/errors"
	"errors"
	"reflect"
	"testing"
	"time"
)

// Re-use sampleTxTestHash if it's general enough, or make a new one for block tests
// func sampleBlockTestHash(val byte) types.Hash { ... }

func TestBlockHeader_SerializeForHashing(t *testing.T) {
	ts := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	header1 := &types.BlockHeader{
		Version:           1,
		PreviousBlockHash: sampleTxTestHash(1), // Using existing helper for simplicity
		MerkleRoot:        sampleTxTestHash(2),
		Timestamp:         ts,
		Nonce:             123,
		Difficulty:        10,
		Height:            100,
		ProposerAddress:   types.Address("proposer1"),
	}
	header2 := &types.BlockHeader{ // Identical to header1
		Version:           1,
		PreviousBlockHash: sampleTxTestHash(1),
		MerkleRoot:        sampleTxTestHash(2),
		Timestamp:         ts,
		Nonce:             123,
		Difficulty:        10,
		Height:            100,
		ProposerAddress:   types.Address("proposer1"),
	}
	header3 := &types.BlockHeader{ // Different timestamp
		Version:           1,
		PreviousBlockHash: sampleTxTestHash(1),
		MerkleRoot:        sampleTxTestHash(2),
		Timestamp:         ts.Add(time.Second),
		Nonce:             123,
		Difficulty:        10,
		Height:            100,
		ProposerAddress:   types.Address("proposer1"),
	}

	data1, err1 := header1.SerializeForHashing()
	if err1 != nil {
		t.Fatalf("header1.SerializeForHashing() failed: %v", err1)
	}
	data2, err2 := header2.SerializeForHashing()
	if err2 != nil {
		t.Fatalf("header2.SerializeForHashing() failed: %v", err2)
	}
	data3, err3 := header3.SerializeForHashing()
	if err3 != nil {
		t.Fatalf("header3.SerializeForHashing() failed: %v", err3)
	}

	if !bytes.Equal(data1, data2) {
		t.Errorf("SerializeForHashing is not deterministic for identical headers.\nData1: %x\nData2: %x", data1, data2)
	}
	if bytes.Equal(data1, data3) {
		t.Errorf("SerializeForHashing produced identical data for different headers (timestamp changed).\nData1: %x\nData3: %x", data1, data3)
	}
}

func TestBlock_HashTransactions(t *testing.T) {
	tx1ID := sampleTxTestHash(0xA1)
	tx2ID := sampleTxTestHash(0xA2)
	tx1 := types.Transaction{TxID: tx1ID} // Only TxID needed for current HashTransactions
	tx2 := types.Transaction{TxID: tx2ID}

	tests := []struct {
		name         string
		transactions []types.Transaction
		expectPanic  bool // For Tx with no ID
		expectedHash types.Hash
		wantErr      bool
		errType      error
	}{
		{
			name:         "no transactions",
			transactions: []types.Transaction{},
			expectedHash: sha256.Sum256([]byte{}),
			wantErr:      false,
		},
		{
			name:         "one transaction",
			transactions: []types.Transaction{tx1},
			expectedHash: sha256.Sum256(tx1ID[:]), // Simplified hash for one item
			wantErr:      false,
		},
		{
			name:         "multiple transactions",
			transactions: []types.Transaction{tx1, tx2},
			expectedHash: sha256.Sum256(bytes.Join([][]byte{tx1ID[:], tx2ID[:]}, []byte{})),
			wantErr:      false,
		},
		{
			name:         "transaction with no TxID",
			transactions: []types.Transaction{{}}, // Zero TxID
			wantErr:      true,
			errType:      internalerrors.ErrInvalidTransactionID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			block := &types.Block{Transactions: tc.transactions}
			hash, err := block.HashTransactions()

			checkTxError(t, err, tc.errType, tc.wantErr) // Using checkTxError from transaction_test for now
			if !tc.wantErr {
				if hash != tc.expectedHash {
					t.Errorf("HashTransactions() got hash %x, want %x", hash, tc.expectedHash)
				}
			}
		})
	}
}

func TestNewBlock(t *testing.T) {
	prevHash := sampleTxTestHash(10)
	proposer := types.Address("proposerAlice")
	tx1ID := sampleTxTestHash(0xB1)
	tx1 := types.Transaction{TxID: tx1ID, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{{RecipientAddress:types.Address("a"), Amount:1}}} // Minimal valid tx for this test

	// To ensure tx1 itself is valid if NewBlock calls tx.Validate()
	// For now, assuming NewBlock doesn't call tx.Validate() on passed transactions,
    // but it does check if TxID is set.
    // If tx.SetTxID() was called on tx1, it would have its own TxID properly.
    // The current NewBlock expects TxIDs to be pre-set.


	tests := []struct {
		name         string
		version      uint32
		prevHash     types.Hash
		height       uint64
		proposer     types.Address
		difficulty   uint64
		nonce        uint64
		transactions []types.Transaction
		wantErr      bool
		errType      error
	}{
		{
			name:         "valid new block with transactions",
			version:      1, prevHash: prevHash, height: 101, proposer: proposer, difficulty: 20, nonce: 12345,
			transactions: []types.Transaction{tx1},
			wantErr:      false,
		},
		{
			name:         "valid new block with no transactions",
			version:      1, prevHash: prevHash, height: 101, proposer: proposer, difficulty: 20, nonce: 12345,
			transactions: []types.Transaction{},
			wantErr:      false,
		},
		{
			name: "new block with transaction missing TxID",
			version:      1, prevHash: prevHash, height: 101, proposer: proposer, difficulty: 20, nonce: 12345,
			transactions: []types.Transaction{{Type: types.StandardTransaction}}, // No TxID
			wantErr:      true,
			errType:      internalerrors.ErrInvalidTransactionID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			block, err := core.NewBlock(tc.version, tc.prevHash, tc.height, tc.proposer, tc.difficulty, tc.nonce, tc.transactions)
			checkTxError(t, err, tc.errType, tc.wantErr)

			if !tc.wantErr {
				if block == nil {
					t.Fatal("NewBlock() returned nil on success")
				}
				if block.Header.Version != tc.version ||
					block.Header.PreviousBlockHash != tc.prevHash ||
					block.Header.Height != tc.height ||
					!bytes.Equal(block.Header.ProposerAddress, tc.proposer) ||
					block.Header.Difficulty != tc.difficulty ||
					block.Header.Nonce != tc.nonce {
					t.Errorf("NewBlock() header fields mismatch. Got %+v", block.Header)
				}
				if !reflect.DeepEqual(block.Transactions, tc.transactions) {
					t.Errorf("NewBlock() transactions mismatch. Got %+v, want %+v", block.Transactions, tc.transactions)
				}

				// Check MerkleRoot
				expectedMerkleRoot, _ := (&types.Block{Transactions: tc.transactions}).HashTransactions()
				if block.Header.MerkleRoot != expectedMerkleRoot {
					t.Errorf("NewBlock() MerkleRoot mismatch. Got %x, want %x", block.Header.MerkleRoot, expectedMerkleRoot)
				}

				// Check BlockHash (is not zero)
				if block.BlockHash == (types.Hash{}) {
					t.Error("NewBlock() did not set BlockHash")
				}
			}
		})
	}
}
```
