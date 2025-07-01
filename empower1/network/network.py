# Main network communication class for EmPower1 nodes.
# Uses Flask for handling incoming HTTP requests and `requests` for sending.

import threading
import time
from flask import Flask, request, jsonify
import requests # For making HTTP requests to other nodes

from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS, VALIDATOR_WALLETS # Assuming these globals are managed
from empower1.transaction import Transaction
from empower1.block import Block
from empower1.network.node import Node
from empower1.network.messages import MessageType

class Network:
    """
    Manages network interactions for a blockchain node.
    """
    def __init__(self, blockchain: Blockchain, host: str, port: int, node_id: str = None, seed_nodes: set = None):
        self.blockchain = blockchain
        if hasattr(self.blockchain, 'set_network_interface'):
            self.blockchain.set_network_interface(self)
        elif hasattr(self.blockchain, 'network_interface'):
             self.blockchain.network_interface = self

        self.self_node = Node(host=host, port=port, node_id=node_id)
        print(f"Network interface initialized for Node: {self.self_node.node_id} at {self.self_node.address}")

        self.peers = set()
        self.seen_tx_ids_broadcast = set()
        self.seen_block_hashes_broadcast = set()
        self.syncing_in_progress = False # Flag to prevent multiple syncs at once


        self.app = Flask(__name__)
        self._configure_routes()

        self.seed_nodes = seed_nodes or set()
        if self.self_node.address in self.seed_nodes:
            self.seed_nodes.remove(self.self_node.address)

    def _configure_routes(self):
        @self.app.route('/ping', methods=['GET'])
        def ping():
            return jsonify({"message": "pong", "node_id": self.self_node.node_id, "address": self.self_node.address}), 200

        @self.app.route(f"/{str(MessageType.GET_CHAIN)}", methods=['GET'])
        def get_chain_endpoint():
            try:
                # Ensure blocks and transactions within blocks are serializable
                chain_data_dicts = [block.to_dict() for block in self.blockchain.chain]
            except Exception as e:
                print(f"Error serializing chain for GET_CHAIN: {e}")
                # Fallback or error response
                chain_data_dicts = [{"error": "Failed to serialize chain"}]
                return jsonify({"error": "Failed to serialize chain data", "details": str(e)}), 500
            return jsonify({"chain": chain_data_dicts, "length": len(self.blockchain.chain)}), 200

        @self.app.route(f"/{str(MessageType.GET_PEERS)}", methods=['GET'])
        def get_peers_api():
            peer_addresses = [peer.address for peer in list(self.peers)]
            return jsonify({"peers": peer_addresses, "node_id": self.self_node.node_id}), 200

        @self.app.route(f"/{str(MessageType.NEW_PEER_ANNOUNCE)}", methods=['POST'])
        def new_peer_announce_api():
            # ... (same as before)
            data = request.get_json()
            if not data or 'address' not in data:
                return jsonify({"error": "Missing peer address"}), 400
            peer_address = data['address']
            if not isinstance(peer_address, str) or not peer_address.startswith("http://"):
                return jsonify({"error": "Invalid peer address format"}), 400
            if peer_address == self.self_node.address:
                return jsonify({"message": "Cannot add self as peer"}), 400
            try:
                peer_node = Node.from_address_string(peer_address)
                if self.add_peer(peer_node):
                    return jsonify({"message": "Peer added and their peers requested", "peer": peer_node.to_dict()}), 201
                else:
                    return jsonify({"message": "Peer already known or is self (not added)"}), 200
            except ValueError as e:
                return jsonify({"error": f"Invalid peer address: {str(e)}"}), 400
            except Exception as e:
                print(f"[{self.self_node.node_id}] Error processing NEW_PEER_ANNOUNCE from {peer_address}: {e}")
                return jsonify({"error": "Failed to process peer announcement"}), 500


        @self.app.route(f"/{str(MessageType.NEW_TRANSACTION)}", methods=['POST'])
        def new_transaction_api():
            tx_data = request.get_json()
            if not tx_data:
                 return jsonify({"error": "No data provided for new transaction"}), 400

            success = self.handle_received_transaction(tx_data)
            if success:
                return jsonify({"message": "Transaction processed"}), 200
            else:
                return jsonify({"message": "Failed to process transaction"}), 400

        @self.app.route(f"/{str(MessageType.NEW_BLOCK)}", methods=['POST'])
        def new_block_api():
            block_data = request.get_json()
            if not block_data:
                return jsonify({"error": "No data provided for new block"}), 400

            success = self.handle_received_block(block_data)
            if success:
                return jsonify({"message": "Block processed"}), 200
            else:
                return jsonify({"message": "Failed to process block"}), 400

    def start_server(self, threaded=True):
        if threaded:
            server_thread = threading.Thread(
                target=lambda: self.app.run(host=self.self_node.host, port=self.self_node.port, debug=False, use_reloader=False)
            )
            server_thread.daemon = True
            server_thread.start()
            print(f"[{self.self_node.node_id}] HTTP server started on {self.self_node.address} (threaded).")
        else:
            print(f"[{self.self_node.node_id}] HTTP server starting on {self.self_node.address} (blocking).")
            self.app.run(host=self.self_node.host, port=self.self_node.port, debug=True)

    def _send_http_request(self, method: str, peer_address: str, endpoint_path: str, json_data: dict = None, timeout=5) -> dict | None: # Increased timeout for chain sync
        try:
            url = f"{peer_address}{endpoint_path}"
            if method.upper() == 'GET':
                response = requests.get(url, timeout=timeout)
            elif method.upper() == 'POST':
                response = requests.post(url, json=json_data, timeout=timeout)
            else:
                return None
            response.raise_for_status()
            return response.json()
        except requests.exceptions.HTTPError: pass
        except requests.exceptions.ConnectionError: pass
        except requests.exceptions.Timeout: pass
        except requests.exceptions.RequestException: pass
        return None

    def connect_to_peer(self, peer_address: str) -> bool:
        if peer_address == self.self_node.address: return False
        ping_response = self._send_http_request('GET', peer_address, '/ping')
        if ping_response and ping_response.get("address") == peer_address:
            try:
                peer_node = Node.from_address_string(peer_address)
                added = self.add_peer(peer_node)
                # If successfully added a new peer, or if it was already known, attempt sync if needed
                if added or peer_node in self.peers: # Check if peer_node is now in self.peers
                    if len(self.blockchain.chain) == 1 and not self.syncing_in_progress: # Only genesis block
                         print(f"[{self.self_node.node_id}] New node or short chain, attempting sync with {peer_node.address}")
                         self.request_chain_from_peer(peer_node)
                return added # Return true if it was newly added by add_peer
            except ValueError as e:
                print(f"[{self.self_node.node_id}] Error parsing address string {peer_address} from ping response: {e}")
        return False

    def connect_to_seed_nodes(self):
        print(f"[{self.self_node.node_id}] Connecting to seed nodes: {self.seed_nodes}")
        for seed_address in list(self.seed_nodes):
            self.connect_to_peer(seed_address)

    def add_peer(self, peer_node: Node) -> bool:
        if peer_node.address == self.self_node.address: return False
        if peer_node in self.peers: return False

        self.peers.add(peer_node)
        print(f"[{self.self_node.node_id}] Added peer: {peer_node.address}. Total peers: {len(self.peers)}")
        self.request_peers_from(peer_node) # Discover more peers from this new peer
        return True

    def request_peers_from(self, peer_node: Node):
        response_data = self._send_http_request('GET', peer_node.address, f"/{str(MessageType.GET_PEERS)}")
        if response_data and 'peers' in response_data:
            newly_discovered_peer_addresses = response_data['peers']
            for new_peer_addr_str in newly_discovered_peer_addresses:
                # Check if not self and not already a known peer by address string
                if new_peer_addr_str != self.self_node.address and \
                   not any(p.address == new_peer_addr_str for p in self.peers):
                    self.connect_to_peer(new_peer_addr_str)

    def get_known_peers_addresses(self) -> list[str]:
        return [peer.address for peer in list(self.peers)]

    def broadcast_transaction(self, transaction: Transaction):
        if transaction.transaction_id in self.seen_tx_ids_broadcast: return
        tx_data = transaction.to_dict()
        self.seen_tx_ids_broadcast.add(transaction.transaction_id)
        # print(f"[{self.self_node.node_id}] Broadcasting tx {transaction.transaction_id} to {len(self.peers)} peers.")
        for peer_node in list(self.peers):
            self._send_http_request('POST', peer_node.address, f"/{str(MessageType.NEW_TRANSACTION)}", json_data=tx_data)

    def broadcast_block(self, block: Block):
        if block.hash in self.seen_block_hashes_broadcast: return
        try:
            block_data = block.to_dict()
        except AttributeError:
            block_data = {"block_repr": repr(block), "hash": block.hash, "index": block.index}

        # print(f"[{self.self_node.node_id}] Broadcasting block {block.hash} (Idx: {block.index}) to {len(self.peers)} peers.")
        self.seen_block_hashes_broadcast.add(block.hash)
        for peer_node in list(self.peers):
            self._send_http_request('POST', peer_node.address, f"/{str(MessageType.NEW_BLOCK)}", json_data=block_data)

    def handle_received_transaction(self, tx_data: dict) -> bool:
        # ... (implementation from previous step, ensure it's robust)
        try:
            required_fields = ['sender_address', 'receiver_address', 'amount', 'signature_hex', 'transaction_id']
            if not all(field in tx_data for field in required_fields):
                # print(f"[{self.self_node.node_id}] Received tx_data missing required fields: {tx_data}")
                return False
            transaction_id = tx_data['transaction_id']
            if any(tx.transaction_id == transaction_id for tx in self.blockchain.pending_transactions) or \
               any(any(tx.transaction_id == transaction_id for tx in b.transactions) for b in self.blockchain.chain):
                return True
            transaction = Transaction.from_dict(tx_data)
            if transaction.transaction_id != transaction_id: return False
            sender_public_key_hex = USER_PUBLIC_KEYS.get(transaction.sender_address)
            if not sender_public_key_hex: return False

            original_bc_net_interface = self.blockchain.network_interface
            self.blockchain.network_interface = None
            success = self.blockchain.add_transaction(transaction, sender_public_key_hex, received_from_network=True)
            self.blockchain.network_interface = original_bc_net_interface

            if success:
                # print(f"[{self.self_node.node_id}] Added transaction {transaction.transaction_id} from network to pending pool.")
                if transaction.transaction_id not in self.seen_tx_ids_broadcast:
                     self.broadcast_transaction(transaction)
                return True
            return False
        except Exception: return False


    def handle_received_block(self, block_data: dict) -> bool:
        # ... (implementation from previous step, ensure it's robust for basic extension)
        try:
            received_block = Block.from_dict(block_data)
            if any(b.hash == received_block.hash for b in self.blockchain.chain): return True

            current_chain_length = len(self.blockchain.chain)
            last_block_hash = self.blockchain.last_block.hash if self.blockchain.last_block else "0" # Should always have genesis

            # Basic case: direct extension of current chain
            if received_block.index == current_chain_length and received_block.previous_hash == last_block_hash:
                # Full validation (similar to blockchain.is_chain_valid for a single block and its contents)
                validator_pub_key = USER_PUBLIC_KEYS.get(received_block.validator_address)
                if not validator_pub_key: return False
                if received_block.hash != received_block._calculate_block_hash() or \
                   not received_block.verify_block_signature(validator_pub_key):
                    return False
                for tx in received_block.transactions:
                    tx_sender_pub_key = USER_PUBLIC_KEYS.get(tx.sender_address)
                    if not tx_sender_pub_key or not tx.verify_signature(tx_sender_pub_key): return False

                self.blockchain.chain.append(received_block)
                self.blockchain.pending_transactions = [
                    p_tx for p_tx in self.blockchain.pending_transactions if p_tx.transaction_id not in {tx.transaction_id for tx in received_block.transactions}
                ]
                print(f"[{self.self_node.node_id}] Added block {received_block.hash} (Index: {received_block.index}) from network.")
                if received_block.hash not in self.seen_block_hashes_broadcast:
                    self.broadcast_block(received_block)
                return True
            # Potential fork or this node is behind
            elif received_block.index >= current_chain_length and not self.syncing_in_progress :
                print(f"[{self.self_node.node_id}] Received block {received_block.hash} (Idx: {received_block.index}) indicates possible fork or being behind. Requesting chain from sender.")
                # Sender info isn't directly available here. Need to get it from request context or message.
                # For now, just pick a peer to sync from if list is not empty
                if self.peers:
                    self.request_chain_from_peer(list(self.peers)[0]) # Simplistic: sync from first peer
                return False # Not added yet, sync will handle
            else: # Block is older or irrelevant for now
                # print(f"[{self.self_node.node_id}] Received block {received_block.hash} (Idx: {received_block.index}) is not a direct extension or is older. Ignoring for now.")
                return False
        except Exception as e:
            print(f"[{self.self_node.node_id}] Error processing received block: {e}")
        return False

    def request_chain_from_peer(self, peer_node: Node):
        if self.syncing_in_progress:
            # print(f"[{self.self_node.node_id}] Sync already in progress. Skipping request to {peer_node.address}.")
            return

        self.syncing_in_progress = True
        print(f"[{self.self_node.node_id}] Requesting full chain from peer {peer_node.address}...")
        response_data = self._send_http_request('GET', peer_node.address, f"/{str(MessageType.GET_CHAIN)}")

        if response_data and 'chain' in response_data and 'length' in response_data:
            print(f"[{self.self_node.node_id}] Received chain of length {response_data['length']} from {peer_node.address}.")
            self.handle_chain_response(response_data['chain'], peer_node)
        else:
            print(f"[{self.self_node.node_id}] Failed to get chain from {peer_node.address} or invalid response.")
        self.syncing_in_progress = False

    def handle_chain_response(self, received_chain_dicts: list[dict], from_peer_node: Node):
        print(f"[{self.self_node.node_id}] Processing chain response from {from_peer_node.address} with {len(received_chain_dicts)} blocks.")
        if not received_chain_dicts:
            print(f"[{self.self_node.node_id}] Received empty chain from {from_peer_node.address}. No action taken.")
            return

        # Basic check: if received chain is shorter or same length as current, ignore (simplistic)
        if len(received_chain_dicts) <= len(self.blockchain.chain):
            print(f"[{self.self_node.node_id}] Received chain (len {len(received_chain_dicts)}) not longer than current (len {len(self.blockchain.chain)}). Ignoring.")
            return

        # Attempt to build and validate the received chain
        prospective_chain = []
        try:
            for block_data in received_chain_dicts:
                prospective_chain.append(Block.from_dict(block_data))
        except Exception as e:
            print(f"[{self.self_node.node_id}] Error deserializing received chain from {from_peer_node.address}: {e}")
            return

        # Full validation of the prospective_chain (from its genesis)
        # This requires a temporary Blockchain instance or a static validation method
        # For simplicity, let's adapt parts of Blockchain.is_chain_valid() here.

        is_valid_prospective_chain = True
        if not prospective_chain: # Should not happen if checked above
            is_valid_prospective_chain = False

        # Check genesis block of prospective chain (must match our known genesis if we have one)
        # This basic sync assumes chains must share the same genesis. More complex sync handles divergent histories.
        if self.blockchain.chain and prospective_chain and prospective_chain[0].hash != self.blockchain.chain[0].hash:
            print(f"[{self.self_node.node_id}] Received chain from {from_peer_node.address} has different genesis block. Sync failed.")
            is_valid_prospective_chain = False

        if is_valid_prospective_chain:
            for i in range(len(prospective_chain)):
                current_block = prospective_chain[i]
                if current_block.hash != current_block._calculate_block_hash(): is_valid_prospective_chain = False; break
                if i > 0: # Non-genesis blocks
                    previous_block = prospective_chain[i-1]
                    if current_block.previous_hash != previous_block.hash: is_valid_prospective_chain = False; break
                    validator_pub_key = USER_PUBLIC_KEYS.get(current_block.validator_address)
                    if not validator_pub_key or not current_block.verify_block_signature(validator_pub_key): is_valid_prospective_chain = False; break
                elif current_block.signature_hex: # Genesis block signature check
                    validator_pub_key = USER_PUBLIC_KEYS.get(current_block.validator_address)
                    if not validator_pub_key or not current_block.verify_block_signature(validator_pub_key): is_valid_prospective_chain = False; break

                for tx in current_block.transactions:
                    tx_sender_pub_key = USER_PUBLIC_KEYS.get(tx.sender_address)
                    if not tx_sender_pub_key or not tx.verify_signature(tx_sender_pub_key): is_valid_prospective_chain = False; break
                if not is_valid_prospective_chain: break

        if is_valid_prospective_chain:
            print(f"[{self.self_node.node_id}] Received chain from {from_peer_node.address} is valid and longer. Updating local chain.")
            self.blockchain.chain = prospective_chain
            # Clear pending transactions as they might be outdated or included
            self.blockchain.pending_transactions = []
            # Clear broadcast history as we have a new chain state
            self.seen_block_hashes_broadcast.clear()
            self.seen_tx_ids_broadcast.clear()
            # Add blocks from new chain to broadcast history to prevent immediate re-broadcast of old blocks
            for block in self.blockchain.chain:
                self.seen_block_hashes_broadcast.add(block.hash)
                for tx in block.transactions:
                    self.seen_tx_ids_broadcast.add(tx.transaction_id)

            # Potentially announce our new chain head or re-evaluate connections
        else:
            print(f"[{self.self_node.node_id}] Received chain from {from_peer_node.address} is invalid or not longer. Local chain preserved.")


