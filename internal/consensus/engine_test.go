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
	"log" // Used in setupEngineWithMocks for dummy handler, could be t.Log
	"reflect"
	"sort" // Used in ConsensusState for proposer selection
	"sync"
	"testing"
	"time"

	"empower1.com/empower1blockchain/internal/blockchain"
	"empower1.com/empower1blockchain/internal/core"
	"empower1.com/empower1blockchain/internal/mempool" // Needed by ProposerService, thus by engine through it
	"empower1.com/empower1blockchain/internal/network"
)

// --- Mock Implementations ---

// MockProposerService
type MockProposerService struct {
	CreateProposalBlockFunc func(height int64, prevHash []byte, proposerAddress []byte) (*core.Block, error)
	CreateProposalBlockCalled bool
	LastHeightCalledWith    int64
	mtx                     sync.Mutex
}

func (m *MockProposerService) CreateProposalBlock(height int64, prevHash []byte, proposerAddress []byte) (*core.Block, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.CreateProposalBlockCalled = true
	m.LastHeightCalledWith = height
	if m.CreateProposalBlockFunc != nil {
		return m.CreateProposalBlockFunc(height, prevHash, proposerAddress)
	}
	// Return a generic valid block with a unique hash for testing
	hash := sha256.Sum256([]byte(fmt.Sprintf("mockBlock%d_hash", height)))
	return &core.Block{Height: height, Hash: hash[:], ProposerAddress: proposerAddress, Transactions: []core.Transaction{}}, nil
}

// MockValidationService
type MockValidationService struct {
	ValidateBlockFunc   func(block *core.Block) error
	ValidateBlockCalled bool
	mtx                 sync.Mutex
}

func (m *MockValidationService) ValidateBlock(block *core.Block) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.ValidateBlockCalled = true
	if m.ValidateBlockFunc != nil {
		return m.ValidateBlockFunc(block)
	}
	return nil // Assume valid by default
}

// MockBlockchain
type MockBlockchain struct {
	GetLatestBlockFunc func() *core.Block
	AddBlockFunc       func(block *core.Block) error
	CurrentHeightFunc  func() int64
	AddBlockCalledWith *core.Block
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
	return -1 // Default to empty chain
}

// MockSimulatedNetwork
type MockSimulatedNetwork struct {
	BroadcastFunc                func(messageType string, data []byte)
	GetBlockReceptionChannelFunc func() <-chan []byte
	RegisterMessageHandlerFunc   func(handler network.MessageHandler)
	BroadcastCalledWithMessage   string
	BroadcastCalledWithData      []byte
	messageHandlerRegistered     bool
	mtx                          sync.Mutex
}

func (m *MockSimulatedNetwork) Broadcast(messageType string, data []byte) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.BroadcastCalledWithMessage = messageType
	m.BroadcastCalledWithData = data
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
	// Return a closed channel by default if not set, to avoid blocking tests indefinitely
	ch := make(chan []byte)
	// close(ch) // Closing immediately might be too restrictive for some tests.
	// Let tests provide a specific channel via the Func.
	// For a generic mock, a small buffered channel might be better.
	return make(chan []byte, 1) // Small buffer
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
func setupEngineWithMocks(t *testing.T, nodeValAddrRaw []byte) (*ConsensusEngine, *MockProposerService, *MockValidationService, *ConsensusState, *MockBlockchain, *MockSimulatedNetwork) {
	mockPS := &MockProposerService{}
	mockVS := &MockValidationService{}

	// Use real ConsensusState for GetProposerForHeight logic
	cs := NewConsensusState()

	// Setup dummy validators in CS
	// Ensure addresses are distinct and allow for deterministic sorting if needed for proposer selection tests
	val1AddrBytes := []byte("validator1_test_addr_alpha") // Raw address bytes
	val2AddrBytes := []byte("validator2_test_addr_beta")  // Raw address bytes

	cs.LoadInitialValidators([]*Validator{ // LoadInitialValidators handles hex encoding for map keys
		{Address: val1AddrBytes, Stake: 1000, Reputation: 1.0},
		{Address: val2AddrBytes, Stake: 1500, Reputation: 0.9},
	})

	// Default nodeValAddr to val1AddrBytes if nil
	if nodeValAddrRaw == nil {
		nodeValAddrRaw = val1AddrBytes
	}

	mockBC := &MockBlockchain{
		CurrentHeightFunc: func() int64 { return -1 }, // Default: Start with empty chain for most tests
		GetLatestBlockFunc: func() *core.Block { return nil },
		AddBlockFunc: func(b *core.Block) error { return nil }, // Default: success
	}

	// Default block reception channel for mockNet
	blockChanForTest := make(chan []byte, 10) // Buffered channel for tests to send blocks
	mockNet := &MockSimulatedNetwork{
		GetBlockReceptionChannelFunc: func() <-chan []byte {
			return blockChanForTest
		},
	}

	// Initialize ProposerService with a dummy key, as it's a dependency for ConsensusEngine
	// This ProposerService is the real one, but its dependencies (mempool, blockchain) might be mocked for its own tests.
	// For engine tests, we often mock the ProposerService itself (mockPS).
	// The engine takes the *mocked* ProposerService (mockPS).
	// However, the real ProposerService constructor needs a key.
	dummyPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// We don't need a real mempool for engine tests if PS is mocked
	realProposerServiceForEngineDeps := NewProposerService(dummyPrivKey, mempool.NewMempool(), mockBC, cs)


	engine := NewConsensusEngine(realProposerServiceForEngineDeps, mockVS, cs, mockBC, mockNet, nodeValAddrRaw)
	if engine == nil {
		t.Fatalf("NewConsensusEngine returned nil. Check dependencies: PS: %v, VS: %v, CS: %v, BC: %v, Net: %v, Addr: %v",
			realProposerServiceForEngineDeps, mockVS, cs, mockBC, mockNet, nodeValAddrRaw)
	}
    // Replace the engine's proposerService with the mock for fine-grained control in tests
    engine.proposerService = mockPS

	return engine, mockPS, mockVS, cs, mockBC, mockNet
}

