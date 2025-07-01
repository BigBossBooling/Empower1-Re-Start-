import pytest
from empower1.network.node import Node
import time

def test_node_creation_basic():
    """Test basic Node creation with host and port."""
    host, port = "127.0.0.1", 5000
    node = Node(host=host, port=port)
    assert node.host == host
    assert node.port == port
    expected_address = f"http://{host}:{port}"
    assert node.address == expected_address
    assert node.node_id == expected_address # Default node_id
    assert isinstance(node.last_seen, float)

def test_node_creation_with_node_id():
    """Test Node creation with a custom node_id."""
    host, port, node_id = "localhost", 5001, "MyCustomNode123"
    node = Node(host=host, port=port, node_id=node_id)
    assert node.host == host
    assert node.port == port
    assert node.address == f"http://{host}:{port}"
    assert node.node_id == node_id

def test_node_to_dict():
    """Test the to_dict method."""
    host, port = "192.168.0.1", 8080
    node = Node(host=host, port=port)
    node_dict = node.to_dict()

    expected_keys = ["node_id", "address", "host", "port", "last_seen"]
    assert all(key in node_dict for key in expected_keys)
    assert node_dict["address"] == node.address
    assert node_dict["host"] == host
    assert node_dict["port"] == port
    assert node_dict["node_id"] == node.address # Default node_id

def test_node_from_address_string_valid():
    """Test creating a Node from a valid address string."""
    address_str = "http://10.0.0.5:5005"
    node = Node.from_address_string(address_str)
    assert node.address == address_str
    assert node.host == "10.0.0.5"
    assert node.port == 5005
    assert node.node_id == address_str

@pytest.mark.parametrize("invalid_address", [
    "127.0.0.1:5000",           # Missing http://
    "http://127.0.0.1",         # Missing port
    "http://:5000",             # Missing host
    "http://localhost:badport", # Invalid port number
    "ftp://localhost:5000"      # Invalid scheme
])
def test_node_from_address_string_invalid(invalid_address):
    """Test creating a Node from invalid address strings."""
    with pytest.raises(ValueError):
        Node.from_address_string(invalid_address)

def test_node_equality():
    """Test Node equality (__eq__ method)."""
    node1_a = Node("127.0.0.1", 5000)
    node1_b = Node("127.0.0.1", 5000) # Same address
    node2 = Node("127.0.0.1", 5001)   # Different port
    node3 = Node("localhost", 5000)   # Different host (potentially same IP but distinct for Node logic)

    assert node1_a == node1_b
    assert node1_a != node2
    assert node1_a != node3 # "127.0.0.1" vs "localhost" might resolve same but are different strings

    # Test equality with address string
    assert node1_a == "http://127.0.0.1:5000"
    assert node1_a != "http://127.0.0.1:5001"
    assert node1_a != 123 # Not equal to other types

def test_node_hash():
    """Test Node hashing (__hash__ method) for use in sets/dicts."""
    node1_a = Node("127.0.0.1", 5000)
    node1_b = Node("127.0.0.1", 5000)
    node2 = Node("127.0.0.1", 5001)

    assert hash(node1_a) == hash(node1_b)
    assert hash(node1_a) != hash(node2)

    node_set = {node1_a, node1_b, node2}
    assert len(node_set) == 2 # node1_a and node1_b should be treated as the same element

def test_node_last_seen_update():
    """Test that last_seen timestamp is set."""
    node = Node("127.0.0.1", 5000)
    initial_time = node.last_seen
    time.sleep(0.01) # Ensure time progresses
    # last_seen is set on init, not typically updated by itself unless a method does so.
    # This test just confirms it's set. If methods to update it were added, they'd be tested here.
    assert node.last_seen == initial_time
    assert isinstance(node.last_seen, float)

# To run: pytest tests/network/test_node.py
# or just pytest from root if __init__.py files are correctly set up.
