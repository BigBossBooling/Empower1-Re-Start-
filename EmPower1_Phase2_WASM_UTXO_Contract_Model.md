# EmPower1 WASM Smart Contracts in a UTXO Model - Detailed Operational Design

## 1. Introduction

**Purpose:** This document defines the detailed conceptual framework and operational mechanics for WebAssembly (WASM)-based smart contracts within the EmPower1 blockchain's Unspent Transaction Output (UTXO) model. It elaborates on how contracts manage state, interact with value (native EMPWR tokens), identify callers, utilize host functions provided by the EmPower1 runtime, and crucially, how atomicity of complex operations is ensured through a Two-Phase Commit (2PC) protocol.

**Philosophy Alignment:** This model is meticulously designed to ensure integrity, efficiency, and scalability, providing a robust foundation for advanced AI/ML-driven smart contracts and decentralized applications (DApps). It directly reflects EmPower1's Design Philosophy, particularly **"Core Principle 4: The Expanded KISS Principle"** through tenets like **"K - Know Your Core, Keep it Clear (Precision in Every Pixel)"** by defining unambiguous operational mechanics, and **"S - Systematize for Scalability, Synchronize for Synergy (Harmonious Growth)"** by creating a system that can support complex interactions while maintaining core ledger integrity. This detailed design underpins EmPower1's mission as a resilient and trustworthy humanitarian blockchain.

## 2. Core Architectural Principles

The EmPower1 smart contract operational model is built upon the following core architectural principles:

*   **Hybrid State Model (UTXO Set + Contract State Trie):** EmPower1 utilizes the UTXO model for its native EMPWR token, providing clarity and proven security for value transfer. Alongside this, a separate Contract State Trie (e.g., Merkle Patricia Trie) is maintained for persistent, arbitrary key-value data storage required by smart contracts.
*   **Explicit Value Transfer Mechanisms:** Interactions between smart contracts and the native EMPWR token (UTXO set) are handled through explicit mechanisms, ensuring that value transfers are clearly auditable and adhere to the UTXO model's rules, even when initiated by contract logic.
*   **Cryptographically Secure Caller Identity:** Smart contracts can reliably and securely identify their immediate caller (the address that signed the `TxInput` in the `TxContractCall` transaction) for access control and other identity-based logic.
*   **Two-Phase Commit (2PC) Protocol for Atomic Execution:** To ensure that complex operations involving both contract state changes and UTXO manipulations either complete fully or have no effect at all, a 2PC protocol is employed. This guarantees atomicity and prevents partial state updates that could compromise system integrity.

## 3. Detailed Design Components

### 3.1. Smart Contract State Management (Hybrid Model)

EmPower1's state management combines the strengths of the UTXO model with the flexibility of account-like state for smart contracts:

*   **Core UTXO Set:** This is the primary ledger for native EMPWR tokens. It consists of a set of unspent transaction outputs, each specifying a value and a controlling address (represented by a public key hash). Standard UTXO rules (inputs must be unspent, sum of inputs ≥ sum of outputs + fee, valid signatures) apply.
*   **Contract State Trie:** Each smart contract has access to its own persistent key-value storage space. This storage is logically organized as a separate Merkle Patricia Trie (or a similar authenticated data structure) for each contract, or as a global trie with contract-specific prefixes. This allows contracts to store arbitrary data (e.g., balances of contract-issued tokens, application-specific state variables, user records).
*   **Block StateRoot Commitment:** The `Block.StateRoot` field in each EmPower1 block header is a cryptographic commitment (e.g., a Merkle root) to the overall state of the blockchain. This `StateRoot` **must commit to both the current state of the UTXO set AND the root of the global Contract State Trie (or a trie of contract state roots).** This ensures that the complete state of the system, including all contract data, is verifiable and tamper-proof.
*   **Contract State Updates:** Smart contracts manage their state via specialized host functions (e.g., `blockchain_set_storage(key, value)`, `blockchain_get_storage(key)`). When a contract modifies its state, these changes are initially batched within a temporary, in-memory version of its state trie during transaction execution. Only upon successful completion of the entire transaction (including the 2PC protocol) are these changes committed to the global Contract State Trie.