// --- Test Functions ---
func TestConsensusEngine_StartStop(t *testing.T) {
	engine, _, _, _, _, _ := setupEngineWithMocks(t, []byte("validator1_test_addr_alpha"))
	go engine.Start()
	time.Sleep(50 * time.Millisecond)
	engine.Stop()
	t.Log("ConsensusEngine Start/Stop test completed.")
}

func TestConsensusEngine_AttemptBlockProposal_IsProposer(t *testing.T) {
	nodeAddr := []byte("validator_proposer")
	engine, mockPS, _, cs, mockBC, _ := setupEngineWithMocks(t, nodeAddr)

	// Ensure this node is the only validator, thus guaranteed to be proposer for height 0
	cs.Validators = make(map[string]*Validator) // Clear any defaults
	cs.LoadInitialValidators([]*Validator{{Address: nodeAddr, Stake: 100, Reputation: 1.0}})

	mockBC.CurrentHeightFunc = func() int64 { return -1 }
	mockBC.GetLatestBlockFunc = func() *core.Block { return nil }

	var createBlockFuncCalled bool
	mockPS.CreateProposalBlockFunc = func(h int64, ph []byte, pa []byte) (*core.Block, error) {
		createBlockFuncCalled = true
		if h != 0 { t.Errorf("Expected height 0, got %d", h) }
		if !bytes.Equal(pa, nodeAddr) { t.Errorf("Proposer address mismatch. Expected %x, got %x", nodeAddr, pa) }
		hash := sha256.Sum256([]byte(fmt.Sprintf("proposedBlock%d", h)))
		return &core.Block{Height: h, Hash: hash[:], ProposerAddress: pa}, nil
	}

	engine.attemptBlockProposal()

	if !createBlockFuncCalled { // Check our flag, not mockPS.CreateProposalBlockCalled as we replaced the service
		t.Errorf("ProposerService.CreateProposalBlock was not called when it was node's turn")
	}
}

func TestConsensusEngine_AttemptBlockProposal_NotProposer(t *testing.T) {
	myNodeAddr := []byte("my_node_is_validator2")
	otherNodeAddr := []byte("other_node_is_validator1") // This should be alphabetically first if hex encoded

	engine, mockPS, _, cs, mockBC, _ := setupEngineWithMocks(t, myNodeAddr)

	// Setup validators so 'otherNodeAddr' is chosen for height 0 (nextHeight = 0)
	// GetProposerForHeight sorts hex strings of addresses.
	// hex("other_node_is_validator1") should be < hex("my_node_is_validator2")
	cs.Validators = make(map[string]*Validator) // Clear
	cs.LoadInitialValidators([]*Validator{
		{Address: otherNodeAddr, Stake: 100, Reputation: 1.0}, // Expected proposer
		{Address: myNodeAddr, Stake: 100, Reputation: 1.0},
	})

	mockBC.CurrentHeightFunc = func() int64 { return -1 } // Proposing for height 0
	mockBC.GetLatestBlockFunc = func() *core.Block { return nil }

	engine.attemptBlockProposal()

	if mockPS.CreateProposalBlockCalled {
		t.Errorf("ProposerService.CreateProposalBlock was called when it was NOT node's turn")
	}
}

