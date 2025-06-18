package state

import (
	"sync"
	"log"
	"os"
	"errors"
	"fmt" // Added for Sprintf
	"encoding/hex" // Added for hex.EncodeToString
	"bytes"        // Added for bytes.Equal
	"empower1.com/empower1blockchain/internal/core" // Added for Block and Transaction types
)

// Package state manages the global state of the EmPower1 Blockchain.
// This includes the UTXO set, smart contract state (Contract State Trie),
// account balances (derived from UTXOs), and the logic for applying
// transactions to update the state, ultimately producing the Block.StateRoot.

// --- Custom Errors for State Manager ---
var (
	ErrStateInit              = errors.New("state manager initialization error")
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrUTXONotFound           = errors.New("utxo not found")
	ErrUTXOAlreadySpent       = errors.New("utxo already spent")
	ErrInvalidTransactionType = errors.New("invalid transaction type for state update")
	ErrStateCorruption        = errors.New("blockchain state corrupted")
	ErrWealthLevelNotFound    = errors.New("wealth level not found for address") // EmPower1 specific
)

// UTXO represents an unspent transaction output.
// This is a fundamental component for UTXO-based blockchains.
type UTXO struct {
	TxID    []byte // The ID of the transaction that created this output
	Vout    int    // The index of the output in that transaction (using int as per original, though uint32 was in core.TxInput.Vout)
	Value   uint64 // The amount of value in this output
	Address []byte // The recipient's public key hash (address)
}

// Account represents the state associated with an address.
// EmPower1: This is crucial for storing AI-assessed wealth levels.
type Account struct {
	Balance    uint64            // Current total balance from UTXOs
	Nonce      uint64            // Transaction nonce (for account-based models or replay protection)
	WealthLevel map[string]string // AI/ML assessed wealth level (e.g., {"category": "affluent", "last_updated": "timestamp"})
	// V2+: ReputationScore float64 // Derived from on-chain behavior
	// V2+: DID            []byte // Decentralized Identifier
}

// State manages the global, synchronized state of the EmPower1 Blockchain.
// For V1, this is an in-memory UTXO set manager with conceptual Account states.
// In a production system, this would be backed by persistent storage (e.g., LevelDB, RocksDB).
type State struct {
	mu           sync.RWMutex                     // Mutex for concurrent access
	utxoSet      map[string]*UTXO                 // UTXO set: maps UTXO ID (TxID:Vout) to UTXO object
	accounts     map[string]*Account              // Account states: maps address (hex) to Account object
	logger       *log.Logger                      // Dedicated logger for the State instance
}

// NewState creates a new State manager.
// Initializes the in-memory state storage.
func NewState() (*State, error) {
	logger := log.New(os.Stdout, "STATE: ", log.Ldate|log.Ltime|log.Lshortfile)
	state := &State{
		utxoSet:  make(map[string]*UTXO),
		accounts: make(map[string]*Account), // Initialize account map
		logger:   logger,
	}
	if state.logger == nil {
		return nil, ErrStateInit
	}
	state.logger.Println("State manager initialized.")
	return state, nil
}

