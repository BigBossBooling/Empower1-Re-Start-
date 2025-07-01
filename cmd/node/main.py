import sys
import os
import time
import threading
import json

# Adjust path to import from parent directory (empower1)
current_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(os.path.dirname(current_dir))
sys.path.insert(0, parent_dir)

from empower1.blockchain import Blockchain, USER_PUBLIC_KEYS, VALIDATOR_WALLETS
from empower1.wallet import Wallet
from empower1.transaction import Transaction
from empower1.network.network import Network
from empower1.network.messages import MessageType

# Define native currency symbol, ideally from blockchain or a shared config
NATIVE_CURRENCY_SYMBOL = Blockchain.NATIVE_CURRENCY_SYMBOL # Use the one defined in Blockchain

def print_help():
    print("\nEmPower1 Blockchain Node CLI")
    print(f"Usage: python {os.path.join('cmd', 'node', 'main.py')} [host] [port] [seed_node_http_address...]")
    print(f"Example: python {os.path.join('cmd', 'node', 'main.py')} 127.0.0.1 5000 http://127.0.0.1:5001")
    print("\nCommands during runtime:")
    print(f"  transfer <receiver_addr> <amount> [{NATIVE_CURRENCY_SYMBOL}|asset_id] [metadata_json] - Create & broadcast transaction")
    print("  getbalance <address>                         - Display EPC balance of an address")
    print(f"  faucet <address> <amount>                    - (Test Only) Mint {NATIVE_CURRENCY_SYMBOL} to an address")
    print("  mine                                         - Attempt to mine a new block")
    print("  chain                                        - Display the current blockchain")
    print("  pending                                      - Display pending transactions")
    print("  peers                                        - Display known peers")
    print("  addpeer <http_address>                       - Manually add and connect to a peer")
    print("  mywallet                                     - Display this node's wallet info")
    print("  stake <amount>                               - Register/update stake for this node's wallet")
    print("  help                                         - Show this help message")
    print("  exit                                         - Shutdown the node")
    print("-" * 40)

