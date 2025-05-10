package consensus

import (
	"bytes"
	"sync"

	"github.com/nanlour/da/src/block"
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
	if tx.Amount == 0 || bytes.Equal(tx.FromAddress[:], tx.ToAddress[:]) {
		return nil
	}

	bfrom, _ := bc.mainDB.GetAccountBalance(&tx.FromAddress)
	if bfrom < tx.Amount {
		return nil
	}
	bto, _ := bc.mainDB.GetAccountBalance(&tx.ToAddress)

	bc.mainDB.InsertAccountBalance(&tx.FromAddress, bfrom-tx.Amount)
	bc.mainDB.InsertAccountBalance(&tx.ToAddress, bto+tx.Amount)

	return nil
}

func (bc *BlockChain) UNDoTxn(tx *block.Transaction) error {
	if tx.Amount == 0 || bytes.Equal(tx.FromAddress[:], tx.ToAddress[:]) {
		return nil
	}

	bfrom, _ := bc.mainDB.GetAccountBalance(&tx.FromAddress)
	if bfrom < tx.Amount {
		return nil
	}
	bto, _ := bc.mainDB.GetAccountBalance(&tx.ToAddress)

	bc.mainDB.InsertAccountBalance(&tx.FromAddress, bfrom+tx.Amount)
	bc.mainDB.InsertAccountBalance(&tx.ToAddress, bto-tx.Amount)

	return nil
}
