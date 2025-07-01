import sys
import os
import time
import threading
import json # Added for metadata parsing

# Adjust path to import from parent directory (empower1)
current_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(os.path.dirname(current_dir)) # This should be the project root
sys.path.insert(0, parent_dir)

from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS, VALIDATOR_WALLETS
from empower1.wallet import Wallet
from empower1.transaction import Transaction
from empower1.network.network import Network
from empower1.network.messages import MessageType # For printing endpoint paths

def print_help():
    print("\nEmPower1 Blockchain Node CLI")
    print("Usage: python cmd/node/main.py [host] [port] [seed_node_http_address...]")
    print("Example: python cmd/node/main.py 127.0.0.1 5000 http://127.0.0.1:5001")
    print("\nCommands during runtime:")
    print("  tx <receiver_address> <amount> <asset_id> [metadata_json_string] - Create & broadcast transaction")
    print("  mine                                     - Mine a new block (if validator & pending txs)")
    print("  chain                                    - Display the current blockchain")
    print("  pending                                  - Display pending transactions")
    print("  peers                                    - Display known peers")
    print("  addpeer <http_address>                   - Manually add and connect to a peer")
    print("  mywallet                                 - Display this node's wallet info")
    print("  stake <amount>                           - Register/update stake for this node's wallet to be a validator")
    print("  help                                     - Show this help message")
    print("  exit                                     - Shutdown the node")
    print("-" * 40)

