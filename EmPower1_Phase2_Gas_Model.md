# EmPower1 Comprehensive Gas Model - Fueling Intelligent Agreements with Fairness

## 1. Introduction

**Purpose:** This document defines a precise, equitable, and comprehensive gas model for WebAssembly (WASM) smart contract execution within the EmPower1 blockchain. This model is specifically designed to operate in conjunction with EmPower1's hybrid state architecture (UTXO set + Contract State Trie) and its Two-Phase Commit (2PC) protocol for atomic contract operations. The gas model is fundamental to the sustainable and secure operation of the network.

**Philosophy Alignment:** This gas model is a critical component that ensures fair resource pricing, prevents network abuse, and supports the long-term economic sustainability of the EmPower1 "digital ecosystem." It directly aligns with **"Core Principle 4: The Expanded KISS Principle,"** particularly the tenets **"S - Sense the Landscape, Secure the Solution (Proactive Resilience)"** by creating a deterrent against resource exhaustion attacks, and **"K - Know Your Core, Keep it Clear (Precision in Every Pixel)"** by providing a transparent and understandable mechanism for resource accounting.

## 2. Core Objective

The primary objective of the EmPower1 gas model is to meticulously account for and charge for every computational and storage resource consumed during the execution of smart contracts. This ensures:
*   **Fair Resource Pricing:** Users and contracts pay proportionally for the resources they consume.
*   **Prevention of Abuse:** Deters denial-of-service (DoS) attacks and prevents runaway computations from degrading network performance.
*   **Economic Sustainability:** Provides a mechanism to compensate validators/block proposers for the work of processing transactions and securing the network.
*   **Support for Advanced Operations:** Enables complex operations, including AI/ML-driven smart contracts and redistributive transaction types (`StimulusTx`, `TaxTx`), by ensuring their resource implications are properly accounted for.

## 3. Gas Accounting: Comprehensive Cost Attribution

Gas in EmPower1 will be a unit that quantifies the total effort required to process a transaction, particularly those involving smart contract execution. Costs will be attributed to several distinct components:

### 3.1. WASM Operations (CPU Cycles)
*   **Description:** Each individual WebAssembly (WASM) opcode executed by a smart contract will have an associated gas cost. These costs will be carefully calibrated to reflect the relative CPU cycles required for each operation.
*   **Metric:** Per-opcode execution cost (e.g., `add` might cost X gas, `mul` Y gas, `call` Z gas).
*   **Strategic Rationale:** Directly prices the computational effort (CPU time) consumed by smart contract logic, ensuring fair compensation for validators and discouraging computationally intensive, inefficient code.

### 3.2. Contract State Trie Reads/Writes (Storage Access & Trie Manipulation)
*   **Description:** Costs associated with a smart contract accessing (reading) or modifying (writing) data in its persistent state, which is stored in the Contract State Trie.
*   **Metric:**
    *   Per-byte read cost.
    *   Per-byte write cost (typically significantly higher than read cost due to the lasting impact on blockchain state size).
    *   Base cost for trie node access/modification, reflecting the I/O and cryptographic hashing involved in Merkle Patricia Trie operations.
*   **Strategic Rationale:** Accurately prices the use of persistent on-chain storage and the computational overhead of maintaining authenticated state tries. This discourages unnecessary state bloat and incentivizes efficient data management within contracts.

### 3.3. Internal UTXO Transactions (Value & Transactional Overhead from Contracts)
*   **Description:** Costs associated with new internal UTXO-spending transactions that are requested by a smart contract (as detailed in `EmPower1_Phase2_WASM_UTXO_Contract_Model.md` via host functions like `blockchain_create_output_request`).
*   **Metric:**
    *   A base gas cost per internal UTXO transaction initiated by a contract.
    *   Additional costs proportional to the number of inputs consumed and outputs created by this internal transaction, similar to how standard L1 UTXO transactions might be priced.
*   **Strategic Rationale:** Prices the overhead and state manipulation involved when smart contracts interact with the main UTXO set to transfer native EMPWR tokens. Ensures that contract-initiated value transfers are economically accounted for and contribute to network security.

