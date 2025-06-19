package network

import (
	"fmt"
	"log"
	"sync"
	"empower1.com/empower1blockchain/internal/core" // For DeserializeBlock/Transaction
)

// MessageHandler defines the function signature for handling messages received from the network.
type MessageHandler func(peerID string, messageType string, data []byte)

// NetworkMessage is a wrapper for data sent between simulated peers, including message type.
type NetworkMessage struct {
	Type string // e.g., "NEW_BLOCK", "NEW_TRANSACTION"
	Data []byte
	// OriginPeerID string // Could be added if direct peer-to-peer replies are needed later
}

// Peer represents a connected node in the simulated network.
type Peer struct {
	ID               string
	IncomingMessages chan NetworkMessage // Changed to NetworkMessage
	stopChan         chan struct{}
	wg               sync.WaitGroup
	network          *SimulatedNetwork
}

// NewPeer creates a new Peer instance.
func NewPeer(id string, net *SimulatedNetwork) *Peer {
	return &Peer{
		ID:               id,
		IncomingMessages: make(chan NetworkMessage, 100), // Now NetworkMessage
		stopChan:         make(chan struct{}),
		network:          net,
	}
}

// conceptualPeerMessageProcessor is run as a goroutine for each peer.
// It reads NetworkMessage from p.IncomingMessages, deserializes, and routes
// the raw data to the appropriate public channel on the main SimulatedNetwork instance.
func (p *Peer) conceptualPeerMessageProcessor() {
	defer p.wg.Done()
	log.Printf("SIMNET_PEER_PROCESSOR [%s]: Starting message processor for peer connection to [%s]", p.network.NodeID, p.ID)
	for {
		select {
		case msg, ok := <-p.IncomingMessages:
			if !ok {
				log.Printf("SIMNET_PEER_PROCESSOR [%s]: IncomingMessages channel closed for peer [%s]. Processor stopping.", p.network.NodeID, p.ID)
				return
			}

			// log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] received message of type '%s', data size %d. Routing...", p.network.NodeID, p.ID, msg.Type, len(msg.Data))

			switch msg.Type {
			case "NEW_BLOCK":
				// Block is deserialized here just to log its hash, but raw data is sent to the channel
				// as ConsensusEngine expects to deserialize it.
				block, err := core.DeserializeBlock(msg.Data)
				if err != nil {
					log.Printf("SIMNET_PEER_PROCESSOR_ERROR [%s]: Peer [%s] failed to deserialize block for logging: %v", p.network.NodeID, p.ID, err)
					// Still attempt to route raw data, consumer will handle deserialization error
				}
				select {
				case p.network.BlockBroadcastChannel <- msg.Data:
					if block != nil {
						log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] routed block %x to network's BlockBroadcastChannel", p.network.NodeID, p.ID, block.Hash)
					} else {
						log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] routed block (deserialization failed) to network's BlockBroadcastChannel", p.network.NodeID, p.ID)
					}
				default:
					log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] found network's BlockBroadcastChannel full when routing block.", p.network.NodeID, p.ID)
				}
			case "NEW_TRANSACTION":
				// Tx is deserialized here just to log its ID.
				tx, err := core.DeserializeTransaction(msg.Data)
				if err != nil {
					log.Printf("SIMNET_PEER_PROCESSOR_ERROR [%s]: Peer [%s] failed to deserialize transaction for logging: %v", p.network.NodeID, p.ID, err)
				}
				select {
				case p.network.TransactionBroadcastChannel <- msg.Data:
					if tx != nil {
						log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] routed transaction %x to network's TransactionBroadcastChannel", p.network.NodeID, p.ID, tx.ID)
					} else {
						log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] routed transaction (deserialization failed) to network's TransactionBroadcastChannel", p.network.NodeID, p.ID)
					}
				default:
					if tx != nil {
						log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] found network's TransactionBroadcastChannel full when routing transaction %x.", p.network.NodeID, p.ID, tx.ID)
					} else {
						log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] found network's TransactionBroadcastChannel full when routing transaction (deserialization failed).", p.network.NodeID, p.ID)
					}
				}
			default:
				log.Printf("SIMNET_PEER_PROCESSOR [%s]: Peer [%s] received message with unknown type '%s'. Discarding.", p.network.NodeID, p.ID, msg.Type)
				// If generic messageHandler exists and is intended for other types:
				// if p.network.messageHandler != nil {
				//    p.network.messageHandler(p.ID, msg.Type, msg.Data)
				// }
			}

		case <-p.stopChan:
			log.Printf("SIMNET_PEER_PROCESSOR [%s]: Stopping message processor for peer [%s]", p.network.NodeID, p.ID)
			return
		}
	}
}

