import pytest
from unittest.mock import patch, MagicMock
import time
import requests # Added for requests.exceptions

from empower1.network.network import Network
from empower1.network.messages import MessageType # Added
from empower1.block import Block # Added for test_broadcast_block
from empower1.network.node import Node
from empower1.blockchain import Blockchain # Real Blockchain, but its methods might be mocked if they interact heavily
from empower1.transaction import Transaction # For creating dummy transactions
from empower1.wallet import Wallet # For creating wallets for dummy transactions

# Mock Blockchain for tests that don't need its full functionality
@pytest.fixture
def mock_blockchain():
    bc = MagicMock(spec=Blockchain)
    bc.chain = [] # Mock attribute
    bc.pending_transactions = []
    # Mock methods that might be called by Network handlers
    bc.add_transaction = MagicMock(return_value=True)
    bc.last_block = MagicMock()
    bc.last_block.hash = "mock_genesis_hash"
    # If Network calls blockchain.add_block or similar, mock that too.
    return bc

@pytest.fixture
def network_node(mock_blockchain):
    """A Network instance for testing."""
    # Clear USER_PUBLIC_KEYS and VALIDATOR_WALLETS before each test if they are modified globally
    from empower1.blockchain import USER_PUBLIC_KEYS, VALIDATOR_WALLETS
    USER_PUBLIC_KEYS.clear()
    VALIDATOR_WALLETS.clear()

    # The Blockchain instance within Network needs to be initialized correctly,
    # especially its own genesis block and validator, because Network's _configure_routes
    # might indirectly access parts of it (e.g. through block.to_dict())
    # For simplicity, we can use a real Blockchain instance here, assuming its __init__ is stable.
    real_bc_for_network = Blockchain() # This creates its own genesis validator and populates globals

    # We pass this real_bc to Network, but some test assertions might use mock_blockchain for verifying calls.
    # This is a bit mixed; ideally, Network would not rely on global USER_PUBLIC_KEYS etc. directly for its operations.
    # For now, we proceed with this structure.

    return Network(blockchain=real_bc_for_network, host="127.0.0.1", port=5000, seed_nodes=None)

@pytest.fixture
def sample_node1():
    return Node(host="127.0.0.1", port=5001)

@pytest.fixture
def sample_node2():
    return Node(host="127.0.0.1", port=5002)


def test_network_initialization(network_node):
    """Test Network class initialization."""
    assert network_node.self_node.host == "127.0.0.1"
    assert network_node.self_node.port == 5000
    assert network_node.self_node.address == "http://127.0.0.1:5000"
    assert network_node.app is not None # Flask app should be initialized
    assert len(network_node.peers) == 0
    assert network_node.blockchain is not None

def test_add_peer(network_node, sample_node1, sample_node2):
    """Test adding peers."""
    assert network_node.add_peer(sample_node1) is True # New peer
    assert sample_node1 in network_node.peers
    assert network_node.add_peer(sample_node1) is False # Already known

    assert network_node.add_peer(sample_node2) is True # Another new peer
    assert sample_node2 in network_node.peers
    assert len(network_node.peers) == 2

    # Try adding self_node (should be ignored)
    assert network_node.add_peer(network_node.self_node) is False
    assert len(network_node.peers) == 2


@patch('requests.get')
def test_connect_to_peer_success(mock_requests_get, network_node, sample_node1):
    """Test successful connection to a peer."""
    # Mock the response from the peer's /ping endpoint
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"message": "pong", "node_id": sample_node1.node_id, "address": sample_node1.address}
    mock_requests_get.return_value = mock_response

    # Mock request_peers_from to prevent further network calls in this isolated test
    with patch.object(network_node, 'request_peers_from', MagicMock()) as mock_req_peers:
        assert network_node.connect_to_peer(sample_node1.address) is True
        assert sample_node1 in network_node.peers
        mock_requests_get.assert_called_once_with(f"{sample_node1.address}/ping", timeout=2)
        mock_req_peers.assert_called_once_with(sample_node1) # Ensure it tries to get more peers

@patch('requests.get')
def test_connect_to_peer_failure_ping(mock_requests_get, network_node, sample_node1):
    """Test failed connection due to ping failure."""
    mock_requests_get.side_effect = requests.exceptions.ConnectionError("Test connection error")

    assert network_node.connect_to_peer(sample_node1.address) is False
    assert sample_node1 not in network_node.peers

@patch('requests.get')
def test_connect_to_peer_failure_bad_status(mock_requests_get, network_node, sample_node1):
    """Test failed connection due to non-200 status from ping."""
    mock_response = MagicMock()
    mock_response.status_code = 404
    mock_response.text = "Not Found"
    mock_requests_get.return_value = mock_response

    assert network_node.connect_to_peer(sample_node1.address) is False
    assert sample_node1 not in network_node.peers

@patch.object(Network, '_send_http_request') # Mock the internal HTTP sending method
def test_request_peers_from_success(mock_send_req, network_node, sample_node1, sample_node2):
    """Test successfully requesting and processing peers from another node."""
    # sample_node1 is the peer we are requesting from
    # sample_node2 is a new peer that sample_node1 knows about

    # Simulate sample_node1 returning sample_node2 in its /GET_PEERS response
    mock_send_req.return_value = {"peers": [sample_node2.address]}

    # We also need to mock connect_to_peer for the newly discovered peer (sample_node2)
    # to avoid actual network calls when processing the response.
    with patch.object(network_node, 'connect_to_peer', MagicMock(return_value=True)) as mock_connect_new_peer:
        network_node.request_peers_from(sample_node1)

        mock_send_req.assert_called_once_with('GET', sample_node1.address, f"/{str(MessageType.GET_PEERS)}")
        # Check if connect_to_peer was called for sample_node2
        mock_connect_new_peer.assert_called_once_with(sample_node2.address)

