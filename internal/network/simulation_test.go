package network

import (
	"bytes" // For comparing message data
	"sync"
	"testing"
	"time"
	// "log" // Not typically used directly in tests for pass/fail
)

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
		t.Error("Peers map is nil")
	}
    if cap(sn.BlockBroadcastChannel) != 100 {
        t.Errorf("BlockBroadcastChannel capacity = %d, want 100", cap(sn.BlockBroadcastChannel))
    }
    if cap(sn.TransactionBroadcastChannel) != 100 {
        t.Errorf("TransactionBroadcastChannel capacity = %d, want 100", cap(sn.TransactionBroadcastChannel))
    }
}

func TestSimulatedNetwork_ConnectDisconnectPeer(t *testing.T) {
	sn1 := NewSimulatedNetwork("node1")
	sn2 := NewSimulatedNetwork("node2")
	sn3 := NewSimulatedNetwork("node3")

	// Connect sn2 to sn1
	sn1.ConnectPeer(sn2)
	if _, exists := sn1.peers[sn2.NodeID]; !exists {
		t.Errorf("Peer %s not found in sn1.peers after ConnectPeer", sn2.NodeID)
	}
    if len(sn1.peers) != 1 {
        t.Errorf("sn1.peers length = %d, want 1", len(sn1.peers))
    }

	// Connect sn3 to sn1
	sn1.ConnectPeer(sn3)
	if _, exists := sn1.peers[sn3.NodeID]; !exists {
		t.Errorf("Peer %s not found in sn1.peers after ConnectPeer", sn3.NodeID)
	}
    if len(sn1.peers) != 2 {
        t.Errorf("sn1.peers length = %d, want 2", len(sn1.peers))
    }

	// Attempt to connect self (should not add)
	sn1.ConnectPeer(sn1)
	if len(sn1.peers) != 2 {
        t.Errorf("sn1.peers length = %d after self-connect attempt, want 2", len(sn1.peers))
    }

    // Attempt to connect nil peer (should not add)
    sn1.ConnectPeer(nil)
	if len(sn1.peers) != 2 {
        t.Errorf("sn1.peers length = %d after nil-connect attempt, want 2", len(sn1.peers))
    }

	// Attempt to connect existing peer (should not change length)
	sn1.ConnectPeer(sn2)
	if len(sn1.peers) != 2 {
		t.Errorf("sn1.peers length = %d after re-connecting existing peer, want 2", len(sn1.peers))
	}


	// Disconnect sn2
	sn1.DisconnectPeer(sn2.NodeID)
	if _, exists := sn1.peers[sn2.NodeID]; exists {
		t.Errorf("Peer %s still found in sn1.peers after DisconnectPeer", sn2.NodeID)
	}
    if len(sn1.peers) != 1 {
        t.Errorf("sn1.peers length = %d after disconnect, want 1", len(sn1.peers))
    }

    // Disconnect non-existent peer
    sn1.DisconnectPeer("nonExistentNode")
    if len(sn1.peers) != 1 {
        t.Errorf("sn1.peers length = %d after disconnecting non-existent, want 1", len(sn1.peers))
    }
}

func TestSimulatedNetwork_Broadcast(t *testing.T) {
	broadcaster := NewSimulatedNetwork("broadcasterNode")
	peer1 := NewSimulatedNetwork("peer1")
	peer2 := NewSimulatedNetwork("peer2")

	broadcaster.ConnectPeer(peer1)
	broadcaster.ConnectPeer(peer2)

	blockData := []byte("this_is_a_block")
	txData := []byte("this_is_a_transaction")

	// Test broadcasting a block
	broadcaster.Broadcast("NEW_BLOCK", blockData)

	// Test broadcasting a transaction
	broadcaster.Broadcast("NEW_TRANSACTION", txData)

    // Test broadcasting unknown type
    broadcaster.Broadcast("UNKNOWN_TYPE", []byte("unknown_data"))


	var wg sync.WaitGroup
	// Expecting one message on each relevant channel for each peer
	wg.Add(4) // peer1-block, peer1-tx, peer2-block, peer2-tx

	receivedBlockOnPeer1 := false
	receivedTxOnPeer1 := false
	receivedBlockOnPeer2 := false
	receivedTxOnPeer2 := false

	// Peer 1 listeners
	go func() {
		defer wg.Done() // For block channel
		select {
		case data := <-peer1.GetBlockReceptionChannel():
			if bytes.Equal(data, blockData) {
				receivedBlockOnPeer1 = true
			} else {
                t.Errorf("Peer1 received wrong block data: %s", string(data))
            }
		case <-time.After(100 * time.Millisecond):
			// This t.Error will be from a goroutine, use t.Log or ensure test fails if flags not set
		}
	}()
    go func() {
        defer wg.Done() // For transaction channel
        select {
        case data := <-peer1.GetTransactionReceptionChannel():
			if bytes.Equal(data, txData) {
				receivedTxOnPeer1 = true
			} else {
                 t.Errorf("Peer1 received wrong tx data: %s", string(data))
            }
		case <-time.After(100 * time.Millisecond):
        }
	}()

	// Peer 2 listeners
	go func() {
		defer wg.Done() // For block channel
		select {
		case data := <-peer2.GetBlockReceptionChannel():
			if bytes.Equal(data, blockData) {
				receivedBlockOnPeer2 = true
			} else {
                t.Errorf("Peer2 received wrong block data: %s", string(data))
            }
		case <-time.After(100 * time.Millisecond):
		}
	}()
    go func() {
        defer wg.Done() // For transaction channel
        select {
        case dataTx := <-peer2.GetTransactionReceptionChannel():
			if bytes.Equal(dataTx, txData) {
				receivedTxOnPeer2 = true
			} else {
                t.Errorf("Peer2 received wrong tx data: %s", string(dataTx))
            }
		case <-time.After(100 * time.Millisecond):
        }
	}()

	wg.Wait()

	if !receivedBlockOnPeer1 {
		t.Error("Peer1 did not confirm block reception flag")
	}
	if !receivedTxOnPeer1 {
		t.Error("Peer1 did not confirm transaction reception flag")
	}
    if !receivedBlockOnPeer2 {
		t.Error("Peer2 did not confirm block reception flag")
	}
	if !receivedTxOnPeer2 {
		t.Error("Peer2 did not confirm transaction reception flag")
	}

    // Test broadcast with no peers
    lonelyBroadcaster := NewSimulatedNetwork("lonelyNode")
    // This should just log that there are no peers and not panic.
    lonelyBroadcaster.Broadcast("NEW_BLOCK", []byte("lonely_block"))
}


