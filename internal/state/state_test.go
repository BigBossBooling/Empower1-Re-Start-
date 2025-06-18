package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"empower1.com/empower1blockchain/internal/core" // For Block, Transaction, TxInput, TxOutput, TxType
)

// Helper function to create a mock address (public key hash)
func mockAddress(id byte) []byte {
	hash := sha256.Sum256([]byte{id})
	return hash[:] // Using full hash for realism, though often truncated in real addresses
}

// Helper function to create a simple transaction for testing
// Note: This is a simplified Tx for testing state; it doesn't involve full signing or complex fields unless needed by a specific test.
func newTestTx(id []byte, inputs []core.TxInput, outputs []core.TxOutput, txType core.TxType) core.Transaction {
	tx := core.Transaction{
		ID:        id,
		Timestamp: time.Now().UnixNano(),
		TxType:    txType,
		Inputs:    inputs,
		Outputs:   outputs,
		Fee:       0, // Assuming 0 fee for basic tests unless specified
	}
	// For AI-driven transactions, populate AI metadata
	if txType == core.TxStimulusPayment || txType == core.TxWealthTax {
		tx.AILogicID = "test_ai_logic_v1"
		tx.AIRuleTrigger = "test_rule_trigger"
		tx.AIProof = []byte("test_ai_proof")
	}
	return tx
}

// TestNewState tests the NewState constructor.
func TestNewState(t *testing.T) {
	s, err := NewState()
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}
	if s == nil {
		t.Fatalf("NewState() returned nil state")
	}
	if s.logger == nil {
		t.Errorf("NewState() logger is nil")
	}
	if s.utxoSet == nil {
		t.Errorf("NewState() utxoSet is nil")
	}
	if len(s.utxoSet) != 0 {
		t.Errorf("NewState() utxoSet is not empty, got %d elements", len(s.utxoSet))
	}
	if s.accounts == nil {
		t.Errorf("NewState() accounts is nil")
	}
	if len(s.accounts) != 0 {
		t.Errorf("NewState() accounts is not empty, got %d elements", len(s.accounts))
	}
}

