package consensus

import (
	"sync"

	"github.com/nanlour/da/block"
)

type TransactionPool struct {
	txnMap map[uint64]*block.Transaction
	mu     sync.RWMutex
}

func (tp *TransactionPool) AddTransaction(height uint64, tx *block.Transaction) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.txnMap[height] = tx
}

// Get a transaction from the pool
func (tp *TransactionPool) GetTransaction(height uint64) (*block.Transaction, bool) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	tx, exists := tp.txnMap[height]
	return tx, exists
}
