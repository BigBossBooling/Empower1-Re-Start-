# Main network communication class for EmPower1 nodes.
# Uses Flask for handling incoming HTTP requests and `requests` for sending.

import threading
import time
from flask import Flask, request, jsonify
import requests # For making HTTP requests to other nodes

from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS # Assuming USER_PUBLIC_KEYS is a global or accessible dict
from empower1.transaction import Transaction
from empower1.block import Block
from empower1.network.node import Node
from empower1.network.messages import MessageType

class Network:
    """
    Manages network interactions for a blockchain node.
    """
    def __init__(self, blockchain: Blockchain, host: str, port: int, node_id: str = None, seed_nodes: set = None):
        self.blockchain = blockchain # Blockchain instance is now passed here
        # Link the blockchain back to this network interface if it needs to call broadcast methods
        if hasattr(self.blockchain, 'set_network_interface'):
            self.blockchain.set_network_interface(self)
        elif hasattr(self.blockchain, 'network_interface'): # Or if it just has an attribute
             self.blockchain.network_interface = self

        self.self_node = Node(host=host, port=port, node_id=node_id)
        print(f"Network interface initialized for Node: {self.self_node.node_id} at {self.self_node.address}")

        self.peers = set()

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
                chain_data_dicts = [block.to_dict() for block in self.blockchain.chain]
            except AttributeError:
                chain_data_dicts = [repr(block) for block in self.blockchain.chain]
            return jsonify({"chain": chain_data_dicts, "length": len(self.blockchain.chain)}), 200

        @self.app.route(f"/{str(MessageType.GET_PEERS)}", methods=['GET'])
        def get_peers_api():
            peer_addresses = [peer.address for peer in list(self.peers)]
            return jsonify({"peers": peer_addresses, "node_id": self.self_node.node_id}), 200

        @self.app.route(f"/{str(MessageType.NEW_PEER_ANNOUNCE)}", methods=['POST'])
        def new_peer_announce_api():
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
                    return jsonify({"message": "Peer already known"}), 200
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

            # print(f"[{self.self_node.node_id}] Received potential new transaction via API: {tx_data.get('transaction_id', 'N/A')}")
            success = self.handle_received_transaction(tx_data)
            if success:
                return jsonify({"message": "Transaction processed successfully"}), 201
            else:
                return jsonify({"message": "Failed to process transaction"}), 400

        @self.app.route(f"/{str(MessageType.NEW_BLOCK)}", methods=['POST'])
        def new_block_api():
            block_data = request.get_json()
            if not block_data:
                return jsonify({"error": "No data provided for new block"}), 400

            # print(f"[{self.self_node.node_id}] Received potential new block via API: {block_data.get('hash', 'N/A')}")
            success = self.handle_received_block(block_data)
            if success:
                return jsonify({"message": "Block processed successfully"}), 201
            else:
                return jsonify({"message": "Failed to process block"}), 400

    def start_server(self, threaded=True):
        # ... (server start logic remains the same)
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


    def _send_http_request(self, method: str, peer_address: str, endpoint_path: str, json_data: dict = None, timeout=2) -> dict | None:
        # ... (send http request logic remains the same)
        try:
            url = f"{peer_address}{endpoint_path}"
            if method.upper() == 'GET':
                response = requests.get(url, timeout=timeout)
            elif method.upper() == 'POST':
                response = requests.post(url, json=json_data, timeout=timeout)
            else:
                print(f"[{self.self_node.node_id}] Unsupported HTTP method: {method}")
                return None
            response.raise_for_status()
            return response.json()
        except requests.exceptions.HTTPError as http_err:
            print(f"[{self.self_node.node_id}] HTTP error requesting {url}: {http_err} - Response: {http_err.response.text[:200] if http_err.response else 'No response text'}")
        except requests.exceptions.ConnectionError as conn_err:
            print(f"[{self.self_node.node_id}] Connection error requesting {url}: {conn_err}")
        except requests.exceptions.Timeout as timeout_err:
            print(f"[{self.self_node.node_id}] Timeout requesting {url}: {timeout_err}")
        except requests.exceptions.RequestException as req_err:
            print(f"[{self.self_node.node_id}] General error requesting {url}: {req_err}")
        return None

    def connect_to_peer(self, peer_address: str) -> bool:
        # ... (connect to peer logic remains the same)
        if peer_address == self.self_node.address: return False
        ping_response = self._send_http_request('GET', peer_address, '/ping')
        if ping_response and ping_response.get("address") == peer_address:
            try:
                peer_node = Node.from_address_string(peer_address)
                return self.add_peer(peer_node)
            except ValueError as e:
                print(f"[{self.self_node.node_id}] Error parsing address string {peer_address} from ping response: {e}")
                return False
        return False

    def connect_to_seed_nodes(self):
        # ... (connect to seed nodes logic remains the same)
        print(f"[{self.self_node.node_id}] Connecting to seed nodes: {self.seed_nodes}")
        for seed_address in list(self.seed_nodes):
            self.connect_to_peer(seed_address)

    def add_peer(self, peer_node: Node) -> bool:
        # ... (add peer logic remains the same)
        if peer_node.address == self.self_node.address: return False
        if peer_node in self.peers: return False

        self.peers.add(peer_node)
        print(f"[{self.self_node.node_id}] Added peer: {peer_node.address}. Total peers: {len(self.peers)}")
        self.request_peers_from(peer_node)
        return True

    def request_peers_from(self, peer_node: Node):
        # ... (request peers from logic remains the same)
        response_data = self._send_http_request('GET', peer_node.address, f"/{str(MessageType.GET_PEERS)}")
        if response_data and 'peers' in response_data:
            newly_discovered_peer_addresses = response_data['peers']
            for new_peer_addr_str in newly_discovered_peer_addresses:
                if new_peer_addr_str != self.self_node.address and new_peer_addr_str not in [p.address for p in self.peers]:
                    self.connect_to_peer(new_peer_addr_str)

    def get_known_peers_addresses(self) -> list[str]:
        # ... (get known peers addresses logic remains the same)
        return [peer.address for peer in list(self.peers)]

    def broadcast_transaction(self, transaction: Transaction):
        # ... (broadcast transaction logic remains the same)
        tx_data = transaction.to_dict()
        print(f"[{self.self_node.node_id}] Broadcasting transaction {transaction.transaction_id} to {len(self.peers)} peers.")
        for peer_node in list(self.peers):
            self._send_http_request('POST', peer_node.address, f"/{str(MessageType.NEW_TRANSACTION)}", json_data=tx_data)

    def broadcast_block(self, block: Block):
        # ... (broadcast block logic remains the same)
        try:
            block_data = block.to_dict()
        except AttributeError:
            block_data = {"block_repr": repr(block), "hash": block.hash, "index": block.index}

        print(f"[{self.self_node.node_id}] Broadcasting block {block.hash} (Index: {block.index}) to {len(self.peers)} peers.")
        for peer_node in list(self.peers):
            self._send_http_request('POST', peer_node.address, f"/{str(MessageType.NEW_BLOCK)}", json_data=block_data)

    def handle_received_transaction(self, tx_data: dict) -> bool:
        print(f"[{self.self_node.node_id}] Processing received transaction: {tx_data.get('transaction_id', 'UnknownID')}")
        try:
            required_fields = ['sender_address', 'receiver_address', 'amount', 'signature_hex'] # Add 'asset_id', 'timestamp', 'fee', 'metadata' if strictly needed by from_dict for basic function
            if not all(field in tx_data for field in required_fields): # Check essential fields for from_dict
                print(f"[{self.self_node.node_id}] Received tx_data missing required fields for deserialization: {tx_data}")
                return False

            transaction = Transaction.from_dict(tx_data)

            # Check if transaction already processed or in chain (simple check by ID)
            if any(tx.transaction_id == transaction.transaction_id for tx in self.blockchain.pending_transactions) or \
               any(any(tx.transaction_id == transaction.transaction_id for tx in b.transactions) for b in self.blockchain.chain):
                # print(f"[{self.self_node.node_id}] Transaction {transaction.transaction_id} already known. Skipping.")
                return True # Acknowledge as processed, but not "newly added"

            # Sender's public key must be in tx_data or globally available.
            # The plan was for Blockchain.add_transaction to take sender_public_key_hex.
            # If tx_data contains 'sender_public_key_hex', use it. Otherwise, fallback to USER_PUBLIC_KEYS.
            sender_public_key_hex = tx_data.get('sender_public_key_hex')
            if not sender_public_key_hex:
                 sender_public_key_hex = USER_PUBLIC_KEYS.get(transaction.sender_address)

            if not sender_public_key_hex:
                print(f"[{self.self_node.node_id}] Public key for sender {transaction.sender_address} not found. Cannot verify tx {transaction.transaction_id}.")
                return False

            # The blockchain's add_transaction method now handles signature verification and then broadcasting.
            # We pass False for its internal broadcast flag to prevent immediate re-broadcast from there if called from network handler.
            # This node will decide to re-broadcast based on its own logic (e.g. if it was new to this node).
            # For now, let's assume blockchain.add_transaction is modified to accept a 'broadcast_internally=True' param
            # or we rely on its current behavior. The current blockchain.add_transaction *will* broadcast.
            # This could lead to broadcast storms if not handled carefully (e.g. with a set of seen tx_ids for broadcast).
            # For this step, let's assume add_transaction handles it or we accept potential extra broadcasts.

            # Store original network interface to temporarily disable broadcast from add_transaction
            original_bc_net_interface = self.blockchain.network_interface
            self.blockchain.network_interface = None # Temporarily disable broadcast from blockchain.add_transaction

            success = self.blockchain.add_transaction(transaction, sender_public_key_hex)

            self.blockchain.network_interface = original_bc_net_interface # Restore it

            if success:
                print(f"[{self.self_node.node_id}] Successfully processed and added transaction {transaction.transaction_id} from network.")
                # Decide if this node should re-broadcast (e.g., if it's the first time seeing this tx)
                # For simplicity, we can re-broadcast. More advanced: use a set of tx_ids already broadcasted by this node.
                self.broadcast_transaction(transaction)
                return True
            else:
                print(f"[{self.self_node.node_id}] Failed to add transaction {transaction.transaction_id} from network to blockchain.")
                return False
        except Exception as e:
            print(f"[{self.self_node.node_id}] Error processing received transaction: {e}")
            return False

    def handle_received_block(self, block_data: dict) -> bool:
        print(f"[{self.self_node.node_id}] Processing received block: {block_data.get('hash', 'UnknownHash')}")
        try:
            received_block = Block.from_dict(block_data)

            if any(b.hash == received_block.hash for b in self.blockchain.chain):
                # print(f"[{self.self_node.node_id}] Block {received_block.hash} already exists. Skipping.")
                return True # Acknowledge as processed.

            # Basic validation (more advanced fork resolution would go here)
            if received_block.index == len(self.blockchain.chain) and \
               received_block.previous_hash == self.blockchain.last_block.hash:

                # Full validation of the block itself (hash, signatures)
                validator_pub_key = USER_PUBLIC_KEYS.get(received_block.validator_address)
                if not validator_pub_key:
                    print(f"[{self.self_node.node_id}] Validator public key for {received_block.validator_address} not found.")
                    return False

                if received_block.hash != received_block._calculate_block_hash() or \
                   not received_block.verify_block_signature(validator_pub_key):
                    print(f"[{self.self_node.node_id}] Invalid block signature or hash for block {received_block.hash}.")
                    return False

                for tx in received_block.transactions:
                    tx_sender_pub_key = USER_PUBLIC_KEYS.get(tx.sender_address)
                    if not tx_sender_pub_key or not tx.verify_signature(tx_sender_pub_key):
                        print(f"[{self.self_node.node_id}] Invalid transaction {tx.transaction_id} in block {received_block.hash}.")
                        return False

                # If all checks pass, add to chain
                self.blockchain.chain.append(received_block)
                # Remove transactions in this block from pending pool
                mined_tx_ids = {tx.transaction_id for tx in received_block.transactions}
                self.blockchain.pending_transactions = [
                    p_tx for p_tx in self.blockchain.pending_transactions if p_tx.transaction_id not in mined_tx_ids
                ]
                print(f"[{self.self_node.node_id}] Added block {received_block.hash} from network.")
                # Re-broadcast valid new block
                self.broadcast_block(received_block)
                return True
            else:
                print(f"[{self.self_node.node_id}] Received block {received_block.hash} does not extend current chain or prev_hash mismatch.")
                # TODO: Handle forks / request missing blocks
                return False
        except Exception as e:
            print(f"[{self.self_node.node_id}] Error processing received block: {e}")
            return False