def main():
    print_help()

    default_host = "127.0.0.1"
    default_port = 5000

    host = sys.argv[1] if len(sys.argv) > 1 else default_host
    try:
        port = int(sys.argv[2]) if len(sys.argv) > 2 else default_port
    except (ValueError, IndexError):
        # print(f"Warning: Invalid port or host not specified, using default {default_host}:{default_port}.")
        host = default_host
        port = default_port

    seed_nodes_str = sys.argv[3:] if len(sys.argv) > 3 else []
    seed_nodes = set()
    for sn_addr in seed_nodes_str:
        if sn_addr.startswith("http://") and ":" in sn_addr.split("http://")[1]:
            seed_nodes.add(sn_addr)
        # else: print(f"Warning: Invalid seed node format '{sn_addr}'. Skipping.")

    node_wallet = Wallet()
    print(f"\nInitializing Node Wallet:")
    print(f"  Address: {node_wallet.address}")
    # print(f"  Public Key (short): {node_wallet.get_public_key_hex()[:20]}...")
    USER_PUBLIC_KEYS[node_wallet.address] = node_wallet.get_public_key_hex()

    blockchain = Blockchain()
    network_manager = Network(blockchain=blockchain, host=host, port=port, node_id=node_wallet.address, seed_nodes=seed_nodes)
    blockchain.network_interface = network_manager # Ensure blockchain can call network

    network_manager.start_server(threaded=True)
    time.sleep(0.5)

    if seed_nodes:
        # print(f"Attempting to connect to seed nodes: {seed_nodes}")
        network_manager.connect_to_seed_nodes()
    elif len(blockchain.chain) <=1 and not seed_nodes:
         print(f"[{network_manager.self_node.node_id}] No seed nodes provided. Node started with its genesis block.")


    print(f"\nNode {network_manager.self_node.node_id} listening on {network_manager.self_node.address}")
    # print(f"API Endpoints: /ping, /{str(MessageType.GET_CHAIN)}, /{str(MessageType.GET_PEERS)}, etc.")

    running = True
    while running:
        try:
            cmd_input_str = input(f"Node {port}> ").strip()
            if not cmd_input_str: continue
            cmd_input = cmd_input_str.split()
            command = cmd_input[0].lower()

            if command == "exit":
                running = False
                print("Shutting down node...")
            elif command == "help": print_help()
            elif command == "mywallet":
                print(f"  Address: {node_wallet.address}")
                print(f"  Public Key: {node_wallet.get_public_key_hex()}")
                print(f"  EPC Balance: {blockchain.balances.get(node_wallet.address, 0.0)}")
            elif command == "stake":
                if len(cmd_input) > 1:
                    try:
                        stake_amount = float(cmd_input[1])
                        if stake_amount > 0:
                            blockchain.register_validator_wallet(node_wallet, stake_amount)
                        else: print("Stake amount must be positive.")
                    except ValueError: print("Invalid stake amount.")
                else: print("Usage: stake <amount>")

            elif command == "transfer":
                if len(cmd_input) >= 3:
                    receiver_addr = cmd_input[1]
                    try:
                        amount = float(cmd_input[2])
                        asset_id = cmd_input[3] if len(cmd_input) > 3 else NATIVE_CURRENCY_SYMBOL
                        metadata_str = " ".join(cmd_input[4:]) if len(cmd_input) > 4 else "{}"
                        metadata = json.loads(metadata_str) if metadata_str else {}

                        if amount <=0:
                            print("Transfer amount must be positive.")
                            continue

                        print(f"Creating transaction: {amount} {asset_id} from {node_wallet.address[:10]}... to {receiver_addr[:10]}...")
                        new_tx = Transaction(
                            sender_address=node_wallet.address,
                            receiver_address=receiver_addr,
                            amount=amount,
                            asset_id=asset_id,
                            metadata=metadata
                        )
                        new_tx.sign(node_wallet)

                        if blockchain.add_transaction(new_tx, node_wallet.get_public_key_hex()):
                             print(f"Transaction {new_tx.transaction_id[:10]}... submitted and broadcasted.")
                        else:
                             print(f"Failed to submit transaction {new_tx.transaction_id[:10]}... (check balance/logs).")
                    except ValueError: print("Invalid amount.")
                    except json.JSONDecodeError: print("Invalid metadata JSON string.")
                else: print(f"Usage: transfer <receiver_addr> <amount> [{NATIVE_CURRENCY_SYMBOL}|asset_id] [metadata_json]")

            elif command == "getbalance":
                if len(cmd_input) > 1:
                    addr_to_check = cmd_input[1]
                    balance = blockchain.balances.get(addr_to_check, 0.0)
                    print(f"Balance of {addr_to_check}: {balance} {NATIVE_CURRENCY_SYMBOL}")
                else: print("Usage: getbalance <address>")

            elif command == "faucet": # Test only
                if len(cmd_input) > 2:
                    try:
                        faucet_addr = cmd_input[1]
                        faucet_amount = float(cmd_input[2])
                        if faucet_amount <=0:
                            print("Faucet amount must be positive.")
                            continue

                        # Directly manipulate balances and total supply (ONLY FOR TESTING)
                        blockchain.balances[faucet_addr] = blockchain.balances.get(faucet_addr, 0.0) + faucet_amount
                        blockchain.total_supply_epc += faucet_amount
                        print(f"Added {faucet_amount} {NATIVE_CURRENCY_SYMBOL} to {faucet_addr}. New total supply: {blockchain.total_supply_epc}")
                        # Ensure USER_PUBLIC_KEYS has this address if it's new, for other operations
                        if faucet_addr not in USER_PUBLIC_KEYS and faucet_addr.startswith("Emp1"): # Basic check
                             print(f"Warning: Address {faucet_addr} may not have a known public key for sending transactions from it later.")
                    except ValueError: print("Invalid amount for faucet.")
                else: print(f"Usage: faucet <address> <amount>")

            elif command == "mine":
                if blockchain.pending_transactions:
                    print("Attempting to mine a new block...")
                    mined_block = blockchain.mine_pending_transactions()
                    if mined_block:
                        print(f"Mined Block #{mined_block.index} by {mined_block.validator_address[:10]}... Hash: {mined_block.hash[:10]}...")
                    # else: print("Mining did not result in a new block by this node (check logs).") # mine_pending_transactions prints reasons
                else: print("No pending transactions to mine.")

            elif command == "chain":
                print("\nCurrent Blockchain:")
                for i, block in enumerate(blockchain.chain): print(f"  {block!r}")
                print("-" * 40)

            elif command == "pending":
                print("\nPending Transactions:")
                if blockchain.pending_transactions:
                    for tx in blockchain.pending_transactions: print(f"  {tx!r}")
                else: print("  No pending transactions.")
                print("-" * 40)

            elif command == "peers":
                print("\nKnown Peers:")
                peers_list = network_manager.get_known_peers_addresses()
                if peers_list:
                    for p_addr in peers_list: print(f"  - {p_addr}")
                else: print("  No known peers.")
                print("-" * 40)

            elif command == "addpeer":
                if len(cmd_input) > 1:
                    peer_addr_to_add = cmd_input[1]
                    if network_manager.connect_to_peer(peer_addr_to_add):
                        print(f"Successfully initiated connection with {peer_addr_to_add}.")
                    # else: print(f"Failed to connect to or add peer {peer_addr_to_add}.") # connect_to_peer prints errors
                else: print("Usage: addpeer <http_address_of_peer>")
            else:
                print(f"Unknown command: {command}. Type 'help' for commands.")
        except EOFError: running = False; print("\nShutting down (EOF)...")
        except KeyboardInterrupt: running = False; print("\nShutting down (Ctrl+C)...")
        # except Exception as e: print(f"An unexpected error occurred: {e}") # Keep commented for now

    print(f"Node at {host}:{port} stopped.")

if __name__ == "__main__":
    main()