// UpdateStateFromBlock updates the blockchain state based on the transactions within a new, valid block.
// This is the core state transition function, called by the blockchain after a block is added.
// It directly supports "Systematize for Scalability, Synchronize for Synergy".
func (s *State) UpdateStateFromBlock(block *core.Block) error {
	s.mu.Lock() // Acquire write lock for state modification
	defer s.mu.Unlock()

	s.logger.Printf("STATE: Updating state from block #%d (%x)", block.Height, block.Hash)

	// Process each transaction in the block
	for _, tx := range block.Transactions {
		txIDHex := hex.EncodeToString(tx.ID) // tx.ID comes from core.Transaction

		// 1. Process Inputs (Remove Spent UTXOs)
		// For standard transactions, mark inputs as spent.
		if tx.TxType == core.TxStandard || tx.TxType == core.TxContractCall || tx.TxType == core.TxContractDeploy {
			// Process inputs for debiting conceptual account balances
			for inputIdx, input := range tx.Inputs {
				utxoKey := fmt.Sprintf("%s:%d", hex.EncodeToString(input.TxID), int(input.Vout))

				// Find the UTXO being spent to get its value and address
				spentUTXO, exists := s.utxoSet[utxoKey]
				if !exists {
					s.logger.Printf("STATE_ERROR: Block %d, Tx %s: Input %d (%s) UTXO not found in state. Possible double-spend or invalid block.",
						block.Height, txIDHex, inputIdx, utxoKey)
					return fmt.Errorf("%w: input UTXO %s not found for tx %s in block %d", ErrUTXONotFound, utxoKey, txIDHex, block.Height)
				}
				// Conceptually update (debit) the balance of the input's address
				// This assumes input.PubKey can be reliably converted to the address that controlled the UTXO
				// In a real system, you'd get the address from the spentUTXO.Address
				s.updateAccountBalance(spentUTXO.Address, spentUTXO.Value, false) // false for debit

				delete(s.utxoSet, utxoKey)
				s.logger.Printf("STATE: Removed spent UTXO %s for tx %s", utxoKey, txIDHex)
			}
		}

		// 2. Process Outputs (Add New UTXOs)
		for outputIdx, output := range tx.Outputs { // output is core.TxOutput
			utxoKey := fmt.Sprintf("%s:%d", hex.EncodeToString(tx.ID), outputIdx)
			if _, exists := s.utxoSet[utxoKey]; exists {
				s.logger.Printf("STATE_ERROR: Block %d, Tx %s: Output %d (%s) already exists in state. Possible tx ID collision or state corruption.",
					block.Height, txIDHex, outputIdx, utxoKey)
				return fmt.Errorf("%w: output UTXO %s already exists for tx %s in block %d", ErrUTXOAlreadySpent, utxoKey, txIDHex, block.Height)
			}
			newUTXO := &UTXO{
				TxID:    tx.ID,
				Vout:    outputIdx,
				Value:   output.Value,
				Address: output.PubKeyHash,
			}
			s.utxoSet[utxoKey] = newUTXO
			s.logger.Printf("STATE: Added new UTXO %s for tx %s, value %d to %x", utxoKey, txIDHex, output.Value, output.PubKeyHash)

			// Update conceptual account balance based on new UTXO
			s.updateAccountBalance(output.PubKeyHash, output.Value, true) // true for credit
		}

		// 3. EmPower1 Specific: AI/ML Driven State Updates (Wealth Gap Redistribution)
		if tx.TxType == core.TxStimulusPayment || tx.TxType == core.TxWealthTax {
			s.logger.Printf("STATE: Processing EmPower1 specific transaction Type: %s (ID: %s)", tx.TxType, txIDHex)
            // Conceptual: AI/ML logic would have determined recipients (outputs of Stimulus)
            // or payers (inputs of WealthTax). The actual balance changes are handled by
            // UTXO processing above. This section is for updating Account.WealthLevel.
            // Example for TxWealthTax inputs (payers):
            if tx.TxType == core.TxWealthTax {
                for _, input := range tx.Inputs {
                    // Assuming input.PubKey can be resolved to an address whose wealth level is being affected.
                    // This is a simplification; the AI decision would likely be linked via tx.AILogicID or similar.
                    // For now, we'll just log. A real implementation would fetch the UTXO, get its address,
                    // and then update the Account struct for that address.
                     s.logger.Printf("STATE: Conceptual wealth tax processed for input related to Tx %s. Actual state update for wealth level would occur here.", txIDHex)
                }
            }
            // Example for TxStimulusPayment outputs (recipients):
             if tx.TxType == core.TxStimulusPayment {
                for _, output := range tx.Outputs {
                    // This output is a stimulus payment. The recipient's Account.WealthLevel might be updated
                    // based on this new state, or this payment is a result of a prior WealthLevel assessment.
                    // The AI AuditLog for this transaction (tx.AILogicID, tx.AIRuleTrigger) provides context.
                    s.logger.Printf("STATE: Conceptual stimulus payment processed for output to %x in Tx %s. Actual state update for wealth level would occur here.", output.PubKeyHash, txIDHex)
                }
            }
		}

		// 4. Update Account Nonce (for account-based transactions, conceptual for hybrid)
		// s.updateAccountNonce(tx.From, newNonceValue) // Placeholder
	}
	s.logger.Printf("STATE: State update from block #%d complete. Current UTXOs: %d, Conceptual Accounts: %d", block.Height, len(s.utxoSet), len(s.accounts))
	return nil
}