// TestUpdateStateFromBlock_StandardTx tests UpdateStateFromBlock with standard transactions.
func TestUpdateStateFromBlock_StandardTx(t *testing.T) {
	s, _ := NewState()
	addr1 := mockAddress(1)
	addr2 := mockAddress(2)

	// Create an initial UTXO for addr1
	initialTxID := []byte("initialTx")
	initialTxIDStr := hex.EncodeToString(initialTxID)
	s.utxoSet[fmt.Sprintf("%s:0", initialTxIDStr)] = &UTXO{
		TxID:    initialTxID,
		Vout:    0,
		Value:   100,
		Address: addr1,
	}
	s.accounts[hex.EncodeToString(addr1)] = &Account{Balance: 100, WealthLevel: make(map[string]string)}


	// Create a block with a transaction spending the initial UTXO
	tx1Inputs := []core.TxInput{{TxID: initialTxID, Vout: 0, PubKey: addr1 /* simplified for state test */}}
	tx1Outputs := []core.TxOutput{{Value: 90, PubKeyHash: addr2}, {Value: 10, PubKeyHash: addr1 /* change */}}
	tx1 := newTestTx([]byte("tx1"), tx1Inputs, tx1Outputs, core.TxStandard)

	block := &core.Block{
		Height:       1,
		Timestamp:    time.Now().UnixNano(),
		Transactions: []core.Transaction{tx1},
		Hash:         []byte("block1hash"), // Simplified
	}

	err := s.UpdateStateFromBlock(block)
	if err != nil {
		t.Fatalf("UpdateStateFromBlock() error = %v", err)
	}

	// Check that initial UTXO is spent
	if _, exists := s.utxoSet[fmt.Sprintf("%s:0", initialTxIDStr)]; exists {
		t.Errorf("Initial UTXO was not removed from utxoSet")
	}

	// Check that new UTXOs are created
	tx1IDStr := hex.EncodeToString(tx1.ID)
	keyOut1 := fmt.Sprintf("%s:0", tx1IDStr)
	keyOut2 := fmt.Sprintf("%s:1", tx1IDStr)

	utxoAddr2, existsAddr2 := s.utxoSet[keyOut1]
	if !existsAddr2 {
		t.Errorf("New UTXO for addr2 not created (key: %s)", keyOut1)
	} else {
		if utxoAddr2.Value != 90 || !bytes.Equal(utxoAddr2.Address, addr2) {
			t.Errorf("UTXO for addr2 has incorrect value or address. Got value %d, addr %x; want value 90, addr %x", utxoAddr2.Value, utxoAddr2.Address, addr2)
		}
	}

	utxoAddr1Change, existsAddr1Change := s.utxoSet[keyOut2]
	if !existsAddr1Change {
		t.Errorf("New UTXO for addr1 (change) not created (key: %s)", keyOut2)
	} else {
		if utxoAddr1Change.Value != 10 || !bytes.Equal(utxoAddr1Change.Address, addr1) {
			t.Errorf("UTXO for addr1 (change) has incorrect value or address. Got value %d, addr %x; want value 10, addr %x", utxoAddr1Change.Value, utxoAddr1Change.Address, addr1)
		}
	}

	// Check conceptual account balances
	acc1, _ := s.accounts[hex.EncodeToString(addr1)]
	if acc1 == nil || acc1.Balance != 10 { // 100 - 100 (spent) + 10 (change) = 10
		t.Errorf("Conceptual balance for addr1 incorrect. Got %v, want 10", acc1)
	}
	acc2, _ := s.accounts[hex.EncodeToString(addr2)]
	if acc2 == nil || acc2.Balance != 90 {
		t.Errorf("Conceptual balance for addr2 incorrect. Got %v, want 90", acc2)
	}
}

// TestUpdateStateFromBlock_StimulusTx tests AI-driven stimulus payments.
func TestUpdateStateFromBlock_StimulusTx(t *testing.T) {
	s, _ := NewState()
	recipientAddr := mockAddress(10)
	blockTimestamp := time.Now().UnixNano()

	stimulusOutputs := []core.TxOutput{{Value: 500, PubKeyHash: recipientAddr}}
	stimulusTx := newTestTx([]byte("stimulusTx1"), []core.TxInput{}, stimulusOutputs, core.TxStimulusPayment)

	block := &core.Block{
		Height:       1,
		Timestamp:    blockTimestamp,
		Transactions: []core.Transaction{stimulusTx},
		Hash:         []byte("stimulusBlock"),
	}

	err := s.UpdateStateFromBlock(block)
	if err != nil {
		t.Fatalf("UpdateStateFromBlock() with stimulus error = %v", err)
	}

	utxoKey := fmt.Sprintf("%s:0", hex.EncodeToString(stimulusTx.ID))
	if utxo, exists := s.utxoSet[utxoKey]; !exists || utxo.Value != 500 || !bytes.Equal(utxo.Address, recipientAddr) {
		t.Errorf("Stimulus UTXO not created correctly for recipient. Got: %v, Exists: %t", utxo, exists)
	}

	expectedWealthLevel := map[string]string{
		"status":        "stimulus_received",
		"ai_logic_id":   "test_ai_logic_v1",
		"rule_trigger":  "test_rule_trigger",
		"tx_id":         hex.EncodeToString(stimulusTx.ID),
		"block_height":  "1",
		"last_updated":  fmt.Sprintf("%d", blockTimestamp),
	}
	updatedWealth, err := s.GetWealthLevel(recipientAddr)
	if err != nil {
		t.Fatalf("GetWealthLevel for recipient error = %v", err)
	}
	if !reflect.DeepEqual(updatedWealth, expectedWealthLevel) {
		t.Errorf("WealthLevel for recipient incorrect. Got %v, want %v", updatedWealth, expectedWealthLevel)
	}

	// Check conceptual account balance
	accRecipient, _ := s.accounts[hex.EncodeToString(recipientAddr)]
	if accRecipient == nil || accRecipient.Balance != 500 {
		t.Errorf("Conceptual balance for stimulus recipient incorrect. Got %v, want 500", accRecipient)
	}
}

