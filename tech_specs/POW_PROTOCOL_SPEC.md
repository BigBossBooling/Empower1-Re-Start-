# DigiSocialBlock (Nexus Protocol) - Proof-of-Witness (PoW) Protocol Technical Specification

## 1. Objective

This document provides the detailed technical specifications for the Proof-of-Witness (PoW) Protocol for DigiSocialBlock. The PoW protocol is a decentralized mechanism designed to establish the authenticity, integrity, and approximate first-seen timestamp of user-generated content without relying on a traditional blockchain. It aims to provide a verifiable basis for content originality, assist in mitigating spam/low-quality content, and serve as a foundation for other platform features like content-based rewards.

## 2. Scope

This specification covers the following aspects of the PoW Protocol:
*   The role and selection process for Witness Nodes.
*   The content attestation process performed by Witness Nodes.
*   The data structures for `WitnessAttestation` and `PoWClaim` objects.
*   The generation of `PoWClaim` objects by content originators or interested parties.
*   The verification process for `PoWClaim` objects.
*   Conceptual approaches to dispute resolution.
*   Core Network Messages (conceptual Protobuf definitions) for PoW interactions.

This document builds upon the conceptual design outlined in `tech_specs/dli_echonet_protocol.md` (Section 3).

## 3. Relation to `dli_echonet_protocol.md`

This specification directly elaborates on the principles and components introduced in Section 3 ("Module 1.3: Proof-of-Witness (PoW) Protocol") of the `dli_echonet_protocol.md` document. It aims to provide the next level of detail required for implementation planning and development. The PoW protocol relies on the Distributed Data Stores (DDS) Protocol (Module 1.2) for content availability and retrieval.

---

## 4. Witness Node Role & Selection

Witness Nodes are critical participants in the PoW protocol, responsible for independently verifying and attesting to the existence and integrity of content.

### 4.1. Responsibilities of Witness Nodes

*   **Content Observation:** Actively monitor the DigiSocialBlock network (e.g., by observing new manifest CIDs appearing on the Distributed Hash Table (DHT) used by the DDS, or by receiving explicit requests to witness content).
*   **Content Retrieval & Verification:** Upon observing or being requested to witness a piece of content (identified by its manifest CID):
    *   Retrieve the `ContentManifest` from the DDS.
    *   Retrieve all constituent data chunks from the DDS as listed in the manifest.
    *   Verify the integrity of each data chunk against its CID.
    *   Reassemble the content from the verified chunks.
    *   Verify the integrity of the reassembled content against the `original_content_hash` stored in the manifest.
*   **Attestation Generation:** If content verification is successful, create and cryptographically sign a `WitnessAttestation` object (detailed in Section 5.3). This attestation includes the manifest CID, the observed timestamp, and the verified original content hash.
*   **Attestation Storage & Publication:**
    *   Store their own generated attestations locally.
    *   Make their attestations discoverable/available to the network (e.g., by announcing them to the DHT, associating them with the manifest CID, or responding to direct queries).
*   **Participation in Dispute Resolution (Conceptual - Future):** If disputes arise regarding content originality or timestamps, Witness Nodes may be called upon to provide their attestations or participate in a resolution protocol.
*   **(Optional) Basic Policy Checks:** Witness Nodes *may* perform rudimentary, automated checks against global blocklists or for obviously malicious content signatures before attesting. However, complex content moderation is an application/governance layer concern, not a primary PoW function.

### 4.2. Eligibility & Selection Criteria for Witness Nodes

To ensure the reliability and trustworthiness of the PoW protocol, nodes wishing to act as Witnesses must meet certain criteria. These criteria are part of the broader Nexus Protocol node governance and incentive model.

*   **Stake Requirement (Future - Nexus Protocol):** A minimum stake of platform tokens may be required to register as a Witness Node. This acts as a security deposit and disincentivizes malicious behavior.
*   **Uptime & Reliability (Future - Nexus Protocol):** Witnesses must demonstrate consistent uptime and responsiveness. A reputation system, potentially updated based on successful attestations and participation, will track this.
*   **Bandwidth & Processing Capacity:** Sufficient resources to download, hash, and process content chunks efficiently.
*   **Storage (for Attestations):** Ability to store generated attestations.
*   **Identity:** Must possess a stable, verifiable network identity (e.g., libp2p PeerID linked to a DigiSocial DID).

### 4.3. Witness Selection Process for Content

When a new piece of content (identified by its manifest CID) requires witnessing, a set of Witness Nodes needs to be selected to perform the attestation.

*   **Target Number of Witnesses (M):** The protocol will aim for `M` attestations for a piece of content to be considered robustly witnessed (e.g., `M=5` or `M=7`). This is a configurable network parameter.
*   **Selection Mechanisms (to be refined, MVP might be simpler):**
    1.  **DHT-Based Proximity (Primary for Scalability):**
        *   Witness Nodes could advertise their "witnessing service" on the DHT, perhaps associated with certain keywords, topics, or simply their general availability.
        *   When new content (manifest CID `C_m`) is published, the Originator Node (or other interested nodes) could query the DHT to find Witness Nodes "closest" (e.g., in Kademlia XOR metric space) to `C_m` or to the `AuthorID`. This promotes a somewhat deterministic yet distributed set of witnesses for any given content.
    2.  **Random Selection from Eligible Pool:**
        *   A globally known (or discoverable via DHT) list of currently eligible/active Witness Nodes is maintained.
        *   A deterministic (e.g., hash-based) random selection process based on the `manifest_cid` could choose `M` witnesses from this pool.
    3.  **Application/Community-Driven Selection (Optional Overlay):**
        *   Specific communities or applications built on DigiSocialBlock might designate preferred or trusted Witness Nodes for their content. This would operate as an additional layer on top of the base protocol.
    4.  **Explicit Request by Originator (MVP Initial):**
        *   For MVP, an Originator Node might explicitly select and send "witness requests" (e.g., `SubmitContentForWitnessingRequest`) to a known set of Witness Nodes.
