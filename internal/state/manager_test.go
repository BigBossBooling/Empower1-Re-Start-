package state_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"

	"empower1/internal/core/types"
	"empower1/internal/state"
	internalerrors "empower1/internal/errors"
)

// Helper to create a sample hash for tests
func sampleStateHash(val byte) types.Hash {
	var h types.Hash
	for i := range h {
		h[i] = val
	}
	return h
}

// Helper to make UTXO key for test setup
func makeTestUTXOKey(txID types.Hash, index uint32) string {
	return hex.EncodeToString(txID[:]) + ":" + strconv.FormatUint(uint64(index), 10)
}

// Helper for creating valid length addresses for tests, assuming 20 bytes expected
func makeValidTestAddress(base string, length int) types.Address {
	addr := make([]byte, length)
	// Fill with a pattern for uniqueness if base is too short
	for i := 0; i < length; i++ {
		if i < len(base) {
			addr[i] = base[i]
		} else {
			addr[i] = byte(i % 256) // Fill with a pattern
		}
	}
	return types.Address(addr)
}


func TestUpdateStateFromBlock_UTXOManagement(t *testing.T) {
	initialAddr1 := makeValidTestAddress("address1", 20)

	t.Run("Valid Spend and Creation", func(t *testing.T) {
		smLocal := state.NewStateManager()

		fundingTxID := sampleStateHash(0xAA)
		fundingHeader := types.BlockHeader{
			Version: 1, PreviousBlockHash: sampleStateHash(0xA9), MerkleRoot: sampleStateHash(0xA0), Timestamp: time.Now(), Nonce: 0, Difficulty: 1, Height: 1, ProposerAddress: makeValidTestAddress("proposer", 20), BlockHash: sampleStateHash(0xAB),
		}
		fundingBlock := &types.Block{
			Header: fundingHeader,
			Transactions: []types.Transaction{
				{
					TxID: fundingTxID, Type: types.StandardTransaction,
					Outputs:   []types.TransactionOutput{{RecipientAddress: initialAddr1, Amount: 100}},
					Timestamp: time.Now(), Signature: []byte("sig"), PublicKey: []byte("pubkey"), // Added sig/pubkey
				},
			},
			BlockHash: sampleStateHash(0xAC),
		}
		err := smLocal.UpdateStateFromBlock(fundingBlock)
		if err != nil {
			t.Fatalf("Failed to process funding block: %v", err)
		}

		// To verify, we use GetBalance / FindSpendableOutputs later or would need an exported GetUTXO.
		// For this test, we check side effects via subsequent spends.

		spendTxID := sampleStateHash(2)
		newAddr := makeValidTestAddress("address2", 20)
		spendHeader := types.BlockHeader{
			Version: 1, PreviousBlockHash: fundingBlock.Header.BlockHash, MerkleRoot: sampleStateHash(0xB0), Timestamp: time.Now().Add(1 * time.Minute), Nonce: 0, Difficulty: 1, Height: 2, ProposerAddress: makeValidTestAddress("proposer", 20), BlockHash: sampleStateHash(0xBA),
		}
		blockToProcess := &types.Block{
			Header: spendHeader,
			Transactions: []types.Transaction{
				{
					TxID: spendTxID, Type: types.StandardTransaction,
					Inputs:    []types.TransactionInput{{PreviousTxHash: fundingTxID, OutputIndex: 0, Signature: []byte("sig"), PublicKey: []byte("pubkey")}},
					Outputs:   []types.TransactionOutput{{RecipientAddress: newAddr, Amount: 90}},
					Timestamp: time.Now(), Fee: 10, Signature: []byte("sig2"), PublicKey: []byte("pubkey2"),
				},
			},
			BlockHash: sampleStateHash(0xBB),
		}

		err = smLocal.UpdateStateFromBlock(blockToProcess)
		if err != nil {
			t.Fatalf("UpdateStateFromBlock failed for valid spend: %v", err)
		}

		// Verify with GetBalance
		balNewAddr, _ := smLocal.GetBalance(newAddr)
		if balNewAddr != 90 {
			t.Errorf("New address balance incorrect. Got %d, want 90", balNewAddr)
		}
		balInitialAddr, _ := smLocal.GetBalance(initialAddr1)
		if balInitialAddr != 0 { // 100 was spent
			t.Errorf("Initial address balance incorrect. Got %d, want 0", balInitialAddr)
		}
	})

	t.Run("Spend Non-Existent UTXO", func(t *testing.T) {
		smLocal := state.NewStateManager()
		nonExistentTxID := sampleStateHash(3)
		header := types.BlockHeader{
			Version: 1, PreviousBlockHash: sampleStateHash(0xC9), MerkleRoot: sampleStateHash(0xC0), Timestamp: time.Now(), Nonce: 0, Difficulty: 1, Height: 1, ProposerAddress: makeValidTestAddress("proposer", 20), BlockHash: sampleStateHash(0xCA),
		}
		blockToProcess := &types.Block{
			Header: header,
			Transactions: []types.Transaction{
				{
					TxID: sampleStateHash(4), Type: types.StandardTransaction,
					Inputs:    []types.TransactionInput{{PreviousTxHash: nonExistentTxID, OutputIndex: 0, Signature: []byte("sig"), PublicKey: []byte("pubkey")}},
					Outputs:   []types.TransactionOutput{{RecipientAddress: makeValidTestAddress("address3", 20), Amount: 50}},
					Timestamp: time.Now(), Signature: []byte("sig"), PublicKey: []byte("pubkey"),
				},
			},
			BlockHash: sampleStateHash(0xCB),
		}

		err := smLocal.UpdateStateFromBlock(blockToProcess)
		if !errors.Is(err, internalerrors.ErrUTXONotFound) {
			t.Errorf("Expected ErrUTXONotFound, got %v", err)
		}
	})

	t.Run("Double Spend Attempt in Block", func(t *testing.T) {
		smLocal := state.NewStateManager()
		fundingTxID_ds := sampleStateHash(0xDA)
		addrToFund_ds := makeValidTestAddress("addrToFundDS", 20)

		fundHeader_ds := types.BlockHeader{Version: 1, PreviousBlockHash: sampleStateHash(0xD9), MerkleRoot: sampleStateHash(0xD0), Timestamp: time.Now(), Nonce: 0, Difficulty: 1, Height: 1, ProposerAddress: makeValidTestAddress("proposerDS", 20), BlockHash: sampleStateHash(0xDB)}
		fundingBlock_ds := &types.Block{
			Header: fundHeader_ds,
			Transactions: []types.Transaction{{
				TxID: fundingTxID_ds, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{{RecipientAddress: addrToFund_ds, Amount: 100}}, Timestamp: time.Now(), Signature: []byte("sig"), PublicKey: []byte("pubkey"),
			}}, BlockHash: sampleStateHash(0xDC) }
		if err := smLocal.UpdateStateFromBlock(fundingBlock_ds); err != nil {
			t.Fatalf("Funding block failed for double spend test: %v", err)
		}

		tx1SpendTxID_ds := sampleStateHash(0xE0)
		tx2SpendTxID_ds := sampleStateHash(0xE1)

		doubleSpendHeader_ds := types.BlockHeader{Version: 1, PreviousBlockHash: fundHeader_ds.BlockHash, MerkleRoot: sampleStateHash(0xE0A), Timestamp: time.Now().Add(1 * time.Minute), Nonce: 0, Difficulty: 1, Height: 2, ProposerAddress: makeValidTestAddress("proposerDS2", 20), BlockHash: sampleStateHash(0xEA)}
		blockToProcess_ds := &types.Block{
			Header: doubleSpendHeader_ds,
			Transactions: []types.Transaction{
				{
					TxID: tx1SpendTxID_ds, Type: types.StandardTransaction,
					Inputs:    []types.TransactionInput{{PreviousTxHash: fundingTxID_ds, OutputIndex: 0, Signature: []byte("sig1"), PublicKey: []byte("pubkey1")}},
					Outputs:   []types.TransactionOutput{{RecipientAddress: makeValidTestAddress("addrDouble1", 20), Amount: 50}},
					Timestamp: time.Now(), Signature: []byte("sigTx1"), PublicKey: []byte("pubkeyTx1"),
				},
				{
					TxID: tx2SpendTxID_ds, Type: types.StandardTransaction,
					Inputs:    []types.TransactionInput{{PreviousTxHash: fundingTxID_ds, OutputIndex: 0, Signature: []byte("sig2"), PublicKey: []byte("pubkey2")}},
					Outputs:   []types.TransactionOutput{{RecipientAddress: makeValidTestAddress("addrDouble2", 20), Amount: 30}},
					Timestamp: time.Now(), Signature: []byte("sigTx2"), PublicKey: []byte("pubkeyTx2"),
				},
			},
			BlockHash: sampleStateHash(0xEB),
		}

		err := smLocal.UpdateStateFromBlock(blockToProcess_ds)
		if !errors.Is(err, internalerrors.ErrUTXONotFound) {
			t.Errorf("Expected ErrUTXONotFound for double spend attempt in block, got %v", err)
		}
	})

	t.Run("Duplicate Output UTXO Creation", func(t *testing.T) {
		smLocal := state.NewStateManager()
		existingTxID_dup := sampleStateHash(0xF0)

		preExistingHeader_dup := types.BlockHeader{Version:1, PreviousBlockHash: sampleStateHash(0xF0-1), MerkleRoot: sampleStateHash(0xF0A), Timestamp: time.Now(), Nonce:0, Difficulty:1, Height:1, ProposerAddress: makeValidTestAddress("proposerDup1", 20), BlockHash: sampleStateHash(0xF1)}
		preExistingBlock_dup := &types.Block{
			Header: preExistingHeader_dup,
			Transactions: []types.Transaction{{
				TxID: existingTxID_dup, Type: types.StandardTransaction,
				Outputs:   []types.TransactionOutput{{RecipientAddress: makeValidTestAddress("addrPreDup", 20), Amount: 10}},
				Timestamp: time.Now(), Signature: []byte("sig"), PublicKey: []byte("pubkey"),
			}}, BlockHash: sampleStateHash(0xF2) }
		if err := smLocal.UpdateStateFromBlock(preExistingBlock_dup); err != nil {
			t.Fatalf("Pre-existing block failed for duplicate output test: %v", err)
		}

		collisionHeader_dup := types.BlockHeader{Version:1, PreviousBlockHash: preExistingHeader_dup.BlockHash, MerkleRoot: sampleStateHash(0xF0B), Timestamp: time.Now().Add(1 * time.Minute), Nonce:0, Difficulty:1, Height:2, ProposerAddress: makeValidTestAddress("proposerDup2", 20), BlockHash: sampleStateHash(0xF3)}
		blockToProcess_dup := &types.Block{
			Header: collisionHeader_dup,
			Transactions: []types.Transaction{
				{
					TxID: existingTxID_dup, Type: types.StandardTransaction,
					Outputs:   []types.TransactionOutput{{RecipientAddress: makeValidTestAddress("addrNewDup", 20), Amount: 20}},
					Timestamp: time.Now(), Signature: []byte("sig"), PublicKey: []byte("pubkey"),
				},
			},
			BlockHash: sampleStateHash(0xF4),
		}

		err := smLocal.UpdateStateFromBlock(blockToProcess_dup)
		if !errors.Is(err, internalerrors.ErrCriticalStateCorruption) {
			t.Errorf("Expected ErrCriticalStateCorruption for duplicate output UTXO, got %v", err)
		}
	})
}