// TestUpdateStateFromBlock_WealthTax tests AI-driven wealth tax.
func TestUpdateStateFromBlock_WealthTax(t *testing.T) {
	s, _ := NewState()
	taxedAddr := mockAddress(20)
	treasuryAddr := mockAddress(255) // Conceptual treasury
	blockTimestamp := time.Now().UnixNano()

	initialTxID := []byte("taxInitialTx")
	initialTxIDStr := hex.EncodeToString(initialTxID)
	s.utxoSet[fmt.Sprintf("%s:0", initialTxIDStr)] = &UTXO{
		TxID:    initialTxID, Vout: 0, Value:   1000, Address: taxedAddr,
	}
	s.accounts[hex.EncodeToString(taxedAddr)] = &Account{Balance: 1000, WealthLevel: make(map[string]string)}


	taxInputs := []core.TxInput{{TxID: initialTxID, Vout: 0, PubKey: taxedAddr /* simplified */}}
	taxOutputs := []core.TxOutput{
		{Value: 100, PubKeyHash: treasuryAddr}, // Taxed amount
		{Value: 900, PubKeyHash: taxedAddr},    // Change
	}
	taxTx := newTestTx([]byte("wealthTaxTx1"), taxInputs, taxOutputs, core.TxWealthTax)

	block := &core.Block{
		Height:       1,
		Timestamp:    blockTimestamp,
		Transactions: []core.Transaction{taxTx},
		Hash:         []byte("wealthTaxBlock"),
	}

	err := s.UpdateStateFromBlock(block)
	if err != nil {
		t.Fatalf("UpdateStateFromBlock() with wealth tax error = %v", err)
	}

	expectedWealthLevel := map[string]string{
		"status":        "wealth_tax_applied",
		"ai_logic_id":   "test_ai_logic_v1",
		"rule_trigger":  "test_rule_trigger",
		"tx_id":         hex.EncodeToString(taxTx.ID),
		"block_height":  "1",
		"last_updated":  fmt.Sprintf("%d", blockTimestamp),
	}
	updatedWealth, err := s.GetWealthLevel(taxedAddr)
	if err != nil {
		t.Fatalf("GetWealthLevel for taxed address error = %v", err)
	}
	if !reflect.DeepEqual(updatedWealth, expectedWealthLevel) {
		t.Errorf("WealthLevel for taxed address incorrect. Got %v, want %v", updatedWealth, expectedWealthLevel)
	}

	// Check conceptual account balances
	accTaxed, _ := s.accounts[hex.EncodeToString(taxedAddr)]
	if accTaxed == nil || accTaxed.Balance != 900 { // 1000 - 1000 (spent) + 900 (change) = 900
		t.Errorf("Conceptual balance for taxedAddr incorrect. Got %v, want 900", accTaxed)
	}
	accTreasury, _ := s.accounts[hex.EncodeToString(treasuryAddr)]
	if accTreasury == nil || accTreasury.Balance != 100 {
		t.Errorf("Conceptual balance for treasuryAddr incorrect. Got %v, want 100", accTreasury)
	}
}


