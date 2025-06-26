# DigiSocialBlock (Nexus Protocol) - Distributed Data Stores (DDS) Protocol Technical Specification

## 1. Objective

This document provides the detailed technical specifications for the Distributed Data Stores (DDS) Protocol for DigiSocialBlock. The DDS is responsible for the decentralized storage, replication, discovery, and retrieval of user-generated content and other platform data blobs. Its primary goals are to ensure content resilience, availability, and censorship resistance, forming the foundational layer for data persistence in the DigiSocialBlock ecosystem.

## 2. Scope

This specification covers the following aspects of the DDS Protocol:
*   Data Chunking and Content Addressing (CIDs)
*   Manifest files for large or multi-part content
*   Node Roles and Responsibilities within the DDS network
*   Data Replication Strategy (initial seeding, proactive replication, repair concepts)
*   Data Discovery and Retrieval mechanisms (primarily DHT-based)
*   Basic Security and Access Control Considerations relevant to the DDS layer
*   Core Network Messages (conceptual Protobuf definitions) for DDS interactions

This document builds upon the conceptual design outlined in `tech_specs/dli_echonet_protocol.md` (Section 2).

## 3. Relation to `dli_echonet_protocol.md`

This specification directly elaborates on the principles and components introduced in Section 2 ("Module 1.2: Distributed Data Stores (DDS) Protocol") of the `dli_echonet_protocol.md` document. It aims to provide the next level of detail required for implementation planning and development.

---

## 4. Data Chunking and Content Addressing

Content addressing ensures data integrity and provides a universal way to reference data blobs within the DDS network. Data chunking allows large files to be broken down into manageable pieces for storage and transfer.

### 4.1. Data Chunking

*   **Chunk Size:**
    *   All content will be divided into fixed-size chunks.
    *   **Specification:** The default chunk size shall be **256 KiB (256 * 1024 bytes)**.
    *   **Rationale:** This size offers a balance between minimizing overhead for small files (where a significant portion of the file might be a single chunk) and managing the number of chunks for very large files. It's a common size in systems like IPFS.
    *   The last chunk of a piece of content may be smaller than the default chunk size if the total content size is not an exact multiple of 256 KiB.
*   **Chunking Process:**
    *   Content is read sequentially and split at 256 KiB boundaries.
    *   Implementations should ensure that the chunking process is deterministic.

### 4.2. Content Identifiers (CIDs)

*   **CID Generation:**
    *   Each individual data chunk will be identified by a Content Identifier (CID).
    *   **Specification:** The CID for a data chunk will be the **SHA-256 hash of its raw byte content**.
    *   **Format:** CIDs will typically be represented as a Base58BTC encoded string of the binary hash for human readability and use in network protocols (similar to IPFS CIDs v0 style for simplicity, though multihash/CIDv1 could be adopted later for future-proofing). For internal storage and map keys, the raw binary hash (32 bytes) might be used.
    *   **Rationale:** SHA-256 provides strong collision resistance and integrity verification. Base58BTC is a common and relatively compact encoding for hashes.
*   **CID Uniqueness:** Due to the cryptographic nature of SHA-256, identical chunks will always produce the same CID, enabling data deduplication across the network (though deduplication at the storage node level is an optimization, not a strict protocol requirement for initial MVP). Different chunks will, with overwhelming probability, have different CIDs.

### 4.3. Manifest File for Multi-Chunk Content

For content larger than a single chunk, or to associate metadata with a single chunk representing a complete piece of content, a Manifest File is used.

*   **Purpose:**
    *   Lists the CIDs of all constituent data chunks in their correct order.
    *   Stores the total size of the original content.
    *   Stores the SHA-256 hash of the *original, unchunked* content for end-to-end integrity verification after reassembly.
    *   May contain other high-level metadata about the content (e.g., original filename, MIME type, though this might also be application-layer metadata).
*   **Structure (Conceptual Protobuf Definition):**
    ```protobuf
    // In a conceptual dds_protocol.proto
    message ContentManifest {
      repeated string chunk_cids = 1; // List of Base58BTC encoded CIDs of data chunks
      int64 total_content_size_bytes = 2; // Total size of the original content
      string original_content_hash = 3;  // Base58BTC encoded SHA-256 hash of the unchunked original content
      // string original_filename = 4; // Optional
      // string mime_type = 5;         // Optional
    }
    ```
*   **Serialization:** The `ContentManifest` message will be serialized using Protobuf.
*   **Manifest CID:** The serialized `ContentManifest` itself is treated as a data blob and will also have its own CID (SHA-256 hash of the serialized manifest data). This manifest CID is what users or higher-level applications will typically use to reference and retrieve the entire piece of content.
*   **Recursive Manifests (Future Consideration):** For extremely large content composed of many thousands of chunks, manifests themselves could potentially reference other manifests (forming a tree of CIDs), though this is not in scope for the initial MVP.

**Example Workflow:**
1.  User uploads `MyVideo.mp4` (10MB).
2.  Originator node chunks `MyVideo.mp4` into 40 x 256KiB chunks.
3.  For each chunk, a SHA-256 CID is calculated (e.g., `CID_chunk1`, `CID_chunk2`, ..., `CID_chunk40`).
4.  The SHA-256 hash of the original `MyVideo.mp4` is calculated (e.g., `OriginalHash_MyVideo`).
5.  A `ContentManifest` is created:
    ```json
    // Conceptual JSON representation
    {
      "chunk_cids": ["CID_chunk1", "CID_chunk2", ..., "CID_chunk40"],
      "total_content_size_bytes": 10485760, // 10 * 1024 * 1024
      "original_content_hash": "OriginalHash_MyVideo"
    }
    ```