// GetBalance returns the current confirmed balance for a given address.
// In a pure UTXO model, this sums up all UTXOs belonging to the address.
func (s *State) GetBalance(address []byte) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_ = hex.EncodeToString(address) // Preserving as in original snippet, even if not directly used for logging here.
	totalBalance := uint64(0)
	found := false

	for _, utxo := range s.utxoSet {
		if bytes.Equal(utxo.Address, address) {
			totalBalance += utxo.Value
			found = true
		}
	}

	if !found {
        s.logger.Printf("STATE: GetBalance for address %x: No UTXOs found, returning ErrInsufficientBalance.", address)
		return 0, ErrInsufficientBalance // As per original user snippet's error handling
	}
	s.logger.Printf("STATE: GetBalance for address %x: %d", address, totalBalance)
	return totalBalance, nil
}

// FindSpendableOutputs finds and returns a list of UTXOs that can be spent by an address to cover an amount.
// This is crucial for creating new transactions.
func (s *State) FindSpendableOutputs(address []byte, amount uint64) ([]UTXO, uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var foundOutputs []UTXO // Initialized as an empty slice
	accumulated := uint64(0)

	// Iterate through UTXOs and select those belonging to the address until amount is met.
	// The order of iteration over a map is not guaranteed. For deterministic UTXO selection,
	// UTXOs might need to be collected and sorted first (e.g., by value or TxID).
	// For V1, simple iteration is acceptable as per the original snippet.
	// TODO: Consider deterministic UTXO selection for V2+ to prevent transaction malleability issues
	// related to different input sets being chosen by different nodes for the same logical payment.
	for _, utxo := range s.utxoSet { // utxo is *UTXO from map
		if bytes.Equal(utxo.Address, address) {
			foundOutputs = append(foundOutputs, *utxo) // Append a copy of the UTXO
			accumulated += utxo.Value
			if accumulated >= amount {
				break // Found enough value
			}
		}
	}

	if accumulated < amount {
		s.logger.Printf("STATE: Insufficient balance for address %x. Requested %d, found %d.", address, amount, accumulated)
		return nil, 0, ErrInsufficientBalance
	}
	s.logger.Printf("STATE: Found %d spendable UTXOs for address %x, total value %d, to cover amount %d.", len(foundOutputs), address, accumulated, amount)
	return foundOutputs, accumulated, nil
}