@patch.object(Network, '_send_http_request')
def test_broadcast_transaction(mock_send_req, network_node, sample_node1, sample_node2):
    """Test broadcasting a transaction."""
    # Add some peers
    network_node.peers.add(sample_node1)
    network_node.peers.add(sample_node2)

    # Create a dummy transaction
    dummy_wallet = Wallet()
    tx = Transaction(dummy_wallet.address, "receiver_addr", 1.0)
    tx.sign(dummy_wallet) # Sign it
    tx_data = tx.to_dict()

    network_node.broadcast_transaction(tx)

    # Check that _send_http_request was called for each peer
    assert mock_send_req.call_count == 2
    calls = mock_send_req.call_args_list
    # Note: order of calls to peers (from a set) is not guaranteed
    expected_calls_args = [
        (('POST', sample_node1.address, f"/{str(MessageType.NEW_TRANSACTION)}"), {'json_data': tx_data}),
        (('POST', sample_node2.address, f"/{str(MessageType.NEW_TRANSACTION)}"), {'json_data': tx_data}),
    ]

    # Check if both expected calls were made, regardless of order
    # This is a bit more complex to assert correctly for unordered calls from a set.
    # A simpler check for now:
    for call_args in calls:
        method, address, endpoint = call_args[0]
        data = call_args[1]['json_data']
        assert method == 'POST'
        assert endpoint == f"/{str(MessageType.NEW_TRANSACTION)}"
        assert data == tx_data
        assert address in [sample_node1.address, sample_node2.address]


@patch.object(Network, '_send_http_request')
def test_broadcast_block(mock_send_req, network_node, sample_node1):
    """Test broadcasting a block."""
    network_node.peers.add(sample_node1)

    dummy_validator_wallet = Wallet()
    block = Block(0, [], time.time(), "0", dummy_validator_wallet.address) # Minimal block
    block.sign_block(dummy_validator_wallet) # Sign it
    block_data = block.to_dict()

    network_node.broadcast_block(block)

    mock_send_req.assert_called_once_with(
        'POST', sample_node1.address, f"/{str(MessageType.NEW_BLOCK)}", json_data=block_data
    )

# --- Flask Route Tests (Basic examples using test_client) ---
# These test the Flask app's routing and basic request/response handling.

def test_ping_endpoint(network_node):
    """Test the /ping endpoint."""
    client = network_node.app.test_client()
    response = client.get('/ping')
    assert response.status_code == 200
    json_data = response.get_json()
    assert json_data["message"] == "pong"
    assert json_data["node_id"] == network_node.self_node.node_id
    assert json_data["address"] == network_node.self_node.address

def test_get_peers_endpoint(network_node, sample_node1):
    """Test the /GET_PEERS endpoint."""
    network_node.peers.add(sample_node1)
    client = network_node.app.test_client()
    response = client.get(f"/{str(MessageType.GET_PEERS)}")
    assert response.status_code == 200
    json_data = response.get_json()
    assert network_node.self_node.node_id in json_data["node_id"] # Check if node_id is part of response
    assert sample_node1.address in json_data["peers"]
    assert len(json_data["peers"]) == 1


def test_new_peer_announce_endpoint_success(network_node, sample_node1):
    """Test successful peer announcement via /NEW_PEER_ANNOUNCE."""
    client = network_node.app.test_client()
    # Mock request_peers_from to prevent further calls during this specific test
    with patch.object(network_node, 'request_peers_from', MagicMock()) as mock_req_peers:
        response = client.post(f"/{str(MessageType.NEW_PEER_ANNOUNCE)}", json={"address": sample_node1.address})
        assert response.status_code == 201 # Created
        json_data = response.get_json()
        assert json_data["message"] == "Peer added and their peers requested"
        assert sample_node1 in network_node.peers
        mock_req_peers.assert_called_once() # Check that it tried to get peers from new peer

def test_new_peer_announce_endpoint_invalid(network_node):
    """Test /NEW_PEER_ANNOUNCE with invalid data."""
    client = network_node.app.test_client()
    response = client.post(f"/{str(MessageType.NEW_PEER_ANNOUNCE)}", json={"addr": "bad_format"}) # Wrong key
    assert response.status_code == 400
    response = client.post(f"/{str(MessageType.NEW_PEER_ANNOUNCE)}", json={"address": "not_an_http_address"})
    assert response.status_code == 400


# Placeholder tests for transaction/block receiving endpoints
# Full tests for these would involve mocking blockchain interactions
def test_new_transaction_endpoint_placeholder(network_node):
    client = network_node.app.test_client()
    dummy_tx_data = {"sender_address": "A", "receiver_address": "B", "amount": 1, "signature_hex":"sig"} # minimal
    with patch.object(network_node, 'handle_received_transaction', MagicMock(return_value=True)) as mock_handler:
        response = client.post(f"/{str(MessageType.NEW_TRANSACTION)}", json=dummy_tx_data)
        assert response.status_code == 201 # Assuming handler returns True
        mock_handler.assert_called_once_with(dummy_tx_data)

def test_new_block_endpoint_placeholder(network_node):
    client = network_node.app.test_client()
    dummy_block_data = {"hash": "blockhash", "index":1} # minimal
    with patch.object(network_node, 'handle_received_block', MagicMock(return_value=True)) as mock_handler:
        response = client.post(f"/{str(MessageType.NEW_BLOCK)}", json=dummy_block_data)
        assert response.status_code == 201
        mock_handler.assert_called_once_with(dummy_block_data)

# To run: pytest tests/network/test_network.py
# (Ensure empower1 is in PYTHONPATH or run from project root)
