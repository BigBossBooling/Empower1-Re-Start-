# DigiSocialBlock (Nexus Protocol) - Mobile Node Role Technical Specification

## 1. Objective

This document provides the detailed technical specifications for the Mobile Node role within the DigiSocialBlock (Nexus Protocol) ecosystem. It defines how mobile devices (e.g., smartphones, tablets) participate as active nodes in the network, their capabilities, inherent limitations, and their specific interactions with core protocols such as the Distributed Data Stores (DDS) and Proof-of-Witness (PoW). The goal is to enable true decentralization by leveraging user devices while being mindful of mobile resource constraints.

## 2. Scope

This specification covers:
*   Defined capabilities and limitations of Mobile Nodes, particularly for the Minimum Viable Product (MVP).
*   Detailed interaction flows for Mobile Nodes with the DDS protocol (content creation/upload, content retrieval, local caching, seeding strategy).
*   Detailed interaction flows for Mobile Nodes with the PoW protocol (requesting attestations, generating and verifying PoW Claims).
*   Specific considerations for resource management on mobile devices (storage, data usage, battery, CPU).
*   Mobile-specific networking considerations (connectivity, peer discovery, background operation).

This document builds upon the conceptual design outlined in `tech_specs/dli_echonet_protocol.md` (Section 5).

## 3. Relation to `dli_echonet_protocol.md`

This specification directly elaborates on the principles and components introduced in Section 5 ("Module 1.5: Mobile Node Role Technical Specifications") of the `dli_echonet_protocol.md` document. It provides the next level of detail required for implementation planning and development of mobile client applications that act as participating nodes in the DigiSocialBlock network.

---

## 4. Mobile Node Capabilities & Limitations

Mobile Nodes are first-class citizens in the DigiSocialBlock network, but their operational parameters must respect the inherent constraints of mobile devices.

### 4.1. Capabilities (MVP Focus)

For the Minimum Viable Product (MVP), Mobile Nodes will possess the following core capabilities:

1.  **Content Origination:**
    *   Create and edit core content types (e.g., `ContentPost`, `UserProfile` updates) locally on the device.
    *   Perform client-side hashing to generate Content IDs (e.g., `post_id`) for new content, incorporating the client-asserted `created_at` timestamp, as per the Content Hashing & Timestamping protocol.
2.  **Lightweight Distributed Data Stores (DDS) Interaction:**
    *   **Local Caching:** Maintain a local cache for:
        *   Content authored by the user (both manifest and data chunks).
        *   Content frequently accessed or explicitly saved/bookmarked by the user (manifest and data chunks).
    *   **Content Seeding:** Actively seed their own authored content to a limited set of more robust network nodes (e.g., "Super Hosts" or designated Storage Nodes) to ensure initial availability and persistence beyond the mobile device's own uptime.
    *   **Content Retrieval:** Fetch content (manifests and data chunks) from the DDS network (from any available provider node, mobile or otherwise) for viewing or interaction.
    *   **DHT Participation (Provider):** Advertise the CIDs of content chunks stored in its local cache to the network's DHT, making itself a source for that content (within its resource limits and connectivity).
3.  **Basic Proof-of-Witness (PoW) Protocol Interaction:**
    *   **Attestation Querying:** Query Witness Nodes (or a future attestation discovery service) to retrieve `WitnessAttestation` objects for specific content CIDs (manifest CIDs).
    *   **PoW Claim Generation:** Assemble collected valid attestations into a `PoWClaim` for content they originated or are interested in. This claim can then be used by the mobile node or shared.
    *   **PoW Claim Verification:** Locally verify the validity of `PoWClaim` objects received from other users or services by fetching the referenced content from DDS and cryptographically checking the attestations.
