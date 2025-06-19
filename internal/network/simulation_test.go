package network

import (
	"bytes" // For comparing message data
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"
	"time"

	"empower1.com/empower1blockchain/internal/core"
)

// Helper for creating a simple block for testing
func newTestSimBlock(id string) *core.Block {
	// Ensure all fields that are part of GOB encoding are initialized to avoid panics/errors.
	// Slices and maps should be initialized (e.g., make([]core.Transaction, 0)).
	// Pointer fields should be nil or initialized.
	// Byte slices should be initialized (e.g., make([]byte, X) or actual data).
	hashBytes := sha256.Sum256([]byte(id))
	prevHashBytes := sha256.Sum256([]byte("prev_" + id))
	proposerBytes := sha256.Sum256([]byte("proposer_" + id))
	sigBytes := sha256.Sum256([]byte("sig_" + id))
	aiLogBytes := sha256.Sum256([]byte("ailog_" + id))
	stateRootBytes := sha256.Sum256([]byte("stateroot_" + id))

	return &core.Block{
		Height:          1,
		Timestamp:       time.Now().UnixNano(),
		PrevBlockHash:   prevHashBytes[:],
		Transactions:    make([]core.Transaction, 0), // Initialize empty slice
		ProposerAddress: proposerBytes[:],
		Signature:       sigBytes[:],
		Hash:            hashBytes[:],
		AIAuditLog:      aiLogBytes[:],
		StateRoot:       stateRootBytes[:],
	}
}

// Helper for creating a simple tx for testing
func newTestSimTx(idStr string) *core.Transaction {
	txIDBytes := sha256.Sum256([]byte(idStr))
	return &core.Transaction{
		ID:        txIDBytes[:],
		Timestamp: time.Now().UnixNano(),
		TxType:    core.TxStandard,
		Inputs:    make([]core.TxInput, 0),  // Initialize empty slice
		Outputs:   make([]core.TxOutput, 0), // Initialize empty slice
		Fee:       0,
		Signers:   make([]core.SignerInfo,0), // Initialize for multi-sig if used by gob
		// Initialize other slice/map/pointer fields if they were to be added
	}
}

func TestNewSimulatedNetwork(t *testing.T) {
	nodeID := "testNode1"
	sn := NewSimulatedNetwork(nodeID)
	if sn == nil {
		t.Fatal("NewSimulatedNetwork returned nil")
	}
	if sn.NodeID != nodeID {
		t.Errorf("SimulatedNetwork NodeID = %s, want %s", sn.NodeID, nodeID)
	}
	if sn.BlockBroadcastChannel == nil {
		t.Error("BlockBroadcastChannel is nil")
	}
	if sn.TransactionBroadcastChannel == nil {
		t.Error("TransactionBroadcastChannel is nil")
	}
	if sn.peers == nil { // Check for peers map
		t.Error("Peers map is nil")
	}
	if cap(sn.BlockBroadcastChannel) != 100 {
		t.Errorf("BlockBroadcastChannel capacity = %d, want 100", cap(sn.BlockBroadcastChannel))
	}
	if cap(sn.TransactionBroadcastChannel) != 100 {
		t.Errorf("TransactionBroadcastChannel capacity = %d, want 100", cap(sn.TransactionBroadcastChannel))
	}
}

