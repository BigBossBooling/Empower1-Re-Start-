# EmPower1 Blockchain - Developer SDKs & Tools: Detailed Technical Specification

## 1. SDK Core Architecture & Structure

### 1.1. SDK Language Bindings

The selection of primary programming languages for EmPower1 Software Development Kits (SDKs) is driven by the goals of maximizing developer accessibility, leveraging existing vibrant ecosystems, and providing appropriate tools for various application domains, from user-facing interfaces to backend services and data analysis. Our choices aim to ensure broad developer adoption and align with EmPower1's principle of "Systematize for Scalability."

The initially supported SDK languages will be:

*   **JavaScript/TypeScript**
    *   **Target Use Cases:** Web frontends (dApps, wallets, explorers), mobile applications (via frameworks like React Native, Ionic), Node.js backend services for dApp orchestration.
    *   **Rationale:** JavaScript, particularly with TypeScript for type safety, boasts the largest global developer community. It is essential for building engaging user interfaces and interactive decentralized applications. The vast array of libraries (e.g., ethers.js, web3.js concepts adapted for EmPower1) and frameworks (React, Angular, Vue) accelerates UI/UX development. This directly supports EmPower1's goal to "Stimulate Engagement, Sustain Impact."

*   **Python**
    *   **Target Use Cases:** Backend services, scripting for automation and operational tasks, data analysis (especially for interacting with AI/ML components and interpreting `AIAuditLog` data), building developer tools, and creating robust testing frameworks.
    *   **Rationale:** Python's ease of use, extensive standard library, and its dominance in the data science and AI/ML fields make it a natural fit for EmPower1. It allows for rapid prototyping of tools and services that interact with the blockchain's intelligent features. This aligns with supporting EmPower1's AI/ML integration and providing tools for "Sense the Landscape."

*   **Go (Golang)**
    *   **Target Use Cases:** High-performance backend services that interact directly with EmPower1 nodes (especially if nodes themselves are in Go), building command-line interface (CLI) tools, and Hashing creates a single hash from all transaction IDs in the block.
// In a production blockchain, this would typically be a Merkle Root.
// TODO: Implement proper Merkle Root calculation for transactions.
func (b *Block) HashTransactions() []byte {
    if len(b.Transactions) == 0 {
        emptyHash := sha256.Sum256([]byte{})
        return emptyHash[:]
    }

    var txHashes [][]byte
    for _, tx := range b.Transactions {
        if len(tx.ID) == 0 {
            // This case should ideally not happen if transactions are finalized
            // and signed before being added to a block.
            // For robustness in this V1, hash a placeholder if ID is missing.
            // In a stricter system, this might be an error.
            placeholderForMissingID := sha256.Sum256([]byte("MISSING_TX_ID"))
            txHashes = append(txHashes, placeholderForMissingID[:])
        } else {
            txHashes = append(txHashes, tx.ID)
        }
    }

    // Simple concatenation and hash for V1 - not a proper Merkle root
    // TODO: Implement sorting of txHashes before concatenation for determinism.
    var combinedTxHashes []byte
    for _, txHash := range txHashes {
        combinedTxHashes = append(combinedTxHashes, txHash...)
    }
    finalHash := sha256.Sum256(combinedTxHashes)
    return finalHash[:]
}

// Serialize uses gob encoding to convert the Block struct into a byte slice.
func (b *Block) Serialize() ([]byte, error) {
    var result bytes.Buffer
    encoder := gob.NewEncoder(&result)
    // Ensure all fields of Block, including []Transaction, are gob-encodable.
    // Transaction already has Serialize/Deserialize, but gob directly encodes structs.
    err := encoder.Encode(b)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrBlockSerialization, err)
    }
    return result.Bytes(), nil
}

// DeserializeBlock converts a byte slice (previously gob encoded) back into a Block struct.
func DeserializeBlock(data []byte) (*Block, error) {
    var b Block
    decoder := gob.NewDecoder(bytes.NewBuffer(data))
    err := decoder.Decode(&b)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrBlockDeserialization, err)
    }
    return &b, nil
}


