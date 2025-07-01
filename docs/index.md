# EmPower1 Blockchain - Project Documentation

Welcome to the documentation for the EmPower1 Blockchain project. This document provides an overview of the project structure, components, and how to get started with development and testing.

## 1. Project Overview

EmPower1 Blockchain is a Python-based implementation aiming to create a decentralized cryptocurrency network with a focus on equitable wealth distribution through an Intelligent Redistribution Engine (IRE).

**Core Principles (Expanded KISS):**
*   **K** - Know Your Core, Keep it Clear
*   **I** - Iterate Intelligently, Integrate Intuitively
*   **S** - Systematize for Scalability, Synchronize for Synergy
*   **S** - Sense the Landscape, Secure the Solution
*   **S** - Stimulate Engagement, Sustain Impact

(Refer to the main `README.md` for a full mission statement and vision.)

## 2. Project Structure

The project is organized into the following main directories:

*   `empower1/`: Contains the core source code for the blockchain.
    *   `__init__.py`: Makes `empower1` a Python package.
    *   `block.py`: Defines the `Block` class, representing a single block in the chain.
    *   `blockchain.py`: Defines the `Blockchain` class, managing the chain, transactions, and consensus.
    *   `transaction.py`: Defines the `Transaction` class for all transactions on the network.
    *   `wallet.py`: Defines the `Wallet` class for managing user keys and signing.
    *   `ire/`: Sub-package for the Intelligent Redistribution Engine.
        *   `__init__.py`: Initializes the `ire` sub-package.
        *   `ai_model.py`: Placeholder for the AI/ML decision models used by IRE.
        *   `redistribution.py`: Core logic for tax application and stimulus distribution.
    *   `smart_contracts/`: Sub-package for smart contract implementations.
        *   `__init__.py`: Initializes the `smart_contracts` sub-package.
        *   `base_contract.py`: A base class with common functionalities for all smart contracts.
        *   `stimulus_contract.py`: Placeholder for a contract managing stimulus payments.
        *   `tax_contract.py`: Placeholder for a contract managing tax collection and rules.
*   `tests/`: Contains all unit and integration tests for the project.
    *   `__init__.py`: Makes `tests` a Python package.
    *   `conftest.py`: Shared pytest fixtures used across different test files.
    *   `test_*.py`: Individual test files for each module (e.g., `test_block.py`, `test_transaction.py`).
    *   `network/`: Tests for the network module.
        *   `test_node.py`: Tests for the `Node` class.
        *   `test_network.py`: Tests for the `Network` class and basic P2P interactions.
*   `docs/`: Contains project documentation files (like this one).
*   `cmd/`: Command-line applications.
    *   `node/`: Contains the main script for running a blockchain node.
        *   `main.py`: Script to initialize and run a node.
*   `.gitignore`: Specifies intentionally untracked files that Git should ignore.
*   `LICENSE`: Contains the project's license information (GNU Affero General Public License v3).
*   `README.md`: The main project README file with a high-level overview.
*   `requirements.txt`: Lists project dependencies.

## 3. Getting Started

### 3.1. Prerequisites

*   Python 3.8 or higher.
*   `pip` for package management.
*   `git` for version control.

### 3.2. Setup

