# EmPower1 Blockchain - Core Data Structure Validation Plan

## 1. Objective

To formulate a detailed implementation plan for adding `Validate() error` methods to the core Go data structures defined in `internal/core/types/types.go`. The primary goal is to ensure robust data integrity from the outset of development, preventing invalid data from being processed or persisted.

## 2. Scope

This plan covers the validation logic for the following core Go structs:
*   `Transaction`
*   `TransactionInput`
*   `TransactionOutput`
*   `BlockHeader`
*   `Block`
*   `Account`

It also includes considerations for centralized error handling, conceptual notes on input sanitization/canonicalization, and integration with the unit testing strategy.

## 3. Guiding Principles

The design and implementation of this validation logic will adhere to the following core principles of the EmPower1 project:
*   **GIGO Antidote:** Implement explicit checks to prevent "garbage in, garbage out."
*   **System Reliability:** Ensure that robust validation contributes to overall system stability and predictability.
*   **Foundation for Testing:** Provide a clear, testable interface for data integrity checks.
*   **Constant Progression:** Iteratively build and refine validation logic as the system evolves.
*   **Know Your Core, Keep it Clear:** Validation rules should be explicit, understandable, and directly reflect the intended constraints of the data structures.
*   **Sense the Landscape, Secure the Solution:** Proactively identify potential data inconsistencies and implement checks to mitigate them.
*   **Iterate Intelligently:** Start with fundamental checks and expand as necessary, learning from testing and development.

---

## 4. Per-Struct Validation Logic Specification

Each core data structure will have an associated `Validate() error` method. This method will perform a series of checks and return `nil` if the structure is valid, or a specific error from `internal/errors` if any check fails.

### 4.1. `Transaction.Validate() error`

The `Validate()` method for the `Transaction` struct (`internal/core/types/types.go`) will implement the following checks. This validation is stateless and only checks the internal consistency of the transaction struct itself.

*   **A. Basic Structure & Required Fields:**
    1.  `TxID`: Must not be an all-zero `Hash`.
        *   *Error: `ErrInvalidTransactionID` (e.g., "transaction ID cannot be zero hash")*
    2.  `Type`: Must be a defined `TransactionType` constant (e.g., `StandardTransaction`, `StimulusTransaction`, etc.). An unknown numeric value for `Type` (outside the defined iota range) is invalid.
        *   *Error: `ErrInvalidTransactionType` (e.g., "unknown transaction type: [value]")*
    3.  `Outputs`: The `Outputs` slice must not be `nil`. For most transaction types, it must also not be empty. Specific system transactions or future types might allow empty outputs, which should be explicitly handled by `Type`.
        *   *Error: `ErrNoTransactionOutputs` (e.g., "transaction outputs cannot be nil or empty for type [Type]")*
    4.  `Timestamp`: Must not be a zero value (`time.Time{}`). Consider also checking if the timestamp is too far in the past or future relative to current network time (this might be a consensus rule rather than struct validation, but a sanity check here is useful, e.g., not before EmPower1 epoch start).
        *   *Error: `ErrZeroTimestamp` or `ErrInvalidTimestampRange`*
    5.  `Fee`: (Currently `uint64`, so always non-negative). No specific validation here unless a minimum fee is required for all transactions or certain types, which would be a consensus rule.
    6.  `Nonce`: (Currently `uint64`, so always non-negative). No specific validation here other than presence if used in an account model.