### 3.4. Inter-Contract Calls (Cross-Contract Communication)
*   **Description:** Costs incurred when one smart contract calls a function in another smart contract.
*   **Metric:**
    *   A base gas cost per inter-contract call.
    *   Additional costs proportional to the size of the arguments being passed between contracts.
*   **Strategic Rationale:** Prices the overhead of context switching, argument serialization/deserialization, and managing chained interactions, which can increase complexity and resource usage.

### 3.5. Cryptographic Operations (Host Function Calls)
*   **Description:** Specific gas costs for invoking host functions that perform cryptographic operations (e.g., `blockchain_verify_signature`, hashing functions if exposed directly).
*   **Metric:** A defined per-call cost, potentially varying based on the computational intensity of the specific cryptographic primitive being used (e.g., signature verification might cost more than a simple hash).
*   **Strategic Rationale:** Accurately prices the use of core blockchain security primitives, which can be computationally intensive, ensuring they are used judiciously.

### 3.6. 2PC Overhead (Conceptual Accounting & Failed Executions)
*   **Description:** The gas model must account for *attempted* work even in executions that ultimately fail or are rolled back by the Two-Phase Commit (2PC) protocol.
*   **Metric & Mechanism:**
    *   Gas consumed during Phase 1 (WASM execution) up to the point of any halt (e.g., out-of-gas trap, explicit revert) **is always charged**.
    *   If a `TxContractCall` successfully completes its WASM execution (Phase 1) but an internal UTXO transaction it requested subsequently fails validation during Phase 2 (Commit/Abort stage), the gas for the *entire original `TxContractCall`* (including all WASM operations from Phase 1) **is still charged**. The transaction as a whole fails, and its state changes are reverted, but the resources were consumed.
*   **Strategic Rationale:** This is crucial for preventing spamming and resource exhaustion attacks that rely on triggering failures late in the execution process. It ensures fair compensation to validators for computational effort already expended, regardless of the final outcome of the transaction.

### 3.7. Global `Block.StateRoot` Update (Implicit in Block Production)
*   **Description:** The computational cost of calculating and committing the new global `Block.StateRoot` (which includes the UTXO set root and the Contract State Trie root) at the end of each block.
*   **Metric & Mechanism:** This cost is considered part of the block proposer's overall responsibility and is implicitly covered by block rewards and potentially a portion of aggregated transaction fees. It is **not** typically itemized as a separate gas cost for each individual transaction within the block.
*   **Strategic Rationale:** Keeps per-transaction gas calculations focused on the direct resource consumption of that transaction (especially contract-specific consumption). The global state update is a shared network cost managed at the block level.

## 4. Gas Limit Handling & Fairness on Exhaustion

Effective gas limit handling is essential for network stability and fairness.

### 4.1. Per-Transaction Gas Limit
*   **Description:** Every transaction that can invoke smart contract execution (e.g., `TxContractCall`, `TxContractDeploy`) **must** specify a `gas_limit`. This is the maximum amount of gas the transaction initiator is willing to pay for the execution of this transaction.
*   **Strategic Rationale:** Prevents runaway computations (accidental infinite loops or malicious contracts) from consuming unbounded network resources and halting the chain. Provides predictability for users regarding maximum transaction cost.

### 4.2. Execution Halting on Exhaustion (During Phase 1 - WASM Execution)
*   **Description:** If the accumulated `gas_used` by a contract execution reaches the specified `gas_limit` at any point during its WASM execution (Phase 1 of 2PC), the execution halts immediately (an "out-of-gas" error).
*   **Mechanism:** All provisional state changes made by that contract call within its temporary in-memory trie are discarded. Any buffered internal UTXO transaction requests are also discarded. The transaction is marked as failed.
*   **Gas Charged:** The full `gas_limit` is charged to the transaction initiator, as the network expended resources up to that limit.
*   **Strategic Rationale:** Provides a quick failure mechanism for computationally expensive or potentially malicious contracts, protecting network resources. Charging the full limit disincentivizes setting unrealistically low limits that cause trivial operations to fail.

