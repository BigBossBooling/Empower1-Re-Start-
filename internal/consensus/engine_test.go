package consensus

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect" // For DeepEqual in some cases
	"sort"    // For deterministic proposer setup
	"sync"
	"testing"
	"time"

	"empower1.com/empower1blockchain/internal/blockchain"
	"empower1.com/empower1blockchain/internal/core"
	"empower1.com/empower1blockchain/internal/mempool"
	"empower1.com/empower1blockchain/internal/network"
)

// --- Mock Implementations ---

type MockProposerService struct {
	CreateProposalBlockFunc   func(height int64, prevHash []byte, proposerAddress []byte) (*core.Block, error)
	CreateProposalBlockCalled bool
	LastHeightCalledWith      int64
	LastPrevHashCalledWith    []byte
	LastProposerAddrCalledWith []byte
	mtx                       sync.Mutex
}

func (m *MockProposerService) CreateProposalBlock(height int64, prevHash []byte, proposerAddress []byte) (*core.Block, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.CreateProposalBlockCalled = true
	m.LastHeightCalledWith = height
	m.LastPrevHashCalledWith = prevHash
	m.LastProposerAddrCalledWith = proposerAddress
	if m.CreateProposalBlockFunc != nil {
		return m.CreateProposalBlockFunc(height, prevHash, proposerAddress)
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("mockBlock%d_hash_from_mockPS", height)))
	// Ensure all fields needed for serialization are present
	return &core.Block{
		Height:          height,
		Timestamp:       time.Now().UnixNano(),
		PrevBlockHash:   prevHash,
		Transactions:    []core.Transaction{},
		ProposerAddress: proposerAddress,
		Signature:       []byte("mockSignature"),
		Hash:            hash[:],
		AIAuditLog:      make([]byte, sha256.Size),
		StateRoot:       make([]byte, sha256.Size),
	}, nil
}

type MockValidationService struct {
	ValidateBlockFunc   func(block *core.Block) error
	ValidateBlockCalled bool
	CalledWithBlock     *core.Block
	mtx                 sync.Mutex
}

func (m *MockValidationService) ValidateBlock(block *core.Block) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.ValidateBlockCalled = true
	m.CalledWithBlock = block
	if m.ValidateBlockFunc != nil {
		return m.ValidateBlockFunc(block)
	}
	return nil
}

type MockBlockchain struct {
	GetLatestBlockFunc func() *core.Block
	AddBlockFunc       func(block *core.Block) error
	CurrentHeightFunc  func() int64
	AddBlockCalledWith *core.Block
	AddBlockCallCount  int
	mtx                sync.Mutex
}

func (m *MockBlockchain) GetLatestBlock() *core.Block {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if m.GetLatestBlockFunc != nil {
		return m.GetLatestBlockFunc()
	}
	return nil
}
func (m *MockBlockchain) AddBlock(b *core.Block) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.AddBlockCalledWith = b
	m.AddBlockCallCount++
	if m.AddBlockFunc != nil {
		return m.AddBlockFunc(b)
	}
	return nil
}
func (m *MockBlockchain) CurrentHeight() int64 {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if m.CurrentHeightFunc != nil {
		return m.CurrentHeightFunc()
	}
	return -1
}

type MockSimulatedNetwork struct {
	BroadcastFunc                func(messageType string, data []byte)
	GetBlockReceptionChannelFunc func() <-chan []byte
	RegisterMessageHandlerFunc   func(handler network.MessageHandler)
	BroadcastCalledWithMessage   string
	BroadcastCalledWithData      []byte
	BroadcastCallCount           int
	messageHandlerRegistered     bool
	testBlockChan                chan []byte // Test-controlled channel
	mtx                          sync.Mutex
}

func NewMockSimulatedNetwork(testChanSize int) *MockSimulatedNetwork {
	return &MockSimulatedNetwork{
		testBlockChan: make(chan []byte, testChanSize),
	}
}
func (m *MockSimulatedNetwork) Broadcast(messageType string, data []byte) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.BroadcastCalledWithMessage = messageType
	m.BroadcastCalledWithData = data
	m.BroadcastCallCount++
	if m.BroadcastFunc != nil {
		m.BroadcastFunc(messageType, data)
	}
}
func (m *MockSimulatedNetwork) GetBlockReceptionChannel() <-chan []byte {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if m.GetBlockReceptionChannelFunc != nil {
		return m.GetBlockReceptionChannelFunc()
	}
	return m.testBlockChan // Return the test-controlled channel
}
func (m *MockSimulatedNetwork) RegisterMessageHandler(h network.MessageHandler) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.messageHandlerRegistered = true
	if m.RegisterMessageHandlerFunc != nil {
		m.RegisterMessageHandlerFunc(h)
	}
}

