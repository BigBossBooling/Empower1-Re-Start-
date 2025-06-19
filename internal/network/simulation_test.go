package network

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"
	"time"
	// "reflect" // Not strictly needed for these tests after refinement

	"empower1.com/empower1blockchain/internal/core"
)

// Helper for creating a simple block for testing serialization
func newTestSimBlock(id string) *core.Block {
	hashBytes := sha256.Sum256([]byte("hash_" + id))
	prevHashBytes := sha256.Sum256([]byte("prev_" + id))
	proposerBytes := sha256.Sum256([]byte("proposer_" + id))
	sigBytes := sha256.Sum256([]byte("sig_" + id))
	aiLogBytes := sha256.Sum256([]byte("ailog_" + id))
	stateRootBytes := sha256.Sum256([]byte("stateroot_" + id))

	return &core.Block{
		Height:          1,
		Timestamp:       time.Now().UnixNano(), // Timestamps will differ, not an issue for these tests
		PrevBlockHash:   prevHashBytes[:],
		Transactions:    make([]core.Transaction, 0),
		ProposerAddress: proposerBytes[:],
		Signature:       sigBytes[:],
		Hash:            hashBytes[:],
		AIAuditLog:      aiLogBytes[:],
		StateRoot:       stateRootBytes[:],
	}
}

