# DigiSocialBlock (Nexus Protocol) - Content Hashing & Timestamping Technical Specification

## 1. Objective

This document provides the detailed technical specifications for the Content Hashing and Timestamping protocol within DigiSocialBlock. This protocol defines how user-generated content (primarily `ContentPost` messages) receives a unique, verifiable cryptographic fingerprint (Content ID, e.g., `post_id`) and how its creation time is asserted and subsequently validated by the network. This is fundamental for content immutability, verifiability, and enabling the Proof-of-Witness (PoW) protocol.

## 2. Scope

This specification covers:
*   The definition of a canonical representation for content types (initially `ContentPost`) for the purpose of hashing.
*   The cryptographic hashing algorithm used to generate unique Content IDs.
*   The process and sources of timestamping, distinguishing between client-asserted time and network-observed time (via PoW).
*   The interaction of these mechanisms with other protocols like DDS and PoW.
*   Data structures involved (primarily referencing existing ones and clarifying field usage).

This document builds upon the conceptual design outlined in `tech_specs/dli_echonet_protocol.md` (Section 4).

## 3. Relation to `dli_echonet_protocol.md`

This specification directly elaborates on the principles and components introduced in Section 4 ("Module 1.4: Content Hashing & Timestamping") of the `dli_echonet_protocol.md` document. It provides the detailed technical underpinnings for generating immutable content identifiers and establishing verifiable timestamps.

---

## 4. Canonical Content Representation for Hashing

To ensure that identical semantic content always produces the same Content ID (e.g., `post_id`), a strict canonical representation of the content's core data must be defined before hashing. This representation must be deterministic.

### 4.1. Target Content Type: `ContentPost`

Initially, this specification focuses on the `ContentPost` message type defined in `echonet_v3_core.proto`. Similar principles can be applied to other user-generated content types (e.g., `UserProfile` updates, `Comment`s) in the future.

### 4.2. Fields Included in Canonical Hash for `ContentPost`

The following fields from the `ContentPost` message (as defined in the conceptual `echonet_v3_core.proto`) **MUST** be included in the canonical data used for generating the `post_id`. The order listed here is the **RECOMMENDED** order for serialization if a custom binary format is used. If Protobuf serialization of a subset message is used, the field tag numbers will dictate order.

1.  **`author_id` (string - UUID):** The identifier of the content creator.
2.  **`parent_post_id` (string - UUID, optional):** If the post is a reply, this is the ID of the parent post. An empty string or a designated nil-UUID string if not a reply.
3.  **`community_group_id` (string - UUID, optional):** If the post is targeted at a specific community group. An empty string or nil-UUID if not.
4.  **`created_at` (google.protobuf.Timestamp):** The client-asserted creation timestamp. Serialized as Unix nanoseconds (int64).
5.  **`content_type` (enum - int32):** The type of the content (e.g., TEXT, IMAGE). Serialized as its `int32` value.
6.  **`visibility` (enum - int32):** The visibility setting for the post. Serialized as its `int32` value.
7.  **`text_content` (string, conditional):** The textual content of the post. Only included if `content_type` indicates text (e.g., TEXT, ARTICLE). If not applicable, an empty string should be hashed, or this field is omitted from the canonical message subset.
8.  **`media_url` (string, conditional):** The URL for media content. Only included if `content_type` indicates media (e.g., IMAGE, VIDEO, AUDIO). If not applicable, an empty string should be hashed or omitted.
9.  **`tags` (repeated string, optional):**
    *   Tags **MUST** be normalized (e.g., to lowercase, trimmed whitespace) before inclusion.
    *   The list of normalized tags **MUST** be sorted alphabetically (lexicographically) before serialization.
    *   Serialized as a concatenation or a length-prefixed list of strings.
10. **(Future Consideration) `content_specific_metadata` (map<string, string>, optional):** If a field for structured, content-defining metadata (distinct from DDS manifest metadata or application-layer display metadata) is added to `ContentPost`, its keys MUST be sorted, and key-value pairs serialized deterministically.

### 4.3. Fields Excluded from Canonical Hash for `ContentPost` `post_id`

The following fields are **EXCLUDED** from the data used to generate the `post_id` because they are either derived, stateful, server-managed, or not intrinsic to the core identity of the content itself:

*   `post_id` (this is the hash being generated).
*   `updated_at` (represents last modification time, not creation identity).
*   `reaction_counts` (dynamic, server-aggregated).
*   `comment_count` (dynamic, server-aggregated).
*   `share_count` (dynamic, server-aggregated).
*   Any other purely server-side or mutable operational data.