if __name__ == '__main__':
    # ... (__main__ remains largely the same, ensure DemoBlockchain's Block has to_dict or is minimal)
    class DemoBlockchain:
        def __init__(self):
            genesis_validator = Wallet()
            # Ensure USER_PUBLIC_KEYS is populated for the genesis validator for block verification
            USER_PUBLIC_KEYS[genesis_validator.address] = genesis_validator.get_public_key_hex()

            # Create a minimal valid Block for to_dict()
            # For Block.to_dict() to work, transactions need to_dict().
            # If Block.to_dict() serializes transactions, they must be actual Transaction objects.
            self.chain = [Block(0, [], time.time(), "0", genesis_validator.address)]
            self.chain[0].sign_block(genesis_validator) # Sign it

            self.pending_transactions = [] # Add for handle_received_transaction
            self.network_interface = None # For add_transaction to not fail if it tries to broadcast
            print("DemoBlockchain initialized for Network demo.")

        # Minimal add_transaction for demo handle_received_transaction
        def add_transaction(self, transaction, sender_public_key_hex):
            # Simulate basic validation and adding
            if transaction.verify_signature(sender_public_key_hex):
                self.pending_transactions.append(transaction)
                print(f"[DemoBC] Added pending tx: {transaction.transaction_id}")
                return True
            print(f"[DemoBC] Failed to add tx: {transaction.transaction_id}")
            return False

        @property # Add last_block for handle_received_block
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
    # Link network_manager to demo_bc so its broadcast methods can be called by demo_bc if needed
    demo_bc.network_interface = network_manager # For the broadcast in add_transaction/mine_block

    network_manager.start_server(threaded=True)

    time.sleep(0.5)
    if valid_seeds:
        network_manager.connect_to_seed_nodes()

    print(f"\nNode {network_manager.self_node.node_id} running. API at {network_manager.self_node.address}")
    print(f"Known peers: {network_manager.get_known_peers_addresses()}")
    print(f"Endpoints: /ping, /{str(MessageType.GET_PEERS)}, /{str(MessageType.NEW_PEER_ANNOUNCE)} (POST with {{'address': 'http://otherhost:otherport'}})")
    print(f"  /{str(MessageType.NEW_TRANSACTION)} (POST), /{str(MessageType.NEW_BLOCK)} (POST)")

    try:
        while True:
            time.sleep(15)
    except KeyboardInterrupt:
        print(f"\nNode {network_manager.self_node.node_id} shutting down.")