func TestSimulatedNetwork_PeerLifecycle(t *testing.T) {
	sn := NewSimulatedNetwork("nodeA")
	peerID := "nodeB_id"

	// Connect Peer
	peerB, err := sn.ConnectPeer(peerID)
	if err != nil {
		t.Fatalf("ConnectPeer failed: %v", err)
	}
	if peerB == nil {
		t.Fatal("ConnectPeer returned nil peer without error")
	}
	sn.mu.RLock()
	_, exists := sn.peers[peerID]
	sn.mu.RUnlock()
	if !exists {
		t.Errorf("Peer %s not found in sn.peers after ConnectPeer", peerID)
	}
	if len(sn.peers) != 1 {
		t.Errorf("sn.peers length = %d, want 1", len(sn.peers))
	}
	// Allow some time for peer processor to start, though it's very quick
	time.Sleep(10 * time.Millisecond)

	// Attempt to connect same peer again (should return existing peer, no error or specific 'already connected' error)
	existingPeerB, err := sn.ConnectPeer(peerID)
	if err != nil { // Current ConnectPeer returns nil error if peer exists, just returns existing
		t.Errorf("Connecting to already connected peer %s returned an error: %v", peerID, err)
	}
	if existingPeerB != peerB {
		t.Errorf("Connecting to already connected peer %s returned a different peer instance", peerID)
	}
	if len(sn.peers) != 1 { // Length should remain 1
		t.Errorf("sn.peers length = %d after re-connecting, want 1", len(sn.peers))
	}


	// Disconnect Peer
	sn.DisconnectPeer(peerID)
	sn.mu.RLock()
	_, exists = sn.peers[peerID]
	sn.mu.RUnlock()
	if exists {
		t.Errorf("Peer %s still found in sn.peers after DisconnectPeer", peerID)
	}
	if len(sn.peers) != 0 {
		t.Errorf("sn.peers length = %d after disconnect, want 0", len(sn.peers))
	}
	// StopProcessor is called by DisconnectPeer, which waits on peer.wg.
	// If it doesn't hang, the goroutine is assumed to have exited.
}