### 4.4. Serialization Format for Canonical Representation

*   **Primary Method: Protobuf Subset Message:**
    *   Define a new Protobuf message (e.g., `ContentPostHashingPayload`) that includes *only* the fields listed in Section 4.2, in the specified order (using Protobuf field tag numbers to enforce order).
    *   Populate an instance of this subset message from the full `ContentPost`.
    *   Serialize this `ContentPostHashingPayload` message using standard Protobuf binary serialization. This output is the canonical byte array for hashing.
    *   **Rationale:** Leverages Protobuf's well-defined and deterministic binary serialization rules (for a given schema and field order). Ensures cross-language compatibility if hashing is ever performed by different client implementations.
*   **Alternative (Manual Binary Concatenation - More Error Prone):**
    *   If not using a Protobuf subset message, each field must be serialized to bytes in a strictly defined manner:
        *   Strings (UUIDs, text, URLs): UTF-8 encoded bytes. Prepend with length.
        *   Timestamps (`created_at`): Convert to `int64` Unix nanoseconds, then serialize as fixed-size (e.g., 8 bytes) big-endian or little-endian (must be consistent).
        *   Enums (`content_type`, `visibility`): Serialize their `int32` values as fixed-size (e.g., 4 bytes) big-endian or little-endian.
        *   Repeated strings (`tags`): After normalization and sorting, serialize each tag (length-prefixed UTF-8) and concatenate them. Or, serialize the count of tags, then each tag.
    *   Concatenate these byte arrays in the exact order specified in Section 4.2.
    *   **This method is strongly discouraged due to its complexity and higher risk of non-deterministic output if not implemented perfectly across all clients.**

**Recommendation:** Use the Protobuf Subset Message approach for its robustness and cross-platform determinism.

---

## 5. Hashing Algorithm & Content ID (Post ID)

This section defines the cryptographic hashing algorithm used to generate the primary application-level identifier for content, such as `post_id` for `ContentPost` messages.

### 5.1. Hashing Algorithm

*   **Specification:** The cryptographic hash function to be used is **SHA-256 (Secure Hash Algorithm 2, 256-bit)**.
*   **Input:** The input to the SHA-256 function will be the canonical byte array generated from the content's core fields, as specified in Section 4 (Canonical Content Representation for Hashing).
*   **Output:** The output of SHA-256 is a 256-bit (32-byte) hash value.
*   **Rationale:** SHA-256 is a widely adopted, secure, and well-vetted cryptographic hash function offering strong collision resistance and pre-image resistance. It is suitable for generating unique content identifiers.

### 5.2. Content ID (e.g., `post_id`) Generation

*   **Definition:** The primary application-level Content ID for a piece of content (e.g., `post_id` for a `ContentPost`) is derived directly from its content hash.
*   **Process:**
    1.  Obtain the canonical byte representation of the content's core data (see Section 4.4, preferably using the Protobuf Subset Message method).
    2.  Compute the SHA-256 hash of this canonical byte array. This results in a 32-byte binary hash.
    3.  **Encode the binary hash using Base58BTC.** This encoded string will serve as the human-readable and network-transmissible `post_id` (or equivalent Content ID).
*   **Example:**
    *   Canonical Data Bytes: `[byte_array_of_canonical_content_post_data]`
    *   Binary SHA-256 Hash: `sha256_binary = SHA256(byte_array_of_canonical_content_post_data)` (32 bytes)
    *   `post_id` (string): `Base58BTCEncode(sha256_binary)`
*   **Immutability Link:** Because the `post_id` is a direct cryptographic hash of the content's core data (including the client-asserted `created_at` timestamp), any change to this core data will result in a different `post_id`. This ensures that a given `post_id` always refers to one specific, immutable version of the content as defined at its creation.

### 5.3. Distinction from DDS CIDs

It is crucial to distinguish this application-level Content ID (`post_id`) from the Content Identifiers (CIDs) used in the Distributed Data Stores (DDS) protocol:

*   **DDS CIDs (Section 4.2 of DDS Spec):**
    *   Identify individual data chunks (e.g., 256KiB pieces of the serialized `ContentPost` *message* or its raw content).
    *   Identify the `ContentManifest` file (which lists chunk CIDs).
    *   Used for storage, discovery, and retrieval of data blobs within the DDS.
*   **Content ID / `post_id` (This Specification):**
    *   Identifies the semantic content at the application layer.
    *   Derived from a canonical representation of *core data fields*, not necessarily the full serialized message that might be chunked for DDS.
    *   Used by the application, users, and other protocols (like PoW) to uniquely reference the conceptual piece of content.

