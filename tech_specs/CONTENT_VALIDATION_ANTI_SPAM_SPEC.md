# DigiSocialBlock (Nexus Protocol) - Content Validation & Anti-Spam Technical Specification

## 1. Objective

This document provides the detailed technical specifications for the Content Validation & Anti-Spam module within DigiSocialBlock. This module is central to ensuring content quality, mitigating spam, and forming the basis for decentralized content monetization through Proof-of-Engagement (PoP). It details the PoP mechanics, integration with AI/ML for content scoring, and the conceptual DLI (Decentralized Ledger-Inspired) Consensus mechanism for validating content.

## 2. Scope

This specification covers:
*   The mechanics of Proof-of-Engagement (PoP), including signal generation, weighting (conceptual), and data structures.
*   The integration points and role of Artificial Intelligence/Machine Learning (AI/ML) in content scoring (quality, spam) and reputation adjustment.
*   The high-level design of the DLI Consensus mechanism for decentralized validation of content scores.
*   Conceptual data structures (Protobuf) for PoP records, AI interactions, and DLI Consensus messages.
*   Interaction of this module with other DigiSocialBlock protocols (e.g., User Identity, DDS, PoW).

This document builds upon the conceptual design outlined in `tech_specs/content_validation_anti_spam.md` (Sections 1, 2, and 3).

## 3. Relation to `content_validation_anti_spam.md`

This specification directly elaborates on the principles and components introduced in Sections 1 ("Proof-of-Engagement (PoP)"), 2 ("AI/ML Integration for Content Scoring"), and 3 ("DLI Consensus for Content Validation") of the `tech_specs/content_validation_anti_spam.md` document. It aims to provide the next level of detail required for implementation planning and development of this critical content quality and monetization engine.

---

## 4. Proof-of-Engagement (PoP) Mechanics

Proof-of-Engagement (PoP) is a system that quantifies the value and impact of user interactions with content. These interactions, or "PoP signals," serve as input for content quality scoring and, subsequently, for reward distribution.

### 4.1. PoP Signal Generation

*   **Qualifying Interactions:** Not all interactions generate PoP signals, or they may generate signals of different intrinsic types/weights. Initially, the following interactions from `InteractionLog.interaction_type` (defined in `echonet_v3_core.proto` or a similar core types proto) are considered primary PoP signal generators:
    *   `INTERACTION_TYPE_LIKE`: A direct positive signal.
    *   `INTERACTION_TYPE_COMMENT`: A significant engagement, especially if the comment itself is of quality (see AI/ML section).
    *   `INTERACTION_TYPE_SHARE` (or Repost): Amplifies content reach, indicating perceived value.
    *   `(Future Consideration)` `INTERACTION_TYPE_SAVE`: Indicates user intent to revisit, a strong quality signal.
    *   `(Future Consideration)` `INTERACTION_TYPE_TIP`: Direct monetary engagement, a very strong signal.
*   **Signal Creation:** When a user performs a qualifying interaction on a `ContentPost`:
    1.  An `InteractionLog` entry is created (as per Module 1.1 Core Data Structures).
    2.  Simultaneously, or derived from the `InteractionLog`, a `PoPSignalRecord` (see Section 4.3) is generated.
    3.  This record links the `post_id` of the content, the `user_id` of the interactor, the `user_id` of the content author, the specific `interaction_type`, and a `timestamp`.
*   **Contextual Data:** For interactions like comments, a hash of the comment text (or the text itself if small and privacy allows for AI processing) can be included as `context_hash` in the `PoPSignalRecord`. This allows AI to assess the quality/relevance of the comment when adjusting PoP signal weights.

### 4.2. PoP Signal Weighting (Conceptual Framework)

The "value" of a PoP signal is not uniform. A multi-layered weighting system will be employed:

1.  **Base Interaction Weight:** Each `interaction_type` will have a default base weight.
    *   Example (illustrative, subject to tuning):
        *   LIKE: 1.0
        *   COMMENT: 5.0 (base, can be adjusted by comment quality)
        *   SHARE: 3.0
    *   These base weights are recorded in `PoPSignalRecord.initial_weight`.