func TestSimulatedNetwork_SimulateReceive(t *testing.T) {
	sn := NewSimulatedNetwork("testNode")
	blockData := []byte("sim_block_data")
	txData := []byte("sim_tx_data")
    otherData := []byte("other_sim_data")
    genericMsgType := "GENERIC_MESSAGE"
    genericMsgReceived := false
    var receivedPeerID, receivedMsgType string
    var receivedData []byte

    sn.RegisterMessageHandler(func(pID string, mType string, data []byte){
        genericMsgReceived = true
        receivedPeerID = pID
        receivedMsgType = mType
        receivedData = data
    })


    // Test SimulateReceive for NEW_BLOCK
	sn.SimulateReceive("peerX", "NEW_BLOCK", blockData)
	select {
	case data := <-sn.GetBlockReceptionChannel():
		if !bytes.Equal(data, blockData) {
			t.Errorf("Simulated block data mismatch. Got %s, want %s", string(data), string(blockData))
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Did not receive simulated block data on BlockBroadcastChannel")
	}

    // Test SimulateReceive for NEW_TRANSACTION
    sn.SimulateReceive("peerY", "NEW_TRANSACTION", txData)
    select {
	case data := <-sn.GetTransactionReceptionChannel():
		if !bytes.Equal(data, txData) {
			t.Errorf("Simulated tx data mismatch. Got %s, want %s", string(data), string(txData))
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Did not receive simulated tx data on TransactionBroadcastChannel")
	}

    // Test SimulateReceive for a generic message type (should use the handler)
    sn.SimulateReceive("peerZ", genericMsgType, otherData)
    time.Sleep(50 * time.Millisecond) // Allow time for handler to be called if it was async (it's sync)
    if !genericMsgReceived {
        t.Errorf("Generic message handler not called for message type %s", genericMsgType)
    } else {
        if receivedPeerID != "peerZ" { t.Errorf("Generic handler peerID got %s, want peerZ", receivedPeerID)}
        if receivedMsgType != genericMsgType {t.Errorf("Generic handler msgType got %s, want %s", receivedMsgType, genericMsgType)}
        if !bytes.Equal(receivedData, otherData) {t.Errorf("Generic handler data got %s, want %s", string(receivedData), string(otherData))}
    }

    // Test SimulateReceive for unknown type with no generic handler (after unregistering)
    sn.messageHandler = nil // Unregister handler
    genericMsgReceived = false // Reset flag
    sn.SimulateReceive("peerW", "UNKNOWN_MSG_TYPE_NO_HANDLER", otherData)
    select {
    case <-sn.GetBlockReceptionChannel():
        t.Error("Received unexpected data on BlockBroadcastChannel for UNKNOWN_MSG_TYPE_NO_HANDLER")
    case <-sn.GetTransactionReceptionChannel():
        t.Error("Received unexpected data on TransactionBroadcastChannel for UNKNOWN_MSG_TYPE_NO_HANDLER")
    case <-time.After(50 * time.Millisecond):
        // Expected: no message on typed channels and no panic from handler
    }
    if genericMsgReceived {
        t.Error("Generic message handler was called for UNKNOWN_MSG_TYPE_NO_HANDLER after unregistering")
    }
}

// TODO: Test for channel full scenarios in Broadcast (non-blocking send should drop for that peer).
// TODO: Test RegisterMessageHandler with nil handler.
```