func TestSimulatedNetwork_Broadcast_And_PeerProcessorRouting(t *testing.T) {
	broadcastingNode := NewSimulatedNetwork("broadcastingNode")
	receivingNodeID := "receivingNodeID" // This is just an ID for the Peer object

	// Connect a "peer" to the broadcastingNode.
	// The Peer object within broadcastingNode will process messages sent to receivingNodeID.
	// Its processor will route them to broadcastingNode's *own* public channels.
	_, err := broadcastingNode.ConnectPeer(receivingNodeID)
	if err != nil {
		t.Fatalf("Failed to connect peer: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Allow peer processor to start

	// Test BroadcastBlock
	testBlock := newTestSimBlock("block123")
	serializedBlock, _ := testBlock.Serialize()
	broadcastingNode.BroadcastBlock(testBlock)

	select {
	case receivedData := <-broadcastingNode.GetBlockReceptionChannel():
		if !bytes.Equal(receivedData, serializedBlock) {
			t.Errorf("Received block data mismatch on broadcastingNode's BlockBroadcastChannel. Got %x, want %x", receivedData, serializedBlock)
		} else {
			t.Logf("Successfully received broadcasted block on broadcastingNode's BlockBroadcastChannel.")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Did not receive broadcasted block data on broadcastingNode's BlockBroadcastChannel")
	}

	// Test BroadcastTransaction
	testTx := newTestSimTx("tx456")
	serializedTx, _ := testTx.Serialize()
	broadcastingNode.BroadcastTransaction(testTx)

	select {
	case receivedData := <-broadcastingNode.GetTransactionReceptionChannel():
		if !bytes.Equal(receivedData, serializedTx) {
			t.Errorf("Received tx data mismatch on broadcastingNode's TransactionBroadcastChannel. Got %x, want %x", receivedData, serializedTx)
		} else {
			t.Logf("Successfully received broadcasted transaction on broadcastingNode's TransactionBroadcastChannel.")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Did not receive broadcasted transaction data on broadcastingNode's TransactionBroadcastChannel")
	}

    lonelySn := NewSimulatedNetwork("lonelyNode")
    lonelySn.BroadcastBlock(newTestSimBlock("lonelyBlock"))
    lonelySn.BroadcastTransaction(newTestSimTx("lonelyTx"))
    t.Log("Tested broadcast with no peers (expected internal log messages, no errors here).")

    // Test broadcast to full peer channel
    snPeerTest := NewSimulatedNetwork("snPeerTest")
    peerToFill, _ := snPeerTest.ConnectPeer("peerToFillID")
    // Fill up peerToFill.IncomingMessages by sending NetworkMessage structs
    for i := 0; i < 105; i++ { // Channel capacity is 100
        msg := NetworkMessage{Type: "FILLER", Data: []byte(fmt.Sprintf("filler%d",i))}
        select {
        case peerToFill.IncomingMessages <- msg:
        default:
            break
        }
    }
    snPeerTest.BroadcastBlock(newTestSimBlock("droppedBlock"))
    t.Log("Tested broadcast to full peer channel (expected internal log message about drop).")

    broadcastingNode.DisconnectPeer(receivingNodeID)
    snPeerTest.DisconnectPeer("peerToFillID")
}


func TestSimulatedNetwork_SimulateReceive(t *testing.T) {
	// Keep existing TestSimulatedNetwork_SimulateReceive, ensure it's still valid.
    // It tests injecting messages directly into THIS node's public channels.
	sn := NewSimulatedNetwork("testNode")
	blockData := []byte("sim_block_data_direct") // Unique data
	txData := []byte("sim_tx_data_direct")       // Unique data
    otherData := []byte("other_sim_data_direct")  // Unique data
    genericMsgType := "GENERIC_MESSAGE_DIRECT"

    var handlerCalled bool
    var receivedPeerID, receivedMsgType string
    var receivedHandlerData []byte
    sn.RegisterMessageHandler(func(pID string, mType string, d []byte) {
        handlerCalled = true
        receivedPeerID = pID
        receivedMsgType = mType
        receivedHandlerData = data
    })

	sn.SimulateReceive("peerX", "NEW_BLOCK", blockData)
	select {
	case data := <-sn.GetBlockReceptionChannel():
		if !bytes.Equal(data, blockData) {
			t.Errorf("Simulated block data mismatch. Got %s, want %s", string(data), string(blockData))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive simulated block data on BlockBroadcastChannel via SimulateReceive")
	}

    sn.SimulateReceive("peerY", "NEW_TRANSACTION", txData)
    select {
	case data := <-sn.GetTransactionReceptionChannel():
		if !bytes.Equal(data, txData) {
			t.Errorf("Simulated tx data mismatch. Got %s, want %s", string(data), string(txData))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive simulated tx data on TransactionBroadcastChannel via SimulateReceive")
	}

    handlerCalled = false
    sn.SimulateReceive("peerZ", genericMsgType, otherData)
    time.Sleep(10 * time.Millisecond)
    if !handlerCalled {
        t.Errorf("Generic message handler not called for message type %s via SimulateReceive", genericMsgType)
    } else {
        if receivedPeerID != "peerZ" { t.Errorf("Generic handler peerID got %s, want peerZ", receivedPeerID)}
        if receivedMsgType != genericMsgType {t.Errorf("Generic handler msgType got %s, want %s", receivedMsgType, genericMsgType)}
        if !bytes.Equal(receivedHandlerData, otherData) {t.Errorf("Generic handler data got %s, want %s", string(receivedHandlerData), string(otherData))}
    }

    sn.messageHandler = nil
    handlerCalled = false
    sn.SimulateReceive("peerW", "UNKNOWN_MSG_TYPE_NO_HANDLER", otherData)
    select {
    case <-sn.GetBlockReceptionChannel():
        t.Error("Received unexpected data on BlockBroadcastChannel for UNKNOWN_MSG_TYPE_NO_HANDLER via SimulateReceive")
    case <-sn.GetTransactionReceptionChannel():
        t.Error("Received unexpected data on TransactionBroadcastChannel for UNKNOWN_MSG_TYPE_NO_HANDLER via SimulateReceive")
    case <-time.After(50 * time.Millisecond):
        // Expected
    }
    if handlerCalled {
        t.Error("Generic message handler was called for UNKNOWN_MSG_TYPE_NO_HANDLER after unregistering")
    }
}

// TODO: Test for channel full scenarios in Broadcast (non-blocking send should drop for that peer).
// - Already partially covered in TestSimulatedNetwork_Broadcast_And_PeerProcessorRouting
// TODO: Test RegisterMessageHandler with nil handler.
// TODO: Test DisconnectPeer ensuring peer's conceptualPeerMessageProcessor actually stops.
// - Current test relies on StopProcessor not hanging. More robust check is complex.
// TODO: Test multiple peers broadcasting and receiving concurrently.
