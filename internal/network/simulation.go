package network

import (
	"log" // For logging broadcast/receive actions
	"sync"
)

// MessageHandler defines the function signature for handling messages received from the network.
// peerID might be the sender's NodeID in a real network.
// This handler is now for generic messages, not blocks or transactions if they use specific channels.
type MessageHandler func(peerID string, messageType string, data []byte)

// SimulatedNetwork provides a basic, in-memory simulation of network interactions
// for testing consensus and other components without a full P2P stack.
type SimulatedNetwork struct {
	NodeID                  string
	mu                      sync.RWMutex
	messageHandler          MessageHandler // For general messages if used by consensus engine directly
	BlockBroadcastChannel   chan []byte      // For blocks this node "receives"
	TransactionBroadcastChannel chan []byte // For transactions this node "receives"

	peers                   map[string]*SimulatedNetwork // Key: Peer NodeID
    // TODO: Add a logger instance for more structured logging.
}

// NewSimulatedNetwork creates a new SimulatedNetwork instance.
func NewSimulatedNetwork(nodeID string) *SimulatedNetwork {
	if nodeID == "" {
		nodeID = "default_sim_node" // Default if not provided
	}
	return &SimulatedNetwork{
		NodeID:                  nodeID,
		BlockBroadcastChannel:   make(chan []byte, 100), // Buffer of 100
		TransactionBroadcastChannel: make(chan []byte, 100), // Buffer of 100
		peers:                   make(map[string]*SimulatedNetwork),
	}
}

// ConnectPeer adds another SimulatedNetwork instance to this node's peer list.
func (sn *SimulatedNetwork) ConnectPeer(peer *SimulatedNetwork) {
	if peer == nil || sn.NodeID == peer.NodeID {
		log.Printf("SIMNET [%s]: Attempted to connect to nil or self peer [%s]. Ignoring.", sn.NodeID, peer.NodeID)
		return
	}
	sn.mu.Lock()
	defer sn.mu.Unlock()
	if _, exists := sn.peers[peer.NodeID]; !exists {
		sn.peers[peer.NodeID] = peer
		log.Printf("SIMNET [%s]: Connected to peer [%s]", sn.NodeID, peer.NodeID)
	} else {
		log.Printf("SIMNET [%s]: Already connected to peer [%s]", sn.NodeID, peer.NodeID)
	}
}

// DisconnectPeer removes a peer from the list.
func (sn *SimulatedNetwork) DisconnectPeer(peerID string) {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	if _, exists := sn.peers[peerID]; exists {
		delete(sn.peers, peerID)
		log.Printf("SIMNET [%s]: Disconnected from peer [%s]", sn.NodeID, peerID)
	} else {
		log.Printf("SIMNET [%s]: Peer [%s] not found for disconnection.", sn.NodeID, peerID)
	}
}

// Broadcast sends a message to all connected peers.
// It routes messages to the appropriate channel on the peer based on messageType.
func (sn *SimulatedNetwork) Broadcast(messageType string, data []byte) {
	sn.mu.RLock() // Lock for reading peers map
	// Copy peers to a slice to avoid holding lock during channel send operations
	peersToBroadcastTo := make([]*SimulatedNetwork, 0, len(sn.peers))
	for _, peer := range sn.peers {
		peersToBroadcastTo = append(peersToBroadcastTo, peer)
	}
	sn.mu.RUnlock()

    if len(peersToBroadcastTo) == 0 {
        log.Printf("SIMNET [%s]: No peers connected to broadcast message type '%s'.", sn.NodeID, messageType)
        return
    }
	log.Printf("SIMNET [%s]: Attempting to broadcast message - Type: %s, Size: %d bytes to %d peers", sn.NodeID, messageType, len(data), len(peersToBroadcastTo))

	for _, peer := range peersToBroadcastTo {
		// log.Printf("SIMNET [%s]: Relaying message type '%s' to peer [%s]", sn.NodeID, messageType, peer.NodeID)
		var targetChannel chan<- []byte // Use send-only channel type for safety

		if messageType == "NEW_BLOCK" {
			targetChannel = peer.BlockBroadcastChannel
		} else if messageType == "NEW_TRANSACTION" { // Conceptual message type for transactions
			targetChannel = peer.TransactionBroadcastChannel
		} else {
			log.Printf("SIMNET [%s]: Unknown message type '%s' for channel broadcast to peer [%s]. Message not sent via typed channel.", sn.NodeID, messageType, peer.NodeID)
            // If there's a generic handler on the peer, we could use its SimulateReceive,
            // but Broadcast is primarily for P2P dissemination via channels here.
            // Alternatively, a generic peer.messageHandler could be called if it exists for non-block/tx messages.
			continue // Skip sending to channel if type is unknown for typed channels
		}

		// Non-blocking send to peer's channel
		select {
		case targetChannel <- data:
			// log.Printf("SIMNET [%s]: Sent data to peer [%s]'s channel for type %s", sn.NodeID, peer.NodeID, messageType)
		default:
			log.Printf("SIMNET [%s]: Peer [%s]'s channel full for type %s. Message dropped for this peer.", sn.NodeID, peer.NodeID, messageType)
		}
	}
}

// RegisterMessageHandler sets a handler function for generic messages not handled by specific channels.
func (sn *SimulatedNetwork) RegisterMessageHandler(handler MessageHandler) {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	sn.messageHandler = handler
	log.Printf("SIMNET [%s]: Generic message handler registered.", sn.NodeID)
}

// GetBlockReceptionChannel returns a read-only channel for receiving block broadcasts.
func (sn *SimulatedNetwork) GetBlockReceptionChannel() <-chan []byte {
	return sn.BlockBroadcastChannel
}

// GetTransactionReceptionChannel returns a read-only channel for receiving transaction broadcasts.
func (sn *SimulatedNetwork) GetTransactionReceptionChannel() <-chan []byte {
	return sn.TransactionBroadcastChannel
}


// SimulateReceive is a helper to manually trigger message processing on this node,
// as if it were received from a peer. It now routes to appropriate channels.
func (sn *SimulatedNetwork) SimulateReceive(peerID string, messageType string, data []byte) {
    log.Printf("SIMNET [%s]: Simulating message reception from peer [%s] - Type: %s, Size: %d bytes", sn.NodeID, peerID, messageType, len(data))

    var targetChannel chan<- []byte // Send-only channel type
    if messageType == "NEW_BLOCK" {
        targetChannel = sn.BlockBroadcastChannel
    } else if messageType == "NEW_TRANSACTION" {
        targetChannel = sn.TransactionBroadcastChannel
    } else {
        // For other message types, use the generic messageHandler if registered
        sn.mu.RLock()
        handler := sn.messageHandler
        sn.mu.RUnlock()
        if handler != nil {
            log.Printf("SIMNET [%s]: Routing message type '%s' to generic message handler.", sn.NodeID, messageType)
            handler(peerID, messageType, data) // Call the generic handler
        } else {
            log.Printf("SIMNET [%s]: SimulateReceive: Unknown message type '%s' and no generic handler. Message dropped.", sn.NodeID, messageType)
        }
        return // Exit after handling via generic handler or dropping
    }

    // Send to the typed channel (NEW_BLOCK or NEW_TRANSACTION)
    select {
    case targetChannel <- data:
        log.Printf("SIMNET [%s]: Successfully simulated receive, data sent to channel for type '%s'.", sn.NodeID, messageType)
    default:
        log.Printf("SIMNET [%s]: SimulateReceive: Own reception channel full for type '%s'. Message dropped.", sn.NodeID, messageType)
    }
}
