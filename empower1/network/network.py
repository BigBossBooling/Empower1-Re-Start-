import threading
import time
from flask import Flask, request, jsonify
import requests

from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS, VALIDATOR_WALLETS
from empower1.transaction import Transaction
from empower1.block import Block
from empower1.network.node import Node
from empower1.network.messages import MessageType

class Network:
    def __init__(self, blockchain: Blockchain, host: str, port: int, node_id: str = None, seed_nodes: set = None):
        self.blockchain = blockchain
        if hasattr(self.blockchain, 'set_network_interface'):
            self.blockchain.set_network_interface(self)
        elif hasattr(self.blockchain, 'network_interface'):
             self.blockchain.network_interface = self
        self.self_node = Node(host=host, port=port, node_id=node_id)
        # print(f"Network interface initialized for Node: {self.self_node.node_id} at {self.self_node.address}")
        self.peers = set()
        self.seen_tx_ids_broadcast = set()
        self.seen_block_hashes_broadcast = set()
        self.syncing_in_progress = False
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
            except Exception as e:
                # print(f"Error serializing chain for GET_CHAIN: {e}")
                return jsonify({"error": "Failed to serialize chain data", "details": str(e)}), 500
            return jsonify({"chain": chain_data_dicts, "length": len(self.blockchain.chain)}), 200

        @self.app.route(f"/{str(MessageType.GET_PEERS)}", methods=['GET'])
        def get_peers_api():
            peer_addresses = [peer.address for peer in list(self.peers)]
            return jsonify({"peers": peer_addresses, "node_id": self.self_node.node_id}), 200

        @self.app.route(f"/{str(MessageType.NEW_PEER_ANNOUNCE)}", methods=['POST'])
        def new_peer_announce_api():
            data = request.get_json()
            if not data or 'address' not in data: return jsonify({"error": "Missing peer address"}), 400
            peer_address = data['address']
            if not isinstance(peer_address, str) or not peer_address.startswith("http://"): return jsonify({"error": "Invalid peer address format"}), 400
            if peer_address == self.self_node.address: return jsonify({"message": "Cannot add self as peer"}), 400
            try:
                peer_node = Node.from_address_string(peer_address)
                if self.add_peer(peer_node): return jsonify({"message": "Peer added", "peer": peer_node.to_dict()}), 201
                else: return jsonify({"message": "Peer already known or is self"}), 200
            except ValueError as e: return jsonify({"error": f"Invalid peer address: {str(e)}"}), 400
            except Exception as e: return jsonify({"error": "Failed to process peer announcement"}), 500

        @self.app.route(f"/{str(MessageType.NEW_TRANSACTION)}", methods=['POST'])
        def new_transaction_api():
            tx_data = request.get_json()
            if not tx_data: return jsonify({"error": "No data provided"}), 400
            success = self.handle_received_transaction(tx_data)
            if success: return jsonify({"message": "Transaction processed"}), 200
            else: return jsonify({"message": "Failed to process transaction"}), 400

        @self.app.route(f"/{str(MessageType.NEW_BLOCK)}", methods=['POST'])
        def new_block_api():
            block_data = request.get_json()
            if not block_data: return jsonify({"error": "No data provided"}), 400
            success = self.handle_received_block(block_data)
            if success: return jsonify({"message": "Block processed"}), 200
            else: return jsonify({"message": "Failed to process block"}), 400

    def start_server(self, threaded=True):
        if threaded:
            server_thread = threading.Thread(target=lambda: self.app.run(host=self.self_node.host, port=self.self_node.port, debug=False, use_reloader=False))
            server_thread.daemon = True
            server_thread.start()
        else: self.app.run(host=self.self_node.host, port=self.self_node.port, debug=True)
        print(f"[{self.self_node.node_id}] HTTP server started on {self.self_node.address}.")

    def _send_http_request(self, method: str, peer_address: str, endpoint_path: str, json_data: dict = None, timeout=5) -> dict | None:
        try:
            url = f"{peer_address}{endpoint_path}"
            if method.upper() == 'GET': response = requests.get(url, timeout=timeout)
            elif method.upper() == 'POST': response = requests.post(url, json=json_data, timeout=timeout)
            else: return None
            response.raise_for_status()
            return response.json()
        except (requests.exceptions.HTTPError, requests.exceptions.ConnectionError, requests.exceptions.Timeout, requests.exceptions.RequestException): pass
        return None

    def connect_to_peer(self, peer_address: str) -> bool:
        if peer_address == self.self_node.address: return False
        ping_response = self._send_http_request('GET', peer_address, '/ping')
        if ping_response and ping_response.get("address") == peer_address:
            try:
                peer_node = Node.from_address_string(peer_address)
                added = self.add_peer(peer_node)
                if added or peer_node in self.peers:
                    if len(self.blockchain.chain) == 1 and not self.syncing_in_progress:
                         self.request_chain_from_peer(peer_node)
                return added
            except ValueError: pass
        return False

    def connect_to_seed_nodes(self):
        for seed_address in list(self.seed_nodes): self.connect_to_peer(seed_address)

    def add_peer(self, peer_node: Node) -> bool:
        if peer_node.address == self.self_node.address or peer_node in self.peers: return False
        self.peers.add(peer_node)
        print(f"[{self.self_node.node_id}] Added peer: {peer_node.address}. Total: {len(self.peers)}")
        self.request_peers_from(peer_node)
        return True

    def request_peers_from(self, peer_node: Node):
        response_data = self._send_http_request('GET', peer_node.address, f"/{str(MessageType.GET_PEERS)}")
        if response_data and 'peers' in response_data:
            for new_peer_addr_str in response_data['peers']:
                if new_peer_addr_str != self.self_node.address and not any(p.address == new_peer_addr_str for p in self.peers):
                    self.connect_to_peer(new_peer_addr_str)

    def get_known_peers_addresses(self) -> list[str]: return [p.address for p in list(self.peers)]

    def broadcast_transaction(self, transaction: Transaction):
        if transaction.transaction_id in self.seen_tx_ids_broadcast: return
        tx_data = transaction.to_dict()
        self.seen_tx_ids_broadcast.add(transaction.transaction_id)
        for peer_node in list(self.peers):
            self._send_http_request('POST', peer_node.address, f"/{str(MessageType.NEW_TRANSACTION)}", json_data=tx_data)

    def broadcast_block(self, block: Block):
        if block.hash in self.seen_block_hashes_broadcast: return
        block_data = block.to_dict()
        self.seen_block_hashes_broadcast.add(block.hash)
        for peer_node in list(self.peers):
            self._send_http_request('POST', peer_node.address, f"/{str(MessageType.NEW_BLOCK)}", json_data=block_data)

    def handle_received_transaction(self, tx_data: dict) -> bool:
        try:
            required_fields = ['sender_address', 'receiver_address', 'amount', 'signature_hex', 'transaction_id']
            if not all(field in tx_data for field in required_fields): return False
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
                if transaction.transaction_id not in self.seen_tx_ids_broadcast:
                     self.broadcast_transaction(transaction)
                return True
            return False
        except Exception: return False

    def handle_received_block(self, block_data: dict) -> bool:
        try:
            received_block = Block.from_dict(block_data)
            if any(b.hash == received_block.hash for b in self.blockchain.chain): return True # Known

            current_chain_length = len(self.blockchain.chain)
            last_block = self.blockchain.last_block # Can be None if chain is empty (not possible with current genesis)

            # Validate block structure and signatures first
            validator_pub_key = USER_PUBLIC_KEYS.get(received_block.validator_address)
            if not validator_pub_key:
                print(f"[{self.self_node.node_id}] Validator PK for {received_block.validator_address} not found for received block {received_block.hash}")
                return False
            if received_block.hash != received_block._calculate_block_hash():
                print(f"[{self.self_node.node_id}] Invalid block hash for received block {received_block.hash}")
                return False
            if not received_block.verify_block_signature(validator_pub_key):
                print(f"[{self.self_node.node_id}] Invalid block signature for received block {received_block.hash}")
                return False
            for tx in received_block.transactions:
                tx_sender_pub_key = USER_PUBLIC_KEYS.get(tx.sender_address)
                if not tx_sender_pub_key or not tx.verify_signature(tx_sender_pub_key):
                    print(f"[{self.self_node.node_id}] Invalid tx {tx.transaction_id} in received block {received_block.hash}")
                    return False

            # Check if it extends the current chain
            if received_block.index == current_chain_length and \
               (last_block and received_block.previous_hash == last_block.hash or current_chain_length == 0 and received_block.previous_hash == "0"): # Handles empty local chain for genesis

                # Validate transactions for state changes against a temporary balance state
                # This ensures the block is valid in terms of fund transfers before committing it
                temp_balances = self.blockchain.balances.copy()
                valid_block_transactions = True
                for tx in received_block.transactions:
                    if tx.asset_id == Blockchain.NATIVE_CURRENCY_SYMBOL:
                        sender_bal = temp_balances.get(tx.sender_address, 0.0)
                        if sender_bal < tx.amount:
                            print(f"[{self.self_node.node_id}] Tx {tx.transaction_id} in received block {received_block.hash} invalidates state (insufficient funds).")
                            valid_block_transactions = False
                            break
                        temp_balances[tx.sender_address] = sender_bal - tx.amount
                        temp_balances[tx.receiver_address] = temp_balances.get(tx.receiver_address, 0.0) + tx.amount

                if not valid_block_transactions:
                    return False # Block contains transactions that would make state invalid

                # If all good, apply to actual state and add to chain
                original_bc_net_interface = self.blockchain.network_interface
                self.blockchain.network_interface = None # Prevent broadcast from internal add_block

                for tx in received_block.transactions: # Apply to actual balances
                    self.blockchain._process_transaction_for_state_changes(tx)
                self.blockchain.chain.append(received_block)

                self.blockchain.network_interface = original_bc_net_interface # Restore

                mined_tx_ids = {tx.transaction_id for tx in received_block.transactions}
                self.blockchain.pending_transactions = [
                    p_tx for p_tx in self.blockchain.pending_transactions if p_tx.transaction_id not in mined_tx_ids
                ]
                print(f"[{self.self_node.node_id}] Added block {received_block.hash} (Index: {received_block.index}) from network.")
                if received_block.hash not in self.seen_block_hashes_broadcast:
                    self.broadcast_block(received_block)
                return True

            # Potential fork or this node is behind
            elif received_block.index >= current_chain_length and not self.syncing_in_progress :
                print(f"[{self.self_node.node_id}] Received block {received_block.hash} (Idx: {received_block.index}) indicates possible fork or being behind. Requesting chain.")
                # Simplistic: sync from first available peer if any.
                # In a real scenario, might sync from the peer that sent this block if that info is available.
                if self.peers: self.request_chain_from_peer(list(self.peers)[0])
                return False
            else:
                return False # Older or irrelevant block
        except Exception as e:
            print(f"[{self.self_node.node_id}] Error processing received block: {e}")
        return False

    def request_chain_from_peer(self, peer_node: Node):
        if self.syncing_in_progress: return
        self.syncing_in_progress = True
        # print(f"[{self.self_node.node_id}] Requesting full chain from peer {peer_node.address}...")
        response_data = self._send_http_request('GET', peer_node.address, f"/{str(MessageType.GET_CHAIN)}")
        if response_data and 'chain' in response_data and 'length' in response_data:
            # print(f"[{self.self_node.node_id}] Received chain of length {response_data['length']} from {peer_node.address}.")
            self.handle_chain_response(response_data['chain'], peer_node)
        # else: print(f"[{self.self_node.node_id}] Failed to get chain from {peer_node.address} or invalid response.")
        self.syncing_in_progress = False

    def handle_chain_response(self, received_chain_dicts: list[dict], from_peer_node: Node):
        # print(f"[{self.self_node.node_id}] Processing chain response from {from_peer_node.address} with {len(received_chain_dicts)} blocks.")
        if not received_chain_dicts: return
        if len(received_chain_dicts) <= len(self.blockchain.chain): return

        prospective_chain = []
        try:
            for block_data in received_chain_dicts: prospective_chain.append(Block.from_dict(block_data))
        except Exception as e:
            print(f"[{self.self_node.node_id}] Error deserializing received chain from {from_peer_node.address}: {e}")
            return

        # Validate the prospective_chain (simplified full validation)
        # This requires creating a temporary Blockchain or having a static validation utility.
        # For now, adapt Blockchain.is_chain_valid logic:
        temp_bc = Blockchain(network_interface=None) # Create a temporary, clean blockchain for validation
        temp_bc.chain = [] # Start with an empty chain for this temp instance.
        temp_bc.balances = {} # Reset balances for temp validation

        # Manually set genesis of temp_bc to match the received chain's genesis for validation to pass
        # This assumes received_chain_dicts[0] is the genesis.
        # And that its validator's pubkey is in USER_PUBLIC_KEYS (might need to ensure this for test/real scenarios)
        if prospective_chain:
            # Critical: Ensure the genesis validator's details for the prospective chain are known
            # For simplicity, if our current chain is just genesis, we accept theirs if it's longer and valid from its own genesis.
            # If we have a longer chain, their genesis MUST match ours.
            if len(self.blockchain.chain) > 1 and prospective_chain[0].hash != self.blockchain.chain[0].hash :
                print(f"[{self.self_node.node_id}] Received chain from {from_peer_node.address} has different genesis. Sync failed.")
                return

            # If our chain is only genesis, we trust the peer's genesis for now if it's valid itself.
            # We need to populate USER_PUBLIC_KEYS for validators in prospective_chain if not already known.
            # This is a simulation limitation. For this test, assume they are known or validation will fail.
            temp_bc.chain.append(prospective_chain[0]) # Add peer's genesis to temp_bc
            # Initialize balances for temp_bc based on its genesis (if it has initial allocation logic)
            # This part is tricky without knowing how prospective_chain[0] allocated initial supply
            # For now, let's assume _create_and_sign_genesis_block in the temp_bc will handle it if we re-init.
            # Or, more simply for validation, just ensure the received genesis is valid on its own.
            genesis_val_pk = USER_PUBLIC_KEYS.get(prospective_chain[0].validator_address)
            if not genesis_val_pk or not prospective_chain[0].verify_block_signature(genesis_val_pk):
                print(f"[{self.self_node.node_id}] Received chain's genesis block from {from_peer_node.address} is invalid. Sync failed.")
                return
            # Simulate initial balance for the temp validation
            temp_bc.balances[prospective_chain[0].validator_address] = self.blockchain.total_supply_epc # Assuming total supply is fixed from one genesis type

            # Validate rest of the prospective chain using temp_bc's context
            valid_so_far = True
            for i in range(1, len(prospective_chain)):
                block_to_validate = prospective_chain[i]
                # Simulate adding to temp_bc to validate against its current state (last block, balances)
                temp_bc.pending_transactions = list(block_to_validate.transactions) # Load txs for this block

                # Temporarily set network_interface to None to avoid broadcasts during validation
                original_temp_net_interface = temp_bc.network_interface
                temp_bc.network_interface = None

                # We can't directly call mine_pending_transactions as it selects a validator.
                # We need to validate the block as if it were received.
                # This means checking its structure, then applying its transactions to temp_bc.balances

                # Structural and signature checks (already done partially in handle_received_block)
                # Re-check here in context of the prospective chain
                prev_b = temp_bc.last_block
                if block_to_validate.previous_hash != prev_b.hash or \
                   block_to_validate.index != len(temp_bc.chain) or \
                   block_to_validate.hash != block_to_validate._calculate_block_hash():
                    valid_so_far = False; break

                val_pk = USER_PUBLIC_KEYS.get(block_to_validate.validator_address)
                if not val_pk or not block_to_validate.verify_block_signature(val_pk):
                    valid_so_far = False; break

                for tx in block_to_validate.transactions:
                    tx_sender_pk = USER_PUBLIC_KEYS.get(tx.sender_address)
                    if not tx_sender_pk or not tx.verify_signature(tx_sender_pk):
                        valid_so_far = False; break
                    # Simulate processing tx for state changes on temp_bc.balances
                    if tx.asset_id == Blockchain.NATIVE_CURRENCY_SYMBOL:
                        s_bal = temp_bc.balances.get(tx.sender_address, 0.0)
                        if s_bal < tx.amount: valid_so_far = False; break
                        temp_bc.balances[tx.sender_address] = s_bal - tx.amount
                        temp_bc.balances[tx.receiver_address] = temp_bc.balances.get(tx.receiver_address, 0.0) + tx.amount
                if not valid_so_far: break

                temp_bc.chain.append(block_to_validate) # Add to temp chain to continue validation
                temp_bc.pending_transactions = [] # Clear for next block
                temp_bc.network_interface = original_temp_net_interface


            if valid_so_far:
                print(f"[{self.self_node.node_id}] Received chain from {from_peer_node.address} is valid and longer. Updating local chain.")
                self.blockchain.chain = prospective_chain
                self.blockchain.balances = temp_bc.balances # Adopt the replayed balances
                self.blockchain.pending_transactions = []
                self.seen_block_hashes_broadcast.clear()
                self.seen_tx_ids_broadcast.clear()
                for block in self.blockchain.chain:
                    self.seen_block_hashes_broadcast.add(block.hash)
                    for tx in block.transactions: self.seen_tx_ids_broadcast.add(tx.transaction_id)
            else:
                print(f"[{self.self_node.node_id}] Received chain from {from_peer_node.address} is invalid. Local chain preserved.")
        # else: # Prospective chain was not valid from the start (e.g. bad genesis)
            # print(f"[{self.self_node.node_id}] Initial validation of received chain from {from_peer_node.address} failed.")


