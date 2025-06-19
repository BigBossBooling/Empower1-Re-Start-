package blockchain

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"empower1.com/empower1blockchain/internal/core"
	"empower1.com/empower1blockchain/internal/state"
)

// Helper to create a basic state manager for tests
func newTestState(t *testing.T) *state.State {
	s, err := state.NewState()
	if err != nil {
		t.Fatalf("Failed to create new state for test: %v", err)
	}
	return s
}

// Helper to create a minimal valid block for testing AddBlock
// Assumes block hash and signatures are handled or placeholder for these tests
func newTestCoreBlock(height int64, prevHash []byte, numTxs int) *core.Block {
	txs := make([]core.Transaction, numTxs)
	for i := 0; i < numTxs; i++ {
		// Create minimal valid-looking transactions for state processing
        // (state.UpdateStateFromBlock will primarily look at inputs/outputs)
        txID := []byte(fmt.Sprintf("testTx%d_block%d", i, height))
        // Simple output for state processing. Input processing in state.UpdateStateFromBlock might
        // require inputs to be valid if TxType is Standard, ContractCall, or ContractDeploy.
        // For simplicity, if numTxs > 0, we create a TxStandard with one output.
        // If inputs are needed for a specific test using this helper, they should be added or this helper adapted.
        outputs := []core.TxOutput{{Value: uint64(i + 1), PubKeyHash: []byte(fmt.Sprintf("testAddr%d", i))}}
        txs[i] = core.Transaction{
            ID:      txID, // Must be set for UpdateStateFromBlock to process outputs correctly
            TxType:  core.TxStandard, // Defaulting to standard for simplicity
            Inputs:  []core.TxInput{}, // Empty inputs for simplicity, state.UpdateStateFromBlock handles this
            Outputs: outputs,
        }
	}

	b := &core.Block{
		Height:          height,
		Timestamp:       time.Now().UnixNano(),
		PrevBlockHash:   prevHash,
		Transactions:    txs,
		ProposerAddress: []byte("testProposer"),
        AIAuditLog:      make([]byte, sha256.Size), // Placeholder
        StateRoot:       make([]byte, sha256.Size), // Placeholder
        Signature:       []byte("testSig"),       // Placeholder
	}
    // Calculate and set hash using the function from blockchain package
    blockHash, err := CalculateBlockHash(b)
    if err != nil {
        // This would be a test setup error, panic or fatal is appropriate in a helper if hash calc fails
        panic(fmt.Sprintf("Failed to calculate block hash in test helper: %v", err))
    }
    b.Hash = blockHash
	return b
}


func TestNewBlockchain(t *testing.T) {
	s := newTestState(t)
	bc, err := NewBlockchain(s)
	if err != nil {
		t.Fatalf("NewBlockchain() error = %v", err)
	}
	if bc == nil {
		t.Fatal("NewBlockchain() returned nil")
	}
	if bc.stateManager == nil {
		t.Error("NewBlockchain() did not set stateManager")
	}
	if len(bc.blocks) != 0 {
		t.Errorf("NewBlockchain() blocks slice not empty, len = %d", len(bc.blocks))
	}
    if bc.CurrentHeight() != -1 {
        t.Errorf("NewBlockchain() current height = %d, want -1", bc.CurrentHeight())
    }

    _, err = NewBlockchain(nil)
    if !errors.Is(err, ErrBlockchainInit) {
        t.Errorf("NewBlockchain(nil) expected ErrBlockchainInit, got %v", err)
    }
}

func TestCreateGenesisBlock(t *testing.T) {
	genesis, err := CreateGenesisBlock()
	if err != nil {
		t.Fatalf("CreateGenesisBlock() error = %v", err)
	}
	if genesis == nil {
		t.Fatal("CreateGenesisBlock() returned nil")
	}
	if genesis.Height != 0 {
		t.Errorf("Genesis block height = %d, want 0", genesis.Height)
	}
	expectedPrevHash := make([]byte, sha256.Size)
	if !bytes.Equal(genesis.PrevBlockHash, expectedPrevHash) {
		t.Errorf("Genesis PrevBlockHash = %x, want %x", genesis.PrevBlockHash, expectedPrevHash)
	}
    if len(genesis.Hash) == 0 {
        t.Error("Genesis block hash not set")
    }
}