// TODO: Implement SetHash(hash []byte) method
// TODO: Implement Sign(privateKey []byte) error method (requires crypto library for signing)
// TODO: Implement VerifySignature(publicKey []byte) (requires crypto library for verification)
```

**Part 2: Modify `internal/consensus/proposer.go`**
1.  Change `CreateProposalBlock` to use `core.CalculateBlockHash` (the one defined in `genesis.go` but conceptually should be a core utility or block method).
    *   This means `CalculateBlockHash` needs to be accessible, e.g., by moving it to `internal/core/block.go` as a public function or method, or making `internal/blockchain/genesis.go`'s version public and importing `blockchain` package in `proposer.go`.
    *   **For this subtask, assume `core.CalculateBlockHash` is made available.** The subtask will modify `proposer.go` to *call* `core.CalculateBlockHash(newBlock)`. The actual moving of `CalculateBlockHash` is a separate concern.
    *   The call will replace the local hashing logic currently in `CreateProposalBlock`.

    ```go
    // In internal/consensus/proposer.go, inside CreateProposalBlock:
    // ... after block is assembled and Signature is set ...
    // Replace local hashing logic with:
    blockHash, err := core.CalculateBlockHash(newBlock) // Assumes CalculateBlockHash moved to core
    if err != nil {
        return nil, fmt.Errorf("failed to calculate proposal block hash: %w", err)
    }
    newBlock.Hash = blockHash
    // ...
    ```
2.  Ensure `empower1.com/empower1blockchain/internal/core` is imported in `proposer.go` (it should be already).

**Part 3: Modify `internal/blockchain/genesis.go`**
1.  Move the `CalculateBlockHash` function from `internal/blockchain/genesis.go` to `internal/core/block.go`.
    *   This makes it a general utility for blocks.
    *   Update its signature if it becomes a method on `*core.Block`, e.g., `(b *core.Block) CalculateAndSetHash() error` or keep it as a package function `core.CalculateBlockHash(b *core.Block) ([]byte, error)`. The latter is simpler for now.
    *   Ensure `genesis.go` now calls `core.CalculateBlockHash` (or the block method).

**Part 4: Update `cmd/empower1d/main.go` (Conceptual Check)**
1.  No direct code changes, but ensure the conceptual flow remains valid: `ProposerService` creates blocks with hashes, `ConsensusEngine` uses these blocks, `Blockchain` adds them. The `CalculateBlockHash` being in `core` makes it accessible to all.

**Verification:**
- `internal/core/block.go` has the `CalculateBlockHash` function moved into it (and `Serialize`/`DeserializeBlock` from previous plan).
- `internal/blockchain/genesis.go` no longer defines `CalculateBlockHash` but calls the one from `core`.
- `internal/consensus/proposer.go` in `CreateProposalBlock` now calls `core.CalculateBlockHash`.
- All files compile conceptually.

This refactoring centralizes block hash calculation.
The `errors` import might be needed in `core/block.go` if `CalculateBlockHash` returns an error.
The `CalculateBlockHash` in `genesis.go` was:
```go
func CalculateBlockHash(block *core.Block) ([]byte, error) {
	if block == nil { return nil, errors.New("cannot calculate hash of nil block") }
	headerBytes := block.HeaderForSigning()
	var dataToHash []byte
	dataToHash = append(dataToHash, headerBytes...)
	if block.Signature != nil { dataToHash = append(dataToHash, block.Signature...) }
	hash := sha256.Sum256(dataToHash)
	return hash[:], nil
}
```
This looks good to move. It uses `block.HeaderForSigning()` which is a method of `*core.Block`.
This function is suitable to be a package-level function in `core` or a method on `*core.Block`. Let's make it a package-level function `core.CalculateBlockHash` for now.
The TODOs in `block.go` for `SetHash`, `Sign`, `VerifySignature` are still relevant for future work.
This subtask ensures `CalculateBlockHash` is centralized in `core` and used by `genesis` and `proposer` services.

[end of docs/specifications/EmPower1_SDK_Tools_Specification.md]

[end of docs/specifications/EmPower1_SDK_Tools_Specification.md]

[end of docs/specifications/EmPower1_SDK_Tools_Specification.md]

[end of docs/specifications/EmPower1_SDK_Tools_Specification.md]

[end of docs/specifications/EmPower1_SDK_Tools_Specification.md]
