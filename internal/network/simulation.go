package network

import (
	"log" // For logging broadcast/receive actions
	"sync"
)

// MessageHandler defines the function signature for handling messages received from the network.
// peerID might be the sender's NodeID in a real network.
type MessageHandler func(peerID string, messageType string, data []byte)

// SimulatedNetwork provides a basic, in-memory simulation of network interactions
// for testing consensus and other components without a full P2P stack.
type SimulatedNetwork struct {
	NodeID         string
	mu             sync.RWMutex
	messageHandler MessageHandler
	// TODO: Could add a channel here to simulate message passing between multiple
	//       SimulatedNetwork instances in more complex tests.
	//       For now, Broadcast just logs.
}

// NewSimulatedNetwork creates a new SimulatedNetwork instance.
func NewSimulatedNetwork(nodeID string) *SimulatedNetwork {
	if nodeID == "" {
		nodeID = "default_sim_node" // Default if not provided
	}
	return &SimulatedNetwork{
		NodeID: nodeID,
	}
}

// Broadcast logs a message, simulating it being sent to the network.
// In a more complex simulation, this might put the message on a shared channel.
func (sn *SimulatedNetwork) Broadcast(messageType string, data []byte) {
	log.Printf("SIMNET [%s]: Broadcasting message - Type: %s, Size: %d bytes", sn.NodeID, messageType, len(data))
	// TODO: Implement actual message delivery if simulating multiple nodes.
}

// RegisterMessageHandler sets a handler function to be called when a message is "received".
// In this simple simulation, messages are not actually received from peers yet.
// This sets up the callback that would be used by a real network layer.
func (sn *SimulatedNetwork) RegisterMessageHandler(handler MessageHandler) {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	sn.messageHandler = handler
	log.Printf("SIMNET [%s]: Message handler registered.", sn.NodeID)
}

// SimulateReceive is a helper to manually trigger the message handler for testing.
// This would not be part of a real network interface but is useful for simulation.
func (sn *SimulatedNetwork) SimulateReceive(peerID string, messageType string, data []byte) {
    sn.mu.RLock()
    handler := sn.messageHandler
    sn.mu.RUnlock()

    if handler != nil {
        log.Printf("SIMNET [%s]: Simulating message receive from %s - Type: %s, Size: %d bytes", sn.NodeID, peerID, messageType, len(data))
        handler(peerID, messageType, data)
    } else {
        log.Printf("SIMNET [%s]: SimulateReceive called, but no message handler registered for type %s.", sn.NodeID, messageType)
    }
}
