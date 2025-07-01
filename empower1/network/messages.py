# Defines the structure and types for network messages.
# For a simple HTTP/JSON based approach, these might just be conventions for JSON payloads.

from enum import Enum
import json
# from empower1.transaction import Transaction # For typing, if needed
# from empower1.block import Block # For typing, if needed

class MessageType(Enum):
    """
    Defines the types of messages that can be sent between nodes.
    The string value can be used in the 'type' field of a JSON message.
    """
    # Requests
    GET_CHAIN = "GET_CHAIN"                 # Request the full blockchain from a peer
    GET_PEERS = "GET_PEERS"                 # Request a list of known peers from a node
    GET_BLOCKS = "GET_BLOCKS"               # Request specific blocks (e.g., by index range or hashes)

    # Responses / Information
    CHAIN_RESPONSE = "CHAIN_RESPONSE"       # Response containing (part of) the blockchain
    PEERS_RESPONSE = "PEERS_RESPONSE"       # Response containing a list of peer addresses
    BLOCKS_RESPONSE = "BLOCKS_RESPONSE"     # Response containing requested blocks
    STATUS_RESPONSE = "STATUS_RESPONSE"     # Generic status/ack response
    ERROR_RESPONSE = "ERROR_RESPONSE"       # Response indicating an error occurred

    # Broadcasts / Announcements
    NEW_TRANSACTION = "NEW_TRANSACTION"     # Announce a new transaction
    NEW_BLOCK = "NEW_BLOCK"                 # Announce a newly mined/validated block
    NEW_PEER_ANNOUNCE = "NEW_PEER_ANNOUNCE" # Announce self to a peer (could be part of /add_peer logic)

    def __str__(self):
        return self.value

# --- Helper functions for creating standardized message payloads ---
# These are conceptual. In practice, the Flask route handlers might construct these directly,
# or a more formal serialization/deserialization layer (like Marshmallow or Pydantic) might be used.

def create_message_payload(message_type: MessageType, data: dict = None, error_message: str = None) -> dict:
    """
    Creates a standardized JSON payload for a network message.
    Args:
        message_type (MessageType): The type of the message.
        data (dict, optional): The main data payload of the message.
        error_message (str, optional): An error message, if this is an ERROR_RESPONSE.
    Returns:
        dict: A dictionary ready to be JSON serialized and sent.
    """
    payload = {
        "type": str(message_type) # Use the string value of the enum
    }
    if data is not None:
        payload["data"] = data
    if error_message is not None and message_type == MessageType.ERROR_RESPONSE:
        payload["error"] = error_message

    # Could add timestamp, sender_node_id etc. to all messages if desired
    # import time
    # payload["timestamp"] = time.time()
    return payload

# --- Example Payloads (for illustration and testing) ---

def example_get_chain_payload():
    return create_message_payload(MessageType.GET_CHAIN)

def example_chain_response_payload(chain_data_list: list, length: int):
    return create_message_payload(MessageType.CHAIN_RESPONSE, data={"chain": chain_data_list, "length": length})

def example_new_transaction_payload(transaction_dict: dict):
    # transaction_dict should be the result of transaction.to_dict()
    # It might also need sender_public_key_hex if not included in to_dict() and required by receiver.
    return create_message_payload(MessageType.NEW_TRANSACTION, data=transaction_dict)

def example_new_block_payload(block_dict: dict): # Assuming block.to_dict()
    return create_message_payload(MessageType.NEW_BLOCK, data=block_dict)

def example_get_peers_payload():
    return create_message_payload(MessageType.GET_PEERS)

def example_peers_response_payload(peer_address_list: list[str]):
    return create_message_payload(MessageType.PEERS_RESPONSE, data={"peers": peer_address_list})

def example_error_response_payload(error_msg: str):
    return create_message_payload(MessageType.ERROR_RESPONSE, error_message=error_msg)


if __name__ == '__main__':
    print("--- Network Message Types ---")
    for msg_type in MessageType:
        print(f"- {msg_type.name}: \"{str(msg_type)}\"")

    print("\n--- Example Message Payloads ---")

    print("\n1. GET_CHAIN Request:")
    print(json.dumps(example_get_chain_payload(), indent=2))

    print("\n2. CHAIN_RESPONSE (partial):")
    # Simulate chain data (e.g., list of block hashes or simplified block dicts)
    demo_chain_part = [{"hash": "hash1...", "index": 0}, {"hash": "hash2...", "index": 1}]
    print(json.dumps(example_chain_response_payload(demo_chain_part, 2), indent=2))

    print("\n3. NEW_TRANSACTION Announcement:")
    # Simulate a transaction dictionary
    demo_tx_dict = {
        "transaction_id": "txid_abc123...",
        "sender_address": "Emp1_Alice_Addr...",
        "receiver_address": "Emp1_Bob_Addr...",
        "amount": 10.5,
        "signature_hex": "sig_hex_def456..."
        # Potentially sender_public_key_hex if needed by network protocol
    }
    print(json.dumps(example_new_transaction_payload(demo_tx_dict), indent=2))

    print("\n4. NEW_BLOCK Announcement:")
    # Simulate a block dictionary
    demo_block_dict = {
        "hash": "blockhash_xyz789...",
        "index": 5,
        "validator_address": "Emp1_Validator_Addr...",
        "signature_hex": "block_sig_ghi012...",
        "transactions": [demo_tx_dict] # List of transaction dicts
    }
    print(json.dumps(example_new_block_payload(demo_block_dict), indent=2))

    print("\n5. GET_PEERS Request:")
    print(json.dumps(example_get_peers_payload(), indent=2))

    print("\n6. PEERS_RESPONSE:")
    demo_peer_list = ["http://10.0.0.2:5000", "http://10.0.0.3:5001"]
    print(json.dumps(example_peers_response_payload(demo_peer_list), indent=2))

    print("\n7. ERROR_RESPONSE:")
    print(json.dumps(example_error_response_payload("Requested block not found."), indent=2))

    print("\nMessage definitions demo complete.")
