# EmPower1 Initial Wallet System - Conceptual Design

## 1. Introduction

The purpose of this document is to conceptualize the initial wallet system for the EmPower1 blockchain. This includes the design of a Command Line Interface (CLI) wallet, intended for early adoption, developer use, and core network testing. Furthermore, this document lays the conceptual groundwork for future user-friendly Mobile and Web User Interfaces (UIs), ensuring that the foundational architecture supports broader accessibility and user empowerment.

## 2. Why (Strategic Rationale & Design Philosophy Alignment)

The EmPower1 wallet system is driven by the following strategic considerations:

*   **Early Access & Testability:** A robust CLI wallet is crucial in the initial phases of the EmPower1 blockchain. It provides developers, validators, early adopters, and testers with the essential tools to interact with the network, create transactions, check balances, and verify core functionalities.
*   **Groundwork for Accessibility:** While the initial focus includes a technical CLI, the underlying wallet logic and design principles must anticipate and facilitate the development of intuitive Graphical User Interfaces (GUIs) for mobile and web platforms. This aligns with EmPower1's mission of global accessibility.
*   **Empowerment through Control:** In line with decentralization principles, the wallet system must provide users with direct control over their private keys and, consequently, their funds. Secure key management is paramount.
*   **Phased Rollout:** The wallet system will be developed in phases, starting with essential features on the CLI. Subsequent expansions, including full GUI implementations and advanced features, will be guided by user needs, community feedback, and the overall maturation of the EmPower1 ecosystem.

## 3. What (Conceptual Component, Data Structures, Core Logic)

This section details the core components, data structures, and logic envisioned for the EmPower1 wallet system.

### 3.1. Core Wallet Functionalities (Applicable to CLI and future GUIs)

These functionalities form the backbone of any EmPower1 wallet:

*   **Key Management:**
    *   **Generation:** Creation of new public/private key pairs using standard, secure cryptographic libraries (e.g., ECDSA with secp256k1 curve, or EdDSA as per final protocol decisions).
    *   **Secure Storage:** Robust encryption of private keys when stored in wallet files (e.g., AES-256 encryption, with key derived from user password). Future iterations should plan for hardware wallet integration (e.g., Ledger, Trezor).
    *   **Import/Export:** Support for importing private keys from, and exporting to, standard formats such as Wallet Import Format (WIF) for individual keys and BIP39 mnemonic phrases for seed-based wallets.
    *   **Address Generation:** Derivation of human-readable EmPower1 addresses from public keys, following the defined address format of the blockchain.
*   **Balance Inquiry:**
    *   Ability to query a connected EmPower1 node (via its RPC API) to retrieve UTXO (Unspent Transaction Output) information or account balances associated with the wallet's addresses.
    *   Clear display of total balance, available (spendable) balance, and potentially pending balances or staked amounts.
*   **Transaction Creation & Signing:**
    *   Construction of various transaction types as defined in `EmPower1_Phase1_Transaction_Model.md` (initially `StandardTx`, conceptually extending to `StakeTx`, `VoteTx`, `ContractCallTx`, etc.).
    *   **Input Selection (for UTXO-based models):** Implementation of a UTXO selection strategy to gather sufficient funds for the desired payment amount while optimizing for minimizing fees and managing change outputs effectively.
    *   **Fee Calculation:** Estimation or specification of transaction fees required for network processing.
    *   **Local Signing:** Transactions must be signed locally using the user's private keys. Private keys should never leave the secure environment of the wallet.
    *   Adherence to the canonical transaction formatting rules before signing to ensure consistent transaction IDs.
*   **Transaction Broadcasting:**
    *   Mechanism to submit locally signed, raw transactions to a connected EmPower1 node via its RPC API for propagation to the network.
*   **Transaction History:**
    *   Fetching and displaying a list of past incoming and outgoing transactions associated with the wallet's addresses.
    *   Displaying relevant details for each transaction: Transaction ID (TxID), type, amount, fee paid, number of confirmations, timestamp, and involved addresses.
*   **Multi-Signature Support (Conceptual for initial CLI, Core for Future GUIs):**
    *   Functionality to create multi-signature addresses/wallets requiring M-of-N signatures to authorize transactions.
    *   Workflow for initiating transactions that require multiple signatures.
    *   Ability to import partially signed transactions (PSBT-like functionality).
    *   Interface for adding a signature to a partially signed transaction.
    *   Broadcasting fully signed multi-signature transactions.

### 3.2. CLI Wallet Specifics

The initial CLI wallet will provide core functionalities through text-based commands.

