package core_test

import (
	"bytes"
	"crypto/sha256"
	"empower1/internal/core"
	"empower1/internal/core/types"
	internalerrors "empower1/internal/errors"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"
)

// Helper to create a sample hash for tests
func sampleTxTestHash(val byte) types.Hash {
	var h types.Hash
	for i := range h {
		h[i] = val
	}
	return h
}

// Helper for checkError
func checkTxError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
	t.Helper()
	if wantErr {
		if gotErr == nil {
			t.Errorf("expected an error but got nil")
			return
		}
		if wantErrType != nil && !errors.Is(gotErr, wantErrType) {
			t.Errorf("got error '%v' (type %T), want error to wrap or be of type %T (base error: '%v')", gotErr, gotErr, wantErrType, wantErrType)
		}
	} else if gotErr != nil {
		t.Errorf("did not expect an error but got: %v (type %T)", gotErr, gotErr)
	}
}


func TestTransaction_HashingAndID(t *testing.T) {
	// Basic transaction for consistent hashing test
	tx1 := &types.Transaction{
		Type:      types.StandardTransaction,
		Inputs:    []types.TransactionInput{{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 0}},
		Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}},
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Fee:       10,
		Nonce:     1,
		Metadata:  map[string][]byte{"keyA": []byte("valueA")},
	}

	t.Run("PrepareDataForHashing Determinism", func(t *testing.T) {
		data1, err1 := tx1.PrepareDataForHashing()
		if err1 != nil {
			t.Fatalf("PrepareDataForHashing failed for tx1: %v", err1)
		}

		// Create a semantically identical transaction but with different internal order for maps/slices if not sorted
		tx2 := &types.Transaction{
			Type:      types.StandardTransaction,
			Inputs:    []types.TransactionInput{{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 0}}, // Same order for simplicity now
			Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}}, // Same order
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Fee:       10,
			Nonce:     1,
			Metadata:  map[string][]byte{"keyA": []byte("valueA")}, // Same order
		}
		data2, err2 := tx2.PrepareDataForHashing()
		if err2 != nil {
			t.Fatalf("PrepareDataForHashing failed for tx2: %v", err2)
		}

		if !bytes.Equal(data1, data2) {
			t.Errorf("PrepareDataForHashing is not deterministic for identical semantic content.\nData1: %x\nData2: %x", data1, data2)
		}

		// Test with different metadata order (keys are sorted by PrepareDataForHashing)
		tx3 := &types.Transaction{
			Type:      types.StandardTransaction,
			Inputs:    []types.TransactionInput{{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 0}},
			Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}},
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Fee:       10,
			Nonce:     1,
			Metadata:  map[string][]byte{"keyB": []byte("valueB"), "keyA": []byte("valueA")}, // Different order
		}
		tx4 := &types.Transaction{ // Identical to tx3 but map declared in different order
			Type:      types.StandardTransaction,
			Inputs:    []types.TransactionInput{{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 0}},
			Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}},
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Fee:       10,
			Nonce:     1,
			Metadata:  map[string][]byte{"keyA": []byte("valueA"), "keyB": []byte("valueB")},
		}
		data3, _ := tx3.PrepareDataForHashing()
		data4, _ := tx4.PrepareDataForHashing()
		if !bytes.Equal(data3, data4) {
			t.Errorf("PrepareDataForHashing for metadata is not deterministic for different map declaration orders (should be sorted).\nData3: %x\nData4: %x", data3, data4)
		}
	})

	t.Run("Hash and SetTxID", func(t *testing.T) {
		tx := &types.Transaction{
			Type:      types.StandardTransaction,
			Inputs:    []types.TransactionInput{{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 0, Signature: []byte("s"), PublicKey: []byte("p")}},
			Outputs:   []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}},
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Fee:       10,
			Nonce:     1,
		}

		err := tx.SetTxID()
		if err != nil {
			t.Fatalf("SetTxID failed: %v", err)
		}
		if tx.TxID == (types.Hash{}) {
			t.Error("TxID is zero after SetTxID")
		}

		// Known hash test (hard to do without a reference implementation of gob encoding order)
		// For now, just check it's not zero.
		// A change in PrepareDataForHashing will change this hash.
		// log.Printf("Generated TxID: %x", tx.TxID)

		// Test ErrTxIDAlreadySet
		err = tx.SetTxID()
		checkTxError(t, err, internalerrors.ErrTxIDAlreadySet, true)
	})
}