### 4.3. Exhaustion/Failure During Phase 2 Validation/Commit of Internal Transactions
*   **Description:** This scenario occurs if the original `TxContractCall`'s WASM execution (Phase 1) completed successfully (i.e., did not run out of gas or trap), but one or more of its buffered internal UTXO transaction requests (e.g., a contract trying to send EMPWR tokens) fails the final, rigorous validation in Phase 2 of the 2PC protocol (e.g., the contract has insufficient confirmed UTXO balance at the point of commit, or an input UTXO was already spent by another transaction).
*   **Mechanism:** The *entire original `TxContractCall` is reverted*. This means all its Phase 1 contract state changes (which were still in a temporary buffer) are discarded. The internal UTXO transactions are, of course, not applied. The overall transaction fails.
*   **Gas Charged:** All gas consumed by the `TxContractCall` during its Phase 1 WASM execution **is still charged** to the transaction initiator, up to the `gas_limit` specified.
*   **Strategic Rationale:** This ensures fairness for network resources already expended during the (potentially complex) WASM execution. The fault in this scenario lies with the contract's logic or its assumptions about its available UTXO balance at the time of commitment, which could not be fully resolved until Phase 2. This upholds the integrity of the UTXO set and the atomicity guarantee of 2PC.

### 4.4. Over-payment for Gas (Refund)
*   **Description:** If a transaction successfully completes and its actual `gas_used` is less than the `gas_limit` specified by the initiator, the difference (`gas_limit` - `gas_used`) multiplied by the gas price is refunded to the transaction initiator.
*   **Strategic Rationale:** Encourages users and wallets to set realistic `gas_limit`s that are sufficient for execution but not excessively high, improving predictability and user experience.

## 5. Strategic Implications & Future Enhancements

The gas model has broader strategic implications and potential for future enhancements:

*   **AI/ML Optimization of Gas Usage:**
    *   **Predictive Gas Limits:** AI/ML models could be developed (e.g., integrated into wallets or developer tools) to analyze contract code or transaction history to predict optimal `gas_limit`s for users, reducing overpayment or out-of-gas errors.
    *   **Code Optimization Suggestions:** AI tools (as part of the DApp Dev Tools strategy) could analyze smart contract WASM bytecode to identify gas-inefficient patterns and suggest optimizations to developers.
    *   **Anomalous Gas Usage Detection:** AI monitoring the network (part of Advanced AI/ML Strategy) could identify contracts or transactions exhibiting unusually high or unexpected gas consumption patterns, potentially flagging bugs, inefficiencies, or malicious behavior.
*   **Tiered Gas Pricing (Conceptual - for specific EmPower1 TxTypes):**
    *   For EmPower1's unique transaction types like `TxStimulusPayment` (outgoing stimulus) and `TxWealthTax` (if implemented as an incoming wealth tax collection), a different gas payment model could be considered.
    *   The base gas cost for end-users receiving a `TxStimulusPayment` could be zero or heavily subsidized, with the actual network cost potentially covered by the EmPower1 Treasury, a dedicated fund, or through the inflation model that supports network operations. This ensures that humanitarian distributions are not burdened by transaction costs for the recipients.
    *   Similarly, if a `TxWealthTax` is levied, the gas for its collection might be factored into the tax assessment or covered by the collecting entity.
    *   This requires careful economic modeling and governance approval.
*   **Developer Tooling:**
    *   Provide static analyzers, debuggers, and simulators as part of the SDK to help developers accurately estimate the gas costs of their smart contracts before deployment.
    *   Clear documentation on gas costs for all WASM opcodes and host functions.

## 6. Conclusion

The EmPower1 Comprehensive Gas Model, with its detailed accounting for various resource types and robust handling of execution limits and failures within a UTXO + 2PC environment, serves as a critical piece of "economic unseen code." It is designed to ensure fair resource pricing, prevent network abuse, and provide a sustainable economic foundation for validators and the network as a whole. By meticulously linking computational effort to cost, this gas model underpins the integrity and efficiency required for EmPower1 to fulfill its humanitarian mission and foster a thriving, intelligent, and equitable "digital ecosystem." Its design also anticipates future enhancements through AI/ML, further aligning with EmPower1's commitment to intelligent adaptation and optimization.