6.  This manifest is serialized (e.g., via Protobuf).
7.  The CID of the serialized manifest is calculated (e.g., `CID_Manifest_MyVideo`). This `CID_Manifest_MyVideo` is the primary identifier for retrieving `MyVideo.mp4`.

---

## 5. Node Roles & Responsibilities in DDS

The DDS network consists of nodes performing different, though sometimes overlapping, roles related to content storage and retrieval.

### 5.1. Storage Nodes

*   **Primary Responsibility:** To reliably store data chunks and serve them upon valid retrieval requests.
*   **Core Functions:**
    *   **Store Chunks:** Accept `StoreChunkRequest` messages containing a CID and chunk data. Verify the provided data hashes to the given CID. If valid, store the chunk locally.
    *   **Retrieve Chunks:** Respond to `RetrieveChunkRequest` messages by looking up the chunk by its CID in local storage and returning its data.
    *   **Advertise Stored Content:** Publish the CIDs of chunks they store to the discovery mechanism (e.g., DHT by calling `Provide(cid, self_peer_id)`).
    *   **Participate in Replication:**
        *   Respond to requests from other nodes to replicate chunks they hold.
        *   Proactively identify under-replicated chunks they are responsible for (or interested in) and find new nodes to replicate them to.
        *   Participate in data repair mechanisms if a chunk they store is found to be corrupted (by fetching a valid copy from another replica).
*   **Requirements (Conceptual for MVP, to be detailed further for full network):**
    *   **Storage Commitment:** Nodes should ideally declare a minimum amount of storage they are willing to contribute.
    *   **Uptime:** High uptime is expected to ensure data availability. Mechanisms for incentivizing uptime and penalizing excessive downtime will be part of the broader Nexus Protocol economic model.
    *   **Bandwidth:** Sufficient bandwidth to serve chunks and participate in replication.
*   **Incentives:** Storage nodes will be incentivized through mechanisms defined in the Nexus Protocol's economic model (e.g., token rewards for provable storage and data serving, potentially linked to Proof-of-Witness).

### 5.2. Originator Nodes (Content Creators/Uploaders)

*   **Primary Responsibility:** To prepare content for storage in the DDS and ensure its initial availability by seeding it to the network.
*   **Core Functions:**
    *   **Chunk Content:** Split original content into fixed-size data chunks as per Section 4.1.
    *   **Generate CIDs:** Calculate the CID for each chunk (Section 4.2).
    *   **Create Manifest:** For multi-chunk content, create and serialize the `ContentManifest` (Section 4.3) and calculate its CID.
    *   **Initial Seeding/Replication:**
        *   Identify a set of initial Storage Nodes (e.g., through a discovery service, pre-configured list, or by querying the DHT for nodes with available capacity/willingness).
        *   Send `StoreChunkRequest` messages for each chunk (and the manifest chunk) to these initial Storage Nodes to achieve the initial replication factor (or a subset thereof).
        *   The Originator Node should ideally keep serving the content until it's sufficiently replicated across the network.
*   **Note:** Any node in the DigiSocialBlock network can act as an Originator Node when uploading content.

### 5.3. Retrieval Nodes (Content Consumers/Clients)