func TestNewStandardTransaction(t *testing.T) {
	validInputs := []types.TransactionInput{{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 0, Signature: []byte("s"), PublicKey: []byte("p")}}
	validOutputs := []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}}

	tests := []struct {
		name     string
		inputs   []types.TransactionInput
		outputs  []types.TransactionOutput
		fee      uint64
		nonce    uint64
		metadata map[string][]byte
		wantErr  bool
		errType  error
	}{
		{"valid standard tx", validInputs, validOutputs, 10, 1, nil, false, nil},
		{"no outputs", validInputs, []types.TransactionOutput{}, 10, 1, nil, true, internalerrors.ErrNoTransactionOutputs},
		{"no inputs", []types.TransactionInput{}, validOutputs, 10, 1, nil, true, internalerrors.ErrEmptyInputsForSpendingTx},
		{"nil inputs", nil, validOutputs, 10, 1, nil, true, internalerrors.ErrEmptyInputsForSpendingTx},
		{"nil outputs", validInputs, nil, 10, 1, nil, true, internalerrors.ErrNoTransactionOutputs},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := core.NewStandardTransaction(tc.inputs, tc.outputs, tc.fee, tc.nonce, tc.metadata)
			checkTxError(t, err, tc.errType, tc.wantErr)
			if !tc.wantErr {
				if tx == nil {
					t.Fatal("NewStandardTransaction returned nil transaction on success")
				}
				if tx.TxID == (types.Hash{}) {
					t.Error("NewStandardTransaction did not set TxID")
				}
				if tx.Type != types.StandardTransaction {
					t.Errorf("NewStandardTransaction set incorrect type: got %v, want %v", tx.Type, types.StandardTransaction)
				}
				if !reflect.DeepEqual(tx.Inputs, tc.inputs) { // Note: reflect.DeepEqual with slices
                     t.Errorf("Inputs mismatch. Got %+v, want %+v", tx.Inputs, tc.inputs)
                }
			}
		})
	}
}

func TestNewStimulusTransaction(t *testing.T) {
	validOutputs := []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 100}}

	tests := []struct {
		name      string
		outputs   []types.TransactionOutput
		aiLogicID string
		aiProof   []byte
		wantErr   bool
		errType   error
	}{
		{"valid stimulus tx", validOutputs, "logic1", []byte("proof1"), false, nil},
		{"no outputs", []types.TransactionOutput{}, "logic1", []byte("proof1"), true, internalerrors.ErrNoTransactionOutputs},
		{"output with zero amount", []types.TransactionOutput{{RecipientAddress: types.Address("addr1"), Amount: 0}}, "logic1", []byte("proof1"), true, internalerrors.ErrInvalidOutputAmount},
		{"missing AILogicID", validOutputs, "", []byte("proof1"), true, internalerrors.ErrMissingRequiredMetadata},
		{"missing AIProof", validOutputs, "logic1", []byte{}, true, internalerrors.ErrMissingRequiredMetadata},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := core.NewStimulusTransaction(tc.outputs, tc.aiLogicID, tc.aiProof)
			checkTxError(t, err, tc.errType, tc.wantErr)
			if !tc.wantErr {
				if tx == nil {
					t.Fatal("NewStimulusTransaction returned nil transaction on success")
				}
				if tx.TxID == (types.Hash{}) {
					t.Error("NewStimulusTransaction did not set TxID")
				}
				if tx.Type != types.StimulusTransaction {
					t.Errorf("NewStimulusTransaction set incorrect type: got %v, want %v", tx.Type, types.StimulusTransaction)
				}
				if tx.Metadata["AILogicID"] == nil || string(tx.Metadata["AILogicID"]) != tc.aiLogicID {
					t.Error("AILogicID not set correctly in metadata")
				}
			}
		})
	}
}

