# Defines the Node identity for network interactions.
# For now, this is a simple placeholder.
# A node's identity could be tied to a Wallet for signing network messages,
# or just be its network address (host:port).

import time

class Node:
    """
    Represents a node in the EmPower1 network.
    For this initial version, a node is primarily identified by its network address.
    It might also have a unique ID or be associated with a Wallet for more advanced features.
    """
    def __init__(self, host: str, port: int, node_id: str = None):
        """
        Initializes a Node.
        Args:
            host (str): The hostname or IP address of the node.
            port (int): The port number the node is listening on.
            node_id (str, optional): A unique identifier for the node.
                                     If None, it can be derived from host:port or a UUID.
        """
        self.host = host
        self.port = port
        self.address = f"http://{self.host}:{self.port}" # Full address for HTTP communication

        if node_id:
            self.node_id = node_id
        else:
            # Simple default node_id based on address. Could use UUID for more uniqueness.
            # import uuid
            # self.node_id = str(uuid.uuid4())
            self.node_id = self.address # For simplicity, node_id is its network address

        self.last_seen = time.time() # Timestamp of the last interaction

    def __eq__(self, other):
        if isinstance(other, Node):
            return self.address == other.address
        elif isinstance(other, str): # Allow comparison with address string
            return self.address == other
        return False

    def __hash__(self):
        return hash(self.address)

    def __repr__(self):
        return f"Node(ID: {self.node_id}, Address: {self.address})"

    def to_dict(self):
        """Returns a dictionary representation of the node, useful for sharing peer info."""
        return {
            "node_id": self.node_id,
            "address": self.address, # The full http address
            "host": self.host,
            "port": self.port,
            "last_seen": self.last_seen
        }

    @classmethod
    def from_address_string(cls, address_string: str):
        """
        Creates a Node instance from an address string like "http://host:port".
        Args:
            address_string (str): The node's full HTTP address.
        Returns:
            Node: An instance of the Node class.
        Raises:
            ValueError: If the address string is not in the expected format.
        """
        if not address_string.startswith("http://"):
            raise ValueError("Address string must start with 'http://'")
        try:
            host_port_part = address_string.split("http://")[1]
            if ':' not in host_port_part:
                raise ValueError("Missing port separator ':'")
            host, port_str = host_port_part.split(":", 1) # Split only on the first colon
            if not host:
                raise ValueError("Host part cannot be empty.")
            port = int(port_str)
            return cls(host=host, port=port)
        except (IndexError, ValueError) as e:
            raise ValueError(f"Invalid node address string format: {address_string}. Expected 'http://host:port'. Error: {e}")


if __name__ == '__main__':
    node1 = Node(host="127.0.0.1", port=5000)
    print(f"Node 1: {node1}")
    print(f"Node 1 dict: {node1.to_dict()}")

    node2 = Node(host="localhost", port=5001, node_id="CustomNodeID123")
    print(f"Node 2: {node2}")

    node3_addr = "http://192.168.1.100:5005"
    node3 = Node.from_address_string(node3_addr)
    print(f"Node 3 (from string): {node3}")
    assert node3.address == node3_addr

    # Test equality
    node1_again = Node(host="127.0.0.1", port=5000)
    assert node1 == node1_again
    assert node1 == "http://127.0.0.1:5000"
    assert node1 != node2

    # Test hashing for set usage
    node_set = {node1, node2, node1_again}
    assert len(node_set) == 2 # node1 and node1_again should be treated as same due to __eq__ and __hash__

    print("\nNode class demo complete.")