*   **Primary Responsibility:** To discover and retrieve content from the DDS.
*   **Core Functions:**
    *   **Obtain Manifest CID:** Get the top-level CID (usually the manifest CID) for the desired content (e.g., from an application-layer link, a search result, or a user's content list).
    *   **Discover Providers for Manifest:** Query the discovery mechanism (e.g., DHT via `FindProvidersRequest`) using the manifest CID to find Storage Nodes holding the manifest chunk.
    *   **Retrieve and Parse Manifest:** Fetch the manifest chunk from one of the providers. Deserialize and parse the `ContentManifest`. Verify its integrity if an original content hash of the manifest itself was known.
    *   **Retrieve Data Chunks:** For each `chunk_cid` listed in the manifest:
        *   Discover providers for that `chunk_cid` via the DHT.
        *   Retrieve the data chunk from one or more providers (potentially in parallel or from the "closest" provider).
        *   Verify the integrity of each retrieved chunk by hashing it and comparing to its `chunk_cid`.
    *   **Reassemble Content:** Once all data chunks are retrieved and verified, reassemble them in the correct order to reconstruct the original content.
    *   **Verify Original Content Hash:** Calculate the SHA-256 hash of the reassembled content and verify it against the `original_content_hash` stored in the manifest.
*   **Note:** Any node wishing to view or access content acts as a Retrieval Node. Storage Nodes also act as Retrieval Nodes when participating in replication or repair.

### 5.4. Discovery Nodes (Conceptual - DHT Participants)

*   **If a dedicated DHT (e.g., Kademlia-based via libp2p) is used:**
    *   **Responsibility:** All participating nodes (especially Storage Nodes) act as DHT nodes. They maintain routing tables and store/provide `(CID -> [PeerID])` mappings.
    *   **Functions:** Respond to DHT queries (`FindProvidersRequest`), store provider records (`Provide` announcements).
*   **If simple gossip is used (MVP fallback):**
    *   Nodes would gossip about CIDs they have or are looking for. This is less efficient for discovery at scale.
*   **Initial Plan:** Leverage libp2p's Kademlia DHT implementation as the primary discovery mechanism.

---

## 6. Data Replication Strategy

Ensuring data durability and availability in a decentralized network relies on a robust replication strategy.

### 6.1. Replication Factor (N)

*   **Specification:** The target replication factor `N` for each unique data chunk (including manifest chunks) shall be configurable at the network level, with an initial default of **N=7**.
*   **Rationale:** A higher N increases data redundancy and resilience against node churn or data loss on individual nodes. N=7 provides a good balance for a moderately sized network. This can be adjusted via network governance as the network matures.

### 6.2. Replication Processes

The DDS will employ several mechanisms to achieve and maintain the target replication factor:

1.  **Initial Seeding by Originator Node:**
    *   When new content (and its manifest) is created, the Originator Node is responsible for initially distributing the chunks to a set of `k` distinct Storage Nodes (where `k <= N`, e.g., `k=3`).
    *   The Originator Node may discover suitable Storage Nodes via the DHT (e.g., querying for nodes advertising storage capacity or specific attributes) or use a list of preferred/bootstrap peers.
    *   The Originator sends `StoreChunkRequest` messages to these `k` nodes for each chunk.
    *   The Originator should remain online and serving the content until it confirms these initial `k` replicas are successfully stored and advertised on the DHT.

2.  **Proactive Replication by Storage Nodes:**
    *   Storage Nodes periodically review the chunks they store.
    *   For each chunk `C` they store, a Storage Node can query the DHT (`FindProvidersRequest(C)`) to determine the current perceived replication count for `C`.
    *   If the count is less than `N`, the Storage Node may proactively seek out other suitable Storage Nodes (that do not already store `C`) and instruct them to replicate the chunk. This can be done by:
        *   Sending a `ReplicationInstruction { cid, self_peer_id }` message to a potential new storage node, effectively telling it "fetch chunk `cid` from me."
        *   The receiving node would then issue a `RetrieveChunkRequest` to the instructing node (or any other provider) and then a `StoreChunkRequest` to itself (locally) and advertise it.
    *   This process helps propagate chunks further and maintain the replication factor without relying solely on the originator.

3.  **Repair and Self-Healing (Triggered by Under-Replication):**
    *   **Detection:** Under-replication can be detected by:
        *   Storage Nodes during their periodic checks (as above).
        *   Retrieval Nodes that find fewer than expected providers for a chunk.
        *   (Future) Dedicated "auditor" or "watcher" nodes that periodically query the DHT for popular or critical CIDs.
    *   **Triggering Replication:** When under-replication is detected for a chunk `C` by a node that holds `C`, that node can initiate proactive replication as described above.
    *   If a Retrieval Node detects under-replication for a chunk it needs, it can, after successfully retrieving it, choose to also store it and advertise itself as a provider, thus contributing to repair.

### 6.3. Node Selection for Replication

When a node (Originator or existing Storage Node) needs to select other Storage Nodes for new replicas, the following criteria should be considered (some are more advanced and for future refinement):

*   **Does Not Already Store Chunk:** The target node must not already be advertising the chunk CID.
*   **Availability/Uptime:** Preference for nodes with a history of high uptime (requires reputation or monitoring system - future).
*   **Storage Capacity:** Nodes advertising available and sufficient storage capacity.
*   **Geographic/Network Diversity (Future):** To improve resilience against regional outages, the system could attempt to select nodes in different geographic locations or network segments. This requires nodes to advertise such information and a mechanism to query/filter by it.
*   **Bandwidth:** Nodes with sufficient bandwidth.
*   **Reputation/Stake (Future):** Nodes with higher reputation or stake in the network might be preferred as more reliable storage providers.

For MVP, selection might be simpler (e.g., random selection from available DHT providers not already holding the chunk, or based on proximity in Kademlia XOR space).

---

## 7. Data Discovery & Retrieval

This section details how clients (Retrieval Nodes) find and fetch content within the DDS network.

### 7.1. Discovery Mechanism - Distributed Hash Table (DHT)

*   **Primary Mechanism:** The DDS will primarily rely on a Distributed Hash Table (DHT) for content discovery.
*   **Implementation:** **Kademlia (Kad-DHT)**, as implemented in `libp2p`, is the chosen DHT protocol.
    *   **Rationale:** Kademlia is widely adopted, proven, and provides efficient peer routing and value lookups. `libp2p` offers a ready-to-use implementation.
*   **Provider Records:**
    *   Storage Nodes advertise the CIDs of the chunks they store by publishing "provider records" to the DHT.
    *   A provider record maps a CID to the PeerID (and network addresses) of the node storing that chunk.
    *   `dht.Provide(cid)`: Storage nodes call this (or an equivalent libp2p function) to announce they have a chunk. The DHT network then makes this record discoverable by peers looking for that CID.
*   **Finding Providers:**
    *   Retrieval Nodes use `dht.FindProviders(cid)` (or equivalent) to query the DHT for a given CID.
    *   The DHT lookup process will return a stream or list of PeerIDs of nodes that have advertised that they store the chunk.
*   **Bootstrapping:** Nodes joining the DHT need a list of bootstrap peers to connect to the wider DHT network.

### 7.2. Retrieval Process

1.  **Obtain Manifest CID:** The process starts with the Retrieval Node having the top-level CID of the content it wants, which is typically the CID of the `ContentManifest`.
2.  **Retrieve Manifest:**
    *   The Retrieval Node performs a `FindProvidersRequest` (DHT lookup) for the `CID_Manifest`.
    *   It receives a list of PeerIDs of Storage Nodes holding the manifest.
    *   It connects to one or more of these providers and issues a `RetrieveChunkRequest { cid: CID_Manifest }`.
    *   Upon receiving the manifest data, it deserializes it (e.g., from Protobuf) into a `ContentManifest` structure.
    *   **(Optional Verification):** If the Retrieval Node also knows the expected hash of the *manifest itself* (e.g., if `CID_Manifest` was derived from a pre-image hash), it can verify the integrity of the received manifest data at this stage.
3.  **Retrieve Data Chunks:**
    *   The Retrieval Node iterates through the `chunk_cids` listed in the parsed `ContentManifest`.
    *   For each `chunk_cid`:
        *   It performs a `FindProvidersRequest` (DHT lookup) for this `chunk_cid`.
        *   It receives a list of PeerIDs of Storage Nodes holding this specific data chunk.
        *   It connects to one (or potentially multiple, for parallel fetching or redundancy) of these providers and issues a `RetrieveChunkRequest { cid: chunk_cid }`.
        *   Upon receiving the chunk data, the Retrieval Node **MUST** verify its integrity by:
            1.  Calculating the SHA-256 hash of the received chunk data.
            2.  Comparing this calculated hash with the `chunk_cid` it requested.
            3.  If they do not match, the chunk is considered corrupt or invalid from that provider, and the node should attempt to retrieve it from a different provider. If all providers yield invalid data for a chunk, the content is considered unavailable or corrupted.
4.  **Reassemble Content:**
    *   Once all (or a sufficient number, if erasure coding were used - not in MVP) data chunks are successfully retrieved and individually verified, the Retrieval Node reassembles them in the order specified in the `ContentManifest`.
5.  **Verify Full Content Integrity:**
    *   The Retrieval Node calculates the SHA-256 hash of the fully reassembled content.
    *   It compares this hash with the `original_content_hash` stored in the `ContentManifest`.
    *   If these hashes match, the content is considered successfully and verifiably retrieved. If not, the content is considered corrupt or incomplete.

### 7.3. Caching (Optional Node Behavior)

*   Retrieval Nodes, after successfully fetching and verifying chunks, *may* choose to cache these chunks locally and also advertise themselves as providers on the DHT. This behavior helps:
    *   Increase data availability and redundancy.
    *   Improve retrieval speeds for subsequent requests for the same content, especially if the caching node is closer (in network terms) to future requesters.
    *   Reduce load on initial seeders or less resilient storage nodes.
*   Caching policies (e.g., TTL, storage limits) are local decisions for each node.

---

## 8. Basic Security & Access Control Considerations (DDS Layer)

The DDS layer's primary security focus is on data integrity and availability of public or encrypted data blobs. Fine-grained access control and advanced threat mitigation are typically handled by higher layers of the DigiSocialBlock protocol stack (e.g., application layer, identity layer, consensus/incentive layer).

### 8.1. Data Integrity

*   **Content Addressing (CIDs):** This is the fundamental mechanism for ensuring data integrity.
    *   Any modification to a chunk's content will result in a different CID.
    *   Retrieval Nodes verify each chunk against its CID upon download.
    *   The manifest's `original_content_hash` allows end-to-end verification of the reassembled content.
*   **Secure Hash Algorithm:** SHA-256 is used for CIDs and content hashes, providing strong resistance against pre-image and collision attacks.

### 8.2. Data Availability

*   **Replication:** The replication strategy (Section 6) is the primary means of ensuring data availability despite node churn or failures.
*   **Incentives (External to DDS Protocol):** The broader Nexus Protocol economic model will incentivize Storage Nodes to remain online and reliably serve data, contributing to availability. Proof-of-Witness will play a role here.

### 8.3. Sybil Resistance for Storage Nodes (Conceptual - External Dependency)

*   **Challenge:** A large number of fake (Sybil) Storage Nodes could falsely claim to store data or disrupt the DHT.
*   **Mitigation (Handled by Nexus Protocol Layer):** The DDS protocol itself does not implement Sybil resistance. This will be addressed by the overarching Nexus Protocol through mechanisms such as:
    *   **Stake-based Registration:** Requiring nodes (especially those wishing to be primary storage providers or earn significant rewards) to stake tokens.
    *   **Proof-of-Resources:** Requiring nodes to prove actual storage capacity or bandwidth.
    *   **Reputation System:** Building reputation based on reliable behavior.
*   **DDS Implication:** The DDS can operate with a less trusted set of nodes, but its efficiency and reliability improve significantly if the participating Storage Nodes are generally reputable.

### 8.4. Access Control for Private/Encrypted Content

*   **DDS as Blob Store:** The DDS protocol primarily treats data as opaque blobs identified by CIDs. It is generally agnostic to whether the content of these blobs is encrypted or public.
*   **Encryption at Application Layer:**
    *   For private or access-controlled content (e.g., direct messages, private group posts, user-encrypted assets), encryption **MUST** occur at the application layer *before* the content is chunked and passed to the DDS for storage.
    *   The resulting chunks stored in the DDS would be encrypted blobs.
*   **Key Management (External to DDS Protocol):**
    *   The management, distribution, and revocation of encryption keys are **NOT** the responsibility of the DDS protocol. This will be handled by the DigiSocialBlock identity and access control layers.
    *   For example, users might use their identity keys to encrypt/decrypt content keys, which are then used to encrypt/decrypt the actual content data.
*   **Implication for DDS:** Storage Nodes store these encrypted chunks without needing access to the decryption keys. Only authorized users (with the correct keys) can decrypt the content after retrieving it via the DDS.

### 8.5. Denial of Service (DoS/DDoS) Mitigation (Conceptual)

*   **Challenge:** Nodes (Storage, DHT) can be targeted by DoS/DDoS attacks.
*   **Mitigation (Leveraging libp2p & Network Layer):**
    *   `libp2p` itself incorporates some DoS resistance features (e.g., connection limits, stream multiplexing).
    *   Rate limiting at the node level for requests (StoreChunk, RetrieveChunk, DHT queries).
    *   Firewalling and standard network security practices by node operators.
    *   (Future) More advanced techniques like traffic analysis, reputation-based request prioritization.
*   **DDS Specifics:** The distributed nature of the DDS helps mitigate the impact of DoS attacks on any single Storage Node, as replicas should exist elsewhere. However, widespread attacks on the DHT could still impact discoverability.

### 8.6. Content Moderation & Takedown (Application Layer Concern)

*   **DDS Immutability (by CID):** Once a chunk is in the DDS and identified by its CID, that specific chunk cannot be altered without changing its CID.
*   **Moderation Handling:** Content moderation (e.g., removing illegal or harmful content) is an application-layer and governance concern.
    *   **Mechanism:** If content (identified by its manifest CID) is deemed inappropriate, application clients would cease to resolve or display it. Governance mechanisms might issue "takedown lists" or "blocklists" of manifest CIDs.
    *   **DDS Role:** Storage Nodes are not expected to actively police content *within* opaque chunks they store. However, they may choose to comply with legally binding takedown requests by deleting specific chunks if they can identify them (e.g., if a takedown request includes the CIDs of the chunks to be removed).
    *   The immutability means the original data might still exist on some nodes if not all comply, but access through the primary application interfaces would be cut off.

---

## 9. DDS Network Messages (Conceptual Protobuf Definitions)

The following Protobuf message definitions outline the core interactions between nodes in the DDS network. These would typically reside in a `dds_protocol.proto` file, separate from the `echonet_v3_core.proto` which defines the content structures themselves.

```protobuf
syntax = "proto3";

package dds_protocol;

option go_package = "empower1/pkg/dds_rpc;dds_rpc"; // Example Go package

// Assuming core_types.proto (echonet_v3_core.proto) defines common types if needed,
// or they are simple enough to be defined here (like CIDs as strings).

// Request to store a data chunk.
message StoreChunkRequest {
  string cid = 1;       // Base58BTC encoded CID of the chunk (SHA-256 hash of chunk_data)
  bytes chunk_data = 2; // The raw chunk data (max 256KiB + any overhead)
  // int64 ttl_seconds = 3; // Optional: requested time-to-live for the chunk
}

// Response to a StoreChunkRequest.
message StoreChunkResponse {
  bool success = 1;
  string error_message = 2; // If success is false
}

// Request to retrieve a data chunk.
message RetrieveChunkRequest {
  string cid = 1; // Base58BTC encoded CID of the chunk to retrieve
}

// Response to a RetrieveChunkRequest.
message RetrieveChunkResponse {
  bytes chunk_data = 1;    // The raw chunk data if found
  bool found = 2;          // True if chunk was found and is included
  string error_message = 3; // If not found or other error
}

// Request to find providers for a given CID (DHT lookup).
message FindProvidersRequest {
  string cid = 1; // Base58BTC encoded CID
  // int32 max_providers_to_return = 2; // Optional: limit number of results
}

// Response containing a list of providers for a CID.
message FindProvidersResponse {
  string cid = 1;                      // The CID that was queried
  repeated string provider_peer_ids = 2; // List of PeerIDs (e.g., libp2p PeerID strings)
  // repeated Multiaddr provider_addresses = 3; // Optional: direct network addresses if known by DHT responder
}

// Instruction from one node to another to replicate a chunk.
// Node A (holder) sends this to Node B (target for new replica).
// Node B would then typically issue a RetrieveChunkRequest to Node A (or any provider) for the chunk.
message ReplicationInstruction {
  string cid_to_replicate = 1;   // The CID of the chunk Node B should fetch and store
  string source_peer_id = 2;     // PeerID of Node A (the instructing node, a known provider)
  // repeated string known_providers_for_cid = 3; // Optional: other known providers Node B could try
}

// Response to a ReplicationInstruction (simple ack/nack).
message ReplicationResponse {
  bool accepted = 1; // True if Node B accepts the task (doesn't guarantee completion)
  string message = 2; // e.g., "Replication queued" or "Cannot replicate: no storage"
}

// (Future considerations could include messages for challenging storage, audit logs, etc.)
```

**Notes on Messages:**
*   **CIDs as Strings:** CIDs are represented as strings (Base58BTC encoded SHA-256 hashes) in these messages for broad compatibility and human readability in logs, etc. Internally, nodes might convert these to/from binary representations.
*   **Error Handling:** Simple `success` booleans and `error_message` strings are used. More structured error codes could be added.
*   **Peer Identification:** `provider_peer_ids` and `source_peer_id` would typically be libp2p PeerID strings.
*   **Network Addresses:** `FindProvidersResponse` *could* include `Multiaddr` formatted addresses if the DHT responder has them readily available, reducing an extra lookup step for the requester. This is often part of libp2p DHT responses.
*   **`ReplicationInstruction` Flow:** This is a simplified "pull" model suggestion. Node A suggests Node B pull a chunk. Node B is responsible for fetching it. Alternative "push" models exist but can be more complex with firewalls.

---

## 10. Initial DDS Implementation Plan (Conceptual)

This section outlines a high-level plan for the initial Go implementation of the DDS Protocol (MVP).

### 10.1. Core DDS Go Package Structure (Proposed)

A dedicated top-level package, `dds`, is proposed, likely within an `internal` or `pkg` directory structure depending on overall project layout (e.g., `internal/dds` or `pkg/dds`).

```
dds/
|-- chunker/          // Logic for splitting content into chunks, generating CIDs, creating manifests.
|   |-- chunker.go
|   `-- chunker_test.go
|
|-- storage/          // Local storage management for chunks.
|   |-- interface.go    // Defines StorageManager interface (Store, Retrieve, Has, Delete).
|   |-- filestore.go    // Filesystem-based implementation of StorageManager.
|   `-- filestore_test.go
|
|-- discovery/        // Data discovery mechanisms (DHT interaction).
|   |-- interface.go    // Defines Discovery interface (Provide, FindProviders).
|   |-- kad_dht.go      // Kademlia DHT implementation using libp2p.
|   `-- kad_dht_test.go // (Might be more integration-focused)
|
|-- rpc/              // Network RPC definitions and handlers for DDS messages.
|   |-- dds_protocol.pb.go // Generated from dds_protocol.proto
|   |-- dds_protocol_grpc.pb.go // Generated gRPC bindings
|   |-- handlers.go     // RPC handler implementations (e.g., HandleStoreChunk).
|   `-- rpc_test.go
|
|-- replication/      // Logic for managing data replication.
|   |-- manager.go      // Manages replication tasks, monitors health.
|   `-- manager_test.go
|
|-- dds_node.go       // Main DDS node logic, orchestrating other components.
`-- dds_node_test.go
```

**Package Descriptions:**

*   **`chunker`**: Handles all aspects of content processing before network interaction: splitting files into defined chunks, calculating CIDs for those chunks, and generating/parsing `ContentManifest` files.
*   **`storage`**: Abstract local storage for chunks.
    *   `interface.go`: Defines the `StorageManager` interface.
    *   `filestore.go`: An initial, simple implementation storing chunks as individual files on disk, named by their CIDs (or a transformation thereof).
*   **`discovery`**: Interface and implementation for how nodes discover which peers are storing specific CIDs.
    *   `interface.go`: Defines the `Discovery` interface.
    *   `kad_dht.go`: Wraps and utilizes `libp2p`'s Kademlia DHT functionality.
*   **`rpc`**: Contains Protobuf message definitions (`.proto` file would live here or in a top-level `proto` dir, with generated Go code placed here) and the gRPC/libp2p stream handlers for these messages.
*   **`replication`**: (Potentially for a later phase beyond initial MVP, but good to plan for). Contains logic for nodes to manage proactive replication, check replication factors, and initiate repair.
*   **`dds_node.go`**: The main orchestrator for a node participating in the DDS network. It would initialize and manage the `StorageManager`, `Discovery` client/server, and `RPC` services. It handles incoming requests and outgoing operations like seeding content or fetching content.

This structure promotes modularity and separation of concerns.

---

### 10.2. Core DDS Components - Initial Implementation Plan (MVP)

This outlines the key functions and interfaces for the initial, Minimum Viable Product (MVP) implementation of the DDS components.

**1. Chunker (`dds/chunker/chunker.go`)**

*   **`Split(data []byte, chunkSize int) ([][]byte, error)`:**
    *   Takes raw content data and a chunk size.
    *   Returns a slice of byte slices, where each inner slice is a chunk.
    *   The last chunk may be smaller than `chunkSize`.
    *   Returns an error if data is nil or chunkSize is invalid (e.g., <= 0).
*   **`GenerateCID(chunkData []byte) (string, error)`:**
    *   Takes raw chunk data.
    *   Calculates its SHA-256 hash.
    *   Base58BTC encodes the hash.
    *   Returns the encoded CID string.
    *   Returns an error if `chunkData` is nil.
*   **`CreateManifest(chunkCIDs []string, totalContentSizeBytes int64, originalContentHash string) ([]byte, error)`:**
    *   Takes a list of (Base58BTC encoded) chunk CIDs, total original content size, and the (Base58BTC encoded) hash of the original unchunked content.
    *   Constructs a `ContentManifest` Protobuf message (as defined in Section 4.3 & 9).
    *   Serializes the `ContentManifest` message to a byte slice using Protobuf.
    *   Returns the serialized manifest data.
    *   Returns an error if inputs are invalid (e.g., empty `chunkCIDs` list if content is expected).
*   **`ParseManifest(manifestData []byte) (*ContentManifest, error)`:**
    *   Takes serialized manifest data.
    *   Deserializes it into a `ContentManifest` Protobuf message.
    *   Returns the pointer to the `ContentManifest` struct.
    *   Returns an error if deserialization fails or data is invalid.

**2. Local Storage Manager (`dds/storage/interface.go`, `dds/storage/filestore.go`)**

*   **`StorageManager` Interface (`interface.go`):**
    ```go
    type StorageManager interface {
        Store(cid string, data []byte) error // Stores data, using CID as key. Verifies data matches CID.
        Retrieve(cid string) ([]byte, bool, error) // Retrieves data by CID. Returns data, found (bool), error.
        Has(cid string) (bool, error) // Checks if CID exists in storage.
        Delete(cid string) error      // Deletes data by CID.
        // ListCIDs() ([]string, error) // Optional: for diagnostics, local checks
    }
    ```
*   **`FileStore` Implementation (`filestore.go`):**
    *   Implements `StorageManager`.
    *   Constructor: `NewFileStore(rootDir string) (*FileStore, error)` - initializes storage in a specified directory.
    *   `Store`: Writes chunk data to a file named `base58btc(cid)` (or some transformation to make it fs-safe) within `rootDir`. Before writing, it will re-calculate the hash of `data` and verify it matches the provided `cid` string (after decoding `cid`).
    *   `Retrieve`: Reads data from the corresponding file.
    *   `Has`: Checks for file existence.
    *   `Delete`: Deletes the file.

**3. Basic RPC/Networking (`dds/rpc/handlers.go`)**
    (Leveraging generated code from `dds_protocol.proto` and gRPC/libp2p stream setup)

*   **`HandleStoreChunk(request *dds_rpc.StoreChunkRequest) (*dds_rpc.StoreChunkResponse, error)`:**
    *   Called when a node receives a `StoreChunkRequest`.
    *   Validates the request (e.g., `cid` format, `chunk_data` size).
    *   Verifies that `sha256(request.chunk_data)` matches `request.cid` (after decoding CID).
    *   Calls `StorageManager.Store(request.cid, request.chunk_data)`.
    *   Returns `StoreChunkResponse` indicating success or failure.
*   **`HandleRetrieveChunk(request *dds_rpc.RetrieveChunkRequest) (*dds_rpc.RetrieveChunkResponse, error)`:**
    *   Called when a node receives a `RetrieveChunkRequest`.
    *   Validates the request (e.g., `cid` format).
    *   Calls `StorageManager.Retrieve(request.cid)`.
    *   If found, returns `RetrieveChunkResponse` with `chunk_data` and `found=true`.
    *   If not found, returns `found=false`.
    *   Returns error message on internal errors.

**4. Simple DHT/Discovery (`dds/discovery/interface.go`, `dds/discovery/kad_dht.go`)**

*   **`Discovery` Interface (`interface.go`):**
    ```go
    type Discovery interface {
        // Provide announces to the network that this node has the content for the given CID.
        Provide(cid string) error
        // FindProviders queries the network to find peers who have advertised the given CID.
        // Returns a channel of PeerInfo or a slice of PeerInfo.
        FindProviders(cid string, timeout time.Duration) (<-chan peer.AddrInfo, error) // Example using libp2p types
    }
    ```
*   **`KadDHT` Implementation (`kad_dht.go`):**
    *   Implements `Discovery`.
    *   Constructor: `NewKadDHT(host host.Host, dht *kaddht.IpfsDHT) (*KadDHT, error)` (takes libp2p host and DHT object).
    *   `Provide`: Calls underlying `dht.Provide()`.
    *   `FindProviders`: Calls underlying `dht.FindProvidersAsync()` or similar, returning a channel of peer information.
    *   Handles context for timeouts.

*(This outlines the core interfaces and functions for an MVP. Actual implementation will involve more detailed error handling, logging, and integration with libp2p for networking.)*

---

### 10.3. MVP Scope for DDS

The Minimum Viable Product (MVP) for the DDS Protocol will focus on demonstrating core functionality with a limited set of features. The goal is to have a working system that can store, discover, and retrieve content between a small number of nodes.

**Included in MVP:**

1.  **Data Chunking & Manifests:**
    *   Implementation of `chunker.Split()`, `chunker.GenerateCID()`, `chunker.CreateManifest()`, and `chunker.ParseManifest()`.
    *   Ability to process a local file, chunk it, generate CIDs for all chunks, create a manifest, and generate a CID for the manifest.
2.  **Local Storage (`FileStore`):**
    *   Full implementation of the `StorageManager` interface using `FileStore`: `Store`, `Retrieve`, `Has`, `Delete`.
    *   A node can store chunks provided to it and retrieve them.
    *   Verification of chunk data against CID on `Store`.
3.  **Basic P2P Network Communication (via `libp2p`):**
    *   Nodes can connect to each other.
    *   Implementation of RPC handlers for:
        *   `StoreChunkRequest` / `StoreChunkResponse`: One node can send a chunk to another for storage.
        *   `RetrieveChunkRequest` / `RetrieveChunkResponse`: One node can request and receive a chunk from another.
4.  **Basic DHT Integration (via `libp2p KadDHT`):**
    *   Nodes can join the Kademlia DHT.
    *   **`Provide`:** A node storing a chunk (or manifest) can advertise this fact to the DHT (`dht.Provide(cid)`).
    *   **`FindProviders`:** A node can query the DHT to find peers that are providing a specific CID (`dht.FindProviders(cid)`).
5.  **End-to-End Workflow (Simplified):**
    *   **Node A (Originator):**
        1.  Takes a local file.
        2.  Chunks it, generates all CIDs, creates manifest, gets manifest CID.
        3.  Stores all its own chunks (data + manifest) locally via its `StorageManager`.
        4.  Advertises all these CIDs to the DHT (`Provide`).
        5.  (Optional for MVP, manual for now) "Sends" (via `StoreChunkRequest`) the manifest and a few data chunks to Node B.
    *   **Node B (Storage/Retrieval):**
        1.  (Optional) Receives chunks from Node A and stores them, then Provides them to DHT.
        2.  **Retrieval Test:** Given `CID_Manifest_From_Node_A`:
            *   Node B uses DHT to `FindProviders` for `CID_Manifest_From_Node_A`. Should find Node A (and itself if it stored it).
            *   Node B sends `RetrieveChunkRequest` to a provider (e.g., Node A) for the manifest chunk.
            *   Node B parses the manifest.
            *   For each `chunk_cid` in the manifest, Node B uses DHT to `FindProviders`, then `RetrieveChunkRequest` from a provider. Verifies each chunk.
            *   Node B reassembles the content and verifies the final original content hash.
6.  **Basic CLI or Test Application:** A simple command-line tool or a set of integration tests to demonstrate the above workflow (e.g., "put file", "get file by manifest CID").

**Excluded from MVP:**

*   **Advanced Replication Strategies:** No proactive replication by storage nodes beyond initial seeding, no complex repair/self-healing mechanisms. Replication factor `N` might effectively be 1 or what's manually seeded.
*   **Advanced Node Selection for Replication/Seeding:** Simple peer selection or manual configuration.
*   **Robust Security Measures:** No advanced Sybil resistance, minimal DoS protection beyond what libp2p offers by default. Access control/encryption is an application concern.
*   **Incentive Mechanisms:** No integration with tokenomics or reward systems.
*   **Content Moderation Hooks:** No specific features for takedowns at the DDS layer.
*   **Complex Caching Strategies:** Nodes will not implement sophisticated caching beyond potentially storing what they retrieve.
*   **Scalability and Performance Optimization:** Focus is on correctness and core P2P flow, not high performance under load.
*   **Full Network Bootstrapping Complexity:** Assumes a small, potentially pre-configured or easily discoverable set of initial peers for DHT bootstrapping.

The MVP aims to prove the fundamental store, discover, and retrieve capabilities of the DDS using content addressing and a DHT.

---

### 10.4. Unit and Basic Integration Test Strategy for DDS MVP

Testing is crucial to ensure the correctness of the DDS MVP components.

**1. Unit Tests:**

*   **`chunker` Package (`dds/chunker/chunker_test.go`):**
    *   **`TestSplit`:**
        *   Test with data smaller than chunk size (should produce 1 chunk).
        *   Test with data equal to chunk size (1 chunk).
        *   Test with data that is an exact multiple of chunk size (multiple full chunks).
        *   Test with data that results in a final partial chunk.
        *   Test with empty data (should return empty slice or error).
        *   Test with nil data (should error).
        *   Test with invalid `chunkSize` (0 or negative, should error).
    *   **`TestGenerateCID`:**
        *   Test with known chunk data and verify against a pre-calculated SHA-256 hash (Base58BTC encoded).
        *   Test that different data produces different CIDs.
        *   Test that identical data produces identical CIDs.
        *   Test with nil/empty chunk data (should error or return defined CID for empty data if applicable).
    *   **`TestCreateManifest`:**
        *   Test with a list of mock CIDs, size, and original hash; verify the serialized output can be parsed and contains the correct fields.
        *   Test with empty `chunkCIDs` list (should error or produce a specific manifest for empty content).
    *   **`TestParseManifest`:**
        *   Test with valid serialized manifest data; verify the resulting `ContentManifest` struct has correct fields.
        *   Test with malformed/corrupt manifest data (should error).

*   **`storage` Package (`dds/storage/filestore_test.go`):**
    *   (Assuming `FileStore` implementation)
    *   **`TestFileStore_StoreRetrieve`:**
        *   Store a chunk with a valid CID (data matches CID).
        *   Retrieve it and verify data integrity.
        *   Attempt to store with mismatched CID and data (should error).
    *   **`TestFileStore_Has`:**
        *   Test `Has` for an existing CID (should be true).
        *   Test `Has` for a non-existent CID (should be false).
    *   **`TestFileStore_Delete`:**
        *   Store a chunk, then delete it. Verify `Has` returns false and `Retrieve` fails to find it.
    *   **Edge Cases:** Storing empty data, retrieving non-existent CID, concurrent access (if applicable, though `FileStore` might be simple initially). Test directory creation/permissions if `NewFileStore` handles this.

**2. Basic Integration Tests:**

These tests will involve setting up minimal `libp2p` hosts and testing interactions. They might reside in a top-level `dds_test` package or within specific component test packages if they can mock dependencies.

*   **RPC Integration (`dds/rpc/rpc_test.go` or `dds_test`):**
    *   **`TestStoreChunkRPC`:**
        *   Setup two libp2p nodes (Node A - client, Node B - server with `StorageManager`).
        *   Node A sends `StoreChunkRequest` to Node B.
        *   Verify Node B stores the chunk correctly (check its `StorageManager`).
        *   Verify Node A receives a success response.
        *   Test with invalid data (e.g., CID mismatch) - Node B should reject and Node A gets error response.
    *   **`TestRetrieveChunkRPC`:**
        *   Setup Node A (client) and Node B (server with a pre-stored chunk).
        *   Node A sends `RetrieveChunkRequest` to Node B.
        *   Verify Node A receives the correct chunk data.
        *   Test requesting a non-existent CID - Node A should receive `found=false`.

*   **DHT/Discovery Integration (`dds/discovery/kad_dht_test.go` or `dds_test`):**
    *   **`TestProvideAndFindProviders`:**
        *   Setup multiple libp2p nodes connected in a DHT.
        *   Node A stores a chunk and calls `Discovery.Provide(cid)`.
        *   Node B calls `Discovery.FindProviders(cid)`.
        *   Verify Node B discovers Node A as a provider for the CID.
        *   Test `FindProviders` for a CID that has not been Provided (should return no providers or timeout).
    *   **Timeout Handling:** Test that `FindProviders` respects timeouts.

*   **End-to-End MVP Workflow Test (`dds/dds_node_test.go` or `dds_test`):**
    *   Simulate the E2E workflow described in Section 10.3, Step 5:
        *   Node A chunks a file, creates manifest, stores locally, Provides all CIDs.
        *   Node B (knowing only the manifest CID) discovers manifest providers, retrieves manifest.
        *   Node B then discovers and retrieves all data chunks listed in the manifest.
        *   Node B reassembles and verifies the content.
    *   This test is crucial for verifying that all MVP components integrate correctly.

These tests will form the basis for ensuring the DDS MVP is functional and correct according to the specifications.

---
*(End of Document: DDS Protocol Specification & Initial Implementation Plan)*
