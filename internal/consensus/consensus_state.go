package consensus

import (
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
	Validators     map[string]*Validator `json:"validators"` // Keyed by hex string of Validator.Address ideally
	// TODO: Add more fields like last block proposer, pending state transitions related to consensus
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
			// In a real system, ensure addresses are hex encoded for map keys consistently.
			// For now, directly using string(val.Address) might lead to issues if addresses are not printable strings
			// or if byte slices representing the same logical address have different string conversions.
			// A better key would be hex.EncodeToString(val.Address).
			// This will be refined when actual crypto keys are used.
			cs.Validators[string(val.Address)] = val
		}
	}
}

// GetValidator returns a validator by address.
// TODO: Implement proper hex encoding for address keys for lookup consistency.
func (cs *ConsensusState) GetValidator(address []byte) (*Validator, bool) {
    cs.mu.RLock()
    defer cs.mu.RUnlock()
    // Key should ideally be hex.EncodeToString(address) if that's how they are stored in LoadInitialValidators.
    // For now, maintaining consistency with the current simple string(address) keying.
    val, ok := cs.Validators[string(address)]
    return val, ok
}