// StartProcessor starts the peer's message processor goroutine.
func (p *Peer) StartProcessor() {
	p.wg.Add(1)
	go p.conceptualPeerMessageProcessor()
}

// StopProcessor signals the peer's message processor to stop and waits for it.
func (p *Peer) StopProcessor() {
	close(p.stopChan)
	p.wg.Wait()
}

// SimulatedNetwork provides a basic, in-memory simulation of network interactions.
type SimulatedNetwork struct {
	NodeID                      string
	mu                          sync.RWMutex
	messageHandler              MessageHandler
	BlockBroadcastChannel       chan []byte
	TransactionBroadcastChannel chan []byte
	peers                       map[string]*Peer
}

// NewSimulatedNetwork creates a new SimulatedNetwork instance.
func NewSimulatedNetwork(nodeID string) *SimulatedNetwork {
	if nodeID == "" {
		nodeID = "default_sim_node"
	}
	return &SimulatedNetwork{
		NodeID:                      nodeID,
		BlockBroadcastChannel:       make(chan []byte, 100),
		TransactionBroadcastChannel: make(chan []byte, 100),
		peers:                       make(map[string]*Peer),
	}
}

// ConnectPeer adds another node to this node's peer list.
func (sn *SimulatedNetwork) ConnectPeer(peerNodeID string) (*Peer, error) {
	if peerNodeID == "" {
		return nil, fmt.Errorf("SIMNET [%s]: cannot connect to peer with empty ID", sn.NodeID)
	}
	if sn.NodeID == peerNodeID {
		return nil, fmt.Errorf("SIMNET [%s]: cannot connect to self", sn.NodeID)
	}
	sn.mu.Lock()
	defer sn.mu.Unlock()

	if existingPeer, exists := sn.peers[peerNodeID]; exists {
		return existingPeer, nil
	}

	peer := NewPeer(peerNodeID, sn)
	peer.StartProcessor()
	sn.peers[peerNodeID] = peer
	log.Printf("SIMNET [%s]: Connected to peer [%s] and started its processor.", sn.NodeID, peerNodeID)
	return peer, nil
}

// DisconnectPeer removes a peer from the list and stops its processor.
func (sn *SimulatedNetwork) DisconnectPeer(peerNodeID string) {
	sn.mu.Lock()
	peer, exists := sn.peers[peerNodeID]
	if !exists {
		sn.mu.Unlock()
		log.Printf("SIMNET [%s]: Peer [%s] not found for disconnection.", sn.NodeID, peerNodeID)
		return
	}
	delete(sn.peers, peerNodeID)
	sn.mu.Unlock()

	log.Printf("SIMNET [%s]: Disconnecting from peer [%s]...", sn.NodeID, peerNodeID)
	peer.StopProcessor()
	log.Printf("SIMNET [%s]: Disconnected from peer [%s] and stopped its processor.", sn.NodeID, peerNodeID)
}

// sendToPeers is an internal helper to send a NetworkMessage to all peers.
func (sn *SimulatedNetwork) sendToPeers(msg NetworkMessage) {
	sn.mu.RLock()
	peersToNotify := make([]*Peer, 0, len(sn.peers))
	for _, p := range sn.peers {
		peersToNotify = append(peersToNotify, p)
	}
	sn.mu.RUnlock()

	if len(peersToNotify) == 0 {
        // log.Printf("SIMNET [%s]: No peers connected to broadcast message type '%s'.", sn.NodeID, msg.Type) // Redundant with BroadcastBlock/Tx logs
        return
    }
	// log.Printf("SIMNET [%s]: Relaying message type '%s' (size %d) to %d peers", sn.NodeID, msg.Type, len(msg.Data), len(peersToNotify))


	for _, peer := range peersToNotify {
		select {
		case peer.IncomingMessages <- msg:
			// Successfully sent
		default:
			log.Printf("SIMNET [%s]: Peer [%s]'s IncomingMessages channel full for type %s. Message dropped.", sn.NodeID, peer.ID, msg.Type)
		}
	}
}

