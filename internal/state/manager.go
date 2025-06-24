package state

import (
	"sync"
	// "log" // Standard logger, or a custom one, can be added later.

	// Adjust import path based on actual module structure
	"bytes"
	"empower1/internal/core/types"
	internalerrors "empower1/internal/errors"
	"encoding/hex"
	"fmt"
	"strconv"
)

// UpdateStateFromBlock processes all transactions in a block and updates the UTXO set.
// Account balances and other state aspects will be handled in subsequent steps.
// This function aims for atomicity for the block's changes to the UTXO set:
// either all valid transactions update the UTXO set, or none do if an error occurs.
// Note: True atomicity with complex state might require more advanced patterns (e.g., journaling, temporary sets).
func (sm *StateManager) UpdateStateFromBlock(block *types.Block) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// --- TODO: Implement True Atomicity for Block Processing ---
	// The current implementation modifies state (utxoSet, accounts) in place.
	// If an error occurs midway through processing transactions in a block,
	// the state will be left in an inconsistent partial state.
	//
	// Proper atomicity requires:
	// 1. Creating temporary copies/caches of the state portions being modified (e.g., a temporary utxoSet and accounts map for this block).
	// 2. Applying all changes to these temporary structures.
	// 3. If any transaction processing fails: discard the temporary structures; the original state remains untouched.
	// 4. If all transactions process successfully: commit the changes by replacing the main state structures with the temporary ones (or merging them).
	//
	// This is a significant undertaking and will be addressed in a dedicated future implementation phase.
	// For now, the risk of inconsistency is noted if a transaction within a pre-validated block still causes an error here.
	// --- End Atomicity TODO ---

	for _, tx := range block.Transactions {
		// 1. Consume Inputs
		// For UTXO-based transactions (e.g., StandardTransaction, if it spends UTXOs)
		if tx.Type == types.StandardTransaction { // Assuming StandardTransaction always spends UTXOs
			if tx.Inputs == nil || len(tx.Inputs) == 0 {
				// This should ideally be caught by tx.Validate(), but good to have defense here.
				return fmt.Errorf("transaction %s of type %s has no inputs: %w", hex.EncodeToString(tx.TxID[:]), tx.Type.String(), internalerrors.ErrEmptyInputsForSpendingTx)
			}
			for _, input := range tx.Inputs {
				utxoKey := makeUTXOKey(input.PreviousTxHash, input.OutputIndex)
				if _, exists := sm.utxoSet[utxoKey]; !exists {
					// This indicates a serious issue if block validation passed:
					// - A double spend within the same block that wasn't caught by more sophisticated validation.
					// - Or an invalid transaction referencing a non-existent/spent UTXO.
					return fmt.Errorf("input UTXO %s for tx %s not found: %w", utxoKey, hex.EncodeToString(tx.TxID[:]), internalerrors.ErrUTXONotFound)
				}
				// --- TODO: Implement Account Balance Update (Consumption) ---
				// spentUTXO := sm.utxoSet[utxoKey] // Get the UTXO before deleting
				// spenderAddrHex := hex.EncodeToString(spentUTXO.RecipientAddress)
				// if account, ok := sm.accounts[spenderAddrHex]; ok {
				// 	if account.Balance < spentUTXO.Amount { // Should not happen if UTXOs are source of truth for balance
				// 		return fmt.Errorf("account %s balance %d less than spent UTXO amount %d for tx %s: %w", spenderAddrHex, account.Balance, spentUTXO.Amount, hex.EncodeToString(tx.TxID[:]), internalerrors.ErrInsufficientBalance)
				// 	}
				// 	account.Balance -= spentUTXO.Amount
				// } else { // Should not happen if accounts are created when UTXOs are received
				// 	return fmt.Errorf("spender account %s not found for input in tx %s: %w", spenderAddrHex, hex.EncodeToString(tx.TxID[:]), internalerrors.ErrAccountNotFound)
				// }
				// --- End Account Balance Update (Consumption) ---
				delete(sm.utxoSet, utxoKey)
			}
		}

		// 2. Create Outputs
		for i, output := range tx.Outputs {
			newUTXO := types.UTXO{
				TxID:             tx.TxID,
				OutputIndex:      uint32(i),
				Amount:           output.Amount,
				RecipientAddress: output.RecipientAddress,
			}
			utxoKey := makeUTXOKey(newUTXO.TxID, newUTXO.OutputIndex)
			if _, exists := sm.utxoSet[utxoKey]; exists {
				return fmt.Errorf("output UTXO %s for tx %s already exists in UTXO set: %w", utxoKey, hex.EncodeToString(tx.TxID[:]), internalerrors.ErrCriticalStateCorruption)
			}
			sm.utxoSet[utxoKey] = newUTXO

			// --- TODO: Implement Account Balance Update (Creation) ---
			// recipientAddrHex := hex.EncodeToString(output.RecipientAddress)
			// if account, ok := sm.accounts[recipientAddrHex]; ok {
			// 	account.Balance += output.Amount
			// } else {
			// 	sm.accounts[recipientAddrHex] = &types.Account{
			// 		Address:         output.RecipientAddress,
			// 		Balance:         output.Amount,
			// 		Nonce:           0, // Initial nonce
			// 		ReputationScore: 0, // Initial reputation
			// 	}
			// }
			// --- End Account Balance Update (Creation) ---
		}

		// --- TODO: Implement AI/ML Driven State Updates (Conceptual) ---
		// if tx.Type == types.StimulusTransaction || tx.Type == types.TaxTransaction {
		//	// Example for Stimulus: Outputs of a stimulus transaction credit accounts.
		//	// The AIProof and AILogicID in metadata would have validated the legitimacy
		//	// of this stimulus distribution during transaction validation or block validation.
		//	// Here, we would primarily focus on applying the state changes.
		//	// For a TaxTransaction, inputs might debit accounts (or UTXOs representing owed tax),
		//	// and the "output" might be to a treasury account or simply burning tokens.
		//
		//	// This logic would also need to interact with sm.accounts similar to balance updates.
		//	// For example, updating a 'LastStimulusReceivedBlock' field or adjusting 'WealthLevel'.
		//	// aiLogicID, _ := tx.Metadata["AILogicID"]
		//	// aiProof,   _ := tx.Metadata["AIProof"]
		//	// logger.Printf("Processing AI-driven transaction type %s for tx %s with AILogicID: %s, AIProof: %s",
		//	//	tx.Type.String(), hex.EncodeToString(tx.TxID[:]), string(aiLogicID), string(aiProof))
		//
		//	// For each output (if stimulus) or input (if tax collection from accounts):
		//	//  Get affected account(s)
		//	//  Update Account fields like ReputationScore, custom AI flags, etc., based on tx metadata and rules.
		// }
		// --- End AI/ML Driven State Updates ---
	}

	// TODO: Update currentBlockHeader in StateManager
	// sm.currentBlockHeader = &block.Header

	return nil
}