func TestStateManager_GetBalance(t *testing.T) {
	addr1_gb := makeValidTestAddress("address1_gb", 20)
	addr2_gb := makeValidTestAddress("address2_gb", 20)

	txID1_gb := sampleStateHash(0xA1)
	txID2_gb := sampleStateHash(0xA2)
	txID3_gb := sampleStateHash(0xA3)

	header_gb1 := types.BlockHeader{BlockHash: sampleStateHash(0xB1), ProposerAddress: makeValidTestAddress("proposer_gb1", 20), Timestamp: time.Now(), Version: 1, MerkleRoot: sampleStateHash(0xB1A), Height:1, PreviousBlockHash: sampleStateHash(0xB0)}
	fundingBlock1 := &types.Block{
		Header: header_gb1,
		Transactions: []types.Transaction{
			{TxID: txID1_gb, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{{RecipientAddress: addr1_gb, Amount: 100}}, Timestamp: time.Now(), Signature:[]byte("s"), PublicKey:[]byte("p")},
			{TxID: txID2_gb, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{{RecipientAddress: addr1_gb, Amount: 50}}, Timestamp: time.Now(), Signature:[]byte("s"), PublicKey:[]byte("p")},
		},
		BlockHash: sampleStateHash(0xB1_1),
	}
	header_gb2 := types.BlockHeader{BlockHash: sampleStateHash(0xB2), ProposerAddress: makeValidTestAddress("proposer_gb2", 20), Timestamp: time.Now(), Version: 1, MerkleRoot: sampleStateHash(0xB2A), Height:2, PreviousBlockHash: header_gb1.BlockHash}
	fundingBlock2 := &types.Block{
		Header: header_gb2,
		Transactions: []types.Transaction{
			{TxID: txID3_gb, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{{RecipientAddress: addr2_gb, Amount: 75}}, Timestamp: time.Now(), Signature:[]byte("s"), PublicKey:[]byte("p")},
		},
		BlockHash: sampleStateHash(0xB2_1),
	}

	sm := state.NewStateManager()
	if err := sm.UpdateStateFromBlock(fundingBlock1); err != nil {
		t.Fatalf("Failed to process fundingBlock1 for GetBalance: %v", err)
	}
	if err := sm.UpdateStateFromBlock(fundingBlock2); err != nil {
		t.Fatalf("Failed to process fundingBlock2 for GetBalance: %v", err)
	}

	tests := []struct {
		name            string
		address         types.Address
		expectedBalance uint64
		expectErr       bool
	}{
		{"addr1 has correct balance", addr1_gb, 150, false},
		{"addr2 has correct balance", addr2_gb, 75, false},
		{"unknown address has zero balance", makeValidTestAddress("unknownAddress", 20), 0, false},
		{"empty valid-length address has zero balance", make([]byte, 20), 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := sm.GetBalance(tc.address)
			if (err != nil) != tc.expectErr {
				t.Errorf("GetBalance() error = %v, expectErr %v", err, tc.expectErr)
				return
			}
			if balance != tc.expectedBalance {
				t.Errorf("GetBalance() = %v, want %v", balance, tc.expectedBalance)
			}
		})
	}
}