**Relationship:**
The `post_id` is generated first by the originator. Then, the full `ContentPost` message (which now includes this `post_id`) is prepared for storage in DDS. This might involve serializing the `ContentPost` message to bytes, then chunking *that* serialized message, generating DDS CIDs for those chunks, creating a DDS manifest, etc. The PoW protocol would then typically operate on the manifest CID from the DDS.

---

## 6. Timestamping Process & Sources

Establishing a verifiable and consistent sense of time for content creation is crucial for ordering, discovery, and supporting protocols like Proof-of-Witness (PoW). DigiSocialBlock will utilize two primary timestamps associated with content:

### 6.1. Initial Client Timestamp (`created_at` in `ContentPost`)

*   **Source:** Asserted by the client application or Originator Node at the moment the content is finalized by the user, before its canonical hash (`post_id`) is generated.
*   **Definition:** This timestamp is embedded within the `ContentPost` message as the `created_at` field (of type `google.protobuf.Timestamp`).
*   **Inclusion in Hash:** As specified in Section 4.2, the `created_at` field (serialized as Unix nanoseconds) **IS INCLUDED** in the data used to generate the `post_id`.
    *   **Implication:** This means the `post_id` cryptographically commits to this client-asserted creation time. Any change to this timestamp would result in a different `post_id`.
*   **Trust Assumption:** This timestamp is initially trusted as the user's claimed creation time. Its veracity against "real-world time" is then subject to network validation via the PoW protocol.
*   **Clock Sync:** Clients should endeavor to have reasonably synchronized clocks (e.g., via NTP) to provide accurate initial timestamps. However, the protocol must be resilient to some level of clock skew.

### 6.2. Network Observed Timestamp (`earliest_observed_timestamp` in `PoWClaim`)

*   **Source:** Derived from multiple `WitnessAttestation.observed_timestamp` values collected and validated within a `PoWClaim` object, as defined by the Proof-of-Witness (PoW) protocol (Module 1.3).
*   **Definition:** This represents the earliest time the content (identified by its `manifest_cid`, which is derived from the `ContentPost` that includes the `post_id` and `client_created_at`) was verifiably observed and attested to by a set of `M` independent Witness Nodes.
*   **Calculation:** The exact method for calculating the `earliest_observed_timestamp` from multiple witness attestations (e.g., earliest valid, median after outlier rejection) is defined in the PoW protocol specification (Section 6.3 of `POW_PROTOCOL_SPEC.md`). For MVP, it's the earliest valid `observed_timestamp`.
*   **Relationship to `post_id`:** This timestamp is **NOT** part of the `post_id` hash. It is established *after* the content (and its `post_id`) exists and has been propagated to the DDS and observed by witnesses.
*   **Trust Level:** This timestamp carries a higher degree of network-validated trust regarding the content's "first seen by the network" time, as it's corroborated by multiple independent parties (Witnesses).

### 6.3. Relationship and Usage

*   **`client_created_at` (in `ContentPost`):**
    *   Forms part of the content's immutable identity (`post_id`).
    *   Represents the user's claim of when the content was created.
    *   Can be used for initial sorting or display before full PoW validation.
*   **`network_observed_at` (in `PoWClaim`):**
    *   Provides a network-verified timestamp for when the content (with its specific `post_id` and `client_created_at`) was first widely observed and attested to.
    *   Serves as the primary timestamp for establishing content originality sequence in the context of the DigiSocialBlock network.
    *   Used by ranking algorithms, reward systems (Proof-of-Engagement), and potentially for dispute resolution regarding content appearance order.
*   **Validation Flow:**
    1.  Client creates content, sets `client_created_at`.
    2.  `post_id` is generated, including `client_created_at` in its hash.
    3.  Content (referenced by its DDS manifest CID) is witnessed. Witnesses record their `observed_timestamp`.
    4.  A `PoWClaim` is generated, which includes the `network_observed_at` (e.g., `earliest_observed_timestamp`).
    5.  Applications can then compare `client_created_at` with `network_observed_at`. Significant discrepancies might indicate issues (e.g., client clock wildly off, attempt to backdate content that was held back before publishing). The PoW protocol (Section 8 of `POW_PROTOCOL_SPEC.md`) might define acceptable tolerances.

This dual-timestamp approach provides both a user-asserted immutable creation time (part of content ID) and a network-validated observation time, offering a robust framework for temporal ordering and content authenticity.

---

## 7. Data Structures (Relevant to Hashing & Timestamping)