// Helper function to create a string key for the UTXO set map.
// Key is "TxIDHex:OutputIndex".
func makeUTXOKey(txID types.Hash, index uint32) string {
	return hex.EncodeToString(txID[:]) + ":" + strconv.FormatUint(uint64(index), 10)
}

// StateManager manages the overall state of the blockchain, including the UTXO set and account balances.
// It is responsible for applying transactions and ensuring state consistency.
// For now, it will be an in-memory store. Persistence will be added later.
type StateManager struct {
	mu sync.RWMutex // Read-Write mutex to protect concurrent access to state components

	// utxoSet stores unspent transaction outputs.
	// The key could be a string representation of "TxID:OutputIndex".
	utxoSet map[string]types.UTXO

	// accounts stores account states.
	// The key could be a string representation of the account address (e.g., hex encoded).
	accounts map[string]*types.Account

	// currentBlockHeader stores the header of the latest processed block.
	// This is important for context like current height, timestamp, etc.
	// currentBlockHeader *types.BlockHeader // To be added when processing blocks

	// logger *log.Logger // For logging state manager activities
}

// NewStateManager creates and returns a new StateManager instance.
// It initializes the necessary data structures for state management.
func NewStateManager(/*logger *log.Logger*/) *StateManager {
	return &StateManager{
		utxoSet:  make(map[string]types.UTXO),
		accounts: make(map[string]*types.Account),
		// logger:   logger,
		// currentBlockHeader: nil, // Initialize as nil, set with first block
	}
}