func TestStateManager_FindSpendableOutputs(t *testing.T) {
	addr1_fso := makeValidTestAddress("address1_fso", 20)

	txID1_fso := sampleStateHash(0xC1)
	txID2_fso := sampleStateHash(0xC2)
	txID3_fso := sampleStateHash(0xC3)

	header_fso := types.BlockHeader{BlockHash: sampleStateHash(0xD1), ProposerAddress: makeValidTestAddress("proposer_fso", 20), Timestamp: time.Now(), Version: 1, MerkleRoot: sampleStateHash(0xD1A), Height:1, PreviousBlockHash: sampleStateHash(0xD0)}
	fundingBlock_fso := &types.Block{
		Header: header_fso,
		Transactions: []types.Transaction{
			{TxID: txID1_fso, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{
				{RecipientAddress: addr1_fso, Amount: 20},
				{RecipientAddress: addr1_fso, Amount: 30},
			}, Timestamp: time.Now(), Signature:[]byte("s"), PublicKey:[]byte("p")},
			{TxID: txID2_fso, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{
				{RecipientAddress: addr1_fso, Amount: 50},
			}, Timestamp: time.Now(), Signature:[]byte("s"), PublicKey:[]byte("p")},
			{TxID: txID3_fso, Type: types.StandardTransaction, Outputs: []types.TransactionOutput{
				{RecipientAddress: makeValidTestAddress("otherAddr_fso", 20), Amount: 1000},
			}, Timestamp: time.Now(), Signature:[]byte("s"), PublicKey:[]byte("p")},
		},
		BlockHash: sampleStateHash(0xD1_1),
	}

	sm := state.NewStateManager()
	if err := sm.UpdateStateFromBlock(fundingBlock_fso); err != nil {
		t.Fatalf("Failed to process fundingBlock for FindSpendableOutputs: %v", err)
	}

	tests := []struct {
		name            string
		address         types.Address
		amountNeeded    uint64
		expectedCount   int
		expectedSum     uint64
		expectErr       bool
		expectedErrType error
	}{
		{"find exact amount (single UTXO strategy)", addr1_fso, 20, 1, 20, false, nil},
		{"find exact amount (multiple UTXOs needed)", addr1_fso, 50, 2, 50, false, nil},
		{"find amount requiring all UTXOs", addr1_fso, 100, 3, 100, false, nil},
		{"find amount slightly less than total, gets all", addr1_fso, 90, 3, 100, false, nil},
		{"find amount more than total (insufficient)", addr1_fso, 101, 3, 100, true, internalerrors.ErrInsufficientBalance},
		{"find for address with no UTXOs", makeValidTestAddress("noFundsAddr_fso", 20), 50, 0, 0, true, internalerrors.ErrInsufficientBalance},
		{"find zero amount (should return no UTXOs)", addr1_fso, 0, 0, 0, false, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			utxos, sum, err := sm.FindSpendableOutputs(tc.address, tc.amountNeeded)

			if (err != nil) != tc.expectErr {
				t.Errorf("FindSpendableOutputs() error presence = %v, expectErr %v. Error: %v", (err != nil), tc.expectErr, err)
			}
			if tc.expectErr && tc.expectedErrType != nil {
				if !errors.Is(err, tc.expectedErrType) {
					t.Errorf("FindSpendableOutputs() error type = %T, want %T. Error: %v", err, tc.expectedErrType, err)
				}
			}
			if len(utxos) != tc.expectedCount {
				t.Errorf("FindSpendableOutputs() utxos count = %d, want %d", len(utxos), tc.expectedCount)
			}
			if sum != tc.expectedSum {
				t.Errorf("FindSpendableOutputs() sum = %d, want %d", sum, tc.expectedSum)
			}

			if !tc.expectErr && len(utxos) > 0 {
				for _, utxo := range utxos {
					if !bytes.Equal(utxo.RecipientAddress, tc.address) {
						t.Errorf("FindSpendableOutputs() returned UTXO for wrong address. Got %s, want %s", hex.EncodeToString(utxo.RecipientAddress), hex.EncodeToString(tc.address))
					}
				}
			}
		})
	}
}
```