// Helper for creating a simple tx for testing serialization
func newTestSimTx(idStr string) *core.Transaction {
	txIDBytes := sha256.Sum256([]byte("txid_" + idStr))
	return &core.Transaction{
		ID:        txIDBytes[:],
		Timestamp: time.Now().UnixNano(),
		TxType:    core.TxStandard,
		Inputs:    make([]core.TxInput, 0),
		Outputs:   make([]core.TxOutput, 0),
		Fee:       0,
		Signers:   make([]core.SignerInfo, 0),
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
	if sn.peers == nil {
		t.Error("Peers map is nil (expected initialized map)")
	}
	if _, ok := sn.peers.(map[string]*Peer); !ok && sn.peers != nil {
		t.Errorf("Peers map is not of type map[string]*Peer")
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
	peerNodeID1 := "nodeB"
	peerNodeID2 := "nodeC"

	// Connect first peer
	peerB, err := sn.ConnectPeer(peerNodeID1)
	if err != nil {
		t.Fatalf("ConnectPeer(%s) failed: %v", peerNodeID1, err)
	}
	if peerB == nil {
		t.Fatalf("ConnectPeer(%s) returned nil peer without error", peerNodeID1)
	}
	if peerB.ID != peerNodeID1 {
		t.Errorf("Connected peer ID = %s, want %s", peerB.ID, peerNodeID1)
	}
	if len(sn.peers) != 1 {
		t.Errorf("Peer map length after 1st connect = %d, want 1", len(sn.peers))
	}
	// Allow processor to start
	time.Sleep(10 * time.Millisecond)

	// Attempt to connect same peer again
	_, err = sn.ConnectPeer(peerNodeID1)
	if err != nil { // Expecting nil error as it should return existing peer
		t.Errorf("Re-connecting to peer %s returned error: %v", peerNodeID1, err)
	}
	if len(sn.peers) != 1 {
		t.Errorf("Peer map length after re-connecting existing peer = %d, want 1", len(sn.peers))
	}

	// Connect second peer
	peerC, err := sn.ConnectPeer(peerNodeID2)
	if err != nil {
		t.Fatalf("ConnectPeer(%s) failed: %v", peerNodeID2, err)
	}
	if len(sn.peers) != 2 {
		t.Errorf("Peer map length after 2nd connect = %d, want 2", len(sn.peers))
	}
	time.Sleep(10 * time.Millisecond)


	// Disconnect first peer
	sn.DisconnectPeer(peerNodeID1)
	sn.mu.RLock()
	_, exists := sn.peers[peerNodeID1]
	sn.mu.RUnlock()
	if exists {
		t.Errorf("Peer %s still in map after disconnect", peerNodeID1)
	}
	if len(sn.peers) != 1 {
		t.Errorf("Peer map length after 1st disconnect = %d, want 1", len(sn.peers))
	}

	// Disconnect second peer
	sn.DisconnectPeer(peerNodeID2)
	sn.mu.RLock()
	_, exists = sn.peers[peerNodeID2]
	sn.mu.RUnlock()
	if exists {
		t.Errorf("Peer %s still in map after disconnect", peerNodeID2)
	}
	if len(sn.peers) != 0 {
		t.Errorf("Peer map length after 2nd disconnect = %d, want 0", len(sn.peers))
	}

	// Ensure StopProcessor doesn't hang (implicitly tested by DisconnectPeer completing)
	// To be more explicit, one could try to send to peerB.IncomingMessages and peerC.IncomingMessages
	// after disconnect and expect them to be closed or non-responsive, but that's complex.
	// For now, successful completion of DisconnectPeer is the main check.
}


func TestSimulatedNetwork_Broadcast_And_PeerProcessing(t *testing.T) {
	broadcaster := NewSimulatedNetwork("broadcasterNode")
	peerID := "internalPeerRepresentation" // The ID used by broadcaster to manage the connection

	// Connect a peer. This peer object is internal to `broadcaster`.
	// Its processor will route messages to `broadcaster.BlockBroadcastChannel` etc.
	internalPeer, err := broadcaster.ConnectPeer(peerID)
	if err != nil {
		t.Fatalf("Failed to connect internal peer: %v", err)
	}
	if internalPeer.network != broadcaster {
		t.Fatal("Internal peer's network reference is not the broadcaster")
	}
	time.Sleep(20 * time.Millisecond) // Give peer processor time to start

	t.Run("BroadcastBlock", func(t *testing.T) {
		testBlock := newTestSimBlock("blockToBroadcast")
		serializedBlock, _ := testBlock.Serialize()

		broadcaster.BroadcastBlock(testBlock)

		select {
		case receivedData := <-broadcaster.GetBlockReceptionChannel():
			if !bytes.Equal(receivedData, serializedBlock) {
				t.Errorf("Received block data mismatch. Got %x, want %x", receivedData, serializedBlock)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive broadcasted block on broadcaster's BlockBroadcastChannel")
		}
	})

	t.Run("BroadcastTransaction", func(t *testing.T) {
		testTx := newTestSimTx("txToBroadcast")
		serializedTx, _ := testTx.Serialize()

		broadcaster.BroadcastTransaction(testTx)

		select {
		case receivedData := <-broadcaster.GetTransactionReceptionChannel():
			if !bytes.Equal(receivedData, serializedTx) {
				t.Errorf("Received tx data mismatch on broadcaster's TransactionBroadcastChannel. Got %x, want %x", receivedData, serializedTx)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive broadcasted tx on broadcaster's TransactionBroadcastChannel")
		}
	})

	t.Run("BroadcastToNoPeers", func(t *testing.T) {
		lonelyNode := NewSimulatedNetwork("lonelyNode")
		testBlock := newTestSimBlock("lonelyBlock")
		// This should execute without error and log internally
		lonelyNode.BroadcastBlock(testBlock)
		// Check that no messages arrived on its own channels (as it has no peers to loop back through)
		select {
		case <-lonelyNode.GetBlockReceptionChannel():
			t.Error("Received unexpected block on lonelyNode's BlockBroadcastChannel")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}
	})

	t.Run("BroadcastToFullPeerChannel", func(t *testing.T) {
		nodeWithBusyPeer := NewSimulatedNetwork("nodeWithBusyPeer")
		busyPeerRepresentation, _ := nodeWithBusyPeer.ConnectPeer("busyPeerID")
		time.Sleep(10 * time.Millisecond)


		// Fill up the Peer's IncomingMessages channel
		for i := 0; i < cap(busyPeerRepresentation.IncomingMessages)+5; i++ {
			msg := NetworkMessage{Type: "FILLER", Data: []byte(fmt.Sprintf("filler%d", i))}
			select {
			case busyPeerRepresentation.IncomingMessages <- msg:
			default: // Channel is full
				t.Logf("Filled busyPeer's IncomingMessages channel at iteration %d", i)
				break
			}
		}
		// Now broadcast. The message to this peer should be dropped (and logged internally by SimulatedNetwork).
		nodeWithBusyPeer.BroadcastBlock(newTestSimBlock("blockForBusyPeer"))
		// Check that no new message (or the dropped one) appears on the node's public channel from this busy peer quickly.
		// This test mainly ensures no deadlock and respects non-blocking send.
		select {
		case <- nodeWithBusyPeer.GetBlockReceptionChannel():
			t.Error("Received block on main channel from a peer whose private channel was full; expected drop.")
		case <- time.After(50 * time.Millisecond):
			t.Log("No block received from busy peer as expected (message likely dropped).")
		}
		nodeWithBusyPeer.DisconnectPeer("busyPeerID")
	})


	broadcaster.DisconnectPeer(peerID) // Cleanup
}

func TestSimulatedNetwork_SimulateReceive(t *testing.T) {
	sn := NewSimulatedNetwork("testNode")
	blockData := []byte("sim_block_data_direct_samer")
	txData := []byte("sim_tx_data_direct_samer")
    otherData := []byte("other_sim_data_direct_samer")
    genericMsgType := "GENERIC_MESSAGE_DIRECT_SAMER"

    var handlerCalled bool
    var receivedPeerID, receivedMsgType string
    var receivedHandlerData []byte
    sn.RegisterMessageHandler(func(pID string, mType string, data []byte) {
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

// TODO: Test RegisterMessageHandler with nil handler.
// TODO: Test DisconnectPeer ensuring peer's conceptualPeerMessageProcessor actually stops more explicitly.
// TODO: Test multiple peers broadcasting and receiving concurrently if current tests don't cover race conditions.
```