*   **B. Inputs Validation (if applicable to `Type`):**
    1.  If `Type` indicates spending (e.g., `StandardTransaction` that isn't a coinbase-like issuance):
        *   The `Inputs` slice must not be `nil` or empty.
            *   *Error: `ErrEmptyInputsForSpendingTx` (e.g., "inputs cannot be empty for a spending transaction type")*
        *   Each `TransactionInput` object within the `Inputs` slice must pass its own `input.Validate()` method.
            *   *Error: `ErrTransactionInputValidationFailed` (wrapping the error from `input.Validate()`)*
    2.  If `Type` does *not* imply spending from existing inputs (e.g., a stimulus issuance not funded by specific UTXOs but by treasury):
        *   The `Inputs` slice should typically be empty. If present and not empty, it might be an error depending on the strict definition of that `Type`.
            *   *Error: `ErrUnexpectedInputsForTxType` (e.g., "inputs are not expected for transaction type [Type]")*

*   **C. Outputs Validation:**
    1.  Each `TransactionOutput` object within the `Outputs` slice must pass its own `output.Validate()` method.
        *   *Error: `ErrTransactionOutputValidationFailed` (wrapping the error from `output.Validate()`)*

*   **D. Signature & PublicKey (if applicable to account-based `Type`):**
    1.  If the `Transaction.Type` implies an account-based sender (most types):
        *   `Signature`: Must not be empty `[]byte`.
            *   *Error: `ErrMissingSignature`*
        *   `PublicKey`: Must not be empty `[]byte`.
            *   *Error: `ErrMissingPublicKey`*
    2.  *Conceptual Note: Actual cryptographic signature verification is a separate, more complex step that uses the transaction's digest, signature, and public key. This `Validate()` method only checks for presence.*

*   **E. Metadata Validation:**
    1.  If `Metadata` map is not `nil`:
        *   Keys: Must not be empty strings. Consider a maximum key length (e.g., 64 chars).
            *   *Error: `ErrInvalidMetadataKey` (e.g., "metadata key cannot be empty" or "metadata key exceeds max length")*
        *   Values: Consider a maximum value size for each `[]byte` (e.g., 256 bytes).
            *   *Error: `ErrInvalidMetadataValueSize` (e.g., "metadata value for key [key] exceeds max size")*
    2.  Type-Specific Mandatory Metadata:
        *   If `Type` is `StimulusTransaction` or `TaxTransaction` (or other future types requiring specific AI attestations/references):
            *   Check for the presence of mandatory keys (e.g., `Metadata["AILogicID"]`, `Metadata["AIProof"]`). Values for these keys must not be empty.
            *   *Error: `ErrMissingRequiredMetadataField` (e.g., "missing required metadata field [key] for transaction type [Type]")*

*   **F. State-Dependent Checks (Not part of this stateless `Validate()` method but noted for completeness):**
    *   UTXO model: Sum of input values equals sum of output values + fee. Inputs must be unspent and valid.
    *   Account model: Sender account (from `PublicKey`) must exist and have sufficient `Balance` for `Outputs` + `Fee`. `Nonce` must be correct.
    *   These checks require access to the current blockchain state and are performed during transaction processing by the node, not within this isolated struct validation.

### 4.2. `TransactionInput.Validate() error`

The `Validate()` method for the `TransactionInput` struct (`internal/core/types/types.go`) will implement the following stateless checks:

*   **A. `PreviousTxHash` Validation:**
    1.  Must not be an all-zero `Hash`. The length is inherently validated by its `Hash` type (`[sha256.Size]byte`).
        *   *Error: `ErrInvalidPreviousTxHash` (e.g., "previous transaction hash cannot be zero hash")*

*   **B. `OutputIndex` Validation:**
    1.  (Currently `uint32`). Must be a sensible value. While `0` is a valid index, extremely large values might be suspect, though `uint32` provides a natural cap. No specific check beyond type limits for now unless a practical upper bound (e.g., max outputs per transaction) is defined.
        *   *(No specific error yet beyond type constraints, unless a max index is imposed)*

*   **C. `Signature` Validation:**
    1.  Must not be an empty `[]byte`.
        *   *Error: `ErrMissingInputSignature` (e.g., "input signature cannot be empty")*
    2.  Consider a maximum length for the signature byte slice if appropriate for the chosen signature scheme (e.g., to prevent arbitrarily large data).
        *   *Error: `ErrInvalidSignatureLength` (e.g., "input signature exceeds maximum allowed length")*

*   **D. `PublicKey` Validation:**
    1.  Must not be an empty `[]byte`.
        *   *Error: `ErrMissingInputPublicKey` (e.g., "input public key cannot be empty")*
    2.  Consider a maximum length for the public key byte slice, and potentially a check for the expected length of the chosen elliptic curve's public key (e.g., 33 bytes for compressed SECP256k1, 65 for uncompressed).
        *   *Error: `ErrInvalidPublicKeyLength` (e.g., "input public key has an invalid length")*
    3.  *Conceptual Note: Validating that the public key is a valid point on the chosen elliptic curve is a cryptographic operation beyond basic struct validation.*

### 4.3. `TransactionOutput.Validate() error`

The `Validate()` method for the `TransactionOutput` struct (`internal/core/types/types.go`) will implement the following stateless checks:

*   **A. `RecipientAddress` Validation:**
    1.  Must not be an empty `Address` (empty `[]byte`).
        *   *Error: `ErrInvalidRecipientAddress` (e.g., "recipient address cannot be empty")*
    2.  Consider a maximum length for the address byte slice if it's variable length, or ensure it matches a fixed length if `Address` type changes. (Current `Address []byte` is flexible).
        *   *Error: `ErrInvalidAddressLength` (e.g., "recipient address exceeds maximum allowed length" or "recipient address has incorrect length")*
    3.  *Conceptual Note: Validating checksums or specific address formats (e.g., Bech32) would be a more advanced check if such formats are adopted.*

*   **B. `Amount` Validation:**
    1.  (Currently `uint64`). Must generally be greater than 0. Zero-value outputs might be permissible for specific transaction types (e.g., data embedding, proof-of-existence, or specific contract interactions) but this would be a consensus rule or `Transaction.Type` specific logic. For standard transfers, amount must be positive.
        *   *Error: `ErrInvalidOutputAmount` (e.g., "output amount must be greater than zero" or "zero amount output not permitted for this transaction type")*
    2.  Ensure `Amount` does not exceed a total supply cap or a reasonable maximum transaction value if such limits are defined at the protocol level (this might be more of a consensus/transaction processing rule).

### 4.4. `BlockHeader.Validate() error`

The `Validate()` method for the `BlockHeader` struct (`internal/core/types/types.go`) will implement the following stateless and context-minimally-aware checks:

*   **A. `Version` Validation:**
    1.  Must be a known and supported block version number. (e.g., `if bh.Version != CurrentBlockVersion && bh.Version != PreviousSupportedBlockVersion ...`)
        *   *Error: `ErrInvalidBlockVersion` (e.g., "unsupported block version: [value]")*

*   **B. `PreviousBlockHash` Validation:**
    1.  Must not be an all-zero `Hash`, *unless* this is the genesis block (where `Height` is 0).
        *   *Error: `ErrInvalidPreviousBlockHash` (e.g., "previous block hash cannot be zero hash for non-genesis block")*

*   **C. `MerkleRoot` Validation:**
    1.  Must not be an all-zero `Hash`. The Merkle root is a commitment to the transactions in the block.
        *   *Error: `ErrInvalidMerkleRoot` (e.g., "merkle root cannot be zero hash")*
    2.  *Conceptual Note: Validating that `MerkleRoot` is correctly calculated from the block's transactions is a separate, more complex process done during block processing.*

*   **D. `Timestamp` Validation:**
    1.  Must not be a zero value (`time.Time{}`).
        *   *Error: `ErrZeroTimestamp` (e.g., "block timestamp cannot be zero")*
    2.  Must be reasonably consistent. For example, it should not be significantly before the median timestamp of the last N blocks (a consensus rule, state-dependent) or excessively far into the future from the node's current time (e.g., more than a few hours, to prevent manipulation). A basic sanity check could be ensuring it's not before the EmPower1 epoch start time.
        *   *Error: `ErrInvalidBlockTimestampRange` (e.g., "block timestamp is out of acceptable range")*

*   **E. `Nonce` Validation:**
    1.  (Currently `uint64`). If the consensus mechanism (e.g., PoW) uses the nonce, specific range checks might apply. For PoS, if not directly used in header construction for difficulty, it might always be 0 or carry other consensus-specific data. For basic validation, no check beyond type constraints unless a specific rule is set (e.g., must be 0 for PoS headers at this stage).

*   **F. `Difficulty` Validation:**
    1.  (Currently `uint64`). If PoW, must be greater than 0. For PoS, this field might be repurposed or set to a default (e.g., 0 or 1).
        *   *Error: `ErrInvalidDifficulty` (e.g., "difficulty must be greater than zero for PoW blocks")*

*   **G. `Height` Validation:**
    1.  (Currently `uint64`). Must be >= 0.
        *   *(No specific error beyond type constraints, as `uint64` is always >= 0)*.
    2.  *Conceptual Note: Consistency with `PreviousBlockHash` (i.e., `Height` should be `parent.Height + 1`) is a state-dependent check performed during block acceptance.*

*   **H. `ProposerAddress` Validation:**
    1.  Must not be an empty `Address` (empty `[]byte`).
        *   *Error: `ErrMissingProposerAddress` (e.g., "block proposer address cannot be empty")*
    2.  Consider length checks similar to `TransactionOutput.RecipientAddress`.
        *   *Error: `ErrInvalidAddressLength` (e.g., "proposer address has invalid length")*

### 4.5. `Block.Validate() error`

The `Validate()` method for the `Block` struct (`internal/core/types/types.go`) will orchestrate validation of its constituent parts and the block as a whole:

*   **A. `Header` Validation:**
    1.  The `Header` field itself must not be `nil` (though as a struct type in Go, it won't be `nil` unless it's a pointer; assuming `BlockHeader` is a direct struct member).
    2.  Call `Header.Validate()`. If this returns an error, propagate it.
        *   *Error: `ErrBlockHeaderValidationFailed` (wrapping the error from `Header.Validate()`)*

*   **B. `Transactions` Validation:**
    1.  The `Transactions` slice can be empty (e.g., some consensus mechanisms might allow empty blocks).
    2.  If `Transactions` is not empty, iterate through each `Transaction` object (`tx`) in the slice:
        *   Call `tx.Validate()`. If this returns an error, propagate it.
            *   *Error: `ErrBlockTransactionValidationFailed` (wrapping the error from `tx.Validate()` and possibly including the index or ID of the failing transaction)*
    3.  Consider a maximum number of transactions per block (this is a consensus rule, but a sanity check could be here).
        *   *Error: `ErrMaxTransactionsPerBlockExceeded`*
    4.  Consider a maximum total block size in bytes (also a consensus rule, but a check on the serialized size of transactions + header could be conceptualized here).
        *   *Error: `ErrBlockSizeExceeded`*

*   **C. `BlockHash` Validation:**
    1.  Must not be an all-zero `Hash`.
        *   *Error: `ErrInvalidBlockHash` (e.g., "block hash cannot be zero hash")*
    2.  *Conceptual Note: Validating that `BlockHash` is the correctly calculated hash of the (typically serialized) `Header` and/or `Transactions` is a separate, more complex process performed during block acceptance and processing. This `Validate()` method only checks for presence and basic format.*

*   **D. Overall Block Consistency (Conceptual):**
    1.  *Merkle Root Consistency:* The `Header.MerkleRoot` should correctly correspond to the `Transactions` list. This calculation and check is complex and typically done during block processing/verification, not simple struct validation.
    2.  *State Root Consistency (if `StateRoot` is added to `BlockHeader`):* The `Header.StateRoot` should reflect the state of the system after applying all transactions in the block. This is also a post-processing validation.

### 4.6. `Account.Validate() error`

The `Validate()` method for the `Account` struct (`internal/core/types/types.go`) will implement the following stateless checks:

*   **A. `Address` Validation:**
    1.  Must not be an empty `Address` (empty `[]byte`).
        *   *Error: `ErrInvalidAccountAddress` (e.g., "account address cannot be empty")*
    2.  Consider length checks similar to `TransactionOutput.RecipientAddress`.
        *   *Error: `ErrInvalidAddressLength` (e.g., "account address has invalid length")*

*   **B. `Balance` Validation:**
    1.  (Currently `uint64`). Must be >= 0. This is inherently true for `uint64`. No specific check needed unless a maximum possible balance (less than `MaxUint64`) is defined by the protocol.
        *   *(No specific error beyond type constraints unless a max balance is imposed)*

*   **C. `Nonce` Validation:**
    1.  (Currently `uint64`). Must be >= 0. This is inherently true for `uint64`.
        *   *(No specific error beyond type constraints)*

*   **D. `ReputationScore` Validation:**
    1.  (Currently `int64`). Values can be positive, negative, or zero.
    2.  Consider if there are practical minimum or maximum bounds for the reputation score (e.g., `-1,000,000` to `+1,000,000`). If so, validate against these bounds.
        *   *Error: `ErrInvalidReputationScore` (e.g., "reputation score [value] is outside the valid range [min] to [max]")*

---

## 5. Cross-Cutting Concerns for Validation

These concerns apply globally to the implementation of all `Validate()` methods.

### 5.1. Centralized Error Definitions (Sub-Issue 3.1)

*   **Requirement:** All `Validate()` methods MUST return errors that are defined and managed centrally, presumably within an `internal/errors` package (or a similarly named shared errors module).
*   **Structure:** Errors should be specific and, where appropriate, hierarchical. For instance, `ErrTransactionInputValidationFailed` could wrap a more specific error like `ErrMissingInputSignature`.
*   **Benefits:**
    *   **Consistency:** Ensures uniform error handling across the codebase.
    *   **Clarity:** Provides clear, understandable error messages for developers and potentially for logs/users.
    *   **Testability:** Allows for precise error checking in unit tests (e.g., `errors.Is(err, internalerrors.ErrMissingSignature)`).
    *   **Maintainability:** Centralizes error definitions, making it easier to update or add new error types.
*   **Example Usage (Conceptual):**
    ```go
    // In internal/errors/errors.go
    // var ErrMissingSignature = errors.New("missing signature") // Simple version
    // // Or a more structured error type:
    // type ValidationError struct { Code ErrorCode; Message string; Cause error }
    // func (e *ValidationError) Error() string { return e.Message }
    // func NewValidationError(code ErrorCode, msg string, cause error) error { ... }

    // In types/transaction.go Validate() method:
    // if len(tx.Signature) == 0 {
    //     return internalerrors.New(internalerrors.ErrCodeValidation, "transaction signature cannot be empty", internalerrors.ErrIdMissingSignature)
    // }
    ```

### 5.2. Input Sanitization & Canonicalization (Sub-Issue 3.2)

*   **Validation Scope:** The `Validate()` methods are primarily responsible for checking the structural and semantic integrity of data *as presented*. They should **not** modify the data they are validating.
*   **Sanitization:**
    *   **Definition:** The process of cleaning or filtering input data to prevent common issues (e.g., removing leading/trailing whitespace from user-provided strings, normalizing character encodings).
    *   **Placement:** Sanitization should ideally occur at the point of data ingress (e.g., when receiving data from an RPC call, user interface, or external message) *before* the data is used to construct the core `types` structs.
    *   **Example:** If a `Transaction.Metadata` key is user-supplied, it might be trimmed of whitespace before being stored in the `Transaction` struct. The `Validate()` method would then check the (already trimmed) key's length or character set.
*   **Canonicalization:**
    *   **Definition:** The process of converting data that has more than one possible representation into a standard, "canonical" form. This is critical for deterministic hashing and signature verification.
    *   **Placement:** Canonicalization must occur *before* any data is hashed to create an ID (like `Transaction.TxID`) or signed. This is typically part of a specific serialization routine (e.g., `transaction.SerializeForSigning()`).
    *   **Examples:**
        *   Sorting map keys alphabetically before serializing a map (e.g., `Transaction.Metadata` if it's part of the signed payload).
        *   Using a consistent format for encoding numbers (e.g., fixed-size big-endian).
        *   Normalizing address string representations (e.g., always lowercase or always checksummed).
    *   **`Validate()` Role:** The `Validate()` method itself does not perform canonicalization. It validates the struct based on its current state. However, it operates on the assumption that if the struct is to be hashed or signed, the relevant parts will be canonicalized by a different part of the system.

---

## 6. Integration with Unit Testing Strategy (Sub-Issue 4.1)

Robust unit testing is critical to ensure the correctness and reliability of the `Validate()` methods. The testing strategy will be guided by Test-Driven Development (TDD) principles where practical.

*   **Dedicated Test Files:** Each `types/*.go` file containing core data structures will have a corresponding `types/*_test.go` file (e.g., `transaction_test.go` for `transaction.go` if types are split, or `types_test.go` if they remain in one file).
*   **Test Function per `Validate()` Method:** For each struct `S` with a `S.Validate() error` method, there will be a dedicated test function (e.g., `TestTransaction_Validate(t *testing.T)`).
*   **Table-Driven Tests:** Employ table-driven tests for validating multiple scenarios efficiently. Each entry in the table will represent a distinct test case, including:
    *   A descriptive name for the test case.
    *   An instance of the struct being tested, configured to be valid or invalid in a specific way.
    *   The expected error (e.g., `nil` for valid cases, or a specific error type/instance from `internal/errors` for invalid cases).
*   **Comprehensive Coverage:**
    *   **Positive Cases:** At least one test case demonstrating that a valid instance of the struct passes validation (returns `nil`).
    *   **Negative Cases (Crucial):** For *every distinct validation rule and potential error path* identified in Sections 4.1 through 4.6 (e.g., `ErrInvalidFee`, `ErrMissingTransactionOutput`, `ErrInvalidPreviousTxHash`), a specific negative test case MUST be created. This test case will construct an instance of the struct that violates precisely that rule and verify that:
        1.  The `Validate()` method returns a non-`nil` error.
        2.  The returned error is of the expected type (e.g., using `errors.Is()` to check against the specific error variable from `internal/errors` like `internalerrors.ErrInvalidFee`).
*   **Focus on Boundary Conditions:** Test edge cases and boundary conditions (e.g., empty slices where not allowed, zero values for numeric fields where not allowed, maximum length strings/byte slices).
*   **Independence of Tests:** Ensure that test cases are independent and do not rely on shared mutable state or the outcome of other tests.
*   **Clarity of Failure Messages:** Test assertions should provide clear messages on failure, indicating what was expected versus what was received.
*   **Reference to Test Plan Document:**
    *   This validation testing strategy forms a core part of the broader unit testing plan for core data structures.
    *   Detailed test case descriptions, particularly for complex structs like `Transaction`, would reside in a conceptual document like `testing_strategies/core_data_structures_unit_tests.md`. This `core_data_validation_plan.md` confirms the *approach* to testing `Validate()` methods.
*   **Execution:** These unit tests will be run automatically as part of the CI/CD pipeline to ensure no regressions are introduced.

---
*(End of Document)*