1.  **Clone the repository (if you haven't already):**
    ```bash
    git clone <repository_url>
    cd empower1-blockchain # Or your project's root directory name
    ```

2.  **Create a virtual environment (recommended):**
    ```bash
    python -m venv venv
    source venv/bin/activate  # On Windows: venv\Scripts\activate
    ```

3.  **Install dependencies:**
    Install required Python packages using `requirements.txt`:
    ```bash
    pip install -r requirements.txt
    ```
    This will install `pytest` (for testing), `cryptography` (for cryptographic operations), `Flask` (for the P2P network server), and `requests` (for making P2P network requests).

### 3.3. Running Tests

The project uses `pytest` for testing. To run all tests:

1.  Ensure your virtual environment is activated and `pytest` is installed.
2.  Navigate to the root directory of the project.
3.  Run the following command:
    ```bash
    pytest
    ```
    Or, for more verbose output:
    ```bash
    pytest -v
    ```

    You should see output indicating the number of tests passed.

### 3.4. Running a Node

The primary way to run an EmPower1 node is using the `main.py` script in the `cmd/node/` directory.

1.  **Navigate to the project root directory.**
2.  **Run the node script:**
    ```bash
    python cmd/node/main.py [host] [port] [seed_node_http_address...]
    ```
    *   `[host]` (optional): The host address for the node to listen on (default: `127.0.0.1`).
    *   `[port]` (optional): The port for the node to listen on (default: `5000`).
    *   `[seed_node_http_address...]` (optional): Full HTTP addresses of seed nodes to connect to on startup (e.g., `http://127.0.0.1:5001`).

    **Example (running a single node on default host/port):**
    ```bash
    python cmd/node/main.py
    ```

    **Example (running a node on port 5001 and connecting to a seed node on port 5000):**
    ```bash
    python cmd/node/main.py 127.0.0.1 5001 http://127.0.0.1:5000
    ```

3.  Once running, the node provides a simple Command-Line Interface (CLI) for interactions. Type `help` in the node's CLI for available commands.

    You can run multiple instances of `cmd/node/main.py` on different ports to simulate a network. Use the `addpeer` command or provide seed node addresses to make them aware of each other.

## 4. Core Components Deep Dive

*(This section will be expanded with more detailed explanations of each component's API, logic, and interaction as development progresses.)*

### 4.1. Core Blockchain Logic (`empower1/`)
#### 4.1.1. `Block`
   - Represents a block with an index, list of transactions, timestamp, previous hash, validator address, its own hash, and a validator's signature.

### 4.2. `Transaction`
   - Represents a transaction with sender, receiver, amount, asset ID, fee, timestamp, and signature.

### 4.3. `Wallet`
   - Manages cryptographic key pairs (simplified) and can sign data.

### 4.4. `Blockchain`
   - Manages the chain of blocks, pending transactions, and includes basic Proof-of-Stake (PoS) logic for validator registration and selection.

### 4.5. Intelligent Redistribution Engine (`ire`)
   - `ai_model.py`: Simulates decision-making for wealth categorization, stimulus eligibility, and tax calculation.
   - `redistribution.py`: Orchestrates the tax and stimulus processes using the AI model.

### 4.6. Smart Contracts (`smart_contracts`)
   - `base_contract.py`: Provides shared functionality for contracts.
   - `stimulus_contract.py`: Placeholder for logic governing stimulus fund management and distribution.
   - `tax_contract.py`: Placeholder for logic governing tax rule definition and application.

### 4.7. Network Communication (`empower1/network/`)
   - `node.py`: Defines the `Node` class, representing a participant in the network with a network address (host, port).
   - `messages.py`: Defines `MessageType` enums for different network P2P messages.
   - `network.py`: Contains the `Network` class, which manages:
        - P2P connections and peer list (`peers`, `seed_nodes`, `connect_to_peer`, `request_peers_from`).
        - An HTTP server (Flask) with endpoints for `/ping`, `/GET_PEERS`, `/NEW_PEER_ANNOUNCE`, `/GET_CHAIN`, `/NEW_TRANSACTION`, `/NEW_BLOCK`.
        - Broadcasting of new transactions and blocks to known peers (`broadcast_transaction`, `broadcast_block`).
        - Handling of received transactions and blocks (`handle_received_transaction`, `handle_received_block`), including validation and addition to the local blockchain.
        - Basic chain synchronization: new nodes request the chain from peers, and a simple "longest valid chain wins" approach is used for updates (`request_chain_from_peer`, `handle_chain_response`).

### 4.8. Command-Line Interface (`cmd/node/main.py`)
   - A script to run an EmPower1 node. It initializes the Wallet, Blockchain, and Network components.
   - Provides CLI commands for interacting with the node (e.g., creating transactions, mining, viewing chain/peers, staking).
   - Connects to seed nodes on startup and participates in the P2P network.

## 5. Future Development & Contributions

*   **Robust Network Layer**:
    *   Enhance peer discovery (e.g., DHT, gossip).
    *   Implement more resilient message handling (e.g., retries, error propagation).
    *   Improve block/transaction propagation efficiency (e.g., inventory messages before full data transfer).
    *   Develop more sophisticated fork resolution logic beyond simple longest chain.
*   **Mature Proof-of-Stake**: Develop the PoS mechanism further, including detailed validator rewards, slashing conditions, and dynamic validator set management.
*   **State Management**: Implement robust state management for account balances and smart contract state, including transaction execution logic that modifies state.
*   Full cryptographic implementation for wallets, signatures, and hashing. # (This is largely complete for core objects)
*   Mature Proof-of-Stake implementation.
*   Development of the AI/ML models for the IRE.
*   Full smart contract execution environment.
*   Comprehensive API for interactions.
*   More detailed documentation for each module and function.

Contributions are welcome! Please refer to contribution guidelines (to be added).

---

*This documentation is a work in progress and will be updated as the project evolves.*