4.  **P2P Networking:**
    *   Operate as a `libp2p` node.
    *   Connect to bootstrap peers and participate in the Kademlia DHT for peer discovery and content/provider discovery.
    *   Handle basic RPC requests from other peers (e.g., `RetrieveChunkRequest` for content it's providing from its cache).
5.  **Identity Management:** Securely manage the user's private keys associated with their DigiSocialBlock identity for signing content, attestations (if ever acting as a limited witness in the future), or transactions.

### 4.2. Limitations (MVP Focus)

To ensure usability and respect for device resources, Mobile Nodes in the MVP will have the following limitations:

1.  **Not Full/Permanent Storage Nodes for Third-Party Content:**
    *   Mobile Nodes are **not expected** to act as persistent, high-volume storage providers for arbitrary content from other users in the DDS network.
    *   Their storage contribution is primarily for their own content, a user-controlled cache of consumed content, and potentially opportunistic short-term caching of recently routed data if resources allow (beyond MVP).
2.  **Not Primary Witness Nodes (MVP):**
    *   Mobile Nodes will **not** typically perform the role of a primary, always-on Witness Node in the PoW protocol during the MVP phase.
    *   This is due to potential constraints in uptime, bandwidth, processing power for continuous network monitoring, and battery life.
    *   (Future Consideration): Users might "opt-in" to limited witnessing duties under specific conditions (e.g., when device is charging and on Wi-Fi, for content from followed users or specific communities), but this requires careful design.
3.  **Resource Management Constraints:**
    *   **Storage:** Strict, user-configurable (with sensible defaults) limits on the amount of disk space used for caching DDS content and other protocol data.
    *   **Data Usage:** Prioritization of Wi-Fi over cellular data for non-critical background tasks (like seeding or proactive caching). User controls and "data saver" modes will be essential.
    *   **Battery Life:** Operations (especially network-intensive ones like seeding or DHT participation) must be designed to be power-efficient and minimize background battery drain. Opportunistic execution (e.g., performing tasks when charging) will be preferred.
    *   **CPU Usage:** Hashing, encryption/decryption (for user's own content if E2EE applied locally), and signature verification should be optimized for mobile CPUs.
4.  **Intermittent Connectivity:**
    *   The protocol design and mobile application logic must assume and gracefully handle intermittent network connectivity (Wi-Fi/cellular transitions, periods of no connectivity).
    *   Tasks like content uploading/seeding and large downloads should be resumable.
    *   DHT participation might be less stable compared to always-on nodes.
5.  **Background Operation Restrictions:**
    *   Mobile operating systems (iOS, Android) impose strict limitations on background processing and network activity.
    *   The extent of background DDS/DHT participation will be constrained by these OS policies. Focus will be on efficient foreground operation and opportunistic background tasks when permitted by the OS and user settings.

These capabilities and limitations aim to make Mobile Nodes active and useful participants while ensuring a good user experience on resource-constrained devices.

---

## 5. Mobile Node Interaction with DDS Protocol

Mobile Nodes interact with the Distributed Data Stores (DDS) for creating, storing, and retrieving content. Their interactions are optimized for mobile constraints.

### 5.1. Content Creation and Upload/Seeding Flow

When a user creates content on their mobile device:

1.  **Local Preparation (as per Content Hashing & Timestamping Spec):**
    *   The mobile application populates the `core_types.ContentPost` message (or other content type).
    *   It sets the client-side `created_at` timestamp.
    *   It generates the canonical representation of the core content fields.
    *   It hashes this canonical data to produce the `post_id` (Content ID).
    *   The `post_id` is added to the `ContentPost` message.
    *   (Optional) The user's signature over the `post_id` or canonical hash can be generated and stored/associated.

2.  **DDS Preparation (Local):**
    *   The complete `ContentPost` message (now including `post_id`) is serialized into a single byte array (the "original content blob" for DDS).
    *   The mobile app's DDS component calculates the SHA-256 hash of this "original content blob" (this becomes `original_content_hash` for the DDS manifest).
    *   The "original content blob" is split into 256KiB chunks.
    *   SHA-256 CIDs are generated for each chunk.
    *   A `dds_protocol.ContentManifest` is created, listing all chunk CIDs, the total size, and the `original_content_hash`.
    *   The manifest is serialized, and its CID (`manifest_cid`) is generated.

3.  **Local Caching:**
    *   All generated data chunks and the manifest chunk **MUST** be stored in the Mobile Node's local DDS cache (subject to cache size limits).
    *   This ensures the user can immediately access their own content even if offline and makes the mobile a primary source if connected.

4.  **DHT Advertisement (Provider Records):**
    *   The Mobile Node **SHOULD** advertise itself as a provider for all locally cached CIDs (data chunks and the manifest chunk) to the Kademlia DHT.
    *   This involves calling `Discovery.Provide(cid)` for each CID.
    *   This makes the mobile device discoverable as a source for its own content when it is online.

5.  **Seeding to Super Hosts / Robust Storage Nodes:**
    *   To ensure content persistence and availability even when the mobile device is offline or has limited resources, the Mobile Node **MUST** attempt to seed its content to a small set (e.g., `S=2` to `S=3`) of more robust, always-on "Super Hosts" or designated Storage Nodes.
    *   **Discovery of Super Hosts:**
        *   MVP: May use a pre-configured list of bootstrap Super Host addresses.
        *   Future: Could query the DHT for nodes advertising "Super Host" capabilities or specific storage services.
    *   **Seeding Process:**
        *   For each selected Super Host, the Mobile Node sends `StoreChunkRequest` messages for its manifest chunk and all its data chunks.
        *   This process should be resumable and ideally prioritize Wi-Fi connections.
        *   The mobile app should provide feedback to the user about the seeding progress and status.
    *   **Goal:** Ensure that at least `S` replicas exist on reliable infrastructure nodes in addition to the mobile's own copy.

### 5.2. Content Retrieval Flow

When a user on a Mobile Node wishes to access content:

1.  **Obtain Manifest CID:** The mobile application obtains the `manifest_cid` of the desired content (e.g., from a user feed, a link, a search result).
2.  **Check Local Cache:** The mobile app first checks its local DDS cache for the `manifest_cid`.
    *   If the manifest chunk is found locally, it proceeds to parse it.
3.  **Discover Manifest Providers (DHT Lookup):**
    *   If not in local cache, the Mobile Node queries the DHT using `Discovery.FindProviders(manifest_cid)`.
    *   It receives a list of PeerIDs (which could include other mobiles or Super Hosts).
4.  **Retrieve and Parse Manifest:**
    *   The Mobile Node attempts to retrieve the manifest chunk from one or more discovered providers using `RetrieveChunkRequest`. It may prioritize "closer" or more reliable peers if such information is available.
    *   Once retrieved, the manifest data is deserialized and validated (as per DDS spec).
    *   The manifest chunk, once validated, **SHOULD** be stored in the local DDS cache.
5.  **Retrieve and Verify Data Chunks:**
    *   The mobile app iterates through the `chunk_cids` in the manifest.
    *   For each `chunk_cid`:
        *   It first checks its local DDS cache.
        *   If not cached, it queries the DHT (`FindProviders(chunk_cid)`).
        *   It retrieves the chunk from a provider, verifying its integrity against the `chunk_cid`. If verification fails, it may try other providers.
        *   Successfully retrieved and verified data chunks **SHOULD** be stored in the local DDS cache (especially if the user is actively consuming the content, e.g., streaming video).
6.  **Reassemble and Present Content:**
    *   Once all necessary chunks are available (either from cache or network), the content is reassembled.
    *   The `original_content_hash` from the manifest is used to verify the integrity of the fully reassembled content.
    *   The content is then presented to the user.

### 5.3. Cache Management

*   The Mobile Node's local DDS cache will have a configurable size limit.
*   An eviction policy (e.g., Least Recently Used - LRU, or prioritizing user's own authored content) will be needed to manage the cache when it reaches its limit.
*   Users may have options to "pin" certain content to keep it in the cache or clear the cache.

This interaction model allows mobile nodes to be effective content originators and consumers while offloading the burden of permanent, high-redundancy storage to more capable nodes when necessary.

---

## 6. Mobile Node Interaction with PoW Protocol

Mobile Nodes primarily act as consumers and initiators within the Proof-of-Witness (PoW) ecosystem, rather than as full Witness Nodes themselves (in MVP). Their interactions focus on leveraging PoW for content validation and claim generation.

### 6.1. Requesting Attestations for Content

*   When a Mobile Node has originated content and successfully seeded it to the DDS (obtaining a `manifest_cid`), it needs `WitnessAttestation`s to build a `PoWClaim`.
*   **Process:**
    1.  The Mobile Node identifies the `manifest_cid` of its content.
    2.  It needs to discover Witness Nodes. For MVP, this might be:
        *   Querying a pre-configured list of known Witness Nodes.
        *   (Future) Querying the DHT for peers advertising "witnessing services" (possibly for the content's topic or general availability).
    3.  The Mobile Node sends `pow_protocol.GetAttestationsRequest { manifest_cid: relevant_cid }` messages to selected/discovered Witness Nodes.
    4.  It collects the `WitnessAttestation` objects from the responses.
    5.  The mobile app should handle timeouts and potential unavailability of witnesses, possibly retrying with different witnesses or informing the user.

### 6.2. Generating PoW Claims

*   Once a Mobile Node has collected a sufficient number (target `M`) of valid `WitnessAttestation`s for its content:
    1.  It locally verifies each collected attestation:
        *   Checks the witness's signature (requires access to witness public keys - see Section 6.4).
        *   Ensures the attestation is for the correct `manifest_cid` and `content_hash_from_manifest`.
        *   Checks the plausibility of `observed_timestamp`.
    2.  If enough valid attestations are gathered, it constructs a `pow_protocol.PoWClaim` object, populating:
        *   `claim_id` (new UUID).
        *   `manifest_cid`, `originator_id` (self), `content_hash_from_manifest`.
        *   `earliest_observed_timestamp` (calculated from valid attestations, e.g., the earliest one).
        *   The list of valid `attestations`.
        *   `claim_timestamp` (current time).
    3.  This `PoWClaim` can then be stored locally, and potentially submitted to application-layer services (e.g., for content promotion, reward eligibility).

### 6.3. Verifying PoW Claims for Consumed Content

*   When a Mobile Node retrieves content created by others, it may also retrieve an associated `PoWClaim`.
*   **Process:**
    1.  The Mobile Node has the `PoWClaim` and the `manifest_cid` of the content.
    2.  It performs the full PoW Claim verification process as detailed in the PoW Protocol Specification (Section 7 of `POW_PROTOCOL_SPEC.md`):
        *   Retrieve the actual content (manifest and chunks) from DDS using the `manifest_cid`.
        *   Verify content integrity against `claim.content_hash_from_manifest`.
        *   Verify each `WitnessAttestation` within the claim (signatures, consistency, witness eligibility if possible, timestamp plausibility).
        *   Verify sufficiency of valid attestations.
        *   Verify the claim's `earliest_observed_timestamp`.
    3.  The result of this verification can be displayed to the user, influencing their trust in the content's originality and timestamp.

### 6.4. Witness Identity and Public Key Management (Simplified for MVP)

*   For Mobile Nodes to verify attestation signatures within `PoWClaim`s (either their own or others'), they need access to the public keys of Witness Nodes.
*   **MVP Approach:**
    *   A curated list of initial, trusted Witness Nodes and their public keys could be bundled with the application or fetched from a well-known, trusted endpoint.
    *   This simplifies initial key discovery.
*   **Future Considerations:**
    *   A decentralized PKI or DID-based system for Witness Node identity and key management.
    *   Witnesses could advertise their public keys on the DHT or as part of their node identity.

### 6.5. Non-Participation as Primary Witnesses (MVP)

*   As stated in Section 4.2 (Limitations), Mobile Nodes **will not** act as primary, always-on Witness Nodes that actively monitor the network for new content to attest to during the MVP.
*   They do not run the full "Attestation Service" described for dedicated Witness Nodes. Their PoW interactions are client-focused (requesting attestations, generating claims for their own content, verifying claims for consumed content).

This focused interaction model allows Mobile Nodes to benefit from the PoW protocol's assurances without bearing the resource burden of full witnessing duties.

---

## 7. Resource Management & Constraints for Mobile Nodes

Effective resource management is paramount for Mobile Nodes to ensure a good user experience and responsible device usage. Mobile applications implementing the DigiSocialBlock node functionality must adhere to these principles and provide user controls where appropriate.

### 7.1. Storage Management

*   **Local DDS Cache Limit:**
    *   **Specification:** Mobile Nodes will have a configurable limit for their local DDS cache (storing authored content chunks, consumed content chunks, and manifests).
    *   **Default MVP Limit:** A default limit of **500MB** is proposed for the MVP.
    *   **User Configuration:** Users MUST be able to adjust this limit (e.g., options like 100MB, 500MB, 1GB, 2GB, or custom value within reasonable OS-imposed bounds).
    *   **Display:** The application should clearly display current cache usage and the configured limit.
*   **Cache Eviction Policy:**
    *   **Specification:** When the cache limit is reached, an eviction policy will be triggered to free up space.
    *   **MVP Policy:** A **Least Recently Used (LRU)** policy for non-authored content is recommended. Content authored by the user SHOULD be prioritized and less aggressively evicted, or have a separate quota/guarantee.
    *   **User Controls:** Users could be given options to "pin" specific content items (authored or consumed) to prevent their eviction, or to manually clear parts of the cache (e.g., "clear consumed content cache," "clear all but my content").
*   **Other Storage:** Storage for application data, user identity keys, local PoW claims, and temporary files must also be managed efficiently, but the DDS cache is the primary concern for bulk data.

### 7.2. Data Usage Management

*   **Network Preference:**
    *   **Specification:** Mobile Nodes SHOULD prioritize Wi-Fi connections over cellular/mobile data for DDS operations involving large data transfers (e.g., initial seeding of authored content, fetching large media files, proactive caching of content).
    *   **User Configuration:** Users MUST have settings to control data usage:
        *   "Use Cellular Data for DDS": On/Off (default Off for large transfers).
        *   "Download large files only on Wi-Fi": On/Off (default On).
        *   (Future) Per-content type data usage rules.
*   **Background Data:**
    *   **Specification:** Background network activity (e.g., DHT participation, responding to `RetrieveChunkRequest` for cached items, opportunistic seeding) MUST be minimized and subject to OS limitations and user settings.
    *   **Throttling:** Implement throttling for background data transfers to avoid consuming excessive data allowances.
    *   **User Configuration:** "Allow Background Data Sync": On/Off, or "Wi-Fi Only" for background sync.
*   **Data Saver Mode Awareness:** The application SHOULD attempt to detect if the underlying OS is in a "Data Saver" mode and further restrict its data usage accordingly.
*   **Usage Statistics:** The application SHOULD provide users with an approximate overview of its data consumption (e.g., "DDS data uploaded/downloaded this month").

### 7.3. Battery Usage Management

*   **Minimize Background Activity:**
    *   **Specification:** PoW claim generation, extensive DDS seeding, or proactive caching of non-critical content SHOULD NOT run as intensive background tasks that drain the battery.
    *   **Opportunistic Execution:** These tasks should ideally be performed opportunistically:
        *   When the device is charging.
        *   When the device is connected to Wi-Fi (often correlates with less mobile/battery-sensitive scenarios).
        *   During user-initiated actions (e.g., explicit "publish" or "sync" action).
*   **Network Connection Management:**
    *   Minimize unnecessary keep-alives for P2P connections when the app is backgrounded, relying on OS push notifications (if applicable for social interactions) or resuming connections when the app returns to the foreground.
    *   `libp2p` connection management parameters should be tuned for mobile environments (e.g., shorter idle timeouts).
*   **CPU Usage Throttling for Background Tasks:** Any background processing (e.g., DHT maintenance) must be lightweight and use minimal CPU to conserve power.

### 7.4. CPU Usage Management

*   **Hashing & Cryptography:**
    *   SHA-256 hashing for CIDs and content verification, and signature generation/verification, are CPU-intensive.
    *   **Optimization:** Use efficient Go crypto libraries. Perform these operations asynchronously where possible to avoid blocking the UI.
    *   **User Feedback:** For user-initiated actions that require significant processing (e.g., preparing a large file for upload), provide clear progress indicators.
*   **Data (De)serialization:** Protobuf (de)serialization should be managed efficiently.
*   **Background Processing Limits:** Strictly limit CPU usage for any background tasks to align with OS best practices and prevent device slowdown or excessive battery drain.

Adherence to these resource management principles is crucial for the viability of Mobile Nodes as active and positive contributors to the DigiSocialBlock network.

---

## 8. Mobile-Specific Network Considerations

Mobile Nodes operate in a challenging network environment characterized by intermittent connectivity, changing IP addresses, and network address translation (NAT).

### 8.1. Connectivity Management

*   **NAT Traversal:**
    *   **Requirement:** Mobile Nodes MUST be able to operate effectively behind NATs and firewalls.
    *   **Solution:** Leverage `libp2p`'s NAT traversal capabilities, including:
        *   AutoNAT for discovering public address and NAT status.
        *   Hole punching techniques (e.g., DCUtR - Direct Connection Upgrade through Relay).
        *   Circuit Relay (libp2p Relay V2) as a fallback to allow connections to nodes that are otherwise unreachable directly. A set of community-run or incentivized relay nodes will be necessary.
*   **Intermittent Connections:**
    *   **Resumable Operations:** DDS seeding/uploads and content downloads MUST be resumable to handle network drops.
    *   **Connection Management:** Implement robust connection management with retries and backoff strategies for connecting to peers or Super Hosts.
    *   **Graceful Degradation:** The application should function gracefully (e.g., serving from local cache, queueing outgoing actions) when offline and seamlessly resume network operations when connectivity returns.
*   **Network Switching (Wi-Fi/Cellular):**
    *   The application SHOULD detect network changes and adapt its behavior (e.g., pause large cellular downloads if user preference is Wi-Fi only, re-establish DHT presence).
    *   `libp2p`'s multiaddress mechanism helps in advertising multiple potential endpoints.

### 8.2. Peer Discovery (DHT & Bootstrap)

*   **DHT Participation:** Mobile Nodes will participate in the Kademlia DHT.
    *   **Client Mode vs. Server Mode:** While mobile nodes can provide CIDs for their cached content, their DHT server capabilities might be limited by OS background restrictions and NATs. `libp2p` DHT client mode allows them to query the DHT effectively.
    *   **Optimized DHT Parameters:** Consider mobile-specific configurations for DHT (e.g., potentially fewer open connections, different refresh intervals if backgrounded).
*   **Bootstrap Peers:**
    *   A reliable set of bootstrap peer addresses is crucial for mobile nodes to join the DHT.
    *   This list can be hardcoded, fetched from a trusted discovery service, or dynamically updated.
    *   Bootstrap nodes should be stable and publicly reachable.
*   **Rendezvous Points (Optional):**
    *   For specific communities or use cases, libp2p's rendezvous protocol could be used to discover peers interested in a particular topic or group, supplementing DHT discovery.

### 8.3. Background Operation and OS Constraints

*   **iOS/Android Limitations:** Both operating systems heavily restrict background network activity, CPU usage, and task execution to conserve battery and data.
*   **Strategies:**
    *   **Foreground Service (Android):** For critical ongoing tasks like an active upload/download initiated by the user, an Android foreground service with a persistent notification can be used.
    *   **Opportunistic Background Tasks:** Use OS-provided mechanisms for deferrable tasks (e.g., Android's `WorkManager`, iOS's `BGTaskScheduler`) to perform non-critical background work like:
        *   Periodic (infrequent) DHT refreshes.
        *   Opportunistic seeding of remaining chunks when device is charging and on Wi-Fi.
        *   Checking for updates or new attestations for subscribed content.
    *   **Push Notifications (Application Layer):** For real-time social interactions (mentions, messages), rely on traditional push notification services (FCM, APNS). These can then trigger the app to wake up and perform specific P2P actions if needed and allowed. The P2P layer itself is not primarily for real-time push to backgrounded apps.
    *   **User Expectations:** Clearly communicate to the user what the app can and cannot do in the background based on their settings and OS version.

### 8.4. Libp2p Configuration for Mobile

*   **Transport Protocols:** Prioritize TCP and QUIC. WebSockets might be useful for browser-based light clients interacting with mobile or desktop nodes.
*   **Security Transports:** TLS 1.3 or Noise for securing connections.
*   **Multiplexing:** Yamux or Mplex for stream multiplexing.
*   **Connection Manager:** Configure connection limits (low/high watermarks) suitable for mobile devices to avoid exhausting resources. Shorter idle connection timeouts.
*   **PeerStore:** Persist discovered peer information (addresses, protocols, public keys) to reduce re-discovery overhead.

Careful tuning of `libp2p` and adherence to OS best practices are essential for a stable and resource-friendly mobile node experience.

---

## 9. Initial Mobile Node Implementation Plan (Conceptual)

This section outlines a high-level plan for the initial Go (or relevant mobile language like Swift/Kotlin with Go bindings via Gomobile) implementation of the Mobile Node functionality.

### 9.1. Key Mobile Application Components (Conceptual Modules/Services)

The mobile application acting as a DigiSocialBlock node would logically consist of several key components or services:

1.  **`ContentCreationService`:**
    *   **Responsibilities:** Handles UI interactions for creating/editing content (e.g., `ContentPost`). Populates the respective Protobuf message. Invokes hashing utilities to generate `post_id` (as per Content Hashing & Timestamping spec). Manages local drafts.
    *   **Interfaces with:** UI, `DDSMobileClient` (for initiating storage/seeding).
2.  **`DDSMobileClient`:**
    *   **Responsibilities:**
        *   Manages the local DDS cache (e.g., using an embedded database or file system storage adhering to `StorageManager` interface principles from DDS spec).
        *   Implements cache eviction policies and user controls for cache size.
        *   Handles chunking of outgoing content and manifest creation (using `chunker` utilities).
        *   Interacts with the `P2PManager` to:
            *   Store/seed own content chunks/manifest to local cache and then to selected Super Hosts/Storage Nodes via DDS RPCs (`StoreChunkRequest`).
            *   Retrieve external content chunks/manifest via DDS RPCs (`RetrieveChunkRequest`) from providers found via DHT.
            *   Advertise CIDs of locally cached content to the DHT (`Provide`).
    *   **Interfaces with:** `ContentCreationService`, `ContentConsumptionService` (UI layer), `P2PManager`, local cache storage.
3.  **`PoWMobileClient` (Proof-of-Witness Client Logic):**
    *   **Responsibilities:**
        *   For authored content: Queries Witness Nodes (via `P2PManager` and PoW RPCs like `GetAttestationsRequest`) for `WitnessAttestation`s for a given `manifest_cid`.
        *   Validates received attestations (signatures, consistency).
        *   Generates `PoWClaim` objects from valid attestations.
        *   For consumed content: Verifies received `PoWClaim` objects (involves fetching content from DDS via `DDSMobileClient` for cross-verification).
    *   **Interfaces with:** UI (to display claim status), `P2PManager`, `DDSMobileClient`.
4.  **`P2PManager (Mobile)`:**
    *   **Responsibilities:**
        *   Manages the underlying `libp2p` host instance configured for mobile constraints (transports, connection manager, NAT traversal).
        *   Handles DHT client participation (bootstrapping, `Provide` calls, `FindProviders` queries).
        *   Manages peer connections.
        *   Exposes an interface for sending/receiving DDS and PoW RPC messages (e.g., wrapping `libp2p` stream/protocol handlers).
    *   **Interfaces with:** `DDSMobileClient`, `PoWMobileClient`, OS network APIs (indirectly via libp2p).
5.  **`ResourceManager`:**
    *   **Responsibilities:**
        *   Monitors device resource usage (storage for DDS cache, data consumption, battery level, charging state).
        *   Enforces configured limits (e.g., cache size, data usage policies like Wi-Fi only for large transfers).
        *   Provides signals/callbacks to other services (e.g., `DDSMobileClient`, `P2PManager`) to adapt behavior (e.g., pause seeding if on cellular and policy disallows, throttle background activity if battery low).
    *   **Interfaces with:** OS APIs for resource status, all other P2P-active services.
6.  **`IdentityService (Mobile)`:**
    *   **Responsibilities:** Securely manages the user's cryptographic identity (private/public keys). Provides functions for signing data (e.g., content IDs, attestations if future limited witnessing is enabled).
    *   **Interfaces with:** `ContentCreationService`, `PoWMobileClient`.
7.  **UI/Application Layer:**
    *   Presents user interface for content creation, browsing, interaction.
    *   Displays status of DDS operations (uploads, downloads), PoW claims.
    *   Provides user settings for resource management, privacy, etc.

This component breakdown allows for a modular approach to building the mobile node client.

---

### 9.2. MVP Scope for Mobile Node Functionality

The Minimum Viable Product (MVP) for the Mobile Node will focus on demonstrating core user flows related to content creation, basic P2P storage/retrieval, and PoW interaction, while respecting mobile constraints.

**Included in MVP Functionality:**

1.  **Content Creation & Local Processing:**
    *   User can create a simple text-based `ContentPost` via the mobile UI.
    *   The mobile app correctly generates the `post_id` (using canonical hashing) and sets the `created_at` timestamp.
    *   The mobile app correctly prepares the `ContentPost` for DDS: serializes it, chunks it, generates CIDs for chunks, creates a `ContentManifest`, and generates the `manifest_cid`.
2.  **Local Caching & DDS Provision:**
    *   All chunks and the manifest for the user's authored post are stored in the mobile's local DDS cache.
    *   The mobile node successfully advertises these CIDs to the Kademlia DHT via its `P2PManager`.
3.  **Simplified Seeding to Super Hosts:**
    *   The mobile node attempts to connect to a small, pre-configured list of 1-2 "Super Host" addresses.
    *   If successful, it sends `StoreChunkRequest` messages to these Super Hosts for all chunks (data + manifest) of its newly created post.
    *   Basic error handling and retry for seeding (e.g., retry once if initial attempt fails).
4.  **Content Retrieval from Network:**
    *   Given a `manifest_cid` (e.g., for content authored by another user, potentially stored on Super Hosts or other mobile nodes):
        *   The mobile node can query the DHT to find providers for the manifest.
        *   It can retrieve the manifest chunk from a provider.
        *   It can parse the manifest and then query the DHT for providers of individual data chunks.
        *   It can retrieve and verify (against CIDs) all data chunks.
        *   It can reassemble the content and verify it against the `original_content_hash` from the manifest.
        *   The mobile app can display the retrieved text content.
5.  **Basic PoW Claim Verification:**
    *   If a `PoWClaim` is associated with retrieved content (e.g., fetched alongside or via a link):
        *   The mobile node can perform local verification of the claim. This includes:
            *   Verifying signatures on `WitnessAttestation`s (using a bundled/known list of Witness public keys for MVP).
            *   Ensuring consistency of hashes within the claim and attestations against the retrieved content.
            *   Checking for a minimum number of valid attestations.
    *   Display a simple "verified" / "unverified" status to the user.
6.  **Resource Constraint Stubs & Basic Settings:**
    *   Implement basic stubs for the `ResourceManager` that allow setting a fixed cache size (e.g., 500MB) and a Wi-Fi only policy for seeding. No complex dynamic adjustments for MVP.
7.  **Basic P2P Stack:**
    *   Initialize and run a `libp2p` host.
    *   Join the DHT using bootstrap peers.
    *   Handle basic incoming `RetrieveChunkRequest`s for content it is providing from its cache.

**Excluded from MVP Functionality:**

*   **Mobile Node as Full Witness:** Mobile nodes will not run the PoW `Attester` service.
*   **Advanced/Proactive Replication by Mobile:** Beyond initial seeding of own content.
*   **Complex Caching Strategies:** Simple "store what I created/retrieved" up to a limit, with basic LRU for consumed content.
*   **Sophisticated Resource Management:** No dynamic adjustment to battery/data/CPU conditions beyond basic Wi-Fi preference for seeding.
*   **User Interface for PoW Claim Generation:** While claim verification is in scope, the UI/UX for a mobile user to *initiate* gathering attestations and *generate* a PoW claim for their own content might be simplified or deferred if too complex for initial MVP UI. Focus on verification of existing claims first.
*   **Offline Content Creation Queue:** While desirable, a robust queueing and background sync mechanism for content created while offline is a more advanced feature. MVP assumes online for creation/seeding.
*   **Handling of Diverse Content Types by Mobile:** MVP primarily focuses on text posts for simplicity in creation and display. Retrieval of image/video posts might work if the DDS part is generic, but full handling in UI is secondary.
*   **Decentralized Discovery of Super Hosts:** Relies on a pre-configured list for MVP.

The Mobile Node MVP aims to demonstrate that a user can create content, have it stored decentrally (with help from Super Hosts), and that other users (mobile or desktop) can retrieve and verify this content and its associated PoW claims.

---

### 9.3. Test Strategy for Mobile Node MVP (Conceptual)

Testing the Mobile Node MVP will involve a combination of unit tests for individual components and integration tests for end-to-end workflows, likely in a simulated/mocked P2P environment initially.

**1. Unit Tests:**

*   **`ContentCreationService`:**
    *   Test `post_id` generation logic for various `ContentPost` inputs.
    *   Test correct population of `ContentPost` fields (e.g., `created_at`).
*   **`DDSMobileClient`:**
    *   **Local Cache Logic:**
        *   Test `Store`, `Retrieve`, `Has`, `Delete` operations on the local cache.
        *   Test cache eviction policy (e.g., LRU, priority for authored content) when cache limit is reached.
        *   Test cache size limit enforcement.
    *   **Chunking & Manifest:** (These might call `chunker` utilities directly, so focus on integration)
        *   Verify correct interaction with `chunker` for content preparation.
    *   **Seeding Strategy Logic:**
        *   Test selection of Super Hosts (e.g., from a mock list).
        *   Test resumable upload logic conceptually (e.g., if an upload is interrupted and resumed, it continues correctly). This is harder to unit test without network mocks.
        *   Test Wi-Fi vs. cellular policy for seeding.
*   **`PoWMobileClient`:**
    *   Test logic for assembling `GetAttestationsRequest`.
    *   Test validation of received `WitnessAttestation`s (mocking valid/invalid attestations).
    *   Test `PoWClaim` generation from a set of mock attestations.
    *   Test `PoWClaim` verification logic (mocking DDS responses for content and witness pubkeys).
*   **`P2PManager (Mobile)`:**
    *   Unit tests for specific mobile-tuned configurations if any (e.g., connection manager parameters). Much of libp2p's core is externally tested. Focus on the wrapper logic.
*   **`ResourceManager`:**
    *   Test enforcement of configured policies (e.g., if Wi-Fi only is set, verify seeding attempts on cellular are blocked or queued).
    *   Test correct reading of (mocked) device resource states (battery, network type).

**2. Integration Tests (Simulated Environment):**

These tests require a controlled environment, potentially with multiple in-process libp2p nodes or mocked network interfaces.

*   **Test 1: Content Creation, Local Cache, DHT Provide, and Seed to Super Host:**
    1.  Setup: Mobile Node A, Mock Super Host Node. Both join a test DHT.
    2.  Mobile Node A: User creates a new `ContentPost`.
    3.  Verify:
        *   `post_id` generated correctly.
        *   Content chunked, manifest created.
        *   All chunks (data + manifest) stored in Mobile Node A's local cache.
        *   Mobile Node A successfully advertises all CIDs to the DHT.
        *   Mobile Node A identifies Mock Super Host and successfully sends all chunks via `StoreChunkRequest` RPCs.
        *   Mock Super Host correctly stores the chunks.
*   **Test 2: Content Retrieval from Super Host by another Mobile Node:**
    1.  Setup: Mock Super Host (with content from Test 1), Mobile Node B. Both join test DHT.
    2.  Mobile Node B: Is given the `manifest_cid` of the content.
    3.  Verify:
        *   Mobile Node B queries DHT, discovers Mock Super Host as provider for manifest.
        *   Retrieves and parses manifest from Super Host.
        *   Queries DHT for data chunk providers (should find Super Host).
        *   Retrieves, verifies, and reassembles all data chunks.
        *   Verifies final content against `original_content_hash`.
        *   Content is now in Mobile Node B's local cache.
*   **Test 3: Content Retrieval from Mobile Node A by Mobile Node B (Peer-to-Peer):**
    1.  Setup: Mobile Node A (with locally cached/authored content, providing to DHT), Mobile Node B. Both join test DHT.
    2.  Mobile Node B: Is given `manifest_cid` of content on Mobile Node A.
    3.  Verify: Similar retrieval flow as Test 2, but providers found will include Mobile Node A. Tests direct mobile-to-mobile chunk transfer.
*   **Test 4: PoW Claim Verification on Mobile Node:**
    1.  Setup: Mobile Node C, Mock DDS (populated with content for a given `manifest_cid`), a valid `PoWClaim` object (with mock attestations signed by known test keys), and a way to provide Witness public keys to Mobile Node C.
    2.  Mobile Node C: Receives the `PoWClaim`.
    3.  Verify: Mobile Node C successfully executes the `VerifyPoWClaim` logic:
        *   Fetches content from Mock DDS.
        *   Verifies content hash.
        *   Verifies all attestation signatures and consistency.
        *   Confirms sufficient valid attestations.
        *   (If applicable) Verifies `earliest_observed_timestamp`.
    4.  Test with an invalid claim (e.g., bad signature, mismatched hash) and verify rejection.

These tests will focus on validating the core user stories and protocol interactions for the Mobile Node MVP.

---
*(End of Document: Mobile Node Role Technical Specification & Initial Implementation Plan)*