// --- Helper to setup engine with mocks ---
type TestSetup struct {
	engine *ConsensusEngine
	mockPS *MockProposerService
	mockVS *MockValidationService
	cs     *ConsensusState
	mockBC *MockBlockchain
	mockNet *MockSimulatedNetwork
	nodePrivKey *ecdsa.PrivateKey
	nodeValAddr []byte
}

func setupEngineWithMocks(t *testing.T) TestSetup {
	mockPS := &MockProposerService{}
	mockVS := &MockValidationService{}
	cs := NewConsensusState()

	// Generate a key for this node to be a validator
	nodePrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	nodeValAddr := core.PublicKeyToBytes(&nodePrivKey.PublicKey) // Using core helper

	// Setup dummy validators
	val1Addr := nodeValAddr // Our node
	val2AddrRaw := []byte("validator2_test_addr_beta_raw")
	cs.LoadInitialValidators([]*Validator{
		{Address: val1Addr, Stake: 1000, Reputation: 1.0},
		{Address: val2AddrRaw, Stake: 1500, Reputation: 0.9},
	})

	mockBC := &MockBlockchain{
		CurrentHeightFunc: func() int64 { return -1 },
		GetLatestBlockFunc: func() *core.Block { return nil },
		AddBlockFunc: func(b *core.Block) error { return nil },
	}
	mockNet := NewMockSimulatedNetwork(10)

	engine := NewConsensusEngine(mockPS, mockVS, cs, mockBC, mockNet, nodeValAddr)
	if engine == nil {
		t.Fatalf("NewConsensusEngine returned nil")
	}
	return TestSetup{engine, mockPS, mockVS, cs, mockBC, mockNet, nodePrivKey, nodeValAddr}
}

// --- Test Functions ---
func TestConsensusEngine_StartStop(t *testing.T) {
	ts := setupEngineWithMocks(t)
	go ts.engine.Start()
	time.Sleep(20 * time.Millisecond) // Let ticker run once if interval is short
	ts.engine.Stop()
	// Test passes if it doesn't hang or panic & logs correctly.
	t.Log("ConsensusEngine Start/Stop test completed.")
}

func TestConsensusEngine_AttemptBlockProposal_IsProposer(t *testing.T) {
	ts := setupEngineWithMocks(t)

	// Ensure this node is the only validator, thus guaranteed to be proposer for height 0
	ts.cs.Validators = make(map[string]*Validator) // Clear defaults
	ts.cs.LoadInitialValidators([]*Validator{{Address: ts.nodeValAddr, Stake: 100, Reputation: 1.0}})

	var createdBlock *core.Block
	mockPrevHash := make([]byte, sha256.Size)

	ts.mockPS.CreateProposalBlockFunc = func(height int64, prevHash []byte, proposerAddress []byte) (*core.Block, error) {
		if height != 0 { t.Errorf("Expected height 0, got %d", height) }
		if !bytes.Equal(prevHash, mockPrevHash) { t.Errorf("Expected prevHash %x, got %x", mockPrevHash, prevHash)}
		if !bytes.Equal(proposerAddress, ts.nodeValAddr) { t.Errorf("Proposer address mismatch") }
		// Create a fully initialized block for serialization
		bHash := sha256.Sum256([]byte(fmt.Sprintf("proposedBlock%d", height)))
		createdBlock = &core.Block{ Height: height, PrevBlockHash: prevHash, ProposerAddress: proposerAddress, Hash: bHash[:], Transactions: []core.Transaction{}, AIAuditLog: make([]byte,32), StateRoot: make([]byte,32), Signature: []byte("sig")}
		return createdBlock, nil
	}
	ts.mockBC.CurrentHeightFunc = func() int64 { return -1 } // Propose for height 0 (next is 0)
	ts.mockBC.GetLatestBlockFunc = func() *core.Block { return nil }

	var addBlockCalledOnBC bool
	ts.mockBC.AddBlockFunc = func(b *core.Block) error {
		addBlockCalledOnBC = true
		if !reflect.DeepEqual(b, createdBlock) { t.Errorf("Block added to BC is not the one proposed") }
		return nil
	}

	// Spy on consensus state update
    var csUpdateHeightCalledWith int64 = -1
    originalUpdateHeight := ts.cs.UpdateHeight // Not easy to spy without changing CS or using interface
    // For simplicity, we'll check if AddBlock was called, which should trigger UpdateHeight in proposeBlock.

	ts.engine.attemptBlockProposal()

	ts.mockPS.mtx.Lock()
	if !ts.mockPS.CreateProposalBlockCalled { t.Errorf("ProposerService.CreateProposalBlock was not called") }
	ts.mockPS.mtx.Unlock()

	ts.mockNet.mtx.Lock()
	if ts.mockNet.BroadcastCallCount == 0 { t.Errorf("network.Broadcast was not called") }
	if ts.mockNet.BroadcastCalledWithMessage != "NEW_BLOCK" { t.Errorf("network.Broadcast wrong message type: got %s, want NEW_BLOCK", ts.mockNet.BroadcastCalledWithMessage) }
	serializedCreatedBlock, _ := createdBlock.Serialize()
	if !bytes.Equal(ts.mockNet.BroadcastCalledWithData, serializedCreatedBlock) {t.Errorf("network.Broadcast called with wrong data")}
	ts.mockNet.mtx.Unlock()

	ts.mockBC.mtx.Lock()
	if !addBlockCalledOnBC { t.Errorf("blockchain.AddBlock was not called") }
	ts.mockBC.mtx.Unlock()
    // Due to direct call of cs.UpdateHeight in proposeBlock, we can't easily check it with this CS mock.
    // This test implies that if AddBlock is successful, UpdateHeight was called.
}