func TestConsensusEngine_ProcessIncomingBlock_Valid(t *testing.T) {
	engine, _, mockVS, cs, mockBC, mockNet := setupEngineWithMocks(t, nil) // Node address not relevant for receiving

	testBlock := &core.Block{Height: 0, Hash: []byte("testIncomingBlockValid"), ProposerAddress: []byte("proposer_of_valid_block")}
	serializedBlock, err := testBlock.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize test block: %v", err)
	}

	var updateHeightCalledWith int64 = -1
	// Mock ConsensusState's UpdateHeight or check its effect if possible
	// For now, we'll assume it gets called by checking a side effect or using a more detailed mock if needed.
	// As UpdateHeight currently just logs, we can't easily check it without log capture or callback.
	// We will check that AddBlock was called, which implies UpdateHeight was called after.

	mockVS.ValidateBlockFunc = func(b *core.Block) error { return nil }
	var addBlockCalled bool
	mockBC.AddBlockFunc = func(b *core.Block) error {
		addBlockCalled = true
		mockBC.CurrentHeightFunc = func() int64 { return b.Height } // Simulate height update
		cs.UpdateHeight(b.Height) // Simulate the call that happens in real flow
		updateHeightCalledWith = b.Height
		return nil
	}

	// To test the select case in Start(), we need to send on the channel
	// Get the actual channel from the mock
	blockChan := mockNet.GetBlockReceptionChannelFunc() (chan []byte) // Cast to writeable for test

	go engine.Start() // Start the engine to listen on channels
	defer engine.Stop()

	blockChan <- serializedBlock
	time.Sleep(50 * time.Millisecond) // Give time for processing

	mockVS.mtx.Lock()
	vsCalled := mockVS.ValidateBlockCalled
	mockVS.mtx.Unlock()

	mockBC.mtx.Lock()
	abCalledWith := mockBC.AddBlockCalledWith
	mockBC.mtx.Unlock()


	if !vsCalled { t.Errorf("ValidationService.ValidateBlock not called") }
	if !addBlockCalled { t.Errorf("Blockchain.AddBlock was not called") }
	if abCalledWith == nil || !bytes.Equal(abCalledWith.Hash, testBlock.Hash) {
		t.Errorf("Blockchain.AddBlock not called with correct block")
	}
	if updateHeightCalledWith != 0 {
		t.Errorf("ConsensusState.UpdateHeight not called with correct height. Expected 0, got %d", updateHeightCalledWith)
	}
}

func TestConsensusEngine_ProcessIncomingBlock_InvalidValidation(t *testing.T) {
	engine, _, mockVS, _, mockBC, _ := setupEngineWithMocks(t, nil)
	testBlock := &core.Block{Height: 0, Hash: []byte("testIncomingBlockInvalid")}

	mockVS.ValidateBlockFunc = func(b *core.Block) error { return errors.New("validation failed") }
	var addBlockCalled bool
	mockBC.AddBlockFunc = func(b *core.Block) error { addBlockCalled = true; return nil }

	engine.processIncomingBlock(testBlock) // Direct call for this test case

	mockVS.mtx.Lock()
	vsCalled := mockVS.ValidateBlockCalled
	mockVS.mtx.Unlock()

	if !vsCalled { t.Errorf("ValidationService.ValidateBlock not called") }
	if addBlockCalled { t.Errorf("AddBlock called for invalid block") }
}

func TestConsensusEngine_ProcessIncomingBlock_AddBlockFails(t *testing.T) {
	engine, _, mockVS, _, mockBC, _ := setupEngineWithMocks(t, nil)
	testBlock := &core.Block{Height: 0, Hash: []byte("testIncomingBlockAddFail")}

	mockVS.ValidateBlockFunc = func(b *core.Block) error { return nil }
	var addBlockCalled bool
	mockBC.AddBlockFunc = func(b *core.Block) error { addBlockCalled = true; return errors.New("add block failed") }

	// We need to track if ConsensusState.UpdateHeight was called.
	// For now, we'll infer based on AddBlock not being fully successful.
	// A better mock for ConsensusState would allow checking this.

	engine.processIncomingBlock(testBlock) // Direct call

	mockVS.mtx.Lock()
	vsCalled := mockVS.ValidateBlockCalled
	mockVS.mtx.Unlock()

	if !vsCalled { t.Errorf("ValidationService.ValidateBlock not called") }
	if !addBlockCalled {t.Errorf("AddBlock was not called, but it should have been.")}
	// TODO: Check that consensusState.UpdateHeight was NOT called if AddBlock fails.
	// This requires ConsensusState mock or callback.
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