This module primarily utilizes and clarifies the usage of fields within existing data structures defined in `echonet_v3_core.proto` (for content like `ContentPost`) and `pow_protocol.proto` (for `PoWClaim`). No new primary Protobuf messages are introduced solely for this Content Hashing & Timestamping module.

### 7.1. Key Fields in `core_types.ContentPost`

*   **`post_id` (string):**
    *   **Source:** Generated by the Originator Node.
    *   **Definition:** The Base58BTC encoded SHA-256 hash of the canonical representation of the content's core fields (as defined in Section 4 of this document).
    *   **Role:** The primary, immutable, application-level unique identifier for the content.
*   **`created_at` (google.protobuf.Timestamp):**
    *   **Source:** Set by the Originator Node's client application.
    *   **Definition:** The client-asserted timestamp of content creation.
    *   **Role:** Part of the data that forms the `post_id`, making the `post_id` a commitment to this initial timestamp.

### 7.2. Key Fields in `pow_protocol.PoWClaim`

*   **`manifest_cid` (string):**
    *   **Source:** From the DDS manifest of the content (which is derived from the full `ContentPost` message that includes the `post_id` and `client_created_at`).
    *   **Role:** Links the PoW claim back to the specific version of content stored in DDS.
*   **`content_hash_from_manifest` (string):**
    *   **Source:** Copied from the `ContentManifest.original_content_hash`.
    *   **Role:** Ensures the PoW claim is tied to the verified hash of the full original content.
*   **`earliest_observed_timestamp` (google.protobuf.Timestamp):**
    *   **Source:** Calculated by the PoW Claim generator based on valid `WitnessAttestation.observed_timestamp` values.
    *   **Definition:** The network-validated "first-seen" timestamp for the content associated with `manifest_cid`.
    *   **Role:** The primary timestamp used by the network for establishing temporal order and originality.

### 7.3. Conceptual `ContentPostHashingPayload` (Internal - Not a Network Message)

As recommended in Section 4.4, a Protobuf message could be defined internally (e.g., in a shared utility `.proto` file or directly in Go code if only used by Go services) to ensure deterministic serialization for hashing. This message is **not** typically exchanged over the network itself; it's an intermediate structure for the hashing process.

*   **Example Conceptual Protobuf Definition:**
    ```protobuf
    // internal_hashing_types.proto
    message ContentPostHashingPayload {
      string author_id = 1;
      string parent_post_id = 2;        // Ensure consistent handling of empty/nil
      string community_group_id = 3;    // Ensure consistent handling of empty/nil
      int64 created_at_unix_nano = 4;   // Store as int64 Unix nanoseconds
      int32 content_type = 5;           // Raw enum int32 value
      int32 visibility = 6;             // Raw enum int32 value
      string text_content = 7;          // Empty if not applicable
      string media_url = 8;             // Empty if not applicable
      repeated string sorted_tags = 9;  // Normalized and sorted tags
      // map<string, string> sorted_content_specific_metadata = 10; // If applicable, keys sorted
    }
    ```
*   **Usage:**
    1.  Populate this `ContentPostHashingPayload` from the main `ContentPost` object.
    2.  Serialize `ContentPostHashingPayload` using Protobuf's deterministic binary marshaller.
    3.  SHA-256 hash the resulting bytes.
    4.  Base58BTC encode the hash to get the `post_id`.

This approach ensures that the data included in the hash and its serialization order are strictly defined by a schema, minimizing cross-implementation discrepancies.

---

## 8. Network Interactions & Process Flow

This section outlines the sequence of operations an Originator Node performs, integrating Content Hashing, Timestamping, DDS, and PoW.

**Content Creation & Initial Publication Flow:**