def main():
    print_help()

    default_host = "127.0.0.1"
    default_port = 5000

    host = sys.argv[1] if len(sys.argv) > 1 else default_host
    try:
        port = int(sys.argv[2]) if len(sys.argv) > 2 else default_port
    except (ValueError, IndexError):
        print(f"Warning: Invalid port or host not specified, using default {default_host}:{default_port}.")
        host = default_host # Ensure host is also default if port parsing fails early
        port = default_port

    seed_nodes_str = sys.argv[3:] if len(sys.argv) > 3 else []
    seed_nodes = set()
    for sn_addr in seed_nodes_str:
        if sn_addr.startswith("http://") and ":" in sn_addr.split("http://")[1]:
            seed_nodes.add(sn_addr)
        else:
            print(f"Warning: Invalid seed node format '{sn_addr}'. Skipping.")

    # Each node needs its own wallet for identity, creating transactions, or validating
    node_wallet = Wallet()
    print(f"\nInitializing Node Wallet:")
    print(f"  Address: {node_wallet.address}")
    print(f"  Public Key (short): {node_wallet.get_public_key_hex()[:20]}...")
    # For this node's transactions to be verifiable by others, its public key needs to be known.
    # And for it to validate blocks, its wallet object needs to be in VALIDATOR_WALLETS.
    # This is typically handled by registration or a known identity system.
    # For now, we add it to the global USER_PUBLIC_KEYS for simplicity in this demo.
    USER_PUBLIC_KEYS[node_wallet.address] = node_wallet.get_public_key_hex()


    # Initialize Blockchain and Network components
    # The Blockchain instance will now be passed the network_manager for broadcasting
    blockchain = Blockchain() # network_interface will be set by Network constructor
    network_manager = Network(blockchain=blockchain, host=host, port=port, node_id=node_wallet.address, seed_nodes=seed_nodes)

    # Start the network server in a separate thread
    network_manager.start_server(threaded=True)
    time.sleep(0.5) # Give server a moment to start

    # Connect to seed nodes
    if seed_nodes:
        print(f"Attempting to connect to seed nodes: {seed_nodes}")
        network_manager.connect_to_seed_nodes()

    print(f"\nNode {network_manager.self_node.node_id} listening on {network_manager.self_node.address}")
    print(f"API Endpoints: /ping, /{str(MessageType.GET_CHAIN)}, /{str(MessageType.GET_PEERS)}, /{str(MessageType.NEW_PEER_ANNOUNCE)}, etc.")

    running = True
    while running:
        try:
            cmd_input = input(f"Node {port}> ").strip().split()
            if not cmd_input:
                continue

            command = cmd_input[0].lower()

            if command == "exit":
                running = False
                print("Shutting down node...")
            elif command == "help":
                print_help()
            elif command == "mywallet":
                print(f"Node Wallet Address: {node_wallet.address}")
                print(f"Node Public Key (hex, uncompressed): {node_wallet.get_public_key_hex()}")
            elif command == "stake":
                if len(cmd_input) > 1:
                    try:
                        stake_amount = float(cmd_input[1])
                        if stake_amount > 0:
                            blockchain.register_validator_wallet(node_wallet, stake_amount)
                            # Note: VALIDATOR_WALLETS and USER_PUBLIC_KEYS are global in blockchain.py for now
                            # This makes the current node a potential validator if selected.
                        else:
                            print("Stake amount must be positive.")
                    except ValueError:
                        print("Invalid stake amount.")
                else:
                    print("Usage: stake <amount>")
            elif command == "tx":
                if len(cmd_input) >= 4:
                    receiver_addr = cmd_input[1]
                    try:
                        amount = float(cmd_input[2])
                        asset_id = cmd_input[3]
                        metadata_str = " ".join(cmd_input[4:]) if len(cmd_input) > 4 else "{}"
                        metadata = json.loads(metadata_str) if metadata_str else {}

                        new_tx = Transaction(
                            sender_address=node_wallet.address,
                            receiver_address=receiver_addr,
                            amount=amount,
                            asset_id=asset_id,
                            metadata=metadata
                        )
                        new_tx.sign(node_wallet)

                        # Add to local blockchain's pending pool (this will also verify and broadcast)
                        # The add_transaction method in Blockchain now takes sender_public_key_hex
                        if blockchain.add_transaction(new_tx, node_wallet.get_public_key_hex()):
                             print(f"Transaction {new_tx.transaction_id} created and broadcasted.")
                        else:
                             print(f"Failed to create or add transaction {new_tx.transaction_id}.")

                    except ValueError:
                        print("Invalid amount.")
                    except json.JSONDecodeError:
                        print("Invalid metadata JSON string.")
                else:
                    print("Usage: tx <receiver_address> <amount> <asset_id> [metadata_json_string]")

            elif command == "mine":
                # Node attempts to mine. Blockchain logic will determine if this node's
                # validator identity is selected by ValidatorManager.
                if blockchain.pending_transactions:
                    print("Attempting to mine a new block (if selected as validator)...")
                    mined_block = blockchain.mine_pending_transactions() # No argument needed now
                    if mined_block:
                        # This means this node *was* selected and successfully mined.
                        print(f"Successfully mined Block #{mined_block.index} with hash {mined_block.hash[:10]}... by {mined_block.validator_address}")
                    else:
                        # This could be due to no pending tx, no active validator selected,
                        # or this node not being the selected one.
                        # The mine_pending_transactions method prints more specific reasons.
                        print("Mining process did not result in a new block by this node.")
                else:
                    print("No pending transactions to mine.")

            elif command == "chain":
                print("\nCurrent Blockchain:")
                for i, block in enumerate(blockchain.chain):
                    print(f"Block {i}: {block!r}") # Use repr for detailed block info
                    # for tx_idx, tx in enumerate(block.transactions):
                    #     print(f"  Tx {tx_idx}: {tx!r}")
                print("-" * 40)

            elif command == "pending":
                print("\nPending Transactions:")
                if blockchain.pending_transactions:
                    for tx in blockchain.pending_transactions:
                        print(f"  {tx!r}")
                else:
                    print("  No pending transactions.")
                print("-" * 40)

            elif command == "peers":
                print("\nKnown Peers:")
                peers_list = network_manager.get_known_peers_addresses()
                if peers_list:
                    for p_addr in peers_list:
                        print(f"  - {p_addr}")
                else:
                    print("  No known peers.")
                print("-" * 40)

            elif command == "addpeer":
                if len(cmd_input) > 1:
                    peer_addr_to_add = cmd_input[1]
                    if network_manager.connect_to_peer(peer_addr_to_add):
                        print(f"Successfully initiated connection with {peer_addr_to_add}.")
                    else:
                        print(f"Failed to connect to or add peer {peer_addr_to_add}.")
                else:
                    print("Usage: addpeer <http_address_of_peer>")

            else:
                print(f"Unknown command: {command}. Type 'help' for commands.")

        except EOFError: # Handle Ctrl+D
            running = False
            print("\nShutting down node (EOF)...")
        except KeyboardInterrupt: # Handle Ctrl+C
            running = False
            print("\nShutting down node (KeyboardInterrupt)...")
        except Exception as e:
            print(f"An unexpected error occurred: {e}")
            # import traceback
            # traceback.print_exc()


    print(f"Node at {host}:{port} stopped.")

if __name__ == "__main__":
    main()
