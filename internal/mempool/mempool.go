package mempool

import (
	"sync"
	"fmt" // For error messages
    "encoding/hex" // For using tx ID as map key

	"empower1.com/empower1blockchain/internal/core"
    // "log" // For internal logging if needed, not strictly required for placeholder
)

var (
    ErrTxExists = fmt.Errorf("transaction already exists in mempool")
    // TODO: Add more mempool specific errors like ErrMempoolFull, ErrTxTooLarge, ErrTxInvalid
)

// Mempool holds transactions that are waiting to be included in a block.
// For V1, this is a simple in-memory map.
type Mempool struct {
	mu           sync.RWMutex
	transactions map[string]*core.Transaction // Keyed by hex string of Transaction ID
    // TODO: Add eviction policies, max size, sorting by fee, etc.
}

// NewMempool creates and returns a new Mempool instance.
func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*core.Transaction),
	}
}

// AddTransaction attempts to add a transaction to the mempool.
// For V1, it's a simple add if not already present.
// TODO: Add validation logic (e.g., check signatures, basic structure, non-conflicting inputs with other mempool txs).
func (mp *Mempool) AddTransaction(tx *core.Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if tx == nil || tx.ID == nil {
		return fmt.Errorf("cannot add nil transaction or transaction with nil ID")
	}

    txIDHex := hex.EncodeToString(tx.ID)
	if _, exists := mp.transactions[txIDHex]; exists {
		return fmt.Errorf("%w: %s", ErrTxExists, txIDHex)
	}

	mp.transactions[txIDHex] = tx
	// log.Printf("MEMPOOL: Added transaction %s", txIDHex) // If internal logging desired
	return nil
}

// GetTransactions retrieves a list of transactions from the mempool, up to a given limit.
// For V1, it returns transactions in unspecified order.
// TODO: Implement selection logic (e.g., highest fee, oldest).
func (mp *Mempool) GetTransactions(limit int) []*core.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if limit <= 0 || limit > len(mp.transactions) {
		limit = len(mp.transactions)
	}

	txs := make([]*core.Transaction, 0, limit)
	count := 0
	for _, tx := range mp.transactions {
		if count >= limit {
			break
		}
		txs = append(txs, tx)
		count++
	}
	return txs
}

// RemoveTransaction removes a transaction from the mempool, typically after it's included in a block.
func (mp *Mempool) RemoveTransaction(txID []byte) {
    mp.mu.Lock()
    defer mp.mu.Unlock()
    txIDHex := hex.EncodeToString(txID)
    // log.Printf("MEMPOOL: Removing transaction %s", txIDHex) // If internal logging desired
    delete(mp.transactions, txIDHex)
}

// Count returns the number of transactions currently in the mempool.
func (mp *Mempool) Count() int {
    mp.mu.RLock()
    defer mp.mu.RUnlock()
    return len(mp.transactions)
}