func TestConsensusEngine_AttemptBlockProposal_NotProposer(t *testing.T) {
	ts := setupEngineWithMocks(t)
	otherNodeAddr := []byte("other_node_is_validator_first")

	// Ensure otherNodeAddr is sorted first by GetProposerForHeight for height 0
	ts.cs.Validators = make(map[string]*Validator)
	addrsToLoad := []*Validator{
		{Address: otherNodeAddr, Stake: 100, Reputation: 1.0}, // Should be selected
		{Address: ts.nodeValAddr, Stake: 100, Reputation: 1.0},
	}
    // Force order for test predictability based on current GetProposerForHeight
    sort.Slice(addrsToLoad, func(i, j int) bool {
        return bytes.Compare(addrsToLoad[i].Address, addrsToLoad[j].Address) < 0
    })
    // Ensure 'otherNodeAddr' is indeed first after sorting if its hex is smaller
    // This requires careful choice of addresses or mocking GetProposerForHeight.
    // For this test, ensure otherNodeAddr's hex string sorts before nodeValAddr's hex string.
    // Example: otherNodeAddr = []byte("a"), nodeValAddr = []byte("b")
    if !(hex.EncodeToString(otherNodeAddr) < hex.EncodeToString(ts.nodeValAddr)) {
        // Swap them if our assumption about sort order is wrong for these specific byte slices
        // This is to make otherNodeAddr guaranteed to be the first in sorted list for height 0.
         addrsToLoad[0], addrsToLoad[1] = addrsToLoad[1], addrsToLoad[0]
    }
	ts.cs.LoadInitialValidators(addrsToLoad)

	mockBC := ts.mockBC
	mockBC.CurrentHeightFunc = func() int64 { return -1 }
	mockBC.GetLatestBlockFunc = func() *core.Block { return nil }

	ts.engine.attemptBlockProposal()

	ts.mockPS.mtx.Lock()
	if ts.mockPS.CreateProposalBlockCalled {
		t.Errorf("ProposerService.CreateProposalBlock was called when it was NOT node's turn")
	}
	ts.mockPS.mtx.Unlock()
}