if __name__ == '__main__':
    class DemoBlockchain:
        def __init__(self):
            genesis_validator = Wallet()
            USER_PUBLIC_KEYS[genesis_validator.address] = genesis_validator.get_public_key_hex()
            VALIDATOR_WALLETS[genesis_validator.address] = genesis_validator

            self.chain = []
            self.balances = {} # Init balances
            self.total_supply_epc = 1_000_000.0 # Define total supply for genesis

            genesis_block = Block(0, [], time.time(), "0", genesis_validator.address)
            genesis_block.sign_block(genesis_validator)
            self.chain.append(genesis_block)
            self.balances[genesis_validator.address] = self.total_supply_epc # Allocate in demo

            self.pending_transactions = []
            self.network_interface = None
            # print("DemoBlockchain initialized for Network demo.")

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

        # Minimal _process_transaction_for_state_changes for demo
        def _process_transaction_for_state_changes(self, transaction: Transaction) -> bool:
            if transaction.asset_id == Blockchain.NATIVE_CURRENCY_SYMBOL: # Use class attribute
                sender_balance = self.balances.get(transaction.sender_address, 0.0)
                if sender_balance < transaction.amount: return False
                self.balances[transaction.sender_address] = sender_balance - transaction.amount
                self.balances[transaction.receiver_address] = self.balances.get(transaction.receiver_address, 0.0) + transaction.amount
            return True


    demo_bc = DemoBlockchain()
    import sys
    default_host = "127.0.0.1"
    default_port = 5000

    if len(sys.argv) > 1 and sys.argv[1] == '--help':
        print("Usage: python empower1/network/network.py [host] [port] [seed_node_http_address ...]")
        sys.exit(0)

    host = sys.argv[1] if len(sys.argv) > 1 else default_host
    try: port = int(sys.argv[2]) if len(sys.argv) > 2 else default_port
    except ValueError: port = default_port

    seed_nodes_for_demo = set(sys.argv[3:]) if len(sys.argv) > 3 else set()
    valid_seeds = {sn for sn in seed_nodes_for_demo if sn.startswith("http://") and ":" in sn.split("http://")[1]}

    network_manager = Network(blockchain=demo_bc, host=host, port=port, seed_nodes=valid_seeds)
    demo_bc.network_interface = network_manager

    network_manager.start_server(threaded=True)

    time.sleep(0.5)
    if valid_seeds: network_manager.connect_to_seed_nodes()
    elif len(demo_bc.chain) <=1 and not valid_seeds:
        print(f"[{network_manager.self_node.node_id}] No seed nodes. Started with genesis block.")

    print(f"\nNode {network_manager.self_node.node_id} running. API: {network_manager.self_node.address}")
    print(f"Peers: {network_manager.get_known_peers_addresses()}")

    try:
        while True: time.sleep(15)
    except KeyboardInterrupt: print(f"\nNode {network_manager.self_node.node_id} shutting down.")