1.  **Client-Side Content Preparation:**
    *   User finalizes content using a client application.
    *   Client application populates a `core_types.ContentPost` message, including:
        *   `author_id` (user's ID).
        *   `parent_post_id` (if a reply).
        *   `community_group_id` (if applicable).
        *   `content_type`, `visibility`, `text_content`, `media_url`, `tags`.
        *   **`created_at`:** Set to the current UTC time by the client (`google.protobuf.Timestamp`).
    *   The `post_id` field is initially empty.

2.  **Canonicalization & Content ID (`post_id`) Generation (Originator Node/Client):**
    *   The Originator Node takes the partially filled `ContentPost` data.
    *   It constructs the canonical representation of the core fields (as defined in Section 4, e.g., by populating a `ContentPostHashingPayload` message). This includes converting `created_at` to Unix nanoseconds and normalizing/sorting `tags`.
    *   It serializes this canonical payload to bytes (e.g., using Protobuf binary marshalling).
    *   It computes the SHA-256 hash of these canonical bytes.
    *   It Base58BTC encodes this hash to generate the unique `post_id` string.

3.  **Finalize `ContentPost` Message:**
    *   The generated `post_id` is now set in the `ContentPost` message.
    *   The Originator Node cryptographically signs the `post_id` (or the canonical data hash) using their private key. This signature can be stored as part of the `ContentPost` (e.g., in a `author_signature` field, if added to the proto) or associated with it via other means. *This step is more about author authenticity than the content hash itself and might be a separate layer.* For now, we focus on `post_id` as the content's fingerprint.

4.  **DDS Preparation & Storage (Originator Node -> DDS Network):**
    *   The complete `ContentPost` Protobuf message (now including its `post_id` and `created_at`) is serialized into a byte array. This byte array is the "original content blob" for DDS purposes.
    *   The Originator Node's DDS component:
        *   Calculates the SHA-256 hash of this "original content blob" (this will be the `original_content_hash` for the DDS manifest).
        *   Splits the "original content blob" into 256KiB chunks.
        *   Calculates the SHA-256 CID (Base58BTC encoded) for each chunk.
        *   Creates a `dds_protocol.ContentManifest` containing the list of chunk CIDs, total size, and the `original_content_hash`.
        *   Serializes the `ContentManifest` and calculates its CID (`manifest_cid`).
    *   The Originator Node stores all data chunks and the manifest chunk locally via its `StorageManager`.
    *   The Originator Node then seeds these chunks (data + manifest) to `k` initial Storage Nodes in the DDS network using `StoreChunkRequest` messages.
    *   The Originator Node (and the initial `k` Storage Nodes) advertise all these CIDs (data chunks + manifest chunk) to the DHT using `Provide(cid)`.

5.  **Proof-of-Witness (PoW) Initiation (Originator Node -> PoW Network):**
    *   Once the `manifest_cid` is available and content is seeded to DDS:
        *   The Originator Node (or Witness Nodes passively observing the DHT) identifies the `manifest_cid` for witnessing.
        *   Witness Nodes retrieve the content via DDS (using `manifest_cid`), verify it, and generate `WitnessAttestation` objects (as per PoW Protocol Spec, Section 5). These attestations include their `observed_timestamp`.
        *   The Originator Node (or a client) gathers these attestations (e.g., via `GetAttestationsRequest` to witnesses or a DHT lookup for attestations).
        *   The Originator Node generates a `PoWClaim`, which includes the `manifest_cid`, `content_hash_from_manifest` (which matches the hash of the full serialized `ContentPost`), and the `earliest_observed_timestamp` derived from valid attestations.

**Summary of Identifiers and Timestamps in Flow:**

*   **`ContentPost.created_at`**: Set by client, part of `post_id` hash.
*   **`ContentPost.post_id`**: Hash of core fields including `client_created_at`. Immutable identifier for the semantic content.
*   **DDS Chunk CIDs**: Hashes of individual 256KiB chunks of the *serialized `ContentPost` message*.
*   **DDS `ContentManifest.original_content_hash`**: Hash of the *entire serialized `ContentPost` message*.
*   **DDS `manifest_cid`**: Hash of the *serialized `ContentManifest`*. This is the primary ID used to retrieve content from DDS and for PoW.
*   **`WitnessAttestation.observed_timestamp`**: Set by each witness upon their verification.
*   **`PoWClaim.earliest_observed_timestamp`**: Derived from multiple witness attestations, serves as network-validated "first-seen" time for the content identified by `manifest_cid`.

This flow ensures that the application-level `post_id` is tied to the initial content and client timestamp, while the DDS handles storage of the (potentially larger) full post object, and PoW validates its network observation time.

---

## 9. Initial Implementation Plan (Conceptual)

This section outlines a high-level plan for the initial Go implementation of the Content Hashing & Timestamping logic.

### 9.1. Core Go Package Structure & Key Functions (Proposed)

The core logic for content hashing and ID generation could reside in a utility package (e.g., `internal/contentutil`) or as methods directly on the generated Protobuf types if helper files are used for custom logic.

**Proposed Location:** `internal/contentutil/hasher.go` (or similar)

**Key Functions:**

1.  **`CanonicalizeContentPost(post *core_types.ContentPost) ([]byte, error)`:**
    *   **Input:** A pointer to a `core_types.ContentPost` message (as generated from Protobuf).
    *   **Process:**
        *   Creates an instance of the internal `ContentPostHashingPayload` Protobuf message (or a similar Go struct designed for canonical serialization).
        *   Populates this payload struct strictly with the fields specified in Section 4.2, taken from the input `post`. This involves:
            *   Copying `author_id`, `parent_post_id`, `community_group_id`.
            *   Converting `post.CreatedAt` (`*timestamppb.Timestamp`) to `int64` Unix nanoseconds.
            *   Copying `content_type` and `visibility` enum `int32` values.
            *   Copying `text_content` and `media_url` (handling conditional inclusion based on `content_type` by populating with empty strings if not applicable for the payload).
            *   Normalizing (e.g., lowercase, trim) and sorting `post.Tags` before adding to the payload's `sorted_tags` field.
            *   (If applicable) Normalizing and sorting `post.ContentSpecificMetadata` map keys and adding to payload.
        *   Serializes the populated `ContentPostHashingPayload` message using Protobuf binary marshalling.
    *   **Output:** The canonical byte slice, or an error if population or serialization fails.

2.  **`GenerateContentID(canonicalData []byte) (string, error)`:**
    *   **Input:** The canonical byte slice generated by `CanonicalizeContentPost`.
    *   **Process:**
        *   Computes the SHA-256 hash of `canonicalData`.
        *   Base58BTC encodes the binary hash.
    *   **Output:** The Base58BTC encoded Content ID string, or an error if hashing/encoding fails (unlikely for standard library functions if input is valid).
    *   This function can be generic for any canonical data, not just `ContentPost`.

3.  **`GenerateAndSetPostID(post *core_types.ContentPost) error` (Convenience Helper):**
    *   **Input:** A pointer to a `core_types.ContentPost` (where `post_id` might be empty).
    *   **Process:**
        1.  Calls `CanonicalizeContentPost(post)` to get the canonical data.
        2.  Calls `GenerateContentID(canonicalData)` to get the `post_id` string.
        3.  Sets `post.PostId = generated_post_id_string`.
    *   **Output:** An error if any underlying step fails.

**Helper for Sorting (if not using Protobuf subset for everything):**
*   If manual serialization (discouraged) of repeated fields like `tags` is chosen over a Protobuf subset message, helper functions for normalizing and sorting string slices will be needed (e.g., `validationutils` could host these).

---

### 9.2. Core Component Implementation Details for MVP

This section details the MVP implementation focus for the key functions.

**1. `CanonicalizeContentPost(post *core_types.ContentPost) ([]byte, error)` Implementation:**
    *   **Priority:** High. This is the core of deterministic ID generation.
    *   **Method:**
        *   Define an internal Go struct (e.g., `contentPostHashingPayload`) that mirrors the fields and order specified in Section 4.2 (e.g., `AuthorId string`, `ParentPostId string`, `CreatedAtUnixNano int64`, etc.).
        *   Populate this struct from the input `post *core_types.ContentPost`.
            *   Handle optional fields: if `post.ParentPostId` is empty, the payload's `ParentPostId` should be an empty string.
            *   Timestamps: Convert `post.CreatedAt.AsTime().UnixNano()` (or `post.CreatedAt.GetSeconds()*1e9 + int64(post.CreatedAt.GetNanos())`).
            *   Enums: Use their `int32` values.
            *   `Tags`: Normalize (e.g., `strings.ToLower(strings.TrimSpace(tag))`) then sort the `post.Tags` slice before assigning/encoding.
        *   Use `encoding/gob` (or a similar stable binary encoder) to serialize this `contentPostHashingPayload` struct. Ensure field order is implicitly or explicitly maintained by the encoder or struct definition for determinism if not using Protobuf for this internal step.
        *   **Alternative/Recommended:** If a `.proto` file for `ContentPostHashingPayload` (as suggested in Sec 7.3) is created, use `proto.Marshal` on an instance of that generated Go type for robust, schema-defined deterministic serialization.
    *   **Error Handling:** Return errors for nil input `post`, issues during timestamp conversion (unlikely if `post.CreatedAt` is validated), or serialization errors.

**2. `GenerateContentID(canonicalData []byte) (string, error)` Implementation:**
    *   **Priority:** High.
    *   **Method:**
        1.  `hash := sha256.Sum256(canonicalData)`
        2.  `cidStr := base58.Encode(hash[:])` (using a Base58BTC compatible library, e.g., `github.com/btcsuite/btcutil/base58`).
    *   **Error Handling:** Minimal, as SHA256 and Base58 encoding of fixed-size hash are unlikely to fail with valid inputs. Check for `canonicalData == nil`.

**3. Integration into Originator Node Workflow (Conceptual):**
    *   An Originator Node, upon receiving user content to create a new `ContentPost`:
        1.  Populates a `core_types.ContentPost` struct (`tempPost`) with all user-provided data and sets `tempPost.CreatedAt = timestamppb.Now()`. `tempPost.PostId` is empty.
        2.  `canonicalBytes, err := contentutil.CanonicalizeContentPost(tempPost)`
        3.  `postIDStr, err := contentutil.GenerateContentID(canonicalBytes)`
        4.  `tempPost.PostId = postIDStr`
        5.  (Optional but recommended) `tempPost.AuthorSignature = sign(postIDStr, authorPrivateKey)` // Sign the content ID
        6.  The now complete `tempPost` object is then passed to the DDS module:
            *   DDS module serializes `tempPost` into a single byte blob (`fullPostBytes`).
            *   DDS module calculates `originalContentHash = sha256(fullPostBytes)`.
            *   DDS module chunks `fullPostBytes`, gets chunk CIDs, creates `ContentManifest` (including `originalContentHash`), gets `manifestCid`.
            *   DDS module stores and Provides chunks & manifest.
        7.  The `manifestCid` (from DDS) and the `postIDStr` (Content ID) are now available. The `manifestCid` is typically used to initiate PoW. The `postIDStr` is the application-level ID.

This implementation sequence ensures the Content ID (`post_id`) is fixed based on core content before the full object (which includes this ID) is stored in DDS.

---

### 9.3. MVP Scope for Content Hashing & Timestamping

The Minimum Viable Product (MVP) for this module will focus on establishing the core hashing and client-side timestamping mechanisms for `ContentPost` objects.

**Included in MVP:**

1.  **`ContentPost` Focus:** All canonicalization and Content ID (`post_id`) generation logic will be implemented and tested specifically for the `core_types.ContentPost` message.
2.  **Canonicalization Implementation:**
    *   Full implementation of `CanonicalizeContentPost(post *core_types.ContentPost) ([]byte, error)`.
    *   This includes deterministic serialization of all specified fields: `author_id`, `parent_post_id`, `community_group_id`, `created_at` (as Unix nanoseconds), `content_type` (as int32), `visibility` (as int32), conditional `text_content`/`media_url`, and normalized (lowercase, trimmed) and alphabetically sorted `tags`.
    *   The primary method will be to define and use an internal Protobuf message (e.g., `ContentPostHashingPayload`) for serialization to ensure cross-language determinism if clients in other languages are anticipated.
3.  **Content ID Generation Implementation:**
    *   Full implementation of `GenerateContentID(canonicalData []byte) (string, error)` using SHA-256 and Base58BTC encoding.
4.  **Client-Asserted Timestamp (`created_at`):**
    *   The `ContentPost` struct will include the `created_at` field (type `google.protobuf.Timestamp`).
    *   Originator clients will be responsible for populating this field with the current UTC time before `post_id` generation.
    *   This `created_at` value (as Unix nanoseconds) will be part of the canonical data hashed for `post_id`.
5.  **Integration Point with DDS/PoW:**
    *   The process flow where `post_id` is generated first, then the full `ContentPost` object (containing this `post_id`) is prepared for DDS storage (chunking, manifest creation leading to a `manifest_cid`), and this `manifest_cid` is then used for PoW, will be clearly documented and followed in conceptual integration.
    *   The distinction that `post_id` hashes core semantic content + client timestamp, while `manifest_cid` (for PoW) references the DDS storage object of the *entire* `ContentPost` message, is key.

**Excluded from MVP (for this specific Hashing & Timestamping module):**

*   **Hashing for Other Content Types:** Canonicalization and ID generation for `UserProfile` updates, `Comments`, etc., will be deferred. The pattern established for `ContentPost` will be reused.
*   **Network Observed Timestamp Implementation:** The generation and management of the `network_observed_at` (i.e., `PoWClaim.earliest_observed_timestamp`) is the responsibility of the PoW protocol (Module 1.3) and is not implemented by this module. This module only clarifies its relationship.
*   **Advanced Canonicalization:** Complex normalization for rich text, deep analysis of media content for fingerprinting beyond URL, or handling of evolving schemas with backward compatibility for hashing are out of MVP scope.
*   **Author Signature of Content ID:** While mentioned as a good practice in the workflow (Section 9.2, Step 3 Integration), the actual implementation of cryptographic signing by the author is part of identity management or an application service layer, not strictly this hashing module. This module focuses on generating the ID to *be* signed.

The MVP will ensure that any DigiSocialBlock node or client can reliably and deterministically generate a unique `post_id` for any given `ContentPost` based on its core, immutable attributes and client-provided creation time.

---

### 9.4. Unit Test Strategy for Content Hashing & Timestamping MVP

Unit tests are essential to ensure the correctness, determinism, and robustness of the hashing and ID generation logic.

**Target Package:** `internal/contentutil` (or wherever the core functions are implemented).

**1. Tests for `CanonicalizeContentPost(post *core_types.ContentPost) ([]byte, error)`:**
    *   **`TestCanonicalizeContentPost_Determinism`:**
        *   Create two identical `core_types.ContentPost` structs. Verify that `CanonicalizeContentPost` produces the exact same byte slice for both.
        *   Create two `core_types.ContentPost` structs that are semantically identical but have different internal ordering for `Tags` (e.g., `{"b", "a"}` vs. `{"a", "b"}`) or `ContentSpecificMetadata` map (if included). Verify that `CanonicalizeContentPost` (due to its internal sorting) produces the exact same byte slice.
    *   **`TestCanonicalizeContentPost_FieldInclusion`:**
        *   Create a `ContentPost` with all relevant fields populated (as per Section 4.2). Serialize it. Then, systematically create variants where one core field at a time is slightly different (e.g., different `author_id`, different `created_at` by one nano, one different tag, different `text_content`). Verify that the canonical byte output changes for each variant, confirming all specified fields are part of the canonicalization.
    *   **`TestCanonicalizeContentPost_FieldExclusion`:**
        *   Create a `ContentPost` and populate fields that should be *excluded* from the hash (e.g., `post_id` itself, `updated_at`, `reaction_counts`). Create a variant with these excluded fields changed. Verify that `CanonicalizeContentPost` produces the *same* byte slice, confirming these fields are correctly ignored.
    *   **`TestCanonicalizeContentPost_CorrectSerialization`:**
        *   For a `ContentPost` with known values, manually construct the expected canonical byte sequence based on the defined serialization rules (e.g., field order, `UnixNano` for timestamp, sorted tags, Protobuf encoding of a subset message).
        *   Compare the output of `CanonicalizeContentPost` with this manually constructed expected byte sequence. This is a more complex test but ensures the serialization logic is precisely as specified.
    *   **`TestCanonicalizeContentPost_EdgeCases`:**
        *   Test with `ContentPost` having empty but non-nil `Tags`.
        *   Test with `ContentPost` having nil `Tags`.
        *   Test with `ContentPost` where optional fields like `parent_post_id` or `community_group_id` are empty strings.
        *   Test with `text_content` or `media_url` being empty based on `content_type`.
    *   **`TestCanonicalizeContentPost_ErrorHandling`:**
        *   Test with `nil` input `ContentPost` (should error).
        *   Test with `nil` `CreatedAt` timestamp if it's mandatory (should error).

**2. Tests for `GenerateContentID(canonicalData []byte) (string, error)`:**
    *   **`TestGenerateContentID_KnownVector`:**
        *   Take a known byte slice (`canonicalData`).
        *   Calculate its SHA-256 hash and Base58BTC encode it manually or using trusted external tools.
        *   Verify `GenerateContentID` produces this exact known Content ID string.
    *   **`TestGenerateContentID_Uniqueness`:**
        *   Generate CIDs for two slightly different `canonicalData` byte slices. Verify the CIDs are different.
    *   **`TestGenerateContentID_ErrorHandling`:**
        *   Test with `nil` `canonicalData` (should error).

**3. End-to-End Test for `GenerateAndSetPostID(post *core_types.ContentPost) error`:**
    *   **`TestGenerateAndSetPostID_Workflow`:**
        1.  Create a new `core_types.ContentPost` struct with relevant fields populated but `PostId` empty.
        2.  Call `GenerateAndSetPostID(post)`.
        3.  Verify no error is returned.
        4.  Verify `post.PostId` is non-empty and is a valid Base58BTC string that looks like a hash.
        5.  (Optional, more advanced) Re-run `CanonicalizeContentPost` on the *original data fields* (excluding the newly set `PostId`) and `GenerateContentID` on that data, then verify it matches the `post.PostId` that was set by the helper. This ensures the helper correctly orchestrated the steps.
    *   **`TestGenerateAndSetPostID_ErrorPropagation`:**
        *   Test cases where underlying `CanonicalizeContentPost` would fail (e.g., nil post); verify the error is propagated.

These tests will ensure the core hashing and ID generation logic is deterministic, correct according to specification, and robust.

---
*(End of Document: Content Hashing & Timestamping Specification & Initial Implementation Plan)*