### 3.2. Receiving and Sending Value/Tokens by Contracts

Smart contracts can interact with the native EMPWR token (UTXO) layer:

*   **Receiving Value (EMPWR tokens):**
    *   A `TxContractCall` transaction (which invokes a smart contract) can include one or more `TxOutput`s directed to the smart contract's own address. This effectively transfers EMPWR tokens to the control of the contract.
    *   The contract's internal logic (e.g., within its WASM bytecode) can become aware of this incoming value through host functions like `blockchain_get_contract_balance()` or by inspecting the transaction details made available to it. The contract itself doesn't "hold" UTXOs in a wallet sense; rather, UTXOs are created on the global ledger that are spendable by transactions authorized by the contract's logic (see below).
*   **Sending Value/Tokens (EMPWR tokens) from a Contract:**
    *   To send EMPWR tokens, a smart contract does not directly construct a UTXO transaction in its WASM environment. Instead, it invokes a privileged host function (e.g., `blockchain_create_output_request(value, pub_key_hash, metadata)` or the more comprehensive `blockchain_transfer_value`).
    *   This host function call acts as a **request** to the EmPower1 runtime to create one or more new UTXO outputs. The contract specifies the value, recipient address (public key hash), and any associated metadata for each requested output.
    *   The runtime buffers these requests as "internal transactions" or "output requests." These requests will consume existing UTXOs controlled by the contract's address.
    *   **Transaction Ordering:** The processing and validation of these internal transactions/output requests are critical and are handled within the Two-Phase Commit protocol (see 3.5) to ensure the contract has sufficient balance and to prevent double-spends of the contract's UTXOs.
*   **Contract-Issued Tokens (e.g., ERC-20 like):**
    *   Tokens created and managed entirely by a smart contract (e.g., tokens adhering to an ERC-20 like standard) are **not** native UTXOs.
    *   The balances and ownership of these contract-issued tokens are managed entirely within the contract's own state in the Contract State Trie (using `blockchain_set_storage` and `blockchain_get_storage`). Transfers of such tokens are state changes within the contract, not direct UTXO operations on the L1.

### 3.3. Caller Identity (`blockchain_get_caller_address()`)

Securely identifying the initiator of a contract call is crucial for access control and auditability:

*   When a `TxContractCall` is processed, the EmPower1 runtime makes the public key hash of the signer of the primary input (`TxInput.PubKey` or its hash) that authorized the contract call available to the WASM smart contract environment.
*   This is exposed via a dedicated host function, for example, `blockchain_get_caller_address() -> AddressHash`.
*   **Importance:** This allows smart contracts to implement robust access control mechanisms (e.g., `onlyOwner` modifiers, role-based permissions) and to reliably identify the entity invoking their functions. This is fundamental for the integrity of contract interactions.

### 3.4. Host Functions (Adapted and New for UTXO Interaction)

Smart contracts interact with the EmPower1 blockchain environment through a well-defined set of host functions.

*   **Adapted Host Functions (from typical account-based models):**
    *   `blockchain_get_balance(address_hash: AddressHash) -> Balance`: Queries the global UTXO set to return the total balance of EMPWR tokens controlled by the given `address_hash`.
    *   `blockchain_transfer_value(to_address_hash: AddressHash, amount: Balance) -> bool`: (High-level abstraction) This function would internally use `blockchain_create_output_request` to request the runtime to transfer EMPWR tokens from the contract's own UTXO balance to the specified recipient. Returns success/failure.
    *   `blockchain_get_storage(key: &[u8]) -> Option<Vec<u8>>`: Reads a value from the calling contract's private key-value state in the Contract State Trie.
    *   `blockchain_set_storage(key: &[u8], value: &[u8])`: Writes a value to the calling contract's private key-value state in the Contract State Trie.