*   **Diversity:** The selection process should ideally aim for diversity among selected witnesses (e.g., different operators, geographic locations if known) to enhance the robustness of the PoW claim.

The goal is to have a decentralized and sufficiently random yet reliable process for assigning witness duties, preventing any single entity from easily controlling the attestation process for a large amount of content.

---

## 5. Attestation Process

The attestation process is how Witness Nodes formally vouch for the existence, integrity, and observed time of a piece of content.

### 5.1. Trigger for Attestation

An attestation is typically generated by a Witness Node when one of the following occurs:

1.  **Observation of New Content:** The Witness Node discovers a new `manifest_cid` on the DHT that it has not seen or attested to before. This is a passive observation mode.
2.  **Explicit Witnessing Request:** The Witness Node receives a direct request (e.g., via a `SubmitContentForWitnessingRequest` network message) from an Originator Node or another party to witness a specific `manifest_cid`.
3.  **Re-Attestation/Refresh (Future):** Periodically, or if a previous attestation is expiring (if attestations have TTLs), a Witness might re-verify and re-attest to content it continues to observe as valid.

### 5.2. Verification Steps by Witness Node Before Attesting

Before creating an attestation, a Witness Node **MUST** perform the following verification steps:

1.  **Retrieve ContentManifest:** Fetch the `ContentManifest` data from the DDS using the provided `manifest_cid`. Deserialize it. If retrieval or parsing fails, an attestation cannot be made.
2.  **Retrieve All Data Chunks:** For each `chunk_cid` listed in the `ContentManifest.chunk_cids`:
    *   Fetch the corresponding data chunk from the DDS.
    *   If any chunk is irretrievable from the DDS (after reasonable attempts), the content is considered incomplete, and an attestation cannot be made for the full content.
3.  **Verify Integrity of Each Data Chunk:** For each retrieved data chunk:
    *   Calculate its SHA-256 hash.
    *   Compare the calculated hash with the `chunk_cid` from the manifest.
    *   If any chunk's hash does not match its CID, the content is considered corrupt or tampered, and an attestation **MUST NOT** be made.
4.  **Reassemble Content:** If all chunks are retrieved and individually verified, reassemble the original content by concatenating the chunks in the order specified in `ContentManifest.chunk_cids`.
5.  **Verify Full Content Integrity:**
    *   Calculate the SHA-256 hash of the fully reassembled content.
    *   Compare this hash with the `original_content_hash` stored in the `ContentManifest`.
    *   If the hashes do not match, the content or manifest is inconsistent, and an attestation **MUST NOT** be made.
6.  **(Optional) Basic Policy/Compliance Checks:**
    *   The Witness Node *may* perform lightweight, automated checks against known malicious CIDs, prohibited content signatures, or basic format validations if such policies are defined for Witnesses. This is not a deep content moderation step but a basic filter. If a policy check fails, the Witness may refuse to attest.

### 5.3. `WitnessAttestation` Data Structure

If all verification steps pass, the Witness Node creates a `WitnessAttestation`.

*   **Conceptual Protobuf Definition (`pow_protocol.proto`):**
    ```protobuf
    message WitnessAttestation {
      string manifest_cid = 1;                 // Base58BTC encoded CID of the ContentManifest being attested to.
      string witness_id = 2;                   // PeerID (or DID-based identifier) of the Witness Node.
      google.protobuf.Timestamp observed_timestamp = 3; // Timestamp when the Witness successfully completed verification.
                                                 // Should be as close as possible to the "first seen" time by this witness.
      string content_hash_from_manifest = 4;   // The original_content_hash value taken from the verified manifest.
      bytes signature = 5;                     // Cryptographic signature by the witness_id over a canonical representation of
                                                 // (manifest_cid + witness_id + observed_timestamp.seconds + observed_timestamp.nanos + content_hash_from_manifest).
      // string witness_node_version = 6;      // Optional: version of the witness software.
      // string attestation_format_version = 7; // Optional: version of this attestation message structure.
      // (Future) uint64 epoch_id = 8;         // Network epoch during which observation occurred.
    }
    ```