func TestConsensusEngine_ProcessIncomingBlock_Valid(t *testing.T) {
	ts := setupEngineWithMocks(t)
	testBlock := newTestSimBlock("block_valid_for_processing")
	serializedBlock, _ := testBlock.Serialize()

	var addBlockCalled bool
	var updateHeightCalledOnCS bool
	ts.mockVS.ValidateBlockFunc = func(b *core.Block) error {
		if !bytes.Equal(b.Hash, testBlock.Hash) {t.Errorf("ValidateBlock called with wrong block hash")}
		return nil
	}
	ts.mockBC.AddBlockFunc = func(b *core.Block) error {
		addBlockCalled = true
		if !bytes.Equal(b.Hash, testBlock.Hash) {t.Errorf("AddBlock called with wrong block hash")}
		ts.mockBC.CurrentHeightFunc = func() int64 { return b.Height }
		return nil
	}
    // Spy on ConsensusState.UpdateHeight
    originalUpdateHeight := ts.cs.UpdateHeight
    ts.cs.UpdateHeight = func(h int64) {
        updateHeightCalledOnCS = true
        if h != testBlock.Height {t.Errorf("UpdateHeight called with wrong height. Got %d, want %d", h, testBlock.Height)}
        originalUpdateHeight(h) // Call original to maintain logging
    }

	go ts.engine.Start()
	// Send block to the engine's network channel
	// Need to cast testBlockChan to writeable for the test
	writeableBlockChan := ts.mockNet.testBlockChan
	writeableBlockChan <- serializedBlock
	time.Sleep(50 * time.Millisecond)
	ts.engine.Stop()

	ts.mockVS.mtx.Lock()
	if !ts.mockVS.ValidateBlockCalled { t.Errorf("ValidationService.ValidateBlock not called") }
	ts.mockVS.mtx.Unlock()

	if !addBlockCalled { t.Errorf("Blockchain.AddBlock was not called") }
	if !updateHeightCalledOnCS {t.Errorf("ConsensusState.UpdateHeight was not called")}
}

func TestConsensusEngine_ProcessIncomingBlock_DeserializeError(t *testing.T) {
    ts := setupEngineWithMocks(t)
    go ts.engine.Start()
    defer ts.engine.Stop()

    badData := []byte("this is not a gob encoded block")
    writeableBlockChan := ts.mockNet.testBlockChan
	writeableBlockChan <- badData
    time.Sleep(50 * time.Millisecond) // Allow processing

    // Check that ValidateBlock and AddBlock were not called
    ts.mockVS.mtx.Lock()
    if ts.mockVS.ValidateBlockCalled { t.Errorf("ValidateBlock called after deserialize error") }
    ts.mockVS.mtx.Unlock()
    ts.mockBC.mtx.Lock()
    if ts.mockBC.AddBlockCalledWith != nil { t.Errorf("AddBlock called after deserialize error") }
    ts.mockBC.mtx.Unlock()
    // Log message "Failed to deserialize received block" should have been printed.
}


func TestConsensusEngine_ProcessIncomingBlock_InvalidValidation(t *testing.T) {
	ts := setupEngineWithMocks(t)
	testBlock := newTestSimBlock("block_invalid_validation")
	serializedBlock, _ := testBlock.Serialize()

	ts.mockVS.ValidateBlockFunc = func(b *core.Block) error { return errors.New("validation failed") }
	var addBlockCalled bool
	ts.mockBC.AddBlockFunc = func(b *core.Block) error { addBlockCalled = true; return nil }

	go ts.engine.Start()
	writeableBlockChan := ts.mockNet.testBlockChan
	writeableBlockChan <- serializedBlock
	time.Sleep(50 * time.Millisecond)
	ts.engine.Stop()

	ts.mockVS.mtx.Lock()
	if !ts.mockVS.ValidateBlockCalled { t.Errorf("ValidationService.ValidateBlock not called") }
	ts.mockVS.mtx.Unlock()
	if addBlockCalled { t.Errorf("AddBlock called for invalid block") }
}

func TestConsensusEngine_ProcessIncomingBlock_AddBlockFails(t *testing.T) {
	ts := setupEngineWithMocks(t)
	testBlock := newTestSimBlock("block_add_fails")
    serializedBlock, _ := testBlock.Serialize()

	ts.mockVS.ValidateBlockFunc = func(b *core.Block) error { return nil }
	var addBlockCalled bool
	ts.mockBC.AddBlockFunc = func(b *core.Block) error { addBlockCalled = true; return errors.New("add block failed") }

    var updateHeightCalledOnCS bool
    originalUpdateHeight := ts.cs.UpdateHeight
    ts.cs.UpdateHeight = func(h int64) {
        updateHeightCalledOnCS = true
        originalUpdateHeight(h)
    }

	go ts.engine.Start()
	writeableBlockChan := ts.mockNet.testBlockChan
	writeableBlockChan <- serializedBlock
	time.Sleep(50 * time.Millisecond)
	ts.engine.Stop()

	ts.mockVS.mtx.Lock()
	if !ts.mockVS.ValidateBlockCalled { t.Errorf("ValidationService.ValidateBlock not called") }
	ts.mockVS.mtx.Unlock()
	if !addBlockCalled {t.Errorf("AddBlock was not called (it should have been, but failed).")}
    if updateHeightCalledOnCS {t.Errorf("ConsensusState.UpdateHeight was called even though AddBlock failed.")}
}

