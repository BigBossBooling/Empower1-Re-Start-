package consensus

import (
	"encoding/hex" // Added for consistent map keys
	"fmt"          // Added for error messages
	"log"          // Added for logging in UpdateHeight
	"sort"         // Added for deterministic proposer selection
	"sync"
	// "empower1.com/empower1blockchain/internal/core" // Not directly needed for these structs
)

// Validator defines the structure for a validator in the consensus mechanism.
type Validator struct {
	Address    []byte  `json:"address"`    // Validator's public key hash (address)
	Stake      uint64  `json:"stake"`      // Amount of PTCN staked
	Reputation float64 `json:"reputation"` // AI-assessed reputation score
	// Add other fields like JailedUntil, MissedBlocks, etc. as needed later
}

// ConsensusState holds information relevant to the current state of the consensus,
// such as the current validator set, epoch information, etc.
type ConsensusState struct {
	mu             sync.RWMutex
	CurrentEpoch   uint64                `json:"currentEpoch"`
	Validators     map[string]*Validator `json:"validators"` // Keyed by hex string of Validator.Address
	// TODO: Add more fields like last block proposer, pending state transitions related to consensus
	// TODO: Add a proper logger instance here too, instead of package-level log.
}

// NewConsensusState creates and returns a new ConsensusState instance.
func NewConsensusState() *ConsensusState {
	return &ConsensusState{
		Validators: make(map[string]*Validator),
		// CurrentEpoch might start at 0 or 1
	}
}

// LoadInitialValidators populates the validator set with a predefined list of validators.
// This is typically used at genesis or for testing.
func (cs *ConsensusState) LoadInitialValidators(initialValidators []*Validator) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.Validators == nil {
		cs.Validators = make(map[string]*Validator)
	}
	for _, val := range initialValidators {
		if val != nil && val.Address != nil {
			addressKey := hex.EncodeToString(val.Address) // USE HEX KEY
			cs.Validators[addressKey] = val
		}
	}
}

// GetValidator returns a validator by address.
func (cs *ConsensusState) GetValidator(address []byte) (*Validator, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	addressKey := hex.EncodeToString(address) // USE HEX KEY
	val, ok := cs.Validators[addressKey]
	return val, ok
}

// GetProposerForHeight determines the designated proposer for a given block height.
// For V1, this is a simple round-robin based on the loaded validators.
// TODO: Implement more sophisticated proposer selection logic (e.g., stake-weighted, reputation-based).
func (cs *ConsensusState) GetProposerForHeight(height int64) (*Validator, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if len(cs.Validators) == 0 {
		return nil, fmt.Errorf("no validators available in consensus state")
	}

	// Simple round-robin for V1:
	// Convert map keys (hex addresses) to slice to get a deterministic order for round-robin.
	validatorAddresses := make([]string, 0, len(cs.Validators))
	for addrHex := range cs.Validators {
		validatorAddresses = append(validatorAddresses, addrHex)
	}
	sort.Strings(validatorAddresses) // Sort hex addresses for deterministic proposer selection

	if len(validatorAddresses) == 0 { // Should be caught by earlier check, but defensive
		 return nil, fmt.Errorf("no validator addresses collected for proposer selection")
	}

	proposerIndex := int(height % int64(len(validatorAddresses)))
	selectedAddressHex := validatorAddresses[proposerIndex]

	selectedValidator, ok := cs.Validators[selectedAddressHex]
	if !ok {
		 // This should not happen if keys are consistent and list is derived from map.
		 return nil, fmt.Errorf("could not find validator for selected hex address (inconsistency): %s", selectedAddressHex)
	}

	return selectedValidator, nil
}

// UpdateHeight is called when a new block is successfully added to the chain.
// For V1, it might just log or update an internal consensus view of height.
// The primary chain height is managed by the Blockchain object.
// TODO: Determine how consensus-specific height/epoch tracking should work.
func (cs *ConsensusState) UpdateHeight(newBlockHeight int64) {
    cs.mu.Lock() // Assuming CurrentEpoch might be updated based on height later
    defer cs.mu.Unlock()
    // For now, just log. If ConsensusState needs its own height tracking:
    // cs.CurrentHeight = newBlockHeight // If such a field existed (note: CurrentEpoch exists)
    log.Printf("CONSENSUS_STATE: Notified of new block height: %d. (CurrentEpoch: %d)", newBlockHeight, cs.CurrentEpoch)
}
