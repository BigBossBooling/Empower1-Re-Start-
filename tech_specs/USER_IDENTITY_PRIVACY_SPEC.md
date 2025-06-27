# DigiSocialBlock (Nexus Protocol) - User Identity & Privacy Layer Technical Specification

## 1. Objective

This document provides the detailed technical specifications for the User Identity & Privacy Layer of DigiSocialBlock. This layer is fundamental to establishing user sovereignty, enabling secure interactions, and providing mechanisms for privacy-preserving data sharing. It focuses on a custom Decentralized Identifier (DID) method (`did:echonet`), the integration of Zero-Knowledge Proofs (ZKPs) for selective data disclosure, and a robust consent management framework.

## 2. Scope

This specification covers:
*   The detailed definition of the `did:echonet` DID method, including identifier format, DID Document (DDO) structure, and CRUD (Create, Read, Update, Deactivate/Revoke) operations for DDOs.
*   The proposed application and high-level integration strategy for Zero-Knowledge Proofs (ZKPs) to enhance user privacy, focusing on selective attribute disclosure.
*   The design of a consent management system, including consent records, storage, and verification mechanisms, to empower users with granular control over their data.
*   Conceptual data structures (Protobuf) for DDOs, ZKP interactions, and Consent Records.
*   Interaction of this layer with other DigiSocialBlock protocols (e.g., DDS for DDO storage).

This document builds upon the conceptual design outlined in `tech_specs/user_identity_privacy.md` (Sections 1, 2, and 3).

## 3. Relation to `user_identity_privacy.md`

This specification directly elaborates on the principles and components introduced in Sections 1 ("Decentralized Identity (DID)"), 2 ("Zero-Knowledge Proofs (ZKPs) for Enhanced Privacy"), and 3 ("Consent Management & Data Control") of the `tech_specs/user_identity_privacy.md` document. It aims to provide the next level of detail required for implementation planning and development of the identity and privacy infrastructure for DigiSocialBlock.

---

## 4. Decentralized Identifier (DID) Method: `did:echonet`

The `did:echonet` method provides a decentralized, user-controlled identity mechanism within the DigiSocialBlock ecosystem. It allows users to create and manage their digital identities without reliance on centralized authorities.

### 4.1. DID Method Name

*   **Formal Name:** `echonet`
*   **DID URI Scheme:** `did:echonet:<method-specific-identifier>`

### 4.2. Method-Specific Identifier (MSI)

*   **Generation:** The MSI for a `did:echonet` DID is generated from the user's initial cryptographic key pair.
    1.  A user generates a primary cryptographic key pair (e.g., Ed25519 for signatures, X25519 for key agreement, or a key that can serve both purposes if the scheme allows).
    2.  The MSI will be the **Base58BTC encoding of the SHA-256 hash of the initial primary public key** used for authentication/verification methods in the DID Document.
    *   Example: `MSI = Base58BTC(SHA256(initial_authentication_public_key_bytes))`
*   **Uniqueness:** This approach ensures that the MSI is cryptographically tied to the key pair that initially controls the DID, providing a strong link and preventing arbitrary MSI creation without key ownership. It also ensures a high probability of global uniqueness.
*   **Format:** The resulting MSI will be a Base58BTC encoded string.
*   **Example DID:** `did:echonet:1A2b3C4d5E6f7G8h9I0j` (Illustrative MSI part)

### 4.3. DID Document (DDO)

The DID Document (DDO) contains public keys, service endpoints, and other metadata associated with a `did:echonet` DID, enabling discovery and interaction.

*   **Structure (Conceptual Protobuf Definition - `identity_protocol.proto` or similar):**
    ```protobuf
    syntax = "proto3";

    package identity_protocol; // Or a shared types package

    import "google/protobuf/timestamp.proto";

    option go_package = "empower1/pkg/identity_rpc;identity_rpc"; // Example

    message VerificationMethod {
      string id = 1;                      // Full DID URL fragment, e.g., "did:echonet:xyz#keys-1"
      string type = 2;                    // Key type, e.g., "Ed25519VerificationKey2020", "X25519KeyAgreementKey2019"
      string controller = 3;              // The DID that controls this verification method (usually the DID itself)
      string public_key_multibase = 4;    // Public key encoded in Multibase format (e.g., Base58BTC prefix 'z')
      // bytes public_key_jwk = 5;        // Alternative: JWK format (JSON)
    }

    message ServiceEndpoint {
      string id = 1;                      // Full DID URL fragment, e.g., "did:echonet:xyz#dwn"
      string type = 2;                    // Service type, e.g., "DecentralizedWebNode", "MessagingService"
      string service_endpoint = 3;        // URI for the service (URL, DID, etc.)
      // map<string, google.protobuf.Value> properties = 4; // Additional properties
    }

    message DIDDocument {
      // context field often omitted in Protobuf, handled by application layer if needed for JSON-LD.
      // string context = 1; // e.g., "https://www.w3.org/ns/did/v1"
      string id = 1;                                // The DID itself, e.g., "did:echonet:xyz"
      repeated string controller = 2;               // DID(s) authorized to update this DDO. Can be self.
      repeated VerificationMethod verification_method = 3;
      repeated string authentication = 4;           // References id(s) from verification_method
      repeated string assertion_method = 5;         // Optional: References id(s) from verification_method
      repeated string key_agreement = 6;            // References id(s) from verification_method for encryption
      repeated string capability_invocation = 7;    // Optional: For invoking capabilities
      repeated string capability_delegation = 8;    // Optional: For delegating capabilities
      repeated ServiceEndpoint service = 9;         // Service endpoints
      google.protobuf.Timestamp created = 10;       // Timestamp of DDO creation
      google.protobuf.Timestamp updated = 11;       // Timestamp of last DDO update
      string version_id = 12;                     // Monotonically increasing version number or hash of previous DDO for history
      // string previous_ddo_cid = 13; // Optional: CID of the previous version of this DDO on DDS
    }
    ```
