package consensus

import (
	"crypto/ecdsa"
	"errors"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/p2p"
	"github.com/nanlour/da/rpc"
	"github.com/nanlour/da/util"
)

type Account struct {
	PrvKey  ecdsa.PrivateKey
	PubKey  ecdsa.PublicKey
	Address [32]byte
}

type Config struct {
	ID               Account
	StakeMine        float64
	MiningDifficulty uint64
	DbPath           string
	RPCPort          int
	ListenAddr       string
	InitStake        map[[32]byte]float64
	StakeSum         float64
	InitBank         map[[32]byte]float64
}

type BlockChain struct {
	RPCserver  *rpc.RPCServer
	P2PNode    *p2p.Service
	NodeConfig *Config
	TxnPool    map[uint64]*block.Transaction // Txn pool
	MiningChan chan *block.Block             // Channel for newly mined blocks
	P2PChan    chan *block.Block             // Channel for blocks received via P2P
}

var (
	genesisTx = block.Transaction{
		FromAddress: [32]byte{}, // No sender for genesis block
		ToAddress:   [32]byte{}, // No receiver for genesis block
		Amount:      0,          // No amount transferred
	}

	genesisBlock = &block.Block{
		PreHash:        [32]byte{},                                            // No previous block
		Height:         0,                                                     // Height is 0
		EpochBeginHash: [32]byte{'H', 'E', 'L', 'L', 'O', ',', ' ', 'D', 'A'}, // Initial epoch hash
		Txn:            genesisTx,
		Signature:      [64]byte{'M', 'A', 'D', 'E', ' ', 'B', 'Y', ' ', 'R', 'O', 'N', 'G', 'W', 'A', 'N', 'G'},
		PublicKey:      [64]byte{},
		Proof:          [516]byte{'T', 'h', 'e', 'r', 'e', ' ', 'i', 's', ' ', 'a', 'l', 'w', 'a', 'y', 's', ' ', 's', 'o', 'm', 'e', ' ', 't', 'h', 'a', 't', ' ', 'y', 'o', 'u', ' ', 'c', 'a', 'n', 'n', 'o', 't', ' ', 'p', 'r', 'o', 'o', 'f'},
	}
)

func (bc *BlockChain) SetConfig(config *Config) {
	bc.NodeConfig = new(Config)
	*bc.NodeConfig = *config
}

func (bc *BlockChain) Init() error {
	err := util.InitialDB(bc.NodeConfig.DbPath)
	if err != nil {
		return err
	}

	bc.RPCserver = rpc.NewRPCServer(bc.NodeConfig.RPCPort)
	bc.RPCserver.Start()

	bc.P2PNode, err = p2p.NewService(bc.NodeConfig.ListenAddr, bc)
	if err != nil {
		return err
	}

	bc.P2PChan = make(chan *block.Block, 100)
	bc.MiningChan = make(chan *block.Block, 10)

	// Start mine
	go bc.mine()

	return nil
}

func (bc *BlockChain) Stop() error {
	var lastErr error

	// Close the database
	if err := util.MainDB.Close(); err != nil {
		lastErr = err
	}

	// Stop RPC server
	if err := bc.RPCserver.Stop(); err != nil {
		lastErr = err
	}

	// Stop P2P node
	if err := bc.P2PNode.Stop(); err != nil {
		lastErr = err
	}

	return lastErr
}

func (bc *BlockChain) AddBlock(block *block.Block) error {
	select {
	case bc.P2PChan <- block:
		// Message sent successfully
		return nil
	default:
		// Channel is full or no receiver ready
		return errors.New("channel is full, cannot send block")
	}
}

func (bc *BlockChain) GetBlockByHash(hash []byte) (*block.Block, error) {
	// Retrieve block from database using hash
	return util.MainDB.GetHashBlock(hash)
}

func (bc *BlockChain) GetTipBlock() (*block.Block, error) {
	// First get the hash of the tip block
	tipHash, err := util.MainDB.GetTipHash()
	if err != nil {
		return nil, err
	}

	// Then retrieve the block using the tip hash
	return util.MainDB.GetHashBlock(tipHash)
}
