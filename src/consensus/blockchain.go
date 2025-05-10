package consensus

import (
	"crypto/ecdsa"
	"errors"
	"sync"

	"github.com/nanlour/da/src/block"
	"github.com/nanlour/da/src/db"
	"github.com/nanlour/da/src/ecdsa_da"
	"github.com/nanlour/da/src/p2p"
	"github.com/nanlour/da/src/rpc"
)

type Account struct {
	PrvKey  ecdsa.PrivateKey
	PubKey  ecdsa.PublicKey
	Address [32]byte
}

type Chain struct {
	Hash    [32]byte
	PrvHash [32]byte
}

type Config struct {
	ID               Account
	StakeMine        float64
	MiningDifficulty uint64
	DbPath           string
	RPCPort          int
	P2PListenAddr    string
	BootstrapPeer    []string
	InitStake        map[[32]byte]float64
	StakeSum         float64
	InitBank         map[[32]byte]float64
}

type BlockChain struct {
	RPCserver  *rpc.RPCServer
	P2PNode    *p2p.Service
	NodeConfig *Config
	MiningChan chan *block.Block  // Channel for newly mined blocks
	P2PChan    chan *p2p.P2PBlock // Channel for blocks received via P2P
	TxnPool    TransactionPool
	mainDB     *db.DBManager
	MyChain    []*Chain
}

var (
	genesisTx = block.Transaction{
		FromAddress: [32]byte{}, // No sender for genesis block
		ToAddress:   [32]byte{}, // No receiver for genesis block
		Amount:      0,          // No amount transferred
	}

	genesisBlock = block.Block{
		PreHash:        [32]byte{},                                            // No previous block
		Height:         0,                                                     // Height is 0
		EpochBeginHash: [32]byte{'H', 'E', 'L', 'L', 'O', ',', ' ', 'D', 'A'}, // Initial epoch hash
		Txn:            genesisTx,
		Signature:      [64]byte{'M', 'A', 'D', 'E', ' ', 'B', 'Y', ' ', 'R', 'O', 'N', 'G', 'W', 'A', 'N', 'G'},
		PublicKey:      [64]byte{},
		Proof:          [516]byte{'T', 'h', 'e', 'r', 'e', ' ', 'i', 's', ' ', 'a', 'l', 'w', 'a', 'y', 's', ' ', 's', 'o', 'm', 'e', 't', 'h', 'i', 'n', 'g', ' ', 't', 'h', 'a', 't', ' ', 'y', 'o', 'u', ' ', 'c', 'a', 'n', 'n', 'o', 't', ' ', 'p', 'r', 'o', 'o', 'f'},
	}
)

func (bc *BlockChain) SetConfig(config *Config) {
	bc.NodeConfig = new(Config)
	*bc.NodeConfig = *config
}

func (bc *BlockChain) Init() error {
	dbmanager, err := db.InitialDB(bc.NodeConfig.DbPath)
	if err != nil {
		return err
	}
	bc.mainDB = dbmanager

	bc.MyChain = []*Chain{
		{
			Hash: genesisBlock.Hash(),
		},
	}

	bc.TxnPool.txnMap = make(map[uint64]*block.Transaction)

	bc.P2PChan = make(chan *p2p.P2PBlock, 100)
	bc.MiningChan = make(chan *block.Block, 10)

	// initila db
	for address, balance := range bc.NodeConfig.InitBank {
		bc.mainDB.InsertAccountBalance(&address, balance)
	}

	gBHash := genesisBlock.Hash()
	bc.mainDB.InsertTipHash(&gBHash)
	bc.mainDB.InsertHashBlock(&gBHash, &genesisBlock)

	bc.RPCserver = rpc.NewRPCServer(bc.NodeConfig.RPCPort)
	bc.RPCserver.Start(bc)

	bc.P2PNode, err = p2p.NewService(bc.NodeConfig.P2PListenAddr, bc)
	if err != nil {
		return err
	}

	for _, addr := range bc.NodeConfig.BootstrapPeer {
		bc.P2PNode.AddBootstrapPeer(addr)
	}
	bc.P2PNode.Start()

	var wg sync.WaitGroup
	wg.Add(2)

	// Start mine
	go func() {
		defer wg.Done()
		bc.mine()
	}()

	go func() {
		defer wg.Done()
		bc.TipManager()
	}()

	wg.Wait()

	return nil
}

func (bc *BlockChain) Stop() error {
	var lastErr error

	// Stop RPC server
	if err := bc.RPCserver.Stop(); err != nil {
		lastErr = err
	}

	// Stop P2P node
	if err := bc.P2PNode.Stop(); err != nil {
		lastErr = err
	}

	// Close the database
	if err := bc.mainDB.Close(); err != nil {
		lastErr = err
	}

	return lastErr
}

func (bc *BlockChain) AddBlock(block *p2p.P2PBlock) error {
	select {
	case bc.P2PChan <- block:
		// Message sent successfully
		return nil
	default:
		// Channel is full or no receiver ready
		return errors.New("channel is full, cannot send block")
	}
}

func (bc *BlockChain) AddTxn(txn *block.Transaction) error {
	bc.TxnPool.AddTransaction(txn.Height, txn)
	return nil
}

func (bc *BlockChain) GetBlockByHash(hash []byte) (*block.Block, error) {
	// Retrieve block from database using hash
	return bc.mainDB.GetHashBlock(hash)
}

func (bc *BlockChain) GetTipBlock() (*block.Block, error) {
	// First get the hash of the tip block
	tipHash, err := bc.mainDB.GetTipHash()
	if err != nil {
		return nil, err
	}

	// Then retrieve the block using the tip hash
	return bc.mainDB.GetHashBlock(tipHash)
}

func (bc *BlockChain) GetAddress() ([32]byte, error) {
	return bc.NodeConfig.ID.Address, nil
}

func (bc *BlockChain) SendTxn(dest [32]byte, amount float64) error {
	tip, _ := bc.GetTipBlock()
	txn := &block.Transaction{
		FromAddress: bc.NodeConfig.ID.Address,
		ToAddress:   dest,
		Amount:      amount,
		Height:      tip.Height + 2,
		PublicKey:   ecdsa_da.PublicKeyToBytes(&bc.NodeConfig.ID.PubKey),
	}

	txn.Sign(&bc.NodeConfig.ID.PrvKey)

	bc.TxnPool.AddTransaction(txn.Height, txn)
	return bc.P2PNode.BroadcastTransaction(txn)
}

func (bc *BlockChain) GetAccountBalance(address *[32]byte) (float64, error) {
	return bc.mainDB.GetAccountBalance(address)
}