// GetBalance retrieves the balance for a given address.
// (Conceptual - will interact with accounts map)
func (sm *StateManager) GetBalance(address types.Address) (uint64, error) {
	sm.RLock()
	defer sm.RUnlock()

	// Convert address to string map key (e.g., hex encoded)
	// addrHex := hex.EncodeToString(address)
	// account, exists := sm.accounts[addrHex]
	// if !exists {
	// 	return 0, fmt.Errorf("account not found for address %s: %w", addrHex, internalerrors.ErrAccountNotFound) // Assuming ErrAccountNotFound
	// }
	// return account.Balance, nil

	// TODO: Implement actual logic using hex-encoded address as key
	// For now, implementing the UTXO summation approach as per plan.
	// Note: This is inefficient for frequent balance lookups. An account model or indexed UTXOs would be better.

	var balance uint64 = 0
	// No hex encoding needed for address if comparing directly with types.Address in UTXO.
	// However, []byte cannot be map keys directly. If UTXO.RecipientAddress was string, it'd be simpler.
	// For direct comparison of []byte, a loop is necessary.

	// To compare addresses, we can use bytes.Equal.
	// For efficiency, addresses in maps (like sm.accounts) are often stored as hex strings.
	// If sm.utxoSet used addresses directly as part of a more complex key or value structure,
	// this would be simpler. Given current map[string]types.UTXO, we iterate.

	for _, utxo := range sm.utxoSet {
		if bytes.Equal(utxo.RecipientAddress, address) {
			balance += utxo.Amount
		}
	}

	// If balance is 0, it could mean the account has no UTXOs or truly has a zero balance.
	// The concept of "ErrAccountNotFound" is more relevant if we were looking up an Account struct.
	// For a pure UTXO summation, a balance of 0 is a valid result.
	// If we want to distinguish "no UTXOs ever" from "all UTXOs spent to zero", that's more complex.
	return balance, nil // No error if balance is 0 from UTXO summation.
}

// FindSpendableOutputs finds UTXOs for a given address that can cover a certain amount.
// Returns the list of spendable UTXOs, their total value, and any error.
// (Conceptual - will iterate utxoSet or use an index)
func (sm *StateManager) FindSpendableOutputs(address types.Address, amountNeeded uint64) ([]types.UTXO, uint64, error) {
	sm.RLock()
	defer sm.RUnlock()

	// var spendableUTXOs []types.UTXO
	// var currentTotal uint64
	// addrHexToMatch := hex.EncodeToString(address)

	// for _, utxo := range sm.utxoSet {
	// 	// This linear scan is inefficient. An index (e.g., map[string][]string AddressHex -> []UtxoKey) would be needed.
	// 	utxoAddrHex := hex.EncodeToString(utxo.RecipientAddress)
	// 	if utxoAddrHex == addrHexToMatch {
	// 		spendableUTXOs = append(spendableUTXOs, utxo)
	// 		currentTotal += utxo.Amount
	// 		if currentTotal >= amountNeeded {
	// 			break
	// 		}
	// 	}
	// }
	// if currentTotal < amountNeeded {
	// 	return nil, 0, fmt.Errorf("insufficient funds: needed %d, found %d for address %s: %w", amountNeeded, currentTotal, addrHexToMatch, internalerrors.ErrInsufficientBalance) // Assuming ErrInsufficientBalance
	// }
	// return spendableUTXOs, currentTotal, nil

	// TODO: Implement actual logic with efficient lookup
	// For now, implementing the iteration approach as per plan.
	// Note: This simple iteration is inefficient. A real implementation would use indexes.
	// It also doesn't guarantee optimal UTXO selection (e.g., smallest number of UTXOs, avoiding dust).

	var spendableUTXOs []types.UTXO
	var currentTotal uint64 = 0

	// Iterate over all UTXOs. This is highly inefficient for a large UTXO set.
	// A real system would have indexes by address.
	for _, utxo := range sm.utxoSet {
		if bytes.Equal(utxo.RecipientAddress, address) {
			spendableUTXOs = append(spendableUTXOs, utxo)
			currentTotal += utxo.Amount
			if currentTotal >= amountNeeded {
				break // Found enough UTXOs
			}
		}
	}

	if currentTotal < amountNeeded {
		// Return the UTXOs found so far, even if not enough, along with the error.
		// The caller might decide to use them or inform the user.
		return spendableUTXOs, currentTotal, fmt.Errorf("insufficient funds for address %s: needed %d, found %d: %w", hex.EncodeToString(address), amountNeeded, currentTotal, internalerrors.ErrInsufficientBalance)
	}

	// Trim to exact or slightly more if strategy is to minimize inputs later
	// For now, return all collected if they meet the amount.
	// A more sophisticated version would select the best fit.
	return spendableUTXOs, currentTotal, nil
}


// TODO: Add methods for:
// - Applying a block (which involves applying transactions)
// - Applying a single transaction (updating UTXO set and account balances)
// - Adding/removing UTXOs
// - Creating/updating accounts
// - Handling state rollbacks (more advanced)
// - Persisting/loading state (e.g., to/from disk using a database like BadgerDB or LevelDB)
```