func TestAddBlock(t *testing.T) {
	s := newTestState(t)
	bc, _ := NewBlockchain(s)

	// 1. Add valid genesis block
	genesis, _ := CreateGenesisBlock()
	err := bc.AddBlock(genesis)
	if err != nil {
		t.Fatalf("AddBlock(genesis) error = %v", err)
	}
	if bc.CurrentHeight() != 0 {
		t.Errorf("After adding genesis, height = %d, want 0", bc.CurrentHeight())
	}
	if latest := bc.GetLatestBlock(); latest == nil || !bytes.Equal(latest.Hash, genesis.Hash) {
		t.Error("Latest block hash does not match genesis hash")
	}

	// 2. Add valid second block
	block2 := newTestCoreBlock(1, genesis.Hash, 1)
	err = bc.AddBlock(block2)
	if err != nil {
		t.Fatalf("AddBlock(block2) error = %v", err)
	}
	if bc.CurrentHeight() != 1 {
		t.Errorf("After adding block2, height = %d, want 1", bc.CurrentHeight())
	}
    retrievedBlock2, _ := bc.GetBlockByHeight(1)
    if retrievedBlock2 == nil || !bytes.Equal(retrievedBlock2.Hash, block2.Hash) {
        t.Error("Retrieved block2 hash does not match added block2 hash")
    }

	// 3. Attempt to add block with incorrect height
	block3BadHeight := newTestCoreBlock(3, block2.Hash, 0) // Height should be 2
	err = bc.AddBlock(block3BadHeight)
	if !errors.Is(err, ErrInvalidBlockHeight) {
		t.Errorf("AddBlock with bad height, error = %v, want ErrInvalidBlockHeight", err)
	}

	// 4. Attempt to add block with incorrect PrevBlockHash
	block3BadPrevHash := newTestCoreBlock(2, []byte("wrongprevhash"), 0)
	err = bc.AddBlock(block3BadPrevHash)
	if !errors.Is(err, ErrInvalidPrevHash) {
		t.Errorf("AddBlock with bad prevHash, error = %v, want ErrInvalidPrevHash", err)
	}

    // 5. Attempt to add a block that fails state update
    // Create a transaction that tries to spend a non-existent UTXO
    invalidTxInput := []core.TxInput{{TxID: []byte("nonExistentTx"), Vout: 0, PubKey: []byte("dummyKey"), ScriptSig: []byte("dummySig")}}
    invalidTxOutputs := []core.TxOutput{{Value:10, PubKeyHash: []byte("someAddr")}}
    invalidTx := core.Transaction{ID: []byte("invalidTxForState"), TxType: core.TxStandard, Inputs: invalidTxInput, Outputs: invalidTxOutputs}

    block3BadState := newTestCoreBlock(2, block2.Hash, 0) // Correct height and prevHash, initially no txs
    block3BadState.Transactions = []core.Transaction{invalidTx} // Add the invalid tx
    // Recalculate hash for block3BadState as transactions changed
    block3BadStateHash, errCalc := CalculateBlockHash(block3BadState)
    if errCalc != nil {
        t.Fatalf("Error calculating hash for block3BadState: %v", errCalc)
    }
    block3BadState.Hash = block3BadStateHash

    err = bc.AddBlock(block3BadState)
    if err == nil {
        t.Errorf("AddBlock with tx failing state update expected an error, got nil")
    } else {
        // Check if the error is ErrStateUpdateFailed, which wraps the underlying state error (e.g., ErrUTXONotFound)
        if !errors.Is(err, ErrStateUpdateFailed) {
            t.Errorf("AddBlock with tx failing state update, error = %v, want wrapped ErrStateUpdateFailed", err)
        }
        // Optionally, check for the underlying error if needed for more specific tests
        // if !errors.Is(errors.Unwrap(err), state.ErrUTXONotFound) {
        //    t.Errorf("Underlying error for failing state update was not ErrUTXONotFound, got %v", errors.Unwrap(err))
        // }
        t.Logf("Correctly received error for AddBlock with failing state update: %v", err)
    }
    if bc.CurrentHeight() != 1 { // Chain should not have advanced
        t.Errorf("Chain height advanced after block failing state update. Height = %d, want 1", bc.CurrentHeight())
    }
}