*   **New Host Functions (Conceptual Additions for UTXO & Enhanced Functionality):**
    *   `blockchain_get_contract_address() -> AddressHash`: Returns the address of the currently executing contract.
    *   `blockchain_get_contract_balance() -> Balance`: Returns the total sum of EMPWR token value in UTXOs currently spendable by (i.e., "owned by") the contract's address.
    *   `blockchain_create_output_request(value: Balance, pub_key_hash: AddressHash, metadata: Option<Vec<u8>>) -> bool`: Low-level host function that signals the runtime to stage the creation of a new `TxOutput` for EMPWR tokens. This output will be funded by consuming UTXOs owned by the contract. This is a key part of the 2PC process.
    *   `blockchain_get_utxo_details(tx_id: &[u8], vout: u32) -> Option<UtxoInfo>`: Allows a contract to inspect details of a specific UTXO on the blockchain (e.g., its value, script type), which could be useful for advanced contract logic interacting with specific UTXOs it expects to receive or manage. (`UtxoInfo` would be a struct with relevant details).
    *   `blockchain_verify_signature(pub_key: &[u8], message_hash: &[u8], signature: &[u8]) -> bool`: Allows contracts to verify digital signatures, useful for validating off-chain data or authorizations passed into the contract.
    *   `blockchain_get_block_metadata() -> BlockMetadata`: Provides access to current block information such as block height, timestamp, the hash of the `AIAuditLog` for the current block (or a relevant segment), and the parent `StateRoot`. (`BlockMetadata` would be a struct).
    *   `blockchain_log(message: &[u8])`: Allows contracts to emit log messages, which could be captured and stored (potentially in the `AIAuditLog` or a separate event log system) for debugging and off-chain monitoring.

### 3.5. Atomicity: Two-Phase Commit (2PC) Protocol for Smart Contract Execution

To ensure that operations involving both contract state modifications (in the Contract State Trie) and EMPWR token transfers (UTXO operations) are atomic (either all succeed or all fail), EmPower1 will implement a Two-Phase Commit protocol for `TxContractCall` execution.