*   **Commands (Examples):**
    *   `empower1-cli getnewaddress [--label <label>]`
    *   `empower1-cli getbalance [--address <address>]`
    *   `empower1-cli listunspent [--address <address>] [--minconf <confirmations>]`
    *   `empower1-cli sendtoaddress <to_address> <amount> [--from_address <from_address>] [--fee <fee_amount>]`
    *   `empower1-cli gettransaction <txid>`
    *   `empower1-cli importprivkey <private_key_wif> [--label <label>] [--rescan yes|no]`
    *   `empower1-cli exportprivkey <address>`
    *   `empower1-cli backupwallet <destination_path>`
    *   `empower1-cli createmultisig <num_signatures_required> <pubkey1> <pubkey2> ... <pubkeyN>`
    *   `empower1-cli addmultisigaddress <multisig_address_from_createmultisig> [--label <label>]`
    *   `empower1-cli createtransaction <to_address> <amount> --inputs '[{"txid":"id","vout":n},...]'` (more advanced)
    *   `empower1-cli signrawtransaction <hex_encoded_transaction_data> [--privkeys <private_keys_wif_list_json>] [--sighashtype <type>]`
    *   `empower1-cli sendrawtransaction <signed_hex_encoded_transaction_data>`
*   **Output Format:** Primarily clear, human-readable text. Include a `--json` option for most commands to output data in JSON format, facilitating scriptability and integration with other tools.
*   **Interaction Model:** Predominantly non-interactive commands for ease of scripting. An optional interactive mode (`empower1-cli interactive`) could guide users through common operations like sending transactions or wallet setup.

### 3.3. Conceptual Mobile/Web UI Sketch

While the initial deliverable might be the CLI, the design should pave the way for future Mobile and Web UIs.

*   **Lightweight Client Principles (Nexus Protocol Link):**
    *   Mobile/Web UIs will likely operate as lightweight clients, focusing on essential user actions: checking balances, sending/receiving PTCN, viewing recent transaction history, and interacting with specific EmPower1 features like Stimulus/Tax views.
    *   Designed for minimal resource footprint, making them suitable for running on average smartphones or within web browsers.
    *   These clients would typically interact with "Super-Hosts" (as per the Nexus Protocol concept) or full nodes that provide simplified, secure, and potentially filtered/aggregated data via APIs. They would not store the full blockchain.
*   **Key UI Sections (Conceptual for Mobile/Web):**
    *   **Dashboard/Overview:** A summary screen showing total portfolio value (in PTCN and potentially a fiat equivalent), recent activity highlights, and quick action buttons (e.g., "Send," "Receive," "Stake").
    *   **Send Page:** User-friendly form with fields for recipient's address (with QR code scanning capability), amount, and a fee selection mechanism (e.g., simple sliders for low/medium/high priority, with an advanced option to set custom fees). A clear transaction summary should be presented before final confirmation.
    *   **Receive Page:** Displays the user's primary address clearly, with a QR code for easy sharing and a "Copy Address" button. Option to generate new addresses.
    *   **Transaction History Page:** A scrollable, filterable, and searchable list of all transactions involving the user's wallet. Each entry should be clickable to show detailed transaction information.
    *   **Settings Page:**
        *   Wallet Management: Backup wallet (e.g., display recovery phrase), export keys.
        *   Security: Set/change PIN or password, enable biometric authentication (if supported by device/browser).
        *   Network: Configure connection to EmPower1 nodes/Super-Hosts, select network (mainnet/testnet).
        *   Address Book: Manage saved contacts/addresses.
*   **Stimulus/Tax View (EmPower1 Specific UI Feature):**
    *   A dedicated section within the wallet UI to clearly display incoming `StimulusTx` transactions and any outgoing `TaxTx` transactions that have affected the user's balance.
    *   Each such transaction should ideally provide a brief explanation or category and a direct link to an `AIAuditLog` explorer (conceptual) for users to understand the basis of the transaction (e.g., "Received Low-Income Stimulus Q1," "Paid Carbon Offset Tax"). This aligns with EmPower1's transparency goals.
*   **Security Considerations for UI Wallets:**
    *   Emphasize client-side key management. For web wallets, this means leveraging the browser's WebCrypto API for cryptographic operations and storing encrypted keys in secure local storage. For mobile wallets, utilize platform-specific secure enclaves or keychain/keystore services.
    *   Provide extensive user education within the app/website regarding security best practices, phishing awareness, and scam prevention.

## 4. How (High-Level Implementation Strategies & Technologies)