func TestGetBlockByHeightAndHash(t *testing.T) {
	s := newTestState(t)
	bc, _ := NewBlockchain(s)
	genesis, _ := CreateGenesisBlock()
	_ = bc.AddBlock(genesis)
	block2 := newTestCoreBlock(1, genesis.Hash, 0)
	_ = bc.AddBlock(block2)

	// Test GetBlockByHeight
	b0, err0 := bc.GetBlockByHeight(0)
	if err0 != nil || b0 == nil || !bytes.Equal(b0.Hash, genesis.Hash) {
		t.Errorf("GetBlockByHeight(0) failed or returned wrong block. Err: %v, Block: %v", err0, b0)
	}
	b1, err1 := bc.GetBlockByHeight(1)
	if err1 != nil || b1 == nil || !bytes.Equal(b1.Hash, block2.Hash) {
		t.Errorf("GetBlockByHeight(1) failed or returned wrong block. Err: %v, Block: %v", err1, b1)
	}
	_, errOutOfBounds := bc.GetBlockByHeight(2)
	if !errors.Is(errOutOfBounds, ErrBlockNotFound) {
		t.Errorf("GetBlockByHeight(2) err = %v, want ErrBlockNotFound", errOutOfBounds)
	}
    _, errNegative := bc.GetBlockByHeight(-1)
	if !errors.Is(errNegative, ErrBlockNotFound) {
		t.Errorf("GetBlockByHeight(-1) err = %v, want ErrBlockNotFound", errNegative)
	}


	// Test GetBlockByHash
	b0Hash, err0Hash := bc.GetBlockByHash(genesis.Hash)
	if err0Hash != nil || b0Hash == nil || !bytes.Equal(b0Hash.Hash, genesis.Hash) {
		t.Errorf("GetBlockByHash(genesis.Hash) failed or returned wrong block. Err: %v, Block: %v", err0Hash, b0Hash)
	}
	b1Hash, err1Hash := bc.GetBlockByHash(block2.Hash)
	if err1Hash != nil || b1Hash == nil || !bytes.Equal(b1Hash.Hash, block2.Hash) {
		t.Errorf("GetBlockByHash(block2.Hash) failed or returned wrong block. Err: %v, Block: %v", err1Hash, b1Hash)
	}
	_, errNotFound := bc.GetBlockByHash([]byte("nonexistenthash"))
	if !errors.Is(errNotFound, ErrBlockNotFound) {
		t.Errorf("GetBlockByHash(nonexistenthash) err = %v, want ErrBlockNotFound", errNotFound)
	}
}

func TestGetLatestBlockAndCurrentHeight(t *testing.T) {
    s := newTestState(t)
	bc, _ := NewBlockchain(s)

    // Empty chain
    if bc.CurrentHeight() != -1 {
        t.Errorf("Empty chain CurrentHeight() = %d, want -1", bc.CurrentHeight())
    }
    if bc.GetLatestBlock() != nil {
        t.Errorf("Empty chain GetLatestBlock() = %v, want nil", bc.GetLatestBlock())
    }

    // After adding genesis
    genesis, _ := CreateGenesisBlock()
	_ = bc.AddBlock(genesis)
    if bc.CurrentHeight() != 0 {
        t.Errorf("Chain with genesis CurrentHeight() = %d, want 0", bc.CurrentHeight())
    }
    if latest := bc.GetLatestBlock(); latest == nil || !bytes.Equal(latest.Hash, genesis.Hash) {
        t.Errorf("Chain with genesis GetLatestBlock() returned %v or wrong hash", latest)
    }

    // After adding another block
    block2 := newTestCoreBlock(1, genesis.Hash, 0)
	_ = bc.AddBlock(block2)
    if bc.CurrentHeight() != 1 {
        t.Errorf("Chain with 2 blocks CurrentHeight() = %d, want 1", bc.CurrentHeight())
    }
    if latest := bc.GetLatestBlock(); latest == nil || !bytes.Equal(latest.Hash, block2.Hash) {
        t.Errorf("Chain with 2 blocks GetLatestBlock() returned %v or wrong hash", latest)
    }
}