*   **Key Management:** The DDO lists public keys. Private keys are managed securely by the user/DID controller.
*   **Standard Compliance:** Aims to be compatible with W3C DID Core specifications where practical.

### 4.4. CRUD Operations on DID Documents

1.  **Create (Register):**
    *   User generates a new primary key pair(s) locally.
    *   The MSI is derived from the initial public key. The full DID (`did:echonet:<MSI>`) is formed.
    *   A `DIDDocument` is constructed locally, populating `id`, `controller` (self), `verificationMethod` (with the new public keys), `authentication`, `keyAgreement`, `created`, and `updated` timestamps. `version_id` is set to an initial value (e.g., "1" or a hash of initial content).
    *   The DDO is serialized (e.g., Protobuf binary).
    *   This serialized DDO is stored on the DDS, yielding a `DDO_CID`.
    *   **Anchoring/Publication:** The mapping `DID -> DDO_CID` needs to be made discoverable. For MVP, this could be:
        *   The user's client announces this mapping to the Kademlia DHT (e.g., `DHT.Put(key=DID_string, value=DDO_CID_string)`).
        *   Other peers can then resolve the DID by querying the DHT.
    *   No central registry is used. Initial trust in a DID relies on the cryptographic link to its keys and subsequent interactions/attestations.

2.  **Read (Resolve):**
    *   Given a `did:echonet:<MSI>` string.
    *   A resolver queries the Kademlia DHT for the key equal to the DID string.
    *   The DHT lookup should return the `DDO_CID` (the CID of the latest version of the DDO stored on DDS).
    *   The resolver then retrieves the DDO data from DDS using this `DDO_CID`.
    *   The DDO data is deserialized and validated (e.g., check signatures if DDOs themselves are signed, verify structure).
    *   If multiple DDO CIDs are found (e.g., due to network propagation or updates), logic to determine the latest/canonical version is needed (e.g., based on `updated` timestamp within the DDO, or a `version_id` field).

3.  **Update:**
    *   The DID Controller (authorized by a key listed in the current DDO's `controller` and `authentication` fields) creates a new version of the DDO.
    *   The `updated` timestamp is set to the current time.
    *   A new `version_id` is generated (e.g., incrementing a sequence number or hashing the new DDO content).
    *   (Optional) The new DDO can include a `previous_ddo_cid` field pointing to the CID of the version it replaces, forming a chain.
    *   The new DDO is signed by a key authorized in the *previous* DDO version to prove update authority. (This signed update instruction or the new DDO itself is stored).
    *   The new serialized DDO is stored on DDS, yielding a new `New_DDO_CID`.
    *   The mapping in the DHT for `DID -> DDO_CID` is updated to point to `New_DDO_CID`. This update itself must be authorized (e.g., signed by the DID controller, old DHT record replaced if allowed by DHT write rules, or by using a mutable pointer system like IPNS if libp2p IPNS is used, though Kademlia PUTs are simpler for MVP).
    *   For MVP, updating the DHT record might involve a simple PUT operation, relying on nodes fetching the DDO from DDS and checking its internal `updated` timestamp and `version_id` to determine the latest if multiple CIDs are found for a DID over time.

4.  **Deactivate (Revoke - Conceptual):**
    *   **Method 1 (DDO Update):** The DID controller updates the DDO to include a specific property indicating it's deactivated (e.g., a `deactivated: true` boolean field, or by removing all verification methods and service endpoints except for a proof of deactivation). The updated DDO is published as per the Update operation.
    *   **Method 2 (Revocation List - Future):** A separate, decentralized revocation list mechanism (e.g., a Verifiable Credential Revocation List, or a smart contract on a sidechain) could be used. This is more complex and likely beyond MVP.
    *   For MVP, updating the DDO to signal deactivation is preferred. Resolution would still find the DDO, but applications would interpret its state as deactivated.

### 4.5. Storage of DID Documents (DDOs)

*   Serialized DID Documents are stored as immutable, content-addressed objects on the **Distributed Data Stores (DDS)** protocol (Module 1.2).
*   Each version of a DDO will have a unique CID on the DDS.
*   The DID itself resolves (e.g., via DHT) to the CID of the *latest active version* of its DDO on the DDS.

This DID method provides a foundation for user-controlled, cryptographically verifiable identities within DigiSocialBlock.

---

## 5. Zero-Knowledge Proofs (ZKPs) for Enhanced Privacy

Zero-Knowledge Proofs will be integrated into DigiSocialBlock to allow users to prove statements about their identity attributes or credentials without revealing the underlying sensitive data.

### 5.1. Primary Use Cases for ZKPs (MVP Focus)

1.  **Selective Disclosure from Verifiable Credentials (VCs):**
    *   **Scenario:** A user holds a Verifiable Credential (e.g., issued by a trusted entity or self-issued, linked to their `did:echonet` identity) containing multiple attributes (e.g., `{ "type": "AgeCredential", "age_over_18": true, "birth_year": 1990 }`).
    *   **Goal:** The user wants to prove to a service or another user that they are over 18 (`age_over_18: true`) *without* revealing their exact birth year or the full VC.
    *   **ZKP Role:** The user generates a ZKP demonstrating that they possess a valid VC (e.g., signed by a trusted issuer or their own DID) which contains the attribute `age_over_18` with the value `true`.
2.  **Anonymous Verification of Group Membership (Future, Post-MVP):**
    *   Prove membership in a `CommunityGroup` without revealing which specific member one is.
3.  **Private Polls/Voting (Future, Post-MVP):**
    *   Vote on a proposal without revealing one's specific vote, only that one is eligible to vote and has not voted before.

For MVP, the focus will be on the **specification and conceptual integration for Selective Disclosure from VCs.** Full implementation of ZKP circuits and proving systems is a significant undertaking often deferred beyond initial core feature sets.

### 5.2. Proposed ZKP Scheme & Libraries (High-Level)

*   **Type of ZKP:**
    *   **zk-SNARKs (Zero-Knowledge Succinct Non-Interactive Argument of Knowledge):** Schemes like **Groth16** (common, efficient proofs, but requires per-circuit trusted setup) or newer universal/updatable setup schemes (e.g., PLONK, Marlin) are candidates.
    *   **Bulletproofs:** Offer smaller proof sizes than some older zk-SNARKs for certain statements and do not require a trusted setup, but verification can be slower. Good for range proofs.
*   **Selection Criteria for MVP Planning:**
    *   Availability of mature and well-audited Go libraries.
    *   Ease of circuit development for common predicates (e.g., "attribute equals value," "attribute is in range").
    *   Performance (proofing time on mobile, verification time on verifier node).
*   **Potential Go Libraries (to be evaluated):**
    *   **`ConsenSys/gnark`:** A comprehensive Go library for zk-SNARKs (Groth16, PLONK) allowing circuit development in Go.
    *   Other libraries for specific schemes as they mature in the Go ecosystem.
*   **Circuit Definition:** Circuits will be defined for specific statements (e.g., "I have a VC of type 'X' signed by issuer 'Y' where attribute 'Z' has value 'V'").

### 5.3. Integration with DIDs & Verifiable Credentials (VCs)

1.  **Issuance of VCs:**
    *   Users obtain VCs from issuers (or self-issue them). These VCs are cryptographically signed by the issuer and linked to the user's `did:echonet` (e.g., the `credentialSubject.id` in the VC is the user's DID).
    *   VCs are stored by the user (e.g., in a local wallet, encrypted on their private DDS space).