// updateAccountBalance conceptually updates an account's balance based on UTXO changes.
// In a pure UTXO model, balances are always calculated by summing UTXOs (via GetBalance).
// This method updates a conceptual 'Balance' field in the Account struct for convenience
// or for hybrid model features. It should be used with awareness of its conceptual nature.
// This is an unexported helper method.
// It assumes the caller (e.g., UpdateStateFromBlock) already holds the necessary write lock on s.mu.
func (s *State) updateAccountBalance(address []byte, valueChange uint64, isCredit bool) {
	addrHex := hex.EncodeToString(address)
	account, exists := s.accounts[addrHex]
	if !exists {
		account = &Account{
			Balance:     0,
			Nonce:       0,
			WealthLevel: make(map[string]string), // Initialize empty
		}
		s.accounts[addrHex] = account
		s.logger.Printf("STATE: Created new conceptual account for address %s during balance update.", addrHex)
	}

	if isCredit {
		account.Balance += valueChange
		s.logger.Printf("STATE: Conceptual account balance credited for %s: +%d, new conceptual balance: %d", addrHex, valueChange, account.Balance)
	} else {
		// This part is more conceptual for UTXO, as "spending" removes UTXOs, balance is sum of remaining.
		// If this were a true account model, subtraction would happen here.
		if account.Balance >= valueChange {
			account.Balance -= valueChange
			s.logger.Printf("STATE: Conceptual account balance debited for %s: -%d, new conceptual balance: %d", addrHex, valueChange, account.Balance)
		} else {
			// This case should ideally be prevented by prior UTXO checks ensuring sufficient funds are being spent.
			// However, if the conceptual balance somehow diverges, this logs it.
			s.logger.Printf("STATE_WARNING: Conceptual account balance debit attempt for %s: -%d. Insufficient conceptual balance %d. True UTXO balance is via GetBalance.", addrHex, valueChange, account.Balance)
			// Do not let conceptual balance go negative if it's meant to track available funds.
			// For UTXO, this debit is more about reflecting that the value *was* there and is now spent from a UTXO.
			// The actual GetBalance() will always be correct.
			// If we strictly want Account.Balance to reflect GetBalance(), this debit logic might need to be a re-summation
			// or this Account.Balance field is understood to be only an approximation or event log.
			// Given the prompt, we'll allow it to go "negative" conceptually if something is wrong,
			// but log a warning. The true balance is always GetBalance().
			// Let's adjust to prevent negative, as it's confusing for a 'Balance' field.
			account.Balance = 0 // Or just log error and don't change if it would go negative.
                                // For now, set to 0 if an underflow would occur.
            s.logger.Printf("STATE: Conceptual balance for %s reset to 0 due to debit exceeding conceptual balance.", addrHex)

		}
	}
}


// GetWealthLevel retrieves the conceptual AI/ML assessed wealth level for an address.
// EmPower1 specific, leveraging AI/ML integration.
func (s *State) GetWealthLevel(address []byte) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addrHex := hex.EncodeToString(address)
	account, exists := s.accounts[addrHex]
	if !exists || account.WealthLevel == nil || len(account.WealthLevel) == 0 {
		s.logger.Printf("STATE: Wealth level not found for address %s", addrHex)
		return nil, ErrWealthLevelNotFound
	}
	// Return a copy to prevent external modification
	wealthCopy := make(map[string]string)
	for k, v := range account.WealthLevel {
		wealthCopy[k] = v
	}
	s.logger.Printf("STATE: GetWealthLevel for address %s: %v", addrHex, wealthCopy)
	return wealthCopy, nil
}

// UpdateWealthLevel is a conceptual function that would be called by the AI/ML module
// or consensus logic to update an account's wealth level in state.
// This simulates the direct integration of AI/ML insights into the blockchain state.
func (s *State) UpdateWealthLevel(address []byte, level map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addrHex := hex.EncodeToString(address)
	account, exists := s.accounts[addrHex]
	if !exists {
		account = &Account{
			Balance:     0, // Conceptual balance
			Nonce:       0,
			WealthLevel: make(map[string]string),
		}
		s.accounts[addrHex] = account
		s.logger.Printf("STATE: Created new account for address %s during wealth level update.", addrHex)
	}
	// Deep copy the map
	account.WealthLevel = make(map[string]string)
	for k, v := range level {
		account.WealthLevel[k] = v
	}
	s.logger.Printf("STATE: Updated wealth level for %s: %v", addrHex, level)
	return nil
}

// NOTE: Further methods for account state manipulation (e.g., for smart contract state trie)
// would be added here as the system evolves. This provides the basic UTXO and conceptual
// account state layer.
