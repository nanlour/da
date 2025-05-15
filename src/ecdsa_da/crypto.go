package ecdsa_da

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

// GenerateKeyPair creates a new ECDSA keypair
func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// BytesToPublicKey converts a serialized 64-byte public key back to an ecdsa.PublicKey
func BytesToPublicKey(pubKeyBytes [64]byte) (*ecdsa.PublicKey, error) {
	// Extract X and Y coordinates (32 bytes each)
	x := new(big.Int).SetBytes(pubKeyBytes[:32])
	y := new(big.Int).SetBytes(pubKeyBytes[32:])

	// Create and validate the public key with crypto/ecdh package
	// First, encode the point in uncompressed form (0x04 + X + Y)
	ecdhEncoded := make([]byte, 65)
	ecdhEncoded[0] = 0x04 // Uncompressed point format
	copy(ecdhEncoded[1:33], pubKeyBytes[:32])
	copy(ecdhEncoded[33:65], pubKeyBytes[32:])

	// Use ecdh.P256().NewPublicKey which performs on-curve checks
	_, err := ecdh.P256().NewPublicKey(ecdhEncoded)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	// Create ECDSA public key
	pub := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	return pub, nil
}

// PublicKeyToBytes serializes an ecdsa.PublicKey to a 64-byte array
func PublicKeyToBytes(pubKey *ecdsa.PublicKey) [64]byte {
	var pubKeyBytes [64]byte

	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Pad X to 32 bytes and copy
	copy(pubKeyBytes[32-len(xBytes):32], xBytes)
	// Pad Y to 32 bytes and copy
	copy(pubKeyBytes[64-len(yBytes):], yBytes)

	return pubKeyBytes
}

// PublicKeyToBytes serializes an ecdsa.PublicKey to a 64-byte array
func PublicKeyToAddress(pubKey *ecdsa.PublicKey) [32]byte {
	pubKeyBytes := PublicKeyToBytes(pubKey)
	return sha256.Sum256(pubKeyBytes[:])
}

// Sign creates a digital signature of the provided message using the private key
func Sign(privateKey *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	// Hash the message
	hash := sha256.Sum256(message)

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	// Create signature by concatenating r and s
	// Each value gets 32 bytes (P256 curve parameters)
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad r to 32 bytes and copy
	copy(signature[32-len(rBytes):32], rBytes)
	// Pad s to 32 bytes and copy
	copy(signature[64-len(sBytes):], sBytes)

	return signature, nil
}

// Verify checks if the provided signature is valid for the message using the public key
func Verify(publicKey *ecdsa.PublicKey, message []byte, signature []byte) bool {
	// Validate signature length
	if len(signature) != 64 {
		return false
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Extract r and s from signature
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	// Verify the signature
	return ecdsa.Verify(publicKey, hash[:], r, s)
}

// difficulty(Mid creates a combined hash of epoch hash and block height
func DifficultySeed(epohHash *[32]byte, height uint64) [32]byte {
	// Convert height to bytes
	heightBytes := make([]byte, 8)

	// Convert uint64 height to big-endian byte representation
	heightBytes[0] = byte(height >> 56)
	heightBytes[1] = byte(height >> 48)
	heightBytes[2] = byte(height >> 40)
	heightBytes[3] = byte(height >> 32)
	heightBytes[4] = byte(height >> 24)
	heightBytes[5] = byte(height >> 16)
	heightBytes[6] = byte(height >> 8)
	heightBytes[7] = byte(height)

	// Combine epoch hash and height bytes
	combined := make([]byte, 32+8)
	copy(combined[:32], epohHash[:])
	copy(combined[32:], heightBytes)

	// Hash the combined data
	return sha256.Sum256(combined)
}

// Difficulty maps a signature to a Difficulty, evenly distributed way
func Difficulty(signature []byte, StakeSum float64, StakeMine float64, MiningDifficulty uint64) uint64 {
	// Hash the signature to ensure uniform distribution
	signatureHash := sha256.Sum256(signature)

	// Convert first 8 bytes of hash to uint64
	value := uint64(0)
	for i := range 8 {
		value = (value << 8) | uint64(signatureHash[i])
	}

	// Divide by maximum uint64 value to get a float64 between 0 and 1
	rm := math.Log(float64(value) / float64(^uint64(0)))
	t := math.Log(1 - float64(StakeMine/(StakeSum*float64(MiningDifficulty))))

	diff := uint64(rm / t)

	// Ensure diff is smaller than 10 * MiningDifficulty
	maxDiff := uint64(float64(MiningDifficulty) * (10 * StakeSum / StakeMine))
	if diff > maxDiff {
		diff = maxDiff
	}

	return 100 + diff
}