*   **Phase 1: Execution & Staging (The "Prepare" Stage)**
    1.  **Isolated WASM Execution:** The contract's WASM bytecode is executed in a sandboxed environment.
    2.  **Temporary Contract State:** All `blockchain_set_storage` operations modify a temporary, in-memory copy or overlay of the contract's state trie. These changes are not visible to other transactions or contracts yet.
    3.  **Internal Transaction Buffering:** All calls to `blockchain_create_output_request` (or higher-level functions like `blockchain_transfer_value` that use it) do not immediately create UTXOs. Instead, they generate "output requests" or "internal transaction proposals" that are collected in a temporary buffer associated with the current `TxContractCall`.
    4.  **Preliminary Validation:** Basic validation of these buffered output requests occurs (e.g., valid recipient address format, non-negative value). The runtime also tentatively checks if the contract *appears* to have sufficient UTXO balance to cover these requests based on its known UTXOs at the start of the transaction.
    5.  **Outcome of Phase 1:** If the WASM execution completes without errors (e.g., out-of-gas, trap), the outcome is a set of proposed state changes (a diff for the contract's state trie) and a list of proposed internal UTXO transactions (the buffered output requests).

*   **Phase 2: Commit or Rollback (The "Commit/Abort" Stage)**
    1.  **Final UTXO Validation:** The EmPower1 runtime now performs a final, rigorous validation of all buffered internal UTXO transactions against the current *global* blockchain state (specifically, the UTXO set). This includes:
        *   Ensuring the contract's address possesses sufficient unspent UTXO value to fund all requested outputs.
        *   Preventing any double-spending of the contract's UTXOs within this transaction or against already confirmed transactions.
        *   This step is critical and must be performed atomically with respect to other transactions being processed for the same block.
    2.  **Decision Point:**
        *   **SUCCESS (Commit):** If the WASM execution was successful AND all buffered internal UTXO transactions are fully validated, the transaction is committed. This means:
            *   The temporary contract state changes (the diff) are atomically applied to the global Contract State Trie.
            *   The buffered internal UTXO transactions are finalized, meaning new UTXOs are created on the global ledger, and the contract's UTXOs used as inputs are marked as spent. These become part of the block's overall transaction set.
            *   The `TxContractCall` is marked as successful.
        *   **FAILURE (Rollback/Abort):** If the WASM execution failed (e.g., trap, out of gas) OR if any of the buffered internal UTXO transactions fail final validation (e.g., insufficient funds, attempted double-spend), the entire operation is rolled back. This means:
            *   All temporary contract state changes are completely discarded.
            *   All buffered internal UTXO transactions are discarded.
            *   The original `TxContractCall` is marked as failed (and any gas fees up to the point of failure are still charged). No changes are made to the UTXO set or the Contract State Trie as a result of this contract call.

*   **Strategic Rationale for 2PC:**
    *   **Integrity & Atomicity:** Guarantees that complex contract operations that touch both contract state and the UTXO ledger are atomic, preventing inconsistent states.
    *   **Security (Firewall):** The isolation of WASM execution and the buffering of UTXO requests act as a firewall. Bugs or malicious logic within a contract cannot directly manipulate the UTXO set; they can only *request* changes, which are then rigorously validated by the runtime.
    *   **Scalability Support:** By clearly delineating contract logic from core ledger operations, the system can be optimized more effectively. L2 solutions can also more easily interact with a well-defined state transition function.
    *   **GIGO (Garbage In, Garbage Out) Antidote:** The 2PC validation, especially of UTXO operations, ensures that even if a contract attempts invalid actions, they are caught before impacting the global state.
    *   **User Confidence:** Provides users and developers with strong assurances about the reliability and consistency of smart contract interactions with native EMPWR tokens.

## 4. Summary of Operational Flow (Conceptual)

The lifecycle of a `TxContractCall` transaction is as follows:

1.  **Receipt & Basic Validation:** The transaction is received by a node, and basic syntactic validation occurs.
2.  **Phase 1 Execution (Prepare):**
    *   The specified WASM contract code is loaded and executed in a sandboxed environment with the provided input data.
    *   Contract state changes are written to a temporary, in-memory trie.
    *   Requests for EMPWR token transfers (UTXO outputs) are buffered as internal transaction proposals.
3.  **End of Phase 1:** If WASM execution traps or runs out of gas, proceed to rollback. Otherwise, the proposed state diff and buffered UTXO requests are passed to Phase 2.
4.  **Phase 2 Validation & Decision (Commit/Abort):**
    *   The runtime performs final validation on the buffered UTXO requests against the global UTXO set (checking for sufficient funds, no double spends from the contract's UTXOs).
    *   **If validation passes:** The state diff is applied to the global Contract State Trie, and the new UTXOs (from buffered requests) are created on the global ledger, consuming the contract's input UTXOs. The `TxContractCall` is successful.
    *   **If validation fails:** All temporary state changes and buffered UTXO requests are discarded. The `TxContractCall` is marked as failed.
5.  **State Root Update:** The global `StateRoot` (committing to both the UTXO set and the Contract State Trie) is updated to reflect the outcome of all successful transactions in the block.

## 5. Conclusion

This detailed operational model for WASM-based smart contracts within EmPower1's UTXO system, centered around a hybrid state model and a Two-Phase Commit protocol, provides a robust, secure, and highly intelligent foundation. It ensures atomic interactions between contract logic and the native EMPWR token layer, upholds the integrity of the UTXO set, and offers the flexibility needed for sophisticated DApps. This design directly supports EmPower1's mission to deliver advanced, AI-enhanced functionalities for its humanitarian objectives by ensuring that even complex operations are executed with precision, clarity, and unwavering integrity.
