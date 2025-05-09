package consensus

import (
	"sync"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/db"
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

func (bc *BlockChain) DoTxn(tx *block.Transaction) error {
	if tx.Amount == 0 {
		return nil
	}

	bfrom, _ := db.MainDB.GetAccountBalance(&tx.FromAddress)
	if bfrom < tx.Amount {
		return nil
	}
	bto, _ := db.MainDB.GetAccountBalance(&tx.ToAddress)

	db.MainDB.InsertAccountBalance(&tx.FromAddress, bfrom-tx.Amount)
	db.MainDB.InsertAccountBalance(&tx.ToAddress, bto+tx.Amount)

	return nil
}

func (bc *BlockChain) UNDoTxn(tx *block.Transaction) error {
	if tx.Amount == 0 {
		return nil
	}

	bfrom, _ := db.MainDB.GetAccountBalance(&tx.FromAddress)
	if bfrom < tx.Amount {
		return nil
	}
	bto, _ := db.MainDB.GetAccountBalance(&tx.ToAddress)

	db.MainDB.InsertAccountBalance(&tx.FromAddress, bfrom+tx.Amount)
	db.MainDB.InsertAccountBalance(&tx.ToAddress, bto-tx.Amount)

	return nil
}