*   **CLI Wallet:**
    *   **Language:** Go is a strong candidate, especially if the EmPower1 node itself is implemented in Go, allowing for shared libraries and consistency. Python is also suitable due to its wide range of libraries and ease of scripting.
    *   **Libraries:** Standard cryptographic libraries (e.g., Go's `crypto/ecdsa`, `golang.org/x/crypto/ripemd160`), RPC client libraries for node communication, libraries for command-line argument parsing.
*   **Mobile UI:**
    *   **Frameworks:** Cross-platform frameworks like React Native (JavaScript/TypeScript) or Flutter (Dart) allow for code reuse across iOS and Android. Native development (Swift for iOS, Kotlin for Android) can offer optimal performance and platform integration.
    *   **Storage:** Utilize secure storage mechanisms like iOS Keychain and Android Keystore for private keys.
*   **Web UI:**
    *   **Frameworks:** Modern JavaScript frameworks such as React, Vue.js, or Angular.
    *   **Storage & Crypto:** Leverage browser's `localStorage` or `IndexedDB` for storing encrypted private keys, and the `WebCrypto API` for performing cryptographic operations securely within the browser environment.
*   **Backend Interaction:** All wallet versions (CLI, Mobile, Web) will interact with EmPower1 nodes (or Super-Hosts) via a well-defined and secure RPC (Remote Procedure Call) API. This API needs to expose endpoints for querying balances, fetching transaction history, broadcasting transactions, etc.
*   **Core Logic Abstraction:**
    *   It is highly recommended to develop a **core wallet library**. This library would encapsulate all critical logic: key generation, address derivation, transaction construction, signing (including canonicalization), and interaction with cryptographic primitives.
    *   This library could be written in a language like Rust (which can compile to WebAssembly for web use and expose C-bindings for mobile/CLI) or Go (with C-bindings).
    *   Using such a shared library across the CLI, mobile, and web interfaces ensures consistency, reduces code duplication, and centralizes security-critical components for easier auditing and maintenance.

## 5. Synergies

The EmPower1 Wallet System is a key user-facing component with strong synergies across the ecosystem:

*   **CritterChain Node (EmPower1 Node):** Wallets are the primary clients that consume the RPC services exposed by EmPower1 nodes. They rely on nodes for blockchain data and transaction broadcasting.
*   **Transaction Model (`EmPower1_Phase1_Transaction_Model.md`):** Wallets are responsible for correctly constructing, signing (according to canonicalization rules), and interpreting transactions as defined in the transaction model.
*   **Nexus Protocol:** The conceptual Mobile/Web UI directly aligns with the vision of lightweight clients. These clients are prime candidates to run on or interact with "Super-Hosts" developed under the Nexus Protocol, benefiting from their optimized services.
*   **Decentralized Identity (DID) System:** Future versions of the wallet are expected to integrate DID management features, allowing users to create, control, and present their DIDs. This might involve dedicated UI sections and transaction types.
*   **Multi-Signature Wallets Design:** The wallet system is the user-facing component for creating and managing multi-signature wallets and transactions, providing the interface for collecting signatures and broadcasting.
*   **AIAuditLog & Governance:** Through the Stimulus/Tax view, wallets will provide users visibility into AI-driven economic adjustments and governance actions, potentially linking to explorer tools for detailed audit trails.

## 6. Anticipated Challenges & Conceptual Solutions

*   **Security of Private Keys:**
    *   *Challenge:* Protecting user private keys from theft, loss, or unauthorized access is the most critical challenge.
    *   *Conceptual Solution:* Implement robust encryption for stored keys (CLI/Mobile/Web). Promote the use of strong, unique passwords. Educate users relentlessly on backup procedures (e.g., mnemonic phrases) and the risks of malware/phishing. Plan for hardware wallet integration as a high-priority future enhancement. For lightweight clients interacting with Super-Hosts, clearly articulate the trust model and any risks associated with key management.
*   **User Experience (UX) for Non-Technical Users:**
    *   *Challenge:* Cryptocurrencies are complex. Making wallets intuitive and accessible for a broad, global audience (especially for Mobile/Web UIs) is essential for adoption.
    *   *Conceptual Solution:* Adhere strictly to the "Expanded KISS Principle." Employ user-centered design methodologies. Use clear, simple language, minimizing technical jargon. Provide in-app educational resources, tooltips, and FAQs. Conduct thorough user testing with diverse target groups.
*   **Cross-Platform Consistency and Maintenance:**
    *   *Challenge:* Ensuring a consistent user experience, feature set, and security level across different wallet implementations (CLI, various mobile platforms, web browsers) can be difficult and resource-intensive.
    *   *Conceptual Solution:* Utilize a shared core wallet library for critical logic. Develop detailed design specifications and UI guidelines. Implement comprehensive automated testing suites that cover all platforms.
*   **Keeping Wallets Synchronized with Network Updates:**
    *   *Challenge:* Blockchain protocols evolve. Wallets must be updated to support new transaction types, consensus rule changes, or handle network forks gracefully.
    *   *Conceptual Solution:* Implement clear versioning for both the wallet software and the RPC API. Design for backward compatibility where possible. Establish robust update mechanisms for wallet applications, with clear communication to users about changes.
*   **Scalability and Reliability of RPC Backend:**
    *   *Challenge:* EmPower1 nodes (or Super-Hosts) must be able to handle a large number of concurrent RPC requests from potentially millions of wallets without performance degradation.
    *   *Conceptual Solution:* Design efficient RPC endpoints on the node side. Consider deploying dedicated, load-balanced RPC query nodes as the user base grows. Implement rate limiting and sensible caching strategies.
*   **Phishing and Social Engineering Attacks:**
    *   *Challenge:* Users can be tricked into revealing their recovery phrases, private keys, or signing malicious transactions through deceptive websites, messages, or applications.
    *   *Conceptual Solution:* Integrate prominent warnings and educational messages within the wallet UI about common attack vectors. For sensitive operations (e.g., sending large amounts, interacting with new contracts), provide clear, unambiguous confirmation screens. Consider integration with community-maintained databases of known scam addresses or malicious dApps for warnings.
