package consensus

import (
	"crypto/sha256"

	"github.com/nanlour/da/src/block"
	"github.com/nanlour/da/src/ecdsa_da"
	"github.com/nanlour/da/src/vdf_go"
)

func (bc *BlockChain) VerifyBlock(block *block.Block) bool {
	seed := ecdsa_da.DifficultySeed(&block.EpochBeginHash, block.Height)
	publicKey, err := ecdsa_da.BytesToPublicKey(block.PublicKey)
	if err != nil {
		return false
	}

	// Check epoch begin hash
	if block.EpochBeginHash != genesisBlock.Hash() {
		return false
	}

	// Check transaction height matches block height
	if block.Txn.Height != block.Height {
		return false
	}

	// Verify transaction
	if !block.Txn.Verify() {
		return false
	}

	// Verify signature
	if !ecdsa_da.Verify(publicKey, seed[:], block.Signature[:]) {
		return false
	}

	diff := ecdsa_da.Difficulty(block.Signature[:], bc.NodeConfig.StakeSum, bc.NodeConfig.InitStake[sha256.Sum256(block.PublicKey[:])], bc.NodeConfig.MiningDifficulty)

	vdf := vdf_go.New(int(diff), block.HashwithoutProof())

	var zeroProof [516]byte
	if block.Proof == zeroProof {
		return false
	}

	return vdf.Verify(block.Proof)
}