2.  **AI-Adjusted Weight:** An AI/ML model (detailed in Section 5) will analyze PoP signals in conjunction with other data (interactor's reputation, content context, comment sentiment/quality, etc.) to produce an `ai_adjusted_weight`.
    *   This allows the system to differentiate between low-effort spam interactions and genuine, high-quality engagement.
    *   For example, a insightful comment from a high-reputation user might see its base weight significantly amplified, while a spam comment might have its weight reduced to near zero.
3.  **User Reputation Factor (Interactor):** The reputation score of the user performing the interaction can act as a multiplier or modifier on the signal's weight. Higher reputation users generate more impactful PoP signals.
4.  **Content Age / Decay (Future):** The impact of PoP signals might decay over time to keep content scoring dynamic.
5.  **Network Saturation / Velocity Limits (Future):** Mechanisms to prevent abuse via rapid, low-quality signal generation (e.g., rate limiting PoP signal impact per user per content item over time).

For MVP, the focus will be on capturing `initial_weight` and providing a field for `ai_adjusted_weight` to be populated by (initially stubbed) AI models. Complex dynamic weighting is a post-MVP refinement.

### 4.3. `PoPSignalRecord` Data Structure

This structure captures an individual instance of a Proof-of-Engagement signal.

*   **Conceptual Protobuf Definition (`pop_protocol.proto` or similar, potentially part of `content_validation_anti_spam.proto`):**
    ```protobuf
    syntax = "proto3";

    package pop_protocol; // Or appropriate package

    import "google/protobuf/timestamp.proto";
    // import "core_types/interaction_log.proto"; // If InteractionType enum is there

    // Assuming InteractionType is defined elsewhere, e.g., in a core_types enum
    // enum InteractionType { ... } from echonet_v3_core.proto

    message PoPSignalRecord {
      string signal_id = 1;                 // UUID for this specific signal instance.
      string post_id = 2;                   // Content ID (e.g., post_id) of the content interacted with.
      string interacting_user_id = 3;       // DID/UUID of the user who performed the interaction.
      string content_author_id = 4;         // DID/UUID of the author of the content.
      /* core_types.InteractionType */ int32 interaction_type = 5; // The type of interaction (e.g., LIKE, COMMENT). Using int32 for direct enum mapping.
      google.protobuf.Timestamp timestamp = 6;  // When the interaction occurred.
      double initial_weight = 7;            // Base weight of the interaction type.
      double ai_adjusted_weight = 8;        // Weight after AI/ML model scoring (populated later).
      string context_hash = 9;              // Optional: Hash of contextual data, e.g., hash of comment text for a COMMENT interaction.
                                            // Allows AI to verify/score context without needing full raw data in this record.
      // string session_id = 10;            // Optional: For tracking user sessions if relevant to spam detection.
      // string platform_client_id = 11;    // Optional: Identifier of the client application that generated the signal.
    }
    ```
*   **Key Fields:**
    *   `signal_id`: Unique ID for the PoP event.
    *   `post_id`: Links to the content.
    *   `interacting_user_id`, `content_author_id`: Participants.
    *   `interaction_type`: The core action.
    *   `timestamp`: Time of interaction.
    *   `initial_weight`: Protocol-defined base value.
    *   `ai_adjusted_weight`: Dynamically scored value, crucial for quality assessment.
    *   `context_hash`: Allows AI to factor in the substance of interactions like comments without storing full comment text here.

### 4.4. Storage and Aggregation of PoP Signals

*   **Initial Generation:** `PoPSignalRecord`s are generated by nodes where the interaction occurs (typically the interactor's client or a node processing their action).
*   **Temporary Storage/Batching:** Nodes might temporarily store or batch these signals locally.
*   **Submission to DLI Consensus Network:** For a `ContentPost` to be validated and scored by the DLI Consensus (Section 6), its associated `PoPSignalRecord`s (or a summary/aggregate of them) need to be made available to the DLI Consensus Nodes.
    *   This could involve Originator Nodes (or any interested party) gathering PoP signals for a specific `post_id` (e.g., by querying interaction logs or a dedicated PoP signal service) and submitting them as part of a `ContentValidationProposal` to the DLI network.
    *   Alternatively, PoP signals could be gossiped or published to a stream that DLI nodes subscribe to.
*   **Aggregation:** DLI Consensus Nodes will aggregate these signals (likely using `ai_adjusted_weight`) to contribute to the overall `content_quality_score` for a `ContentPost`.

The PoP mechanism transforms raw user interactions into weighted signals that quantitatively reflect engagement, forming a key input for decentralized content validation and reward systems.

---

## 5. AI/ML Integration for Content Scoring & Anti-Spam

Artificial Intelligence (AI) and Machine Learning (ML) play a crucial role in enhancing the PoP mechanism, providing nuanced content scoring, and identifying spam or low-quality interactions.

### 5.1. Objectives of AI/ML Integration

*   **Enhance PoP Signal Quality:** Adjust the weight of PoP signals based on deeper contextual understanding (e.g., quality of a comment, reputation of the interactor).
*   **Content Quality Scoring:** Generate a holistic `content_quality_score` for each `ContentPost`.
*   **Spam Detection:** Identify and score potential spam content and interactions, generating a `spam_probability_score`.
*   **User Reputation Feedback:** Contribute to the dynamic adjustment of user reputation scores based on their content quality and engagement patterns.
*   **Adaptive Anti-Abuse:** Learn from network activity to adapt to new spam techniques or coordinated inauthentic behavior.

### 5.2. Input Features for AI Models

AI models will process a variety of features, including but not limited to:

1.  **`ContentPost` Data:**
    *   Textual content (`text_content`): For NLP analysis (sentiment, topic, coherence, toxicity, originality if compared against a corpus).
    *   `media_url`: Metadata from media if available (e.g., image tags, video length - future).
    *   `tags`: Relevance and appropriateness.
    *   `client_created_at`: As part of content context.
    *   `author_id`: To link with author's reputation and history.
2.  **`PoPSignalRecord` Data:**
    *   `interaction_type`: Type and frequency of different interactions.
    *   `interacting_user_id`: To link with interactor's reputation.
    *   `initial_weight`: Base value of the interaction.
    *   `context_hash`: Link to comment text (via its hash) or other contextual data for deeper analysis.
3.  **`UserProfile` Data (Interactor & Author):**
    *   `reputation_score`: Existing reputation of the users involved.
    *   Historical activity patterns (e.g., frequency of posting/interacting, types of interactions, past content quality - requires careful privacy considerations and may use aggregated/anonymized data or ZKPs for access).
    *   Account age, verification status.
4.  **`PoWClaim` Data (Proof-of-Witness):**
    *   `earliest_observed_timestamp`: Network-validated timestamp.
    *   Number and diversity of witnesses: As a proxy for initial content credibility or visibility.
5.  **Network-Level Data (Future):**
    *   Interaction graphs between users and content.
    *   Temporal patterns of engagement.

### 5.3. AI Model Outputs (Conceptual)

The AI/ML system will produce several key outputs:

1.  **`content_quality_score` (for each `ContentPost`):**
    *   A normalized score (e.g., 0.0 to 1.0) representing the AI's assessment of the content's quality, relevance, and originality.
2.  **`spam_probability_score` (for each `ContentPost` and potentially individual `PoPSignalRecord`s):**
    *   A score (e.g., 0.0 to 1.0) indicating the likelihood that the content or interaction is spam, low-quality, or part of a coordinated inauthentic campaign.
3.  **`ai_adjusted_weight` (for each `PoPSignalRecord`):**
    *   The refined weight for an individual PoP signal after AI analysis (e.g., quality of comment, interactor reputation).
4.  **User Reputation Updates (Feedback):**
    *   Signals or scores that can be fed back into the User Reputation system (Module 1.1 from `dli_echonet_protocol.md`, which is linked to `UserProfile.reputation_score`). High-quality content and engagement can boost reputation; spammy behavior can lower it.

### 5.4. AI Model Types (High-Level Examples)

A suite of models may be employed:

*   **Natural Language Processing (NLP) Models:**
    *   For `text_content` and `comment_text` (via `context_hash`): Sentiment analysis, topic modeling, coherence scoring, toxicity detection, spam keyword detection, (future) plagiarism/originality checks against a corpus.
    *   Libraries: TensorFlow/Keras, PyTorch, spaCy, Hugging Face Transformers.
*   **Behavioral Analysis / Anomaly Detection Models:**
    *   To identify unusual patterns of interaction (e.g., bot-like activity, sudden bursts of likes from new/low-reputation accounts).
    *   To analyze user interaction graphs for detecting coordinated inauthentic behavior.
    *   Libraries: Scikit-learn, PyOD.
*   **Reputation-Aware Scoring Models:**
    *   Models that incorporate user reputation scores as features to weigh the significance of their actions (content creation, PoP signals).
*   **Ensemble Models:** Combining outputs from multiple specialized models to produce final scores.

### 5.5. AI Model Integration & Workflow

1.  **Data Collection:** Relevant data (ContentPost, PoPSignals, UserProfile summaries) is collected.
2.  **Preprocessing & Feature Engineering:** Data is cleaned, transformed, and features are extracted for model input.
3.  **Model Inference:** Data is fed into the deployed AI/ML models.
    *   **Triggering:** This can be event-driven (e.g., new post, new PoP signal) for rapid feedback, or batch-processed (e.g., nightly) for more complex analyses or model retraining.
    *   **AI Service Nodes (Conceptual):** Specialized nodes in the network might be responsible for running AI models, or DLI Consensus nodes might run them. For MVP, this could be a centralized service providing an API, with a plan to decentralize.
4.  **Output Handling:** Scores and adjusted weights are generated.
5.  **Feedback to DLI Consensus:** The `content_quality_score` and `spam_probability_score` are provided as key inputs to the DLI Consensus mechanism (Section 6) for content validation.
6.  **Feedback to Reputation System:** Outputs influencing user reputation are fed back to the user identity/reputation module.

### 5.6. Ethical AI, Bias Mitigation, and Explainability (XAI)

*   **Bias Audits:** AI models MUST be regularly audited for potential biases (e.g., against certain topics, user groups, language styles).
*   **Fairness Metrics:** Monitor models for fairness across different user demographics and content types.
*   **Transparency in Scoring (Conceptual):** While full model internals are complex, strive for "Reason Codes" or "Influence Factors" that provide users with some understanding of why content received a particular score or why an interaction was down-weighted. This is an XAI (Explainable AI) goal.
*   **Model Governance:** The selection, training, and updating of AI models should be subject to network governance processes to ensure community trust and alignment with platform values.
*   **Data Privacy in Training:** If user data is used for training, it MUST be anonymized or used with explicit consent, adhering to privacy principles. Federated learning is a future option.

For MVP, AI/ML integration will start with **stubs** for AI scoring functions. These stubs will return predefined or randomized scores to allow the rest of the PoP and DLI Consensus pipeline to be built and tested. True AI model development and deployment is a significant, iterative effort.

---

## 6. DLI Consensus Mechanism for Content Validation

The Decentralized Ledger-Inspired (DLI) Consensus mechanism provides a decentralized way for the network to agree on the quality and spam scores of content, based on aggregated Proof-of-Engagement (PoP) signals and AI/ML assessments. This is **not** a full blockchain consensus for all state, but a specialized consensus for content validation metrics.

### 6.1. Purpose and Goals

*   To establish a network-agreed `content_quality_score` and `spam_probability_score` for each `ContentPost`.
*   To do so in a decentralized manner, involving multiple DLI Consensus Nodes.
*   To provide a transparent and verifiable basis for subsequent actions (e.g., reward distribution, content ranking/filtering).
*   To be more lightweight and faster than full blockchain consensus for every piece of content.

### 6.2. Participants: DLI Consensus Nodes

*   **Role:** These nodes are responsible for participating in the DLI consensus rounds to validate content scores.
*   **Eligibility (Nexus Protocol Integration):** Similar to Witness Nodes, DLI Consensus Nodes will likely need to meet certain criteria defined by the broader Nexus Protocol:
    *   Stake Requirement.
    *   Reputation Score.
    *   Uptime and Performance metrics.
*   **Selection for a Round:** For any given content validation round, a subset of eligible DLI Consensus Nodes might be chosen (e.g., randomly, or based on proximity/sharding if the network scales significantly). For MVP, a smaller, potentially fixed set of known DLI nodes might participate.

### 6.3. DLI Consensus Process (High-Level, Round-Based Example)

The following outlines a conceptual round-based approach. The exact consensus algorithm (e.g., a simplified BFT variant, Raft-like leader election for rounds, or a voting/staking-weighted aggregation) needs further research and specification based on desired trade-offs (speed, consistency, complexity).

1.  **Content Validation Proposal:**
    *   **Trigger:** A `ContentPost` (identified by its `post_id` or `manifest_cid`) becomes eligible for DLI consensus, e.g., after:
        *   A certain amount of time has passed since its PoW claim was established.
        *   It has accumulated a minimum number of PoP signals.
        *   Explicit submission by the originator or an interested party.
    *   **Proposal Message (`ContentValidationProposal`):** Contains `post_id`, `proposer_id`, `timestamp`. This is broadcast to participating DLI Consensus Nodes for the current round/batch.

2.  **Evidence Gathering & Local Evaluation by DLI Nodes:**
    *   Each participating DLI Node, upon receiving a `ContentValidationProposal`:
        *   Retrieves the `ContentPost` data from DDS (using `post_id` to find `manifest_cid` if necessary, or directly if `manifest_cid` is proposed).
        *   Gathers associated `PoPSignalRecord`s (e.g., from a PoP aggregation service or by querying).
        *   Obtains AI/ML scores (`content_quality_score`, `spam_probability_score`, adjusted PoP weights) for the content and its interactions. This might involve:
            *   Querying dedicated AI Service Nodes.
            *   Running local instances of (potentially lighter) AI models if DLI nodes are also AI capable.
            *   For MVP: Using scores from a (stubbed) centralized AI service.
        *   Performs its own evaluation based on the PoP signals (e.g., sum of `ai_adjusted_weight`), AI scores, and potentially other heuristics (e.g., author reputation).

3.  **Voting/Attestation by DLI Nodes:**
    *   Each DLI Node forms its opinion on the `content_quality_score` and `spam_probability_score`.
    *   It creates and signs a `DLINodeVote` message containing `post_id`, its `dli_node_id`, its calculated scores, and its signature.
    *   These votes are broadcast to other DLI Consensus Nodes participating in the round for that content, or to a round leader/coordinator if applicable.

4.  **Aggregation & Consensus:**
    *   **Mechanism (TBD in detail, options):**
        *   **Leader-Based:** A temporary leader for the round collects votes. If a supermajority (e.g., 2/3+ of participating DLI stake/reputation) agrees on scores within a certain tolerance, the leader finalizes.
        *   **Quorum Voting:** Votes are gossiped. Once a node sees a quorum (e.g., >M of N) of consistent votes, it considers the scores finalized.
        *   **Score Averaging/Median:** If scores are continuous, an aggregation function (e.g., weighted average based on DLI node stake/reputation, or median) could be used once enough votes are collected.
    *   The goal is to arrive at a single, agreed-upon `agreed_quality_score` and `agreed_spam_score` for the `ContentPost`.

5.  **Result Publication & Finality:**
    *   The final agreed-upon scores, along with evidence (e.g., a list of supporting `DLINodeVote` CIDs or the votes themselves), are packaged into a `FinalContentScores` record.
    *   This record is made publicly available and verifiable (e.g., stored on DDS, announced on DHT, linked from the original `ContentPost` via an application-layer update).
    *   Once finalized by DLI consensus, these scores are considered the network's official validation for that content for that period/round. They can then be used by the Rewards module, content ranking algorithms, etc.

### 6.4. Data Structures for DLI Consensus (Conceptual Protobuf)

```protobuf
// In a conceptual dli_consensus_protocol.proto
// Assuming google.protobuf.Timestamp is imported

message ContentValidationProposal {
  string proposal_id = 1; // UUID for this proposal instance
  string post_id = 2;     // The Content ID (post_id) of the content to be validated
  // string manifest_cid = 3; // Or the manifest_cid from DDS
  string proposer_node_id = 4; // PeerID/DID of the node proposing validation
  google.protobuf.Timestamp proposal_timestamp = 5;
  // repeated pop_protocol.PoPSignalRecord pop_signals_summary = 6; // Or CIDs of batches of PoP signals
  // string ai_scores_reference_cid = 7; // CID of an AI scoring report on DDS
}

message DLINodeVote {
  string vote_id = 1;         // UUID for this vote
  string proposal_id = 2;     // Links to the ContentValidationProposal
  string post_id = 3;
  string dli_node_id = 4;     // PeerID/DID of the voting DLI node
  double calculated_quality_score = 5; // e.g., 0.0 to 1.0
  double calculated_spam_score = 6;    // e.g., 0.0 to 1.0
  google.protobuf.Timestamp vote_timestamp = 7;
  bytes signature = 8;        // DLI Node's signature over (proposal_id, post_id, scores, timestamp)
}

message FinalContentScores {
  string record_id = 1;        // UUID for this result record
  string post_id = 2;
  double agreed_quality_score = 3;
  double agreed_spam_score = 4;
  string dli_consensus_round_id = 5; // Identifier for the consensus round
  google.protobuf.Timestamp finalization_timestamp = 6;
  repeated string supporting_vote_cids = 7; // CIDs of DLINodeVote objects on DDS, or embedded votes
  // map<string, string> consensus_metadata = 8; // e.g., number of participants, algorithm used
}
```

### 6.5. Frequency and Batching

*   DLI Consensus for content validation does not need to run in real-time for every new piece of content.
*   It can be performed in batches (e.g., hourly, daily) or triggered when content reaches certain thresholds (e.g., number of PoP signals, views, age).
*   This allows for efficient processing and reduces network overhead compared to per-content immediate consensus.

### 6.6. MVP Simplifications

*   **Single "Coordinator" Node:** For MVP, the DLI "consensus" might be simulated by a single, trusted coordinator node that gathers PoP/AI data and assigns final scores. This allows testing the data pipeline.
*   **Simplified Voting/Aggregation:** If multiple nodes are simulated, a very simple majority vote or averaging of scores might be used, without complex BFT guarantees.
*   The focus is on the data flow from PoP/AI through to a `FinalContentScores` record.

This DLI Consensus mechanism, while lighter than full blockchain consensus, is key to achieving decentralized agreement on content value within DigiSocialBlock.

---

## 7. Network Messages (Conceptual Protobuf)

This section outlines conceptual Protobuf messages for interactions related to PoP signal handling, AI score integration, and the DLI Consensus process. These would typically reside in one or more `.proto` files (e.g., `pop_protocol.proto`, `ai_integration_protocol.proto`, `dli_consensus_protocol.proto`).

The data structures `PoPSignalRecord`, `ContentValidationProposal`, `DLINodeVote`, and `FinalContentScores` defined in Sections 4.3 and 6.4 are themselves key messages used in these flows. Additional RPC-style request/response messages are outlined below.

### 7.1. PoP Signal Related Messages

*   **`SubmitPoPSignalRequest`**:
    *   Sent by a client/node to a PoP aggregation service or directly to DLI nodes (if gossiped).
    *   Contains one or more `PoPSignalRecord`s.
    ```protobuf
    message SubmitPoPSignalRequest {
      repeated pop_protocol.PoPSignalRecord signals = 1;
    }
    ```
*   **`SubmitPoPSignalResponse`**:
    *   Acknowledgement of receipt.
    ```protobuf
    message SubmitPoPSignalResponse {
      bool success = 1;
      string message = 2; // e.g., "Signals received for processing"
      // repeated string accepted_signal_ids = 3;
      // repeated string rejected_signal_ids_with_reason = 4;
    }
    ```
*   **`GetAggregatedPoPRequest`**:
    *   Sent to a PoP aggregation service or DLI node to get engagement data for a post.
    ```protobuf
    message GetAggregatedPoPRequest {
      string post_id = 1;
      // google.protobuf.Timestamp since_timestamp = 2; // Optional filter
    }
    ```
*   **`GetAggregatedPoPResponse`**:
    *   Returns aggregated/summarized PoP data for a post.
    ```protobuf
    message GetAggregatedPoPResponse {
      string post_id = 1;
      int64 total_likes = 2;        // Example aggregate
      int64 total_comments = 3;     // Example aggregate
      double aggregated_pop_score = 4; // Example combined score from PoP signals
      // repeated pop_protocol.PoPSignalRecord raw_signals = 5; // Optional: if returning raw signals too
    }
    ```

### 7.2. AI/ML Service Interaction Messages (Conceptual)

*   **`GetAIScoresRequest`**:
    *   Sent by a DLI Node (or coordinator) to an AI Service Node.
    ```protobuf
    message GetAIScoresRequest {
      string post_id = 1;
      // core_types.ContentPost post_data = 2; // Full post data, or CID to fetch from DDS
      // repeated pop_protocol.PoPSignalRecord associated_pop_signals = 3; // Relevant PoP signals
      // core_types.UserProfile author_profile_summary = 4; // Summary of author's profile/reputation
    }
    ```
*   **`GetAIScoresResponse`**:
    *   Response from AI Service Node.
    ```protobuf
    message GetAIScoresResponse {
      string post_id = 1;
      double content_quality_score = 2; // e.g., 0.0 to 1.0
      double spam_probability_score = 3; // e.g., 0.0 to 1.0
      // repeated AIScoredPoPSignal adjusted_pop_signals = 4;
      // message AIScoredPoPSignal { string signal_id = 1; double ai_adjusted_weight = 2; }
      string error_message = 5;
    }
    ```

### 7.3. DLI Consensus Network Messages

The primary data structures (`ContentValidationProposal`, `DLINodeVote`, `FinalContentScores`) defined in Section 6.4 are the core "messages" exchanged during the DLI consensus process (e.g., via gossip, broadcast to participants, or sent to/from a round leader).

Additional RPC wrappers might be:

*   **`SubmitContentValidationProposalRequest`**:
    *   Contains `ContentValidationProposal`.
*   **`SubmitContentValidationProposalResponse`**:
    *   `bool accepted`, `string message`.
*   **`BroadcastDLINodeVoteRequest`**:
    *   Contains `DLINodeVote`.
*   **`BroadcastDLINodeVoteResponse`**:
    *   `bool acknowledged`.
*   **`GetFinalContentScoresRequest`**:
    *   `string post_id`.
*   **`GetFinalContentScoresResponse`**:
    *   Contains `FinalContentScores` (if available), `bool found`, `string error_message`.

These messages facilitate the structured exchange of information required for the Content Validation & Anti-Spam module to function.

---

## 8. Initial Implementation Plan (Conceptual)

This section outlines a high-level plan for the initial Go implementation of the Content Validation & Anti-Spam module (MVP).

### 8.1. Core Go Package Structure (Proposed)

The functionalities for this module will be organized into several Go packages, likely under `internal/content_validation` or similar.

```
content_validation/
|-- pop/                          // Proof-of-Engagement logic
|   |-- signal.go                   // PoPSignalRecord struct (if not from common proto) & related logic
|   |-- aggregator_interface.go     // Interface for PoP signal aggregation
|   |-- basic_aggregator.go         // Simple in-memory aggregator for MVP
|   `-- pop_test.go
|
|-- ai_integration/               // AI/ML model interaction
|   |-- interface.go                // Defines AIScoringService interface (e.g., GetContentScores)
|   |-- stub_service.go             // MVP stub implementation returning dummy scores
|   `-- stub_service_test.go
|
|-- dli_consensus/                // DLI Consensus logic
|   |-- types.go                    // DLI-specific types like Proposal, Vote, FinalScores (if not from common proto)
|   |-- coordinator.go              // MVP: Single coordinator logic for simulating consensus
|   |-- node_interface.go           // What a DLI node needs to implement (e.g., ProcessProposal, CastVote)
|   `-- dli_consensus_test.go
|
|-- rpc/                          // Network RPC definitions and handlers (if module-specific)
|   |-- content_validation_proto.pb.go // Generated from relevant .proto files
|   |-- handlers.go                 // RPC handlers for e.g. SubmitPoPSignal, GetAIScores
|   `-- rpc_test.go
|
|-- validation_service.go         // Main service orchestrating PoP, AI, and DLI for a piece of content
`-- validation_service_test.go

// protos/ (or a shared location)
// |-- pop_protocol.proto
// |-- ai_integration_protocol.proto
// |-- dli_consensus_protocol.proto
```

**Package Descriptions:**

*   **`pop`**: Handles Proof-of-Engagement signals.
    *   `signal.go`: May contain the Go struct for `PoPSignalRecord` if it's not directly used from a `pb.go` file, plus any helper methods.
    *   `aggregator_interface.go` & `basic_aggregator.go`: Defines how PoP signals are collected and aggregated (e.g., summed up per post) before being fed to AI or DLI consensus. MVP uses a simple in-memory aggregator.
*   **`ai_integration`**: Manages interaction with AI/ML models.
    *   `interface.go`: Defines an `AIScoringService` interface with methods like `GetContentScores(postData, popData) (AIScoreResult, error)`.
    *   `stub_service.go`: An MVP implementation of `AIScoringService` that returns hardcoded or randomized scores, allowing the rest of the system to be built without waiting for full AI model development.
*   **`dli_consensus`**: Contains the logic for the DLI Consensus mechanism.
    *   `types.go`: Holds Go structs for `ContentValidationProposal`, `DLINodeVote`, `FinalContentScores` if not directly using Protobuf generated types.
    *   `coordinator.go`: For MVP, this could be a simplified, single-node "coordinator" that simulates the consensus process by receiving PoP/AI data and outputting `FinalContentScores`.
    *   `node_interface.go`: Defines what a participating DLI node would do (relevant for future multi-node MVP).
*   **`rpc`**: (If needed for this module specifically, or part of a larger RPC service). Handles network messages related to submitting PoP signals, or DLI consensus messages if not handled by a generic P2P gossip layer.
*   **`validation_service.go`**: A higher-level service that orchestrates the overall flow: receives a request to validate content, gathers PoP, queries AI (stubs), triggers DLI consensus (simulated), and records/returns the `FinalContentScores`.

This structure aims to separate concerns for PoP processing, AI interaction, and the consensus mechanism itself.

---

### 8.2. Core Component Implementation Details for MVP

This details the MVP implementation focus for the key functions and interfaces within the Content Validation & Anti-Spam module.

**1. PoP Signal Handling (`content_validation/pop/`)**

*   **`PoPSignalRecord` Struct:**
    *   Utilize the Go struct generated from the `PoPSignalRecord` Protobuf definition (Section 4.3).
*   **`BasicAggregator` (`basic_aggregator.go`):**
    *   Implements `AggregatorInterface`.
    *   `NewBasicAggregator() *BasicAggregator`: Initializes an in-memory map (e.g., `map[string][]*PoPSignalRecord` // post_id -> signals).
    *   `AddSignal(signal *pop_protocol.PoPSignalRecord) error`: Adds a signal to the in-memory store for the relevant `post_id`. Performs basic validation on the signal itself (e.g., required fields present).
    *   `GetSignalsForPost(postID string) ([]*pop_protocol.PoPSignalRecord, error)`: Retrieves all stored signals for a post.
    *   `AggregateSignalsForPost(postID string) (PopSummary, error)`:
        *   `PopSummary` (struct): `{ TotalInitialWeight float64, TotalAIAdjustedWeight float64, SignalCountByType map[InteractionType]int }`.
        *   Calculates basic aggregates from stored signals (e.g., sum of initial weights, count per interaction type). For MVP, `TotalAIAdjustedWeight` might just be sum of `initial_weight` if AI stubs don't adjust yet.

**2. AI Integration Stubs (`content_validation/ai_integration/`)**

*   **`AIScoringService` Interface (`interface.go`):**
    ```go
    type AIScoreResult struct {
        ContentQualityScore    float64
        SpamProbabilityScore   float64
        AdjustedPoPSignals []*pop_protocol.PoPSignalRecord // Optional: if AI adjusts individual signal weights
    }

    type AIScoringService interface {
        GetContentScores(
            postData *core_types.ContentPost, // From echonet_v3_core.proto
            popSignals []*pop_protocol.PoPSignalRecord,
            // authorReputation float64, // Optional additional inputs
            // interactorReputations map[string]float64,
        ) (*AIScoreResult, error)
    }
    ```
*   **`StubAIService` (`stub_service.go`):**
    *   Implements `AIScoringService`.
    *   `GetContentScores(...)`:
        *   Returns dummy/random scores for `ContentQualityScore` (e.g., 0.75) and `SpamProbabilityScore` (e.g., 0.1).
        *   For MVP, it might not modify `AdjustedPoPSignals`, simply passing them through or copying `initial_weight` to `ai_adjusted_weight`.
        *   Logs the request for inspection.

**3. DLI Consensus - MVP Coordinator (`content_validation/dli_consensus/coordinator.go`)**

*   **`Coordinator` Struct:**
    *   `NewCoordinator(popAggregator pop.AggregatorInterface, aiService ai_integration.AIScoringService)`
*   **`ProcessContentValidation(postID string, postData *core_types.ContentPost) (*dli_consensus_protocol.FinalContentScores, error)`:**
    1.  **(PoP):** Call `popAggregator.GetSignalsForPost(postID)` to get raw PoP signals (or `AggregateSignalsForPost` for summary).
    2.  **(AI Stub):** Call `aiService.GetContentScores(postData, rawPopSignals, ...)` to get AI-derived scores.
    3.  **(Simplified "Consensus"):**
        *   Directly use the `ContentQualityScore` and `SpamProbabilityScore` from the `AIScoreResult` as the `agreed_quality_score` and `agreed_spam_score`.
        *   No actual multi-node voting or complex aggregation in MVP. The "coordinator" effectively dictates the final scores based on (stubbed) AI output.
    4.  Construct and return a `dli_consensus_protocol.FinalContentScores` message, populating:
        *   `record_id` (new UUID).
        *   `post_id`.
        *   `agreed_quality_score`, `agreed_spam_score`.
        *   `dli_consensus_round_id` (e.g., a timestamp or simple counter).
        *   `finalization_timestamp` (`time.Now()`).
        *   `supporting_vote_cids` can be empty or contain a self-attestation from the coordinator.

**4. Orchestration (`content_validation/validation_service.go`)**

*   **`ValidationService` Struct:**
    *   Holds instances of `pop.AggregatorInterface`, `ai_integration.AIScoringService`, `dli_consensus.Coordinator`.
*   **`ValidateAndScoreContent(post *core_types.ContentPost) (*dli_consensus_protocol.FinalContentScores, error)`:**
    1.  (Assumes `post` is already stored in DDS and has PoP signals being generated for its `post.PostId`).
    2.  Calls `dli_consensusCoordinator.ProcessContentValidation(post.PostId, post)`.
    3.  Returns the `FinalContentScores` or error.
    4.  (Future) This service could also handle persistence of `FinalContentScores` (e.g., to DDS).

This MVP implementation focuses on establishing the data flow and interfaces between PoP signal collection, (stubbed) AI scoring, and a (simulated) DLI consensus, resulting in final scores for content.

---

### 8.3. MVP Scope for Content Validation & Anti-Spam Module

The MVP for this module will focus on establishing the end-to-end data pipeline from PoP signal generation to a final (simulated) consensus score for content, using stubs for complex components like AI and true decentralized consensus.

**Included in MVP Functionality:**

1.  **PoP Signal Generation & Basic Aggregation:**
    *   Ability to create `PoPSignalRecord`s for basic interactions (e.g., Likes, Comments on a `ContentPost`).
    *   An in-memory `BasicAggregator` that can receive these signals and provide a simple summary for a given `post_id` (e.g., count of likes, sum of initial weights).
2.  **AI Integration Stubs:**
    *   Implement the `AIScoringService` interface with a `StubAIService`.
    *   This stub will accept `ContentPost` data and PoP signal data (or summary) as input.
    *   It will return predefined or randomized `content_quality_score` and `spam_probability_score`.
    *   It will pass through or minimally adjust `ai_adjusted_weight` in `PoPSignalRecord`s (e.g., set it equal to `initial_weight`).
3.  **DLI Consensus Simulation (Single Coordinator):**
    *   Implement the `dli_consensus.Coordinator` as described in Section 8.2.
    *   The `ProcessContentValidation` method will take a `post_id` and its data, fetch aggregated PoP signals (from `BasicAggregator`), get scores from the `StubAIService`, and then directly generate a `FinalContentScores` record.
    *   This simulates the outcome of a consensus process without implementing multi-node voting or agreement protocols.
4.  **Orchestration Service (`ValidationService`):**
    *   Implement the `ValidationService` to tie these components together: trigger PoP aggregation, call the AI stub, and then call the DLI coordinator to produce `FinalContentScores`.
5.  **Data Structures:**
    *   Go structs generated from all conceptual Protobuf messages defined (PoPSignalRecord, AI interaction messages, DLI consensus messages).
6.  **Basic API/Interface:** Expose a basic internal API through `ValidationService` (e.g., `ValidateAndScoreContent(post)`) that can be called by other parts of the system (e.g., after a post is made and some initial PoP signals are collected).

**Excluded from MVP Functionality:**

*   **Real AI/ML Model Implementation:** No actual training or deployment of machine learning models. All AI scoring is via stubs.
*   **Complex PoP Signal Weighting:** Dynamic adjustment of PoP signal weights based on user reputation, content age, velocity limits, etc., is excluded. Only `initial_weight` is used significantly by PoP aggregator.
*   **Robust, Multi-Node DLI Consensus:** No implementation of BFT, leader election (beyond the single coordinator), or distributed voting/aggregation.
*   **Integration with Reward Distribution Module:** While `FinalContentScores` are generated, the actual distribution of rewards based on these scores is handled by a separate module and is out of scope for this MVP.
*   **Network Communication for DLI Consensus:** Since MVP uses a single coordinator, complex network messages for proposal dissemination, voting, and result broadcasting are not implemented. RPCs for PoP signal submission and AI score retrieval (if AI service were externalized from DLI node) would be simple.
*   **Persistence of PoP Signals & Final Scores:** In-memory storage for PoP signals in `BasicAggregator` and no defined persistence for `FinalContentScores` beyond what the caller of `ValidationService` might do.
*   **Advanced Anti-Spam Heuristics:** Beyond the stubbed `spam_probability_score`.

The MVP aims to prove the architectural concept and data flow for content validation, from raw engagement signals through a (simulated) AI and consensus pipeline to a final set of content scores.

---

### 8.4. Unit Test Strategy for Content Validation & Anti-Spam MVP

Unit tests will ensure the core logic of each MVP component functions as expected.

**Target Packages:** `content_validation/pop`, `content_validation/ai_integration`, `content_validation/dli_consensus`, `content_validation`.

**1. `pop` Package Tests (`pop_test.go`):**
    *   **`TestPoPSignalRecordValidation` (if `PoPSignalRecord` has a `Validate()` method):**
        *   Test creation with valid/invalid fields (post_id, user_ids, type, timestamp, initial_weight).
    *   **`TestBasicAggregator_AddSignal`:**
        *   Test adding valid signals for new and existing post_ids.
        *   Test adding an invalid signal (e.g., missing post_id) - should error or be rejected.
        *   Verify internal map structure reflects added signals.
    *   **`TestBasicAggregator_GetSignalsForPost`:**
        *   Test retrieving signals for a post with multiple signals.
        *   Test retrieving for a post with no signals (should return empty list, no error).
        *   Test retrieving for a non-existent post_id.
    *   **`TestBasicAggregator_AggregateSignalsForPost`:**
        *   Test with various signals (different types, weights) for a post. Verify `PopSummary` (TotalInitialWeight, SignalCountByType) is calculated correctly.
        *   Test aggregation for a post with no signals.

**2. `ai_integration` Package Tests (`stub_service_test.go`):**
    *   **`TestStubAIService_GetContentScores`:**
        *   Verify the stub returns the expected dummy/randomized (within defined behavior) `ContentQualityScore` and `SpamProbabilityScore`.
        *   Verify it correctly passes through or sets `ai_adjusted_weight` in `AdjustedPoPSignals` as per MVP stub logic (e.g., equals `initial_weight`).
        *   Test that it logs input or can be inspected to show it received data.

**3. `dli_consensus` Package Tests (`dli_consensus_test.go`):**
    *   **Data Structure Validation (if applicable):** Tests for `Validate()` methods on `ContentValidationProposal`, `DLINodeVote`, `FinalContentScores` (checking required fields, basic formats).
    *   **`TestCoordinator_ProcessContentValidation`:**
        *   Mock `pop.AggregatorInterface` to provide various PoP summaries/signal lists.
        *   Mock `ai_integration.AIScoringService` to provide various AI score results.
        *   Test that the `Coordinator` correctly calls these mocks with expected inputs (e.g., `post_id`, `postData`).
        *   Verify the generated `FinalContentScores` record contains:
            *   The correct `post_id`.
            *   `agreed_quality_score` and `agreed_spam_score` directly from the (mocked) AI service output for MVP.
            *   A valid `record_id` (UUID) and `finalization_timestamp`.
            *   Correct `dli_consensus_round_id` (if applicable for MVP coordinator).
        *   Test error propagation if mocks return errors.

**4. `content_validation` Package Tests (`validation_service_test.go`):**
    *   **`TestValidationService_ValidateAndScoreContent`:**
        *   Mock all dependencies (`pop.AggregatorInterface`, `ai_integration.AIScoringService`, `dli_consensus.Coordinator`).
        *   Provide a `core_types.ContentPost`.
        *   Verify the service calls its internal components in the correct order (PoP aggregation -> AI scoring -> DLI coordination).
        *   Verify the final `FinalContentScores` from the coordinator is returned.
        *   Test error handling and propagation from each internal component.

These unit tests will focus on the logic within each component and the correct interaction with (mocked) dependencies, ensuring the data pipeline for the MVP is sound.

---
*(End of Document: Content Validation & Anti-Spam Specification & Initial Implementation Plan)*