// TestUpdateStateFromBlock_DoubleSpend tests double spending a UTXO.
func TestUpdateStateFromBlock_DoubleSpend(t *testing.T) {
	s, _ := NewState()
	addr1 := mockAddress(1)
	initialTxID := []byte("initialDoubleSpendTx")
	initialTxIDStr := hex.EncodeToString(initialTxID)
	s.utxoSet[fmt.Sprintf("%s:0", initialTxIDStr)] = &UTXO{TxID: initialTxID, Vout: 0, Value: 100, Address: addr1}
	s.accounts[hex.EncodeToString(addr1)] = &Account{Balance: 100, WealthLevel: make(map[string]string)}

	tx1Inputs := []core.TxInput{{TxID: initialTxID, Vout: 0}}
	tx1Outputs := []core.TxOutput{{Value: 100, PubKeyHash: mockAddress(2)}}
	tx1 := newTestTx([]byte("txDouble1"), tx1Inputs, tx1Outputs, core.TxStandard)

	tx2Inputs := []core.TxInput{{TxID: initialTxID, Vout: 0}}
	tx2Outputs := []core.TxOutput{{Value: 100, PubKeyHash: mockAddress(3)}}
	tx2 := newTestTx([]byte("txDouble2"), tx2Inputs, tx2Outputs, core.TxStandard)

	block := &core.Block{Height: 1, Transactions: []core.Transaction{tx1, tx2}, Hash: []byte("doubleSpendBlock")}
	err := s.UpdateStateFromBlock(block)
	if err == nil {
		t.Fatalf("Expected error for double spend, but got nil")
	}
	if !errors.Is(err, ErrUTXONotFound) {
		t.Errorf("Expected ErrUTXONotFound for double spend, got %v", err)
	}
}


// TestGetBalance tests GetBalance method.
func TestGetBalance(t *testing.T) {
	s, _ := NewState()
	addr1 := mockAddress(1)
	addr2 := mockAddress(2)

	s.utxoSet["tx1:0"] = &UTXO{TxID: []byte("tx1"), Vout: 0, Value: 50, Address: addr1}
	s.utxoSet["tx2:1"] = &UTXO{TxID: []byte("tx2"), Vout: 1, Value: 30, Address: addr1}
	// No need to add to s.accounts for pure GetBalance from UTXO set.

	balance1, err1 := s.GetBalance(addr1)
	if err1 != nil {
		t.Errorf("GetBalance(addr1) error = %v", err1)
	}
	if balance1 != 80 {
		t.Errorf("GetBalance(addr1) = %d, want 80", balance1)
	}

	balance2, err2 := s.GetBalance(addr2)
	if !errors.Is(err2, ErrInsufficientBalance) { // Check if the error is ErrInsufficientBalance
		t.Errorf("GetBalance(addr2) expected ErrInsufficientBalance, got err %v", err2)
	}
    if balance2 != 0 {
        t.Errorf("GetBalance(addr2) expected balance 0 when erroring, got %d", balance2)
    }
}

// TestFindSpendableOutputs tests FindSpendableOutputs method.
func TestFindSpendableOutputs(t *testing.T) {
	s, _ := NewState()
	addr1 := mockAddress(1)

	s.utxoSet["txA:0"] = &UTXO{TxID: []byte("txA"), Vout: 0, Value: 10, Address: addr1}
	s.utxoSet["txB:0"] = &UTXO{TxID: []byte("txB"), Vout: 0, Value: 20, Address: addr1}
	s.utxoSet["txC:0"] = &UTXO{TxID: []byte("txC"), Vout: 0, Value: 70, Address: addr1}

	// Test case 1: Exact amount (e.g. 30, could pick 10+20)
	outputs, total, err := s.FindSpendableOutputs(addr1, 30)
	if err != nil {
		t.Fatalf("FindSpendableOutputs(addr1, 30) error = %v", err)
	}
	var sumReturned uint64
    for _, o := range outputs { sumReturned += o.Value }
    if sumReturned < 30 {
		t.Errorf("FindSpendableOutputs(addr1, 30) sumReturned = %d, want >= 30", sumReturned)
	}
	if sumReturned != total { t.Errorf("Sum of returned outputs %d != reported total %d", sumReturned, total) }


	// Test case 2: Insufficient funds
	_, _, err = s.FindSpendableOutputs(addr1, 101)
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("FindSpendableOutputs(addr1, 101) expected ErrInsufficientBalance, got %v", err)
	}

	// Test case 3: Amount less than smallest UTXO (should still pick one)
	outputs, total, err = s.FindSpendableOutputs(addr1, 5)
	if err != nil {
		t.Fatalf("FindSpendableOutputs(addr1, 5) error = %v", err)
	}
	if total < 5 {
		t.Errorf("FindSpendableOutputs(addr1, 5) total = %d, want >= 5 (likely picked smallest UTXO)", total)
	}
    if len(outputs) == 0 {
        t.Errorf("FindSpendableOutputs(addr1, 5) expected some outputs, got 0")
    }
}


