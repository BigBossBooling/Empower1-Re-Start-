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
*   `docs/`: Contains project documentation files (like this one).
*   `.gitignore`: Specifies intentionally untracked files that Git should ignore.
*   `LICENSE`: Contains the project's license information (GNU Affero General Public License v3).
*   `README.md`: The main project README file with a high-level overview.
*   `requirements.txt` (To be added): Will list project dependencies.

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
    Currently, the main external dependency for running tests is `pytest`.
    ```bash
    pip install pytest
    ```
    (A `requirements.txt` file will be added later to streamline this: `pip install -r requirements.txt`)

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

### 3.4. Running the Blockchain (Basic Simulation)

Currently, the blockchain components can be run in a simulated way via their `if __name__ == '__main__':` blocks. For example, to run the `blockchain.py` demo:

```bash
python empower1/blockchain.py
```

This will demonstrate basic blockchain operations like creating transactions, mining blocks, and validating the chain as per the example script in that file. Similar runnable examples exist in `block.py`, `transaction.py`, etc.

## 4. Core Components Deep Dive (Placeholders)

*(This section will be expanded with more detailed explanations of each component's API, logic, and interaction as development progresses.)*

### 4.1. `Block`
   - Represents a block with an index, list of transactions, timestamp, previous hash, validator, and its own hash.

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

## 5. Future Development & Contributions

*   Full cryptographic implementation for wallets, signatures, and hashing.
*   Robust network layer for P2P communication between nodes.
*   Mature Proof-of-Stake implementation.
*   Development of the AI/ML models for the IRE.
*   Full smart contract execution environment.
*   Comprehensive API for interactions.
*   More detailed documentation for each module and function.

Contributions are welcome! Please refer to contribution guidelines (to be added).

---

*This documentation is a work in progress and will be updated as the project evolves.*