func TestConsensusEngine_Start_Loop_ProposerRotation(t *testing.T) {
    nodeAddr1 := []byte("val_addr_one") // hex will be 76616c5f616464725f6f6e65
    nodeAddr2 := []byte("val_addr_two") // hex will be 76616c5f616464725f74776f

    // Ensure nodeAddr1's hex sorts before nodeAddr2's hex for deterministic testing
    if !(hex.EncodeToString(nodeAddr1) < hex.EncodeToString(nodeAddr2)) {
        t.Fatalf("Test setup error: nodeAddr1 hex should be less than nodeAddr2 hex for predictable proposer order.")
    }

    ts := setupEngineWithMocks(t) // This will use nodeAddr1 as default nodeValAddr due to setup logic
    ts.engine.nodeValidatorAddress = nodeAddr1 // Explicitly set for clarity

    // Setup CS with these two validators. nodeAddr1 will be proposer for height 0, 2, ...
    // nodeAddr2 will be proposer for height 1, 3, ...
    ts.cs.Validators = make(map[string]*Validator)
    ts.cs.LoadInitialValidators([]*Validator{
        {Address: nodeAddr1, Stake: 100, Reputation: 1.0},
        {Address: nodeAddr2, Stake: 100, Reputation: 1.0},
    })

    var proposalHeights []int64
    ts.mockPS.CreateProposalBlockFunc = func(height int64, prevHash []byte, proposerAddress []byte) (*core.Block, error) {
        ts.mockPS.mtx.Lock()
        proposalHeights = append(proposalHeights, height)
        ts.mockPS.mtx.Unlock()
        // Return a valid block so the engine's proposeBlock continues
        hash := sha256.Sum256([]byte(fmt.Sprintf("block_h%d", height)))
        return &core.Block{Height: height, Hash: hash[:], ProposerAddress: proposerAddress, Transactions: []core.Transaction{}, AIAuditLog: make([]byte,32), StateRoot: make([]byte,32), Signature: []byte("sig")}, nil
    }

    // Simulate a few ticks of the engine's attemptBlockProposal
    // Tick 1: CurrentHeight = -1, NextHeight = 0. Proposer should be nodeAddr1.
    ts.mockBC.CurrentHeightFunc = func() int64 { return -1 }
    ts.mockBC.GetLatestBlockFunc = func() *core.Block { return nil }
    ts.engine.attemptBlockProposal()

    // Tick 2: Simulate block 0 added. CurrentHeight = 0, NextHeight = 1. Proposer should be nodeAddr2.
    // Our node (nodeAddr1) should NOT propose.
    // We need to simulate block 0 being added to advance height for the next proposal attempt.
    // For this test, we'll directly manipulate the mock blockchain's height.
    ts.mockBC.CurrentHeightFunc = func() int64 { return 0 }
    // For GetLatestBlock, we need a mock block 0
    block0Hash := sha256.Sum256([]byte("block_h0"))
    ts.mockBC.GetLatestBlockFunc = func() *core.Block { return &core.Block{Height: 0, Hash: block0Hash[:]}}
    ts.engine.attemptBlockProposal()

    // Tick 3: CurrentHeight = 1, NextHeight = 2. Proposer should be nodeAddr1 again.
    ts.mockBC.CurrentHeightFunc = func() int64 { return 1 }
    block1Hash := sha256.Sum256([]byte("block_h1"))
    ts.mockBC.GetLatestBlockFunc = func() *core.Block { return &core.Block{Height: 1, Hash: block1Hash[:]}}
    ts.engine.attemptBlockProposal()

    ts.mockPS.mtx.Lock()
    defer ts.mockPS.mtx.Unlock()

    expectedProposalHeights := []int64{0, 2}
    if !reflect.DeepEqual(proposalHeights, expectedProposalHeights) {
        t.Errorf("Proposals made at incorrect heights. Got %v, want %v", proposalHeights, expectedProposalHeights)
    }
}

// TODO: More tests:
// - ProposeBlock detail testing (transactions from mempool) - needs ProposerService mock refinement.
// - Full Start() loop test with multiple ticks and block receptions from multiple peers (more complex integration test).
// - Error handling in NewConsensusEngine for nil dependencies (already partly covered by t.Fatal in setup).
// - Test for when GetProposerForHeight returns an error.
// - Test for when proposerService.CreateProposalBlock returns an error or nil block.
// - Test for when proposedBlock.Serialize() returns an error.
// - Test for when ce.blockchain.AddBlock(proposedBlock) in proposeBlock fails.
// - Test for when block reception channel is closed in Start() loop.
// - Test for when core.DeserializeBlock fails for received block data.
```