*   **Key Fields Explained:**
    *   `manifest_cid`: Links the attestation directly to the specific version of the content's structure.
    *   `witness_id`: Identifies the attestor.
    *   `observed_timestamp`: Crucial for establishing a "first-seen" timeline. Witnesses should use reliable, synchronized time (e.g., NTP). The exact method for determining this timestamp (e.g., time of successful verification completion) should be consistent.
    *   `content_hash_from_manifest`: Re-stating this hash in the attestation confirms what version of the content (as per the manifest's claim) was verified.
    *   `signature`: Provides non-repudiation and integrity for the attestation itself. The signature scheme (e.g., ECDSA, Ed25519) will be tied to the Witness Node's identity. The data to be signed must be canonicalized (e.g., concatenating specific field values in a defined order).

### 5.4. Attestation Storage and Propagation/Publication

*   **Local Storage:** Witness Nodes **MUST** store all attestations they generate locally for their own records and potential future dispute resolution.
*   **Publication/Discovery:** To be useful for `PoWClaim` generation, attestations must be discoverable. Mechanisms include:
    1.  **DHT Announcement (Primary for MVP):**
        *   The Witness Node can publish a provider record to the DHT where the "key" is the `manifest_cid` (or a composite key like `manifest_cid:attestation_type`) and the "value" indicates that this `witness_id` has an attestation for it.
        *   Alternatively, the Witness could store the `WitnessAttestation` object (or its CID, if attestations are also stored in DDS) directly in the DHT, associated with the `manifest_cid`. This might be less scalable for many attestations per content item.
        *   A common approach is for the DHT to map `manifest_cid` to a list of `witness_id`s that have attested to it. Parties seeking attestations then query these witnesses directly.
    2.  **Direct Response to Queries:** Witness Nodes should respond to network requests (e.g., `GetAttestationsRequest`) for attestations they hold for a given `manifest_cid`.
    3.  **Push to Aggregators (Future):** Witnesses might push their attestations to designated aggregator services or a sidechain/ledger designed for PoW claims, though this is beyond MVP.

The goal is for Originators or other parties to be able to efficiently discover and collect a sufficient number of valid attestations for a given piece of content.

---

## 6. Proof-of-Witness (PoW) Claim Generation

A Proof-of-Witness (PoW) Claim aggregates multiple `WitnessAttestation` objects to provide strong, verifiable evidence of a piece of content's existence, integrity, and approximate first-seen time.

### 6.1. Claim Creator

*   Typically, the **Originator Node** (content creator/uploader) is responsible for gathering attestations and generating the `PoWClaim` for their content.
*   However, any interested third party could also perform this role if they can discover and collect the necessary attestations.

### 6.2. Gathering Attestations

1.  The Claim Creator identifies the `manifest_cid` of the content for which a PoW Claim is desired.
2.  They query the network (e.g., via DHT lookups as described in Section 5.4, or by directly querying known witnesses) to find and retrieve `WitnessAttestation` objects for that `manifest_cid`.
3.  They collect attestations until they have at least `M` (the target number of witnesses, e.g., M=5) valid and distinct attestations.
    *   **Validity Checks on Collected Attestations:**
        *   Each attestation's signature must be valid and from a recognized/eligible Witness Node.
        *   The `manifest_cid` and `content_hash_from_manifest` in each attestation must match the target content.
        *   `observed_timestamp` should be plausible. (Outlier timestamps might be flagged or require more attestations).

### 6.3. `PoWClaim` Data Structure

*   **Conceptual Protobuf Definition (`pow_protocol.proto`):**
    ```protobuf
    message PoWClaim {
      string claim_id = 1;                       // UUID for this specific claim object.
      string manifest_cid = 2;                   // Base58BTC encoded CID of the ContentManifest.
      string originator_id = 3;                  // PeerID/UUID of the original content creator/uploader.
      string content_hash_from_manifest = 4;     // The original_content_hash from the manifest.
      google.protobuf.Timestamp earliest_observed_timestamp = 5; // Represents the "effective first seen" time derived from the attestations.
                                                     // Could be the earliest, median, or a weighted average of `observed_timestamp`
                                                     // from the collected valid attestations. Policy TBD. For MVP, earliest valid observed_timestamp.
      repeated WitnessAttestation attestations = 6;  // The list of M (or more) valid WitnessAttestation objects.
      google.protobuf.Timestamp claim_timestamp = 7; // Timestamp when this PoWClaim object was created.
      // map<string, string> additional_claim_data = 8; // Optional: For future use or application-specific claim details.
    }
    ```
*   **Key Fields Explained:**
    *   `claim_id`: A unique identifier for this claim instance.
    *   `manifest_cid`, `originator_id`, `content_hash_from_manifest`: Identify the content and its creator.
    *   `earliest_observed_timestamp`: This is a critical piece of information derived from the set of attestations. The method for determining this (e.g., earliest, median of valid attestations after discarding outliers) needs to be defined by protocol policy. For MVP, using the earliest valid `observed_timestamp` from the `attestations` list is a straightforward start.
    *   `attestations`: The collected evidence.
    *   `claim_timestamp`: When the claim itself was packaged.

### 6.4. PoW Claim Usage

*   A generated `PoWClaim` can be:
    *   Stored by the originator.
    *   Submitted to application-layer services within DigiSocialBlock (e.g., a reputation system, a content reward distribution module, a search indexer that prioritizes witnessed content).
    *   Shared with other users as proof of originality/timestamp.
    *   (Future) Recorded on a dedicated ledger or sidechain for PoW claims if high-level global consensus on claims is desired.

The `PoWClaim` serves as a portable, verifiable assertion about the content's history.

---

## 7. Verification of PoW Claims

Any node in the DigiSocialBlock network can independently verify a `PoWClaim` to ascertain the validity of the claimed content originality and timestamp.

### 7.1. Verification Process

To verify a `PoWClaim`, a Verifier Node performs the following steps:

1.  **Obtain the `PoWClaim` object.**
2.  **Retrieve Original Content (via DDS):**
    *   Using the `manifest_cid` from the `PoWClaim`, retrieve the `ContentManifest` from the DDS.
    *   Using the `ContentManifest`, retrieve all constituent data chunks from the DDS.
    *   Reassemble the original content.
    *   If any part of this retrieval or reassembly fails, the claim cannot be fully verified against the live network data, and the verification may fail or be considered incomplete.
3.  **Verify Content Integrity Against Claim:**
    *   Calculate the SHA-256 hash of the reassembled content.
    *   Compare this hash with the `content_hash_from_manifest` field in the `PoWClaim`.
    *   If they do not match, the claim is invalid or refers to different content. Verification fails.
4.  **Verify Individual Attestations:**
    *   For each `WitnessAttestation` included in `PoWClaim.attestations`:
        *   **Check Consistency:** Verify that `attestation.manifest_cid` matches `PoWClaim.manifest_cid` and `attestation.content_hash_from_manifest` matches `PoWClaim.content_hash_from_manifest`. If not, this specific attestation is inconsistent or irrelevant.
        *   **Verify Witness Identity & Eligibility (External Dependency):**
            *   Check if `attestation.witness_id` corresponds to a known and currently eligible Witness Node. This requires access to a list or registry of valid Witness Nodes (potentially maintained by the Nexus Protocol's governance/staking system).
            *   If the witness is unknown or ineligible at `attestation.observed_timestamp`, the attestation may be considered invalid or carry less weight.
        *   **Verify Attestation Signature:**
            *   Reconstruct the canonical data that the witness signed (typically: `manifest_cid + witness_id + observed_timestamp.seconds + observed_timestamp.nanos + content_hash_from_manifest`).
            *   Using the public key associated with `attestation.witness_id`, verify `attestation.signature` against the reconstructed signed data.
            *   If the signature is invalid, the attestation is fraudulent or corrupt. Verification of this attestation fails.
        *   **Check Timestamp Plausibility:** Ensure `attestation.observed_timestamp` is reasonable (not in the distant past before system epoch, not excessively in the future beyond network time + clock skew tolerance).
5.  **Verify Sufficiency and Diversity of Valid Attestations:**
    *   Count the number of valid, distinct `WitnessAttestation` objects (after performing checks in step 4).
    *   Ensure this count meets or exceeds the required minimum `M` (e.g., M=5).
    *   **(Future Refinement):** Check for diversity among the valid witnesses (e.g., are they operated by different entities, geographically distributed, etc., if such information is available and part of the protocol's trust model).
6.  **Verify `earliest_observed_timestamp` in Claim:**
    *   Based on the set of valid `WitnessAttestation` objects, recalculate the `earliest_observed_timestamp` according to the defined protocol policy (e.g., the actual earliest `observed_timestamp` among valid attestations, or a median after discarding outliers).
    *   Compare this recalculated timestamp with the `earliest_observed_timestamp` field in the `PoWClaim`. They should match (within a small tolerance if a complex calculation like median is used).

### 7.2. Outcome of Verification

*   **Valid Claim:** If all steps pass, the `PoWClaim` is considered valid. The verifier can have high confidence that the content identified by `manifest_cid` (matching `content_hash_from_manifest`) existed at least by the `earliest_observed_timestamp` and was witnessed by `M` or more distinct, valid witnesses.
*   **Invalid Claim:** If any critical step fails (e.g., content hash mismatch, insufficient valid attestations, signature failures), the `PoWClaim` is considered invalid.

This verification process allows any node to independently confirm the assertions made by a PoW Claim without trusting the claim issuer.

---

## 8. Dispute Resolution (Conceptual)

Disputes in the PoW system can arise if there are conflicting claims about content originality, timestamping, or the validity of witness attestations. A full-fledged dispute resolution mechanism is complex and may involve layers of social consensus or governance beyond the core PoW protocol. This section outlines conceptual approaches.

### 8.1. Types of Disputes

*   **Timestamp Conflicts:** Multiple `PoWClaim` objects for the *same content* (identical `manifest_cid` and `content_hash_from_manifest`) exist with significantly different `earliest_observed_timestamp` values.
*   **Witness Integrity Challenges:** A Witness Node is accused of falsely attesting (e.g., backdating timestamps, signing attestations for content they didn't verify correctly).
*   **Content Originality/Plagiarism (Application Layer):** While PoW establishes "first seen" by the network, disputes about true intellectual property or plagiarism of content that existed *outside* DigiSocialBlock before being witnessed are largely an application and community moderation issue. PoW can provide evidence for when it was first seen *within* the DigiSocialBlock network.
*   **Claim Tampering:** A `PoWClaim` itself is suspected of being malformed or containing fraudulent attestations. (The verification process in Section 7 aims to catch this).

### 8.2. Conceptual Resolution Mechanisms

1.  **More Evidence (More Witnesses):**
    *   If conflicting `earliest_observed_timestamp` values exist, parties can be encouraged to gather more `WitnessAttestation` objects. A larger set of attestations might provide a clearer statistical picture of the true first-seen time (e.g., by identifying a cluster of timestamps).
    *   The protocol could define a threshold where a claim with significantly more (valid) witnesses supersedes one with fewer, especially if timestamp discrepancies are minor.

2.  **Reputation-Weighted Attestations (Future - Nexus Protocol Integration):**
    *   If Witness Nodes have a reputation score (from the broader Nexus Protocol), attestations from higher-reputation witnesses could be given more weight in resolving disputes or calculating the `earliest_observed_timestamp`.

3.  **Challenge Protocol (Future - Advanced):**
    *   A mechanism where a party can formally challenge a `PoWClaim` by submitting counter-evidence (e.g., an earlier `PoWClaim` for the same content, proof of witness misbehavior).
    *   This might involve a period where additional witnesses can weigh in, or a subset of high-reputation "Arbiter Nodes" review the conflicting evidence.
    *   Penalties (e.g., stake slashing for Witnesses found to be malicious) would be necessary to disincentivize false claims and attestations.

4.  **Community Governance / Moderation (Application Layer):**
    *   For issues like plagiarism of off-platform content or subjective assessments of content originality, final decisions may fall to community governance or platform moderation teams.
    *   PoW claims serve as one input/piece of evidence for these processes.

5.  **Forking of Timelines (Decentralized Approach - Contentious):**
    *   In extreme cases with no clear resolution and deeply divided witness sets, different parts of the network or different applications might effectively "choose" which PoW claim (and thus which timestamp) they consider canonical for a piece of content. This is similar to how blockchain forks can occur. This is generally undesirable but a possibility in a sufficiently decentralized system.

**MVP Scope:**
*   For the MVP, dispute resolution will be **minimal and likely off-protocol**.
*   The primary mechanism will be the transparency of `PoWClaim` objects. Users and applications can inspect claims and make their own judgments if multiple conflicting claims for the same content appear.
*   The focus is on robust generation and verification of individual claims. Formal, automated on-protocol dispute resolution is a future enhancement.

The strength of a PoW claim inherently comes from the number, diversity, and (eventually) reputation of the witnesses supporting it.

---

## 9. PoW Network Messages (Conceptual Protobuf Definitions)

The following Protobuf message definitions outline potential interactions for the PoW protocol. These would reside in a `pow_protocol.proto` file, likely importing `google/protobuf/timestamp.proto` and potentially the `WitnessAttestation` and `PoWClaim` definitions if they are in a shared types proto. For this example, `WitnessAttestation` and `PoWClaim` are assumed to be defined within this same conceptual `pow_protocol.proto`.

```protobuf
syntax = "proto3";

package pow_protocol;

import "google/protobuf/timestamp.proto";

option go_package = "empower1/pkg/pow_rpc;pow_rpc"; // Example Go package

// --- Data Structures (already defined conceptually in Sections 5.3 and 6.3) ---

message WitnessAttestation {
  string manifest_cid = 1;
  string witness_id = 2;
  google.protobuf.Timestamp observed_timestamp = 3;
  string content_hash_from_manifest = 4;
  bytes signature = 5;
  // string witness_node_version = 6;
  // string attestation_format_version = 7;
  // uint64 epoch_id = 8;
}

message PoWClaim {
  string claim_id = 1;
  string manifest_cid = 2;
  string originator_id = 3;
  string content_hash_from_manifest = 4;
  google.protobuf.Timestamp earliest_observed_timestamp = 5;
  repeated WitnessAttestation attestations = 6;
  google.protobuf.Timestamp claim_timestamp = 7;
  // map<string, string> additional_claim_data = 8;
}

// --- Network Messages ---

// Optional: Request for a Witness Node to attest to a piece of content.
message SubmitContentForWitnessingRequest {
  string manifest_cid = 1;
  // string originator_id = 2; // Optional: Who is making the request
  // int32 priority = 3;      // Optional: Requested priority
}

message SubmitContentForWitnessingResponse {
  bool accepted = 1;         // True if the witness node has queued the content for verification and potential attestation.
  string message = 2;        // e.g., "Queued for witnessing" or "Witnessing not accepted: [reason]"
  // string estimated_completion_time = 3; // Optional
}

// Optional: If witnesses proactively push their attestations to a collector or directly to originator.
// More likely, attestations are pulled via GetAttestationsRequest.
message PublishAttestationRequest {
  WitnessAttestation attestation = 1;
}

message PublishAttestationResponse {
  bool success = 1;
  string message = 2; // e.g., "Attestation received" or "Invalid attestation format"
}

// Request to retrieve attestations for a given manifest_cid.
// This would be sent to Witness Nodes or a DHT/service that stores/indexes attestations.
message GetAttestationsRequest {
  string manifest_cid = 1;
  // int32 min_witnesses_required = 2; // Optional: Client can specify how many they are looking for.
  // google.protobuf.Timestamp observed_after = 3; // Optional: Filter for attestations made after a certain time.
}

message GetAttestationsResponse {
  string manifest_cid = 1;
  repeated WitnessAttestation attestations = 2; // List of found attestations.
  bool partial_results = 3; // True if more results might be available (e.g., due to query limits).
  string error_message = 4; // If an error occurred during the query.
}

// Optional: For submitting a generated PoWClaim to a service that processes/validates/stores them.
// This is more application-specific (e.g., submitting to a rewards module).
message SubmitPoWClaimRequest {
  PoWClaim claim = 1;
}

message SubmitPoWClaimResponse {
  string claim_id = 1;        // The ID of the submitted claim.
  string processing_status = 2; // e.g., "PENDING_VERIFICATION", "VERIFIED", "REJECTED"
  string message = 3;
}

```

**Notes on Messages:**

*   **`WitnessAttestation` and `PoWClaim`:** These are duplicated here for context but would ideally be defined once in a shared types Protobuf file and imported if `pow_protocol.proto` is separate from `dds_protocol.proto` (which defines `ContentManifest`). For simplicity of this section, they are shown inline.
*   **Explicit vs. Implicit Flows:**
    *   `SubmitContentForWitnessingRequest`: Supports an explicit push model from originators to witnesses.
    *   `GetAttestationsRequest`: Supports a pull model where claim generators query for existing attestations. The MVP might rely more on witnesses passively observing new CIDs on the DDS/DHT and generating attestations, which are then discovered via `GetAttestationsRequest`.
*   **Storage of Attestations:** The `GetAttestationsRequest` implies that attestations are stored somewhere queryable. This could be:
    *   Each Witness Node stores its own and responds directly.
    *   Attestations (or references to them) are published to the same DHT as content provider records, associated with the `manifest_cid`.
*   **`SubmitPoWClaimRequest`:** This is highly dependent on what system will consume these PoW claims. It's included as a conceptual endpoint.

These messages provide a basic set for the PoW protocol's operations. Further messages might be needed for advanced dispute resolution or witness management.

---

## 10. Initial PoW Implementation Plan (Conceptual)

This section outlines a high-level plan for the initial Go implementation of the Proof-of-Witness (PoW) Protocol (MVP).

### 10.1. Core PoW Go Package Structure (Proposed)

A dedicated top-level package, `pow`, is proposed, likely within an `internal` or `pkg` directory structure (e.g., `internal/pow` or `pkg/pow`).

```
pow/
|-- attestation/      // Logic for Witness Nodes to create, sign, and manage attestations.
|   |-- attester.go     // Interface and implementation for attestation generation.
|   `-- attester_test.go
|
|-- claim/            // Logic for generating and managing PoWClaim objects.
|   |-- generator.go    // Interface and implementation for PoWClaim generation.
|   `-- generator_test.go
|
|-- verification/     // Logic for verifying WitnessAttestations and PoWClaims.
|   |-- verifier.go     // Interface and implementation for verification.
|   `-- verifier_test.go
|
|-- rpc/              // Network RPC definitions and handlers for PoW messages.
|   |-- pow_protocol.pb.go // Generated from pow_protocol.proto
|   |-- pow_protocol_grpc.pb.go // Generated gRPC bindings
|   |-- handlers.go     // RPC handler implementations (e.g., HandleGetAttestations).
|   `-- rpc_test.go
|
|-- pow_node.go       // Main PoW node logic/service, orchestrating other components.
|                     // Could be part of a general node or a specialized witness service.
`-- pow_node_test.go
```

**Package Descriptions:**

*   **`attestation`**: Contains the logic for Witness Nodes. This includes observing content (via DDS interface), performing the verification steps (Section 5.2), creating `WitnessAttestation` structs, signing them using the node's identity, and storing/publishing them.
*   **`claim`**: Handles the generation of `PoWClaim` objects. This involves discovering and collecting `WitnessAttestation`s from the network (via an interface to query witnesses or a DHT), validating them, and assembling them into a `PoWClaim` struct.
*   **`verification`**: Provides functions to verify the cryptographic signatures on individual `WitnessAttestation`s and to perform the full verification of a `PoWClaim` object (as detailed in Section 7). This package would depend on DDS access to retrieve content for verification.
*   **`rpc`**: Contains Protobuf message definitions (from `pow_protocol.proto`) and the gRPC/libp2p stream handlers for these messages (e.g., handling `GetAttestationsRequest`).
*   **`pow_node.go`**: (Or integrated into a main node structure). Orchestrates the PoW functionalities for a node. If it's a Witness Node, it runs the attestation service. If it's an Originator Node, it uses the `claim` package to generate PoW Claims. Any node can use the `verification` package.

This structure promotes modularity and clear separation of PoW-specific concerns.

---

### 10.2. Core PoW Components - Initial Implementation Plan (MVP)

This outlines the key functions and interfaces for the initial MVP implementation of PoW components. Dependencies on a running DDS and node identity/crypto services are assumed.

**1. Attestation Service (`pow/attestation/attester.go`)**
   (Runs on Witness Nodes)

*   **`Attester` Interface/Struct:**
    *   `NewAttester(nodeID string, privateKey crypto.Signer, ddsClient DDSAccessor, attestationStore AttestationStorage) *Attester`
    *   `ProcessContentForAttestation(manifestCID string) (*pow_rpc.WitnessAttestation, error)`
        1.  **(DDS Interaction):** Use `ddsClient` to fetch `ContentManifest` and all data chunks for `manifestCID`.
        2.  Perform verification steps as per Section 5.2 (chunk integrity, reassembly, original content hash match).
        3.  If verification successful:
            *   Record `observed_timestamp` (current UTC time).
            *   Create `pow_rpc.WitnessAttestation` struct (filling `manifest_cid`, `witness_id` (self), `observed_timestamp`, `content_hash_from_manifest`).
            *   Canonicalize and sign the attestation data fields using `privateKey`. Store signature in `WitnessAttestation.signature`.
            *   **(Storage):** Use `attestationStore.StoreAttestation()` to save it locally.
            *   **(Discovery - DHT):** Announce availability of this attestation on the DHT (e.g., `dht.Provide(manifestCID + "/attestation/" + witness_id)` or similar keying scheme).
            *   Return the generated attestation.
        4.  If verification fails, log and return an appropriate error (e.g., `ErrContentVerificationFailed`).
*   **`AttestationStorage` Interface (could be in `pow/attestation` or a shared `storage` package):**
    *   `StoreAttestation(attestation *pow_rpc.WitnessAttestation) error`
    *   `GetAttestationsByManifestCID(manifestCID string) ([]*pow_rpc.WitnessAttestation, error)`
    *   (Initial implementation could be in-memory or simple file-based).
*   **`DDSAccessor` Interface (a local interface abstracting DDS interactions needed by PoW):**
    *   `RetrieveManifest(manifestCID string) (*dds_rpc.ContentManifest, error)` // Assuming dds_rpc has ContentManifest
    *   `RetrieveChunk(chunkCID string) ([]byte, error)`

**2. Claim Generation Logic (`pow/claim/generator.go`)**
   (Typically runs on Originator/Client Nodes)

*   **`ClaimGenerator` Interface/Struct:**
    *   `NewClaimGenerator(nodeID string, attestationQuerier AttestationQuerier)`
    *   `GeneratePoWClaim(manifestCID string, originatorID string, contentHashFromManifest string, minAttestations int) (*pow_rpc.PoWClaim, error)`
        1.  Use `attestationQuerier.GetAttestations(manifestCID, minAttestations)` to fetch attestations.
        2.  Validate each received attestation (signature, consistency with `manifestCID` and `contentHashFromManifest`).
        3.  If fewer than `minAttestations` valid attestations are found, return error (e.g., `ErrInsufficientAttestations`).
        4.  Determine `earliest_observed_timestamp` from the valid attestations (e.g., earliest one).
        5.  Construct and return `pow_rpc.PoWClaim` with a new `claim_id` (UUID) and `claim_timestamp`.
*   **`AttestationQuerier` Interface:**
    *   `GetAttestations(manifestCID string, minRequired int) ([]*pow_rpc.WitnessAttestation, error)`
    *   (Implementation would use PoW RPCs like `GetAttestationsRequest` to query multiple witnesses or a DHT/index).

**3. Claim Verification Logic (`pow/verification/verifier.go`)**
   (Callable by any node)

*   **`VerifyPoWClaim(claim *pow_rpc.PoWClaim, ddsClient DDSAccessor, witnessVerifier WitnessIdentityVerifier) error`**
    1.  Validate `claim` struct itself (e.g., non-empty fields, sufficient attestations count as per claim).
    2.  **(DDS Interaction):** Use `ddsClient` to fetch content (manifest & chunks) based on `claim.manifest_cid`.
    3.  Verify reassembled content hash against `claim.content_hash_from_manifest`. If mismatch, return `ErrContentHashMismatch`.
    4.  For each `attestation` in `claim.attestations`:
        *   Verify `attestation.manifest_cid` and `attestation.content_hash_from_manifest` match the claim's.
        *   Use `witnessVerifier.GetPublicKey(attestation.witness_id)` to get witness's public key.
        *   Reconstruct the signed payload of the attestation.
        *   Verify `attestation.signature`. If invalid, return `ErrInvalidAttestationSignature`.
        *   Check `attestation.observed_timestamp` plausibility.
    5.  (Optional) Verify `claim.earliest_observed_timestamp` logic based on the verified attestations.
    6.  If all checks pass, return `nil`.
*   **`WitnessIdentityVerifier` Interface (abstracts checking witness validity/pubkey retrieval):**
    *   `GetPublicKey(witnessID string) (crypto.PublicKey, error)` // Returns public key for signature verification
    *   `IsWitnessEligible(witnessID string, atTime time.Time) (bool, error)` // Checks if witness was eligible at attestation time

**4. Basic RPC/Networking (`pow/rpc/handlers.go`)**

*   **`HandleGetAttestations(request *pow_rpc.GetAttestationsRequest) (*pow_rpc.GetAttestationsResponse, error)`:**
    *   (Runs on Witness Nodes or nodes that store/index attestations).
    *   Retrieves locally stored/indexed attestations for the given `request.manifest_cid` using `AttestationStorage`.
    *   Filters based on optional request parameters (e.g., `observed_after`).
    *   Returns `GetAttestationsResponse`.

*(Crypto interfaces like `crypto.Signer`, `crypto.PublicKey` are from Go's standard library or a chosen crypto library. `peer.AddrInfo`, `host.Host`, `kaddht.IpfsDHT` are from `libp2p` context).*

---

### 10.3. MVP Scope for PoW Protocol

The Minimum Viable Product (MVP) for the PoW Protocol will focus on establishing the core mechanics of content attestation, claim generation, and claim verification.

**Included in MVP:**

1.  **Content Attestation by Witnesses:**
    *   A Witness Node can be manually configured or explicitly requested to witness a specific `manifest_cid`.
    *   The Witness Node successfully retrieves the content (manifest and chunks) from the DDS (assuming DDS MVP is functional for retrieval).
    *   The Witness Node performs all local verification steps (chunk integrity, reassembled content hash against manifest's `original_content_hash`).
    *   The Witness Node generates a valid `WitnessAttestation` structure, including a correct `observed_timestamp` and signs it with its node identity key.
    *   The Witness Node stores this attestation locally.
2.  **Attestation Discovery (Simplified):**
    *   Implement a basic `GetAttestationsRequest` RPC handler on Witness Nodes that allows querying for attestations they hold for a given `manifest_cid`.
    *   No complex DHT-based discovery of attestations for MVP; direct querying of known/selected witnesses is sufficient.
3.  **PoW Claim Generation by Originator/Client:**
    *   An Originator Node (or a client tool) can query one or more (manually specified) Witness Nodes for attestations for a given `manifest_cid`.
    *   It collects these attestations.
    *   It validates the received attestations (checks signatures against known witness public keys, consistency of hashes and CIDs).
    *   It assembles a valid `PoWClaim` structure if a sufficient number (e.g., a small, fixed number like 2-3 for MVP testing) of valid attestations are gathered.
    *   The `earliest_observed_timestamp` in the claim is determined simply (e.g., the very earliest from the valid set).
4.  **PoW Claim Verification by Any Node:**
    *   Any node, given a `PoWClaim` object and access to the DDS, can perform the full verification process outlined in Section 7.
    *   This includes fetching content from DDS, re-verifying content integrity, and re-verifying all included witness attestations (signatures, consistency).
5.  **Basic Crypto Primitives:** Integration of basic digital signature capabilities (e.g., using standard Go crypto libraries for ECDSA or Ed25519) for signing attestations and verifying signatures. Node identity keys are assumed to be available.
6.  **Data Structures:** Implementation of Go structs corresponding to `WitnessAttestation` and `PoWClaim` Protobuf messages.

**Excluded from MVP:**

*   **Automated Witness Selection:** The process of how witnesses are chosen or assigned to content will be manual or pre-configured for MVP. No dynamic DHT-based selection or complex algorithms.
*   **Robust Witness Eligibility & Management:** No integration with staking, reputation, or formal registration/de-registration of Witness Nodes. Assume a known, trusted set of witnesses for initial testing.
*   **Complex Dispute Resolution:** No on-protocol mechanisms for handling conflicting attestations or challenges to witness integrity.
*   **Scalable Attestation Storage/Discovery:** Beyond direct querying of witnesses, no advanced DHT indexing or dedicated storage solutions for attestations.
*   **Integration with Reward Systems:** `PoWClaim`s will be generated and verifiable, but their use in any tokenomic or reward system is outside the PoW MVP scope.
*   **On-chain Recording of Claims:** No interaction with any blockchain or ledger for recording PoW claims.
*   **Advanced Timestamp Synchronization:** Relies on system clocks being reasonably synchronized (NTP); no byzantine fault-tolerant time synchronization protocol.
*   **Sophisticated Policy Checks by Witnesses:** Witnesses will only perform basic content integrity checks, not complex policy enforcement.

The PoW MVP aims to demonstrate the core cycle: content creation -> DDS storage -> witness attestation -> claim generation -> claim verification.

---

### 10.4. Unit and Basic Integration Test Strategy for PoW MVP

Testing will be essential to validate the PoW MVP components and their interactions.

**1. Unit Tests:**

*   **`attestation` Package (`pow/attestation/attester_test.go`):**
    *   **`TestAttester_ProcessContentForAttestation`:**
        *   Mock `DDSAccessor` to provide valid manifest and chunk data. Test successful attestation generation (correct fields, valid signature).
        *   Mock `DDSAccessor` to return errors (e.g., manifest not found, chunk missing, content hash mismatch). Verify attestation is not generated and correct errors are returned.
        *   Mock `AttestationStorage` to verify `StoreAttestation` is called with the correct attestation.
        *   Mock DHT client to verify `Provide` (for attestation availability) is called.
        *   Test signature generation with known key pairs.
*   **`claim` Package (`pow/claim/generator_test.go`):**
    *   **`TestClaimGenerator_GeneratePoWClaim`:**
        *   Mock `AttestationQuerier` to return a set of valid `WitnessAttestation` objects. Verify correct `PoWClaim` generation (fields populated, `earliest_observed_timestamp` calculated correctly based on MVP policy).
        *   Mock `AttestationQuerier` to return insufficient valid attestations. Verify `ErrInsufficientAttestations` is returned.
        *   Mock `AttestationQuerier` to return attestations with mismatched `manifest_cid` or `content_hash`. Verify these are filtered out or cause an error.
        *   Test with attestations having varying `observed_timestamp` values to check `earliest_observed_timestamp` calculation.
*   **`verification` Package (`pow/verification/verifier_test.go`):**
    *   **`TestVerifyPoWClaim`:**
        *   Test with a fully valid `PoWClaim` and mock `DDSAccessor` (to provide matching content) and `WitnessIdentityVerifier` (to provide valid pubkeys and eligibility). Verify returns `nil` error.
        *   Test with invalid `PoWClaim` components:
            *   Content hash mismatch (DDS content doesn't match claim's `content_hash_from_manifest`).
            *   Insufficient number of valid attestations in the claim.
            *   An attestation with an invalid signature.
            *   An attestation from an ineligible or unknown witness.
            *   An attestation for a different `manifest_cid`.
            *   `earliest_observed_timestamp` in claim doesn't match recalculation from attestations.
        *   Ensure specific errors are returned for each failure case.
    *   **`TestVerifyWitnessAttestation` (if factored out as a helper):**
        *   Unit test individual attestation signature verification.

**2. Basic Integration Tests (Conceptual - may use in-memory mocks or minimal libp2p testbeds):**

These tests verify the interaction between different PoW components and with a mocked/minimal DDS.

*   **Attestation Flow Integration Test:**
    1.  Setup: Originator Node (client), Witness Node (with `Attester` service), Mock DDS.
    2.  Originator "publishes" content to Mock DDS (conceptually, just making it available for the Witness to "retrieve").
    3.  Originator requests Witness Node to attest to the content's `manifest_cid`.
    4.  Verify: Witness Node retrieves from Mock DDS, verifies, creates, signs, and stores a `WitnessAttestation`.
    5.  Originator (or another client) queries Witness Node for the attestation and successfully retrieves it.
*   **Claim Generation & Verification Flow Integration Test:**
    1.  Setup: Originator Node, multiple Witness Nodes (producing valid attestations for the same content via Mock DDS), Verifier Node.
    2.  Originator gathers attestations from Witness Nodes.
    3.  Originator generates a `PoWClaim`.
    4.  Verifier Node receives the `PoWClaim`.
    5.  Verifier Node uses Mock DDS to retrieve content and its `WitnessIdentityVerifier` (mocked to know witness pubkeys) to validate the claim.
    6.  Verify: Claim is successfully validated.
    7.  Introduce an invalid attestation into the set gathered by the Originator; verify the Verifier Node rejects the claim.

These tests aim to confirm that the core PoW lifecycle (attest, gather, claim, verify) functions correctly with interacting components, even if network aspects (like real DHT lookups for attestations) are simplified for MVP.

---
*(End of Document: PoW Protocol Specification & Initial Implementation Plan)*