func TestNewTaxTransaction(t *testing.T) {
    // Tax transactions can be varied. This is a basic test.
    // Example: tax collected from no specific input, and output to a treasury (or burned if outputs nil)
    treasuryAddress := types.Address("treasury")
    validTaxOutputs := []types.TransactionOutput{{RecipientAddress: treasuryAddress, Amount: 50}}

    tests := []struct {
        name      string
        inputs    []types.TransactionInput // Can be nil if tax is not from specific UTXOs
        outputs   []types.TransactionOutput // Can be nil if tax is burned
        aiLogicID string
        aiProof   []byte
        metadata  map[string][]byte
        wantErr   bool
        errType   error
    }{
        {"valid tax tx", nil, validTaxOutputs, "tax_logic_v1", []byte("tax_proof_abc"), nil, false, nil},
        {"missing AILogicID", nil, validTaxOutputs, "", []byte("tax_proof_abc"), nil, true, internalerrors.ErrMissingRequiredMetadata},
        {"missing AIProof", nil, validTaxOutputs, "tax_logic_v1", []byte{}, nil, true, internalerrors.ErrMissingRequiredMetadata},
        {"valid with custom metadata", nil, validTaxOutputs, "tax_logic_v1", []byte("tax_proof_abc"), map[string][]byte{"source":[]byte("fees")}, false, nil},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            tx, err := core.NewTaxTransaction(tc.inputs, tc.outputs, tc.aiLogicID, tc.aiProof, tc.metadata)
            checkTxError(t, err, tc.errType, tc.wantErr)
            if !tc.wantErr {
                if tx == nil {
                    t.Fatal("NewTaxTransaction returned nil on success")
                }
                if tx.TxID == (types.Hash{}) {
                    t.Error("NewTaxTransaction did not set TxID")
                }
                if tx.Type != types.TaxTransaction {
                    t.Errorf("NewTaxTransaction set incorrect type: got %v, want %v", tx.Type, types.TaxTransaction)
                }
                 if string(tx.Metadata["AILogicID"]) != tc.aiLogicID {
                    t.Error("AILogicID not set correctly in metadata for tax tx")
                }
            }
        })
    }
}

// Mock for sorting inputs/outputs for testing PrepareDataForHashing determinism more deeply
// (Not strictly needed if PrepareDataForHashing itself sorts copies)
func TestTransaction_PrepareDataForHashing_Determinism_WithUnsortedData(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	inputs1 := []types.TransactionInput{
		{PreviousTxHash: sampleTxTestHash(2), OutputIndex: 0},
		{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 1},
	}
	inputs2 := []types.TransactionInput{ // Same inputs, different order
		{PreviousTxHash: sampleTxTestHash(1), OutputIndex: 1},
		{PreviousTxHash: sampleTxTestHash(2), OutputIndex: 0},
	}

	outputs1 := []types.TransactionOutput{
		{RecipientAddress: types.Address("addrB"), Amount: 200},
		{RecipientAddress: types.Address("addrA"), Amount: 100},
	}
	outputs2 := []types.TransactionOutput{ // Same outputs, different order
		{RecipientAddress: types.Address("addrA"), Amount: 100},
		{RecipientAddress: types.Address("addrB"), Amount: 200},
	}

	metadata1 := map[string][]byte{"keyB": []byte("valB"), "keyA": []byte("valA")}
	metadata2 := map[string][]byte{"keyA": []byte("valA"), "keyB": []byte("valB")}

	txA := &types.Transaction{Type: types.StandardTransaction, Inputs: inputs1, Outputs: outputs1, Timestamp: ts, Fee: 1, Nonce: 1, Metadata: metadata1}
	txB := &types.Transaction{Type: types.StandardTransaction, Inputs: inputs2, Outputs: outputs2, Timestamp: ts, Fee: 1, Nonce: 1, Metadata: metadata2}

	dataA, errA := txA.PrepareDataForHashing()
	if errA != nil { t.Fatalf("txA.PrepareDataForHashing() error = %v", errA) }
	dataB, errB := txB.PrepareDataForHashing()
	if errB != nil { t.Fatalf("txB.PrepareDataForHashing() error = %v", errB) }

	if !bytes.Equal(dataA, dataB) {
		t.Errorf("PrepareDataForHashing not deterministic with unsorted inputs/outputs/metadata map declaration.\nDataA: %x\nDataB: %x", dataA, dataB)
	}
}
```