// BroadcastBlock serializes a block and sends it to all connected peers.
func (sn *SimulatedNetwork) BroadcastBlock(block *core.Block) {
	if block == nil {
		log.Printf("SIMNET [%s]: Attempted to broadcast nil block.", sn.NodeID)
		return
	}
	serializedBlock, err := block.Serialize()
	if err != nil {
		log.Printf("SIMNET [%s]: ERROR serializing block %x for broadcast: %v", sn.NodeID, block.Hash, err)
		return
	}

	log.Printf("SIMNET [%s]: Broadcasting NEW_BLOCK message (size %d) for block %x", sn.NodeID, len(serializedBlock), block.Hash)
	msg := NetworkMessage{Type: "NEW_BLOCK", Data: serializedBlock}
	sn.sendToPeers(msg)
}

// BroadcastTransaction serializes a transaction and sends it to all connected peers.
func (sn *SimulatedNetwork) BroadcastTransaction(tx *core.Transaction) {
	if tx == nil {
		log.Printf("SIMNET [%s]: Attempted to broadcast nil transaction.", sn.NodeID)
		return
	}
	serializedTx, err := tx.Serialize()
	if err != nil {
		log.Printf("SIMNET [%s]: ERROR serializing transaction %x for broadcast: %v", sn.NodeID, tx.ID, err)
		return
	}
	log.Printf("SIMNET [%s]: Broadcasting NEW_TRANSACTION message (size %d) for tx %x", sn.NodeID, len(serializedTx), tx.ID)
	msg := NetworkMessage{Type: "NEW_TRANSACTION", Data: serializedTx}
	sn.sendToPeers(msg)
}


/*
// Old generic Broadcast - to be removed or made private if specific use cases remain.
func (sn *SimulatedNetwork) Broadcast(messageType string, data []byte) {
	sn.mu.RLock()
	peersToNotify := make([]*Peer, 0, len(sn.peers))
	for _, p := range sn.peers {
		peersToNotify = append(peersToNotify, p)
	}
	sn.mu.RUnlock()

	if len(peersToNotify) == 0 {
        log.Printf("SIMNET [%s]: No peers connected to broadcast message type '%s'.", sn.NodeID, messageType)
        return
    }
	log.Printf("SIMNET [%s]: Broadcasting message type '%s' (size %d bytes) to %d peers.", sn.NodeID, messageType, len(data), len(peersToNotify))

	wrappedMsg := NetworkMessage{Type: messageType, Data: data}
	for _, peer := range peersToNotify {
		select {
		case peer.IncomingMessages <- wrappedMsg:
		default:
			log.Printf("SIMNET [%s]: Peer [%s]'s IncomingMessages channel full. Message dropped for this peer.", sn.NodeID, peer.ID)
		}
	}
}
*/

// RegisterMessageHandler sets a handler function for generic messages.
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

// SimulateReceive is a helper to manually trigger message processing on THIS node,
// as if it were received from an external peer. It routes to appropriate internal channels.
func (sn *SimulatedNetwork) SimulateReceive(peerID string, messageType string, data []byte) {
    log.Printf("SIMNET [%s]: Simulating message reception from peer [%s] - Type: %s, Size: %d bytes", sn.NodeID, peerID, messageType, len(data))

    var targetChannel chan<- []byte
    if messageType == "NEW_BLOCK" {
        targetChannel = sn.BlockBroadcastChannel
    } else if messageType == "NEW_TRANSACTION" {
        targetChannel = sn.TransactionBroadcastChannel
    } else {
        sn.mu.RLock()
        handler := sn.messageHandler
        sn.mu.RUnlock()
        if handler != nil {
            log.Printf("SIMNET [%s]: Routing message type '%s' to generic message handler.", sn.NodeID, messageType)
            handler(peerID, messageType, data)
        } else {
            log.Printf("SIMNET [%s]: SimulateReceive: Unknown message type '%s' and no generic handler. Message dropped.", sn.NodeID, messageType)
        }
        return
    }

    select {
    case targetChannel <- data:
        log.Printf("SIMNET [%s]: Successfully simulated receive, data sent to channel for type '%s'.", sn.NodeID, messageType)
    default:
        log.Printf("SIMNET [%s]: SimulateReceive: Own reception channel full for type '%s'. Message dropped.", sn.NodeID, messageType)
    }
}