2.  **ZKP Generation (Prover - User's Client):**
    *   A service or peer (Verifier) requests proof of a specific statement (e.g., "Are you over 18?").
    *   The user's client identifies the relevant VC(s) they hold.
    *   Using the ZKP library and a pre-defined circuit for the requested statement:
        *   The **private inputs** to the circuit are the full VC data (or specific sensitive parts like birth year).
        *   The **public inputs** are non-sensitive elements related to the proof (e.g., hash of the VC schema, issuer's DID, the specific predicate being proven like "age_over_18 is true", a current challenge/nonce from the verifier to prevent replay).
        *   The client generates the ZKP (`proof_bytes`).
3.  **ZKP Verification (Verifier - Service/Peer):**
    *   The User's client sends the `ZKPProof { proof_bytes, public_inputs }` to the Verifier.
    *   The Verifier, using the same circuit definition (or its verification key) and the received `public_inputs`, runs the ZKP verification algorithm against `proof_bytes`.
    *   If verification passes, the Verifier is assured the user can satisfy the statement without learning the private inputs.

### 5.4. Conceptual Data Structures for ZKP Interaction

*   **Conceptual Protobuf Definitions (`zkp_protocol.proto` or similar):**
    ```protobuf
    // In a conceptual zkp_protocol.proto
    message ZKPredicate { // Describes the statement to be proven
      string credential_type_required = 1; // e.g., "AgeCredential"
      string issuer_did_required = 2;     // Optional: DID of the required issuer
      message AttributeClaim {
        string attribute_name = 1;
        string attribute_value_proven = 2; // e.g., "true" for a boolean, or a specific value for equality
        // enum ComparisonType { EQUALS = 0; GREATER_THAN = 1; ... }
        // ComparisonType comparison = 3;
      }
      repeated AttributeClaim claims = 3;
    }

    message ZKPChallengeRequest { // Verifier to Prover
      string session_id = 1;        // Unique ID for this proof session
      ZKPredicate predicate = 2;    // The statement the verifier wants proof of
      string nonce = 3;             // Fresh nonce to prevent replay attacks
    }

    message ZKPProofSubmission { // Prover to Verifier
      string session_id = 1;
      bytes proof_bytes = 2;        // The actual ZKP (e.g., Groth16 proof)
      // Public inputs used for proof generation, structured appropriately for the circuit.
      // This might be a google.protobuf.Struct or specific fields.
      // For example:
      string vc_schema_hash = 3;    // Hash of the schema of the VC used
      string issuer_did_used = 4;   // DID of the issuer of the VC used
      string predicate_satisfied_hash = 5; // Hash of the specific predicate proven
      string verifier_nonce = 6;    // The nonce from ZKPChallengeRequest
    }

    message ZKPVerificationResult { // Verifier to Prover (optional ack)
      string session_id = 1;
      bool verified = 2;
      string message = 3; // e.g., "Proof verified" or "Verification failed"
    }
    ```

### 5.5. MVP Scope & Implementation Notes

*   **MVP Focus:**
    *   Detailed specification of **one primary use case:** e.g., proving possession of an attribute like "is_over_18" from a conceptual VC without revealing the birthdate.
    *   Define the structure of the conceptual VC that would hold such an attribute.
    *   Define the public and private inputs for the ZKP circuit for this use case.
    *   Define the conceptual Protobuf messages for requesting and submitting this specific proof.
*   **Implementation Deferral:** Actual development of ZKP circuits, proving key generation, trusted setups (if needed by the chosen scheme), and integration of ZKP libraries into mobile clients is a complex task. For the initial core DigiSocialBlock MVP, this will likely be:
    *   **Stubbed:** Go interfaces and placeholder functions for `GenerateProof()` and `VerifyProof()` will be created.
    *   These stubs might return `true` with dummy proof data for workflow testing, or `ErrNotImplemented`.
*   **Security:** Emphasize that ZKP security relies entirely on the correctness of the circuit design and the underlying cryptographic scheme. Audits are crucial for real deployments.

This approach allows the overall system architecture to account for ZKP integration from the start, even if the full cryptographic machinery is implemented in a later phase.

---

## 6. Consent Management & Data Control

Empowering users with granular control over their data and how it's shared is a cornerstone of DigiSocialBlock. This section outlines the consent management framework.

### 6.1. Core Principles

*   **User Sovereignty:** Users own their data and have the final say on who can access it and for what purpose.
*   **Explicit Consent:** Access to user data (beyond what's publicly shared by the user) requires explicit, informed consent.
*   **Granularity:** Users should be able to grant consent for specific data scopes, specific requesters, and specific permissions.
*   **Revocability:** Users MUST be able to revoke previously granted consent.
*   **Verifiability:** Consent grants and revocations should be verifiable (e.g., cryptographically signed by the user).

### 6.2. `ConsentRecord` Data Structure

A `ConsentRecord` captures the details of a user's consent grant.

*   **Conceptual Protobuf Definition (`identity_protocol.proto` or similar):**
    ```protobuf
    enum ConsentPermission {
      CONSENT_PERMISSION_UNSPECIFIED = 0;
      CONSENT_PERMISSION_VIEW = 1;         // Permission to read/view the data.
      CONSENT_PERMISSION_EDIT = 2;         // Permission to modify (if data type allows).
      CONSENT_PERMISSION_RESHARE_LIMITED = 3; // Permission to reshare with specific constraints.
      CONSENT_PERMISSION_RESHARE_FULL = 4;  // Permission to reshare broadly.
      CONSENT_PERMISSION_USE_FOR_ANALYTICS = 5; // Permission for data to be included in aggregated, anonymized analytics.
    }

    message ConsentRecord {
      string consent_id = 1;                    // UUID for this consent grant.
      string granter_did = 2;                   // DID of the user granting consent (e.g., "did:echonet:xyz").
      string requester_id = 3;                  // DID of the user or ID of the application/service requesting access.
      repeated string data_scope_references = 4;  // Identifiers for the data being consented to.
                                                // Could be:
                                                // - CIDs of specific DDS objects (manifests or chunks).
                                                // - Tags/categories defined by the user (e.g., "profile.email", "posts.friends_only").
                                                // - Specific field paths within a larger data structure.
      string purpose_description = 5;           // Human-readable description of why access is requested.
      repeated ConsentPermission permissions_granted = 6; // List of specific permissions granted.
      google.protobuf.Timestamp issuance_timestamp = 7;   // When the consent was granted.
      google.protobuf.Timestamp expiry_timestamp = 8;     // Optional: When the consent automatically expires.
      string revocation_signature = 9;          // Optional: If revoked, this is granter's signature over (consent_id + revocation_timestamp).
      google.protobuf.Timestamp revocation_timestamp = 10; // Optional: When consent was revoked.
      bytes granter_signature = 11;               // User's (granter_did) signature over a canonical representation of
                                                // (consent_id + requester_id + data_scope_references + purpose_description + permissions_granted + issuance_timestamp + expiry_timestamp).
      // map<string, string> context_metadata = 12; // Optional: Application-specific context for the consent.
    }
    ```
*   **Key Fields Explained:**
    *   `granter_did`: The user giving consent.
    *   `requester_id`: The entity (user DID or application ID) receiving consent.
    *   `data_scope_references`: Specifies what data is covered. This needs careful design for flexibility (specific CIDs, user-defined tags/labels, or structured data paths).
    *   `purpose_description`: Essential for informed consent.
    *   `permissions_granted`: Granular permissions.
    *   `expiry_timestamp`: Allows for time-limited consent.
    *   `revocation_signature`, `revocation_timestamp`: Mechanism for verifiable revocation.
    *   `granter_signature`: Makes the consent grant itself a verifiable, non-repudiable record.

### 6.3. Storage and Discovery of Consent Records

*   **User-Controlled Storage (Primary):**
    *   `ConsentRecord`s are primarily created and managed by the user's client application.
    *   They SHOULD be stored in a location under the user's control, for example:
        *   Encrypted within the user's private space on the DDS.
        *   In a local wallet or user-managed data pod.
*   **Discoverability for Requesters:**
    *   When a `requester_id` (application or another user) needs to access data, it must present a valid `ConsentRecord` (or a reference to one) that authorizes the access.
    *   **Option 1 (Requester Presents):** The user, upon granting consent, might share the `ConsentRecord` (or its CID if on DDS) directly with the requester. The requester then presents this to the data source or to the user's node when attempting access.
    *   **Option 2 (Service Endpoint in DDO):** A user's DDO could list a `serviceEndpoint` of type `ConsentVerificationService` or `DataSharingService`. Requesters would interact with this endpoint, which would internally check the user's stored `ConsentRecord`s.
    *   **Option 3 (Consent Registry - More Centralized):** A dedicated, potentially decentralized (e.g., DHT-based or sidechain) "Consent Registry" where CIDs of (or hashes of) `ConsentRecord`s are published. This is more complex.
*   **MVP Focus:** For MVP, Option 1 (requester presents consent provided by user) is the simplest to start with. The user's client would manage their consent grants and provide them to services as needed.

### 6.4. Consent Enforcement

*   **Application Layer Responsibility:** The actual enforcement of consent (i.e., allowing or denying access to data based on a valid `ConsentRecord`) is primarily an **application-layer or service-layer responsibility.**
*   **Verification Process by Data Holder/Service:**
    1.  Requester attempts to access data related to `granter_did`.
    2.  Data holder/service demands a `ConsentRecord` (or proof of consent).
    3.  Requester provides the `ConsentRecord`.
    4.  Data holder/service verifies:
        *   `granter_signature` on the `ConsentRecord` using `granter_did`'s public key.
        *   `requester_id` matches the entity attempting access.
        *   Current time is before `expiry_timestamp` (if set).
        *   `revocation_signature` is not present (or if present, `revocation_timestamp` is validly set).
        *   The requested `data_scope_references` and intended action align with `permissions_granted`.
    5.  If all checks pass, access is granted according to the permissions.

### 6.5. Consent Revocation

*   **Mechanism:**
    1.  User (granter) decides to revoke a specific `consent_id`.
    2.  User's client creates a revocation statement (e.g., `consent_id + current_timestamp`).
    3.  User signs this statement, producing `revocation_signature`.
    4.  The original `ConsentRecord` is updated (or a new record linked to it is created) with the `revocation_signature` and `revocation_timestamp`.
*   **Notification:** The system needs a way to notify the `requester_id` that consent has been revoked. This could be:
    *   Proactive notification from the user's client/service endpoint.
    *   Requesters periodically re-validating consent with the user's consent service endpoint.
    *   If a Consent Registry is used, the revocation status can be updated there.
*   **Effect:** Once revoked, the `ConsentRecord` is no longer valid for authorizing data access.

This framework aims to provide a transparent, user-centric approach to data sharing and privacy.

---

## 7. Initial Implementation Plan (Conceptual)

This section outlines a high-level plan for the initial Go implementation of the User Identity & Privacy Layer (MVP).

### 7.1. Core Go Package Structure (Proposed)

The functionalities for this layer will be organized into several Go packages, likely under `internal/identity` or a similar top-level directory.

```
identity/
|-- did/                // DID method implementation (did:echonet)
|   |-- did_echonet.go    // Logic for generating DIDs, creating/updating DDOs.
|   |-- did_resolver.go // Interface and implementation for resolving DIDs (e.g., via DHT/DDS).
|   `-- did_echonet_test.go
|
|-- zkp/                // Zero-Knowledge Proof utilities and integration points.
|   |-- interface.go    // Defines interfaces for Prover and Verifier services.
|   |-- stubs.go        // MVP stub implementations for ZKP generation/verification.
|   `-- zkp_test.go     // Tests for stub interfaces.
|
|-- consent/            // Consent management logic.
|   |-- record.go       // Defines ConsentRecord struct (if not from proto) & management functions.
|   |-- manager.go      // Interface and implementation for consent storage, retrieval, verification.
|   `-- consent_test.go
|
|-- crypto/             // Wrapper for cryptographic operations (key generation, signing, verification)
|   |-- keys.go         // Key generation, loading, saving (interfacing with secure storage).
|   |-- signer.go       // Signing logic.
|   |-- verifier.go     // Verification logic.
|   `-- crypto_test.go
|
|-- proto/              // If .proto files for identity are kept separate
|   |-- identity_protocol.proto
|   |-- zkp_protocol.proto
|   |-- consent_protocol.proto
|   `-- (generated .pb.go files would go into a pkg/identity_rpc or similar)

```
*(Note: The `proto/` subdirectory is illustrative. Protobuf definitions might be organized differently, e.g., in a central `protos` directory with generated Go code in respective `pkg` directories like `pkg/identity_rpc`, `pkg/zkp_rpc`, `pkg/consent_rpc`.)*

**Package Descriptions:**

*   **`did`**: Implements the `did:echonet` method.
    *   `did_echonet.go`: Core logic for generating `did:echonet` DIDs from public keys, constructing, serializing, and potentially signing `DIDDocument` structures.
    *   `did_resolver.go`: Defines an interface for DID resolution (e.g., `Resolve(did string) (*DIDDocument, error)`) and an initial implementation that might use a DHT client (from DDS/P2P layer) to find DDO CIDs and then a DDS client to fetch DDOs.
*   **`zkp`**: Provides interfaces and MVP stubs for ZKP functionalities.
    *   `interface.go`: Defines `Prover` and `Verifier` interfaces (e.g., `Prover.GenerateProof(privateInputs, publicInputs, circuitID) (*ZKPProof, error)`, `Verifier.VerifyProof(proof *ZKPProof, publicInputs, circuitID) (bool, error)`).
    *   `stubs.go`: MVP implementations of these interfaces that return placeholder success/failure or `ErrNotImplemented`. Actual ZKP logic is deferred.
*   **`consent`**: Manages consent records.
    *   `record.go`: Defines the `ConsentRecord` Go struct (if not directly using the generated Protobuf struct) and functions for creating and signing consent records.
    *   `manager.go`: Defines an interface for storing, retrieving, and verifying consent records (e.g., `StoreConsent`, `GetConsent`, `VerifyConsent`, `RevokeConsent`). Initial implementation might be in-memory or rely on user-controlled DDS storage.
*   **`crypto`**: A utility package abstracting cryptographic operations needed by the identity layer. This is crucial for separating crypto primitives from the identity logic itself, allowing for easier updates or changes to crypto schemes.
    *   Handles key pair generation (e.g., Ed25519, X25519).
    *   Provides signing and verification functions.
    *   May include helpers for public key encoding (e.g., to Multibase).

This structure aims for modularity, allowing each concern (DIDs, ZKPs, Consent) to be developed and tested somewhat independently before full integration.

---

### 7.2. Core Component Implementation Details for MVP

This section details the MVP implementation focus for the key functions and interfaces within the User Identity & Privacy Layer.

**1. DID Generation & Management (`identity/did/did_echonet.go`)**

*   **`GenerateDIDKeys() (publicKey crypto.PublicKey, privateKey crypto.PrivateKey, error)` (in `identity/crypto/keys.go`):**
    *   Implement using standard Go crypto libraries (e.g., `ed25519.GenerateKey` for Ed25519 keys).
    *   Returns the public and private key pair.
*   **`GenerateDID(publicKey crypto.PublicKey) (string, error)`:**
    *   Takes a public key.
    *   Serializes the public key to bytes (e.g., raw bytes for Ed25519).
    *   Calculates SHA-256 hash of these bytes.
    *   Base58BTC encodes the hash to form the Method-Specific Identifier (MSI).
    *   Prepends "did:echonet:" to the MSI to form the full DID string.
*   **`CreateInitialDDO(did string, publicKeyAuth crypto.PublicKey, publicKeyEnc crypto.PublicKey) (*identity_rpc.DIDDocument, error)`:**
    *   Takes the DID string, an authentication public key, and an encryption public key.
    *   Constructs a `identity_rpc.DIDDocument` (Protobuf generated struct).
    *   Populates `id`, `controller` (with the DID itself).
    *   Creates `VerificationMethod` entries for the provided public keys (e.g., type "Ed25519VerificationKey2020", "X25519KeyAgreementKey2019", public keys encoded as Multibase).
    *   Populates `authentication` and `keyAgreement` fields referencing the IDs of the appropriate `VerificationMethod`s.
    *   Sets `created` and `updated` timestamps (`timestamppb.Now()`).
    *   Sets an initial `version_id` (e.g., "1" or a hash).
*   **`SerializeDDO(ddo *identity_rpc.DIDDocument) ([]byte, error)`:**
    *   Uses `proto.Marshal` to serialize the `DIDDocument` to bytes for storage (e.g., on DDS).
*   **`DeserializeDDO(data []byte) (*identity_rpc.DIDDocument, error)`:**
    *   Uses `proto.Unmarshal` to deserialize bytes into a `DIDDocument`.

**2. DDO Resolution (`identity/did/did_resolver.go`)**

*   **`Resolver` Interface:**
    ```go
    type Resolver interface {
        Resolve(did string, options ...ResolveOption) (*identity_rpc.DIDDocument, error)
    }
    ```
*   **`DHTDDSRegistryResolver` Struct (MVP Stub):**
    *   Implements `Resolver`.
    *   Constructor: `NewDHTDDSRegistryResolver(dhtClient dds.Discovery, ddsClient dds.StorageManager)` (conceptual DDS interfaces).
    *   `Resolve(did string, ...)`:
        1.  **(MVP Stub):** For MVP, this might initially be a stub returning a hardcoded DDO or `ErrNotImplemented`.
        2.  **(Conceptual Full Logic):**
            *   Query DHT: `ddoCID_string, err := dhtClient.GetValue(context.Background(), "/ddo/"+did)` (assuming DDO CIDs are stored under a path like `/ddo/<did>`).
            *   Fetch from DDS: `ddoBytes, found, err := ddsClient.Retrieve(ddoCID_string)`.
            *   Deserialize: `ddo, err := DeserializeDDO(ddoBytes)`.
            *   Return `ddo`. Handle errors (e.g., `ErrNotFoundInDHT`, `ErrNotFoundInDDS`).

**3. ZKP Stubs (`identity/zkp/stubs.go`, `identity/zkp/interface.go`)**

*   **`Prover` Interface:**
    ```go
    type Prover interface {
        // circuitID identifies the specific ZKP circuit (e.g., "age_over_18_from_vc_type_X")
        GenerateProof(circuitID string, privateInputs map[string]interface{}, publicInputs map[string]interface{}) (*ZKPProofData, error)
    }
    ```
*   **`Verifier` Interface:**
    ```go
    type Verifier interface {
        VerifyProof(circuitID string, proof *ZKPProofData, publicInputs map[string]interface{}) (bool, error)
    }
    ```
*   **`ZKPProofData` Struct (conceptual, actual structure depends on ZKP scheme):**
    ```go
    type ZKPProofData struct {
        ProofBytes   []byte
        PublicInputsStructured map[string]interface{} // Or a specific struct for public inputs
    }
    ```
*   **Stub Implementations:**
    *   `GenerateProof` stub: Returns a dummy `ZKPProofData` and `nil` error, or `ErrNotImplemented`.
    *   `VerifyProof` stub: Returns `true, nil` (always verifies), or `ErrNotImplemented`.

**4. Consent Management Stubs (`identity/consent/manager.go`, `identity/consent/record.go`)**

*   **`ConsentRecord` Struct (Go version of the conceptual Protobuf):**
    *   Defined in `record.go` if not directly using a generated `pb.ConsentRecord`.
*   **`ConsentManager` Interface:**
    ```go
    type ConsentManager interface {
        GrantConsent(record *ConsentRecord) (consentID string, err error)
        VerifyConsent(granterDID, requesterID string, dataScopeReferences []string, permission PermissionType) (isValid bool, record *ConsentRecord, err error)
        RevokeConsent(consentID string, granterDID string, revocationSignature []byte) error
        GetConsent(consentID string) (*ConsentRecord, error)
    }
    ```
*   **Stub Implementations:**
    *   All methods return placeholder values (e.g., dummy consentID, `true` for verify) and `nil` error, or `ErrNotImplemented`.
    *   An in-memory map could be used in the stub for basic grant/get/revoke testing.

*(This outlines the core interfaces and functions for an MVP. Actual implementation will involve more detailed error handling, logging, and integration with underlying crypto and P2P libraries.)*

---

### 7.3. MVP Scope for User Identity & Privacy Layer

The MVP for the User Identity & Privacy Layer will focus on establishing the foundational elements of `did:echonet` and setting up the framework for future ZKP and Consent Management integration.

**Included in MVP:**

1.  **`did:echonet` Core Implementation:**
    *   **Key Generation:** Ability to generate cryptographic key pairs (e.g., Ed25519 for signing, X25519 for encryption) locally within a client/node using standard libraries.
    *   **DID Generation:** Implement `GenerateDID(publicKey)` to create `did:echonet:<MSI>` strings from a public key as specified.
    *   **DDO Creation:** Implement `CreateInitialDDO(...)` to construct a valid `identity_rpc.DIDDocument` Protobuf message in Go. This includes populating `id`, `controller`, `verificationMethod` (with correctly formatted public keys, e.g., Multibase), `authentication`, `keyAgreement`, `created`, `updated`, and an initial `version_id`.
    *   **DDO Serialization/Deserialization:** Implement `SerializeDDO` (using `proto.Marshal`) and `DeserializeDDO` (using `proto.Unmarshal`) for the `DIDDocument`.
2.  **DDO Storage & Retrieval (Conceptual/Simplified for MVP):**
    *   **Storage:** DDOs, once serialized, are intended to be stored on DDS. The MVP will include the serialization step, but the actual DDS `Store` call from the identity module might be a placeholder or log message.
    *   **Resolution Interface:** Define the `Resolver` interface (`Resolve(did string) (*DIDDocument, error)`).
    *   **Resolution Stub:** Provide a stub implementation of the `Resolver` (e.g., `DHTDDSRegistryResolver`) that for MVP might:
        *   Return a hardcoded/test `DIDDocument`.
        *   Or, simply return `nexuserrors.ErrNotImplemented`.
        *   This defers full DHT/DDS lookup logic for DDOs but establishes the interface.
3.  **ZKP Framework (Stubs & Interfaces):**
    *   Define `Prover` and `Verifier` interfaces as outlined in Section 7.2.
    *   Implement stub versions of these interfaces that log calls and return dummy success values or `nexuserrors.ErrNotImplemented`.
    *   Define the conceptual `ZKPProofData` struct (or use `google.protobuf.Struct` for flexible public inputs in stubs).
    *   This prepares the system for future ZKP integration without implementing complex cryptography in MVP.
4.  **Consent Management Framework (Stubs & Interfaces):**
    *   Define the `ConsentRecord` Go struct (or rely on a conceptual Protobuf definition for it).
    *   Define the `ConsentManager` interface (`GrantConsent`, `VerifyConsent`, `RevokeConsent`, `GetConsent`).
    *   Implement stub versions of `ConsentManager` methods. An in-memory map can be used within the stub to simulate basic grant/get/revoke for testing the flow.
5.  **Basic `Validate()` Methods for New Structs:**
    *   Implement basic `Validate()` methods for `DIDDocument`, `VerificationMethod`, `ServiceEndpoint`, and `ConsentRecord` (if defined as Go structs rather than just using proto directly). These would check for required fields, valid UUIDs/URLs where appropriate, and enum correctness, using `nexuserrors`.

**Excluded from MVP:**

*   **Full DDO Resolution via Live DDS/DHT:** While the interface is defined, the full implementation of DHT lookups for DDO CIDs and DDS retrieval will be part of DDS/P2P layer integration.
*   **DDO Update and Deactivation Full Logic:** Specification exists, but MVP implementation will focus on creation and conceptual resolution of the initial DDO version. Version chaining and DHT updates for DDOs are post-MVP.
*   **Actual ZKP Circuit Development & Proving/Verification:** Only interfaces and stubs are in MVP.
*   **Robust Consent Storage & Networked Verification:** MVP consent stubs will be local/in-memory. Distributed storage and verification of consent is post-MVP.
*   **Advanced Key Management:** Secure hardware key storage, key rotation policies, advanced multi-signature control of DIDs. MVP assumes keys are managed locally by the client software.
*   **Cross-DID Communication Security (beyond KeyAgreement setup):** Full protocols for secure messaging using DID key agreement are a separate concern.

The MVP aims to provide a working local DID generation and DDO creation capability, with clearly defined interfaces and stubs for more advanced privacy features, setting a solid foundation for future development.

---

### 7.4. Unit Test Strategy for Identity & Privacy MVP

Unit tests will focus on the components implemented within the MVP scope, primarily DID/DDO generation, serialization, and basic validation of the new data structures.

**Target Packages:** `identity/did`, `identity/crypto`, `identity/consent` (for stubs/structs), `identity/zkp` (for stubs/structs).

**1. `identity/crypto` Package Tests (e.g., `keys_test.go`):**
    *   **`TestGenerateDIDKeys`:**
        *   Verify that valid, distinct key pairs (public and private) are generated for supported schemes (e.g., Ed25519, X25519).
        *   Check key formats/lengths if applicable.
    *   **(Future) Tests for signing and verification wrappers if implemented here.**

**2. `identity/did` Package Tests (e.g., `did_echonet_test.go`):**
    *   **`TestGenerateDID`:**
        *   Test with a known public key and verify against a pre-calculated `did:echonet` string.
        *   Ensure different public keys produce different DIDs.
        *   Test that the MSI part is valid Base58BTC.
    *   **`TestCreateInitialDDO`:**
        *   Verify all required fields (`id`, `controller`, `verificationMethod`, `authentication`, `keyAgreement`, `created`, `updated`, `version_id`) are populated correctly.
        *   Check `verificationMethod` entries: correct IDs (DID URL fragments), types, controller, and public key encoding (Multibase).
        *   Ensure `authentication` and `keyAgreement` correctly reference `verificationMethod` IDs.
        *   Check timestamps are recent and `updated >= created`.
    *   **`TestSerializeDeserializeDDO`:**
        *   Create a `DIDDocument`, serialize it using `SerializeDDO`.
        *   Deserialize the bytes back using `DeserializeDDO`.
        *   Use `reflect.DeepEqual` (or `proto.Equal` for protobuf messages) to ensure the reconstructed DDO is identical to the original.
        *   Test with DDOs having optional fields present and absent (e.g., services, assertionMethod).
    *   **`TestDIDDocument_Validate` (and for `VerificationMethod`, `ServiceEndpoint`):**
        *   Table-driven tests for the `Validate()` method of `DIDDocument` and its nested structures.
        *   Cover valid cases.
        *   Cover invalid cases for each field:
            *   Missing `id`, `controller`, `verificationMethod`.
            *   Invalid UUID format for `id` in `VerificationMethod` or `ServiceEndpoint`.
            *   Missing `type` or `publicKeyMultibase` in `VerificationMethod`.
            *   `authentication`/`keyAgreement` referencing non-existent `verificationMethod` IDs.
            *   Invalid/zero timestamps, `updated` before `created`.
            *   Invalid `serviceEndpoint` URL format.
*   **`Resolver` Stub Tests (`did_resolver_test.go`):**
    *   Test that the MVP stub for `Resolve(did)` returns `ErrNotImplemented` or the hardcoded test DDO as expected.

**3. `identity/zkp` Package Tests (e.g., `zkp_test.go`):**
    *   **Test Stub Interfaces (`Prover`, `Verifier`):**
        *   Verify that `GenerateProof` stub returns dummy proof data or `ErrNotImplemented`.
        *   Verify that `VerifyProof` stub returns `true` (or `ErrNotImplemented`).

**4. `identity/consent` Package Tests (e.g., `consent_test.go`):**
    *   **`TestConsentRecord_Validate` (if `ConsentRecord` has a `Validate()` method):**
        *   Table-driven tests covering:
            *   Valid `ConsentRecord`.
            *   Missing required fields (`consent_id`, `granter_did`, `requester_id`, `data_scope_references`, `permissions_granted`, `issuance_timestamp`, `granter_signature`).
            *   Invalid UUIDs for ID fields.
            *   Empty `data_scope_references` or `permissions_granted` if not allowed.
            *   Invalid enum values for `permissions_granted`.
            *   Invalid timestamps (zero, expiry before issuance, revocation before issuance).
    *   **`ConsentManager` Stub Tests:**
        *   Test that stub methods (`GrantConsent`, `VerifyConsent`, `RevokeConsent`, `GetConsent`) behave as defined for MVP (e.g., return dummy data, manage an in-memory map correctly, or return `ErrNotImplemented`).

These tests will primarily focus on local logic, data structure integrity, and correct functioning of MVP stubs. Full end-to-end resolution and ZKP/consent flows involving network interaction will be part of broader integration testing phases.

---
*(End of Document: User Identity & Privacy Layer Technical Specification & Initial Implementation Plan)*