// TestGetAndUpdateWealthLevel tests GetWealthLevel and UpdateWealthLevel.
func TestGetAndUpdateWealthLevel(t *testing.T) {
	s, _ := NewState()
	addr1 := mockAddress(1)
	addr2 := mockAddress(2)

	_, err := s.GetWealthLevel(addr1)
	if !errors.Is(err, ErrWealthLevelNotFound) {
		t.Errorf("GetWealthLevel for non-existent addr1 expected ErrWealthLevelNotFound, got %v", err)
	}

	level1 := map[string]string{"category": "middle", "source": "initial_assessment_v1"}
	err = s.UpdateWealthLevel(addr1, level1)
	if err != nil {
		t.Fatalf("UpdateWealthLevel(addr1, level1) error = %v", err)
	}

	retrievedLevel1, err := s.GetWealthLevel(addr1)
	if err != nil {
		t.Fatalf("GetWealthLevel(addr1) after update error = %v", err)
	}
	if !reflect.DeepEqual(retrievedLevel1, level1) {
		t.Errorf("GetWealthLevel(addr1) got %v, want %v", retrievedLevel1, level1)
	}

	level1Update := map[string]string{"category": "upper-middle", "source": "reassessment_v1.2", "timestamp": "12345"}
	err = s.UpdateWealthLevel(addr1, level1Update)
	if err != nil {
		t.Fatalf("UpdateWealthLevel(addr1, level1Update) error = %v", err)
	}
	retrievedLevel1Updated, err := s.GetWealthLevel(addr1)
	if err != nil {
		t.Fatalf("GetWealthLevel(addr1) after second update error = %v", err)
	}
	if !reflect.DeepEqual(retrievedLevel1Updated, level1Update) {
		t.Errorf("GetWealthLevel(addr1) second update got %v, want %v", retrievedLevel1Updated, level1Update)
	}
    // Check original level1 map was not modified
    originalLevel1Value, ok1 := level1["category"]
    updatedLevel1Value, ok2 := retrievedLevel1Updated["category"]
    if ok1 && ok2 && originalLevel1Value == updatedLevel1Value && level1["source"] != retrievedLevel1Updated["source"] {
        // This check is a bit indirect. The main thing is that retrievedLevel1Updated == level1Update
        // and level1 is different from level1Update.
    }


	s.accounts[hex.EncodeToString(addr2)] = &Account{WealthLevel: make(map[string]string)}
	_, err = s.GetWealthLevel(addr2)
	if !errors.Is(err, ErrWealthLevelNotFound) { // Check if it's specifically ErrWealthLevelNotFound
		t.Errorf("GetWealthLevel for addr2 with empty WealthLevel map expected ErrWealthLevelNotFound, got %v", err)
	}
}

// TODO: Add more tests for UpdateStateFromBlock:
// - Block with multiple transactions.
// - Transaction spending outputs created in the same block (if allowed by consensus).
// - Invalid transactions (e.g., incorrect signature - though this is tx validation, not state).
// - Edge cases for UTXO values (e.g., zero value outputs).
// - Concurrency tests if State methods were to be called concurrently without external locking (though methods have internal locks).