if __name__ == '__main__':
    class DemoBlockchain:
        def __init__(self):
            genesis_validator = Wallet()
            USER_PUBLIC_KEYS[genesis_validator.address] = genesis_validator.get_public_key_hex()
            VALIDATOR_WALLETS[genesis_validator.address] = genesis_validator # Store wallet for genesis signing

            self.chain = []
            # Create actual genesis block for DemoBlockchain
            genesis_block = Block(0, [], time.time(), "0", genesis_validator.address)
            genesis_block.sign_block(genesis_validator)
            self.chain.append(genesis_block)

            self.pending_transactions = []
            self.network_interface = None
            print("DemoBlockchain initialized for Network demo.")

        def add_transaction(self, transaction, sender_public_key_hex, received_from_network=False):
            if transaction.verify_signature(sender_public_key_hex):
                if not any(tx.transaction_id == transaction.transaction_id for tx in self.pending_transactions):
                    self.pending_transactions.append(transaction)
                if self.network_interface and not received_from_network:
                    self.network_interface.broadcast_transaction(transaction)
                return True
            return False

        @property
        def last_block(self):
            return self.chain[-1] if self.chain else None

    demo_bc = DemoBlockchain()
    import sys
    default_host = "127.0.0.1"
    default_port = 5000

    if len(sys.argv) > 1 and sys.argv[1] == '--help':
        print("Usage: python empower1/network/network.py [host] [port] [seed_node1_address seed_node2_address ...]")
        print(f"Example: python empower1/network/network.py 127.0.0.1 5000 http://127.0.0.1:5001")
        sys.exit(0)

    host = sys.argv[1] if len(sys.argv) > 1 else default_host
    try:
        port = int(sys.argv[2]) if len(sys.argv) > 2 else default_port
    except ValueError:
        print(f"Warning: Invalid port '{sys.argv[2]}', using default {default_port}.")
        port = default_port

    seed_nodes_for_demo = set(sys.argv[3:]) if len(sys.argv) > 3 else set()

    valid_seeds = set()
    for sn in seed_nodes_for_demo:
        if sn.startswith("http://") and ":" in sn.split("http://")[1]:
            valid_seeds.add(sn)
        else:
            print(f"Warning: Invalid seed node format '{sn}'. Skipping.")

    network_manager = Network(blockchain=demo_bc, host=host, port=port, seed_nodes=valid_seeds)
    demo_bc.network_interface = network_manager

    network_manager.start_server(threaded=True)

    time.sleep(0.5)
    if valid_seeds: # If seeds provided, new node will try to sync
        network_manager.connect_to_seed_nodes()
    elif len(demo_bc.chain) <=1 and not valid_seeds: # If no seeds, and it's a new node
        print(f"[{network_manager.self_node.node_id}] No seed nodes provided. Node started with its genesis block.")


    print(f"\nNode {network_manager.self_node.node_id} running. API at {network_manager.self_node.address}")
    print(f"Known peers: {network_manager.get_known_peers_addresses()}")
    print(f"Endpoints: /ping, /{str(MessageType.GET_CHAIN)}, /{str(MessageType.GET_PEERS)}, /{str(MessageType.NEW_PEER_ANNOUNCE)}")
    print(f"  /{str(MessageType.NEW_TRANSACTION)} (POST), /{str(MessageType.NEW_BLOCK)} (POST)")

    try:
        while True:
            time.sleep(15)
    except KeyboardInterrupt:
        print(f"\nNode {network_manager.self_node.node_id} shutting down.")
