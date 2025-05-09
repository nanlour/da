package consensus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigConversion(t *testing.T) {
	// Create a test config
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	var address [32]byte
	copy(address[:], []byte("test-address-12345678901234567890"))

	var address2 [32]byte
	copy(address2[:], []byte("test-address-22222222222222222222"))

	config := &Config{
		ID: Account{
			PrvKey:  *privateKey,
			PubKey:  privateKey.PublicKey,
			Address: address,
		},
		StakeMine:        1.5,
		MiningDifficulty: 10,
		DbPath:           "/test/path",
		RPCPort:          8000,
		P2PListenAddr:    "localhost:9000",
		BootstrapPeer:    []string{"peer1:9001", "peer2:9002"},
		InitStake: map[[32]byte]float64{
			address:  100.0,
			address2: 200.0,
		},
		StakeSum: 300.0,
		InitBank: map[[32]byte]float64{
			address:  1000.0,
			address2: 2000.0,
		},
	}

	// Convert to JSON and back
	configJSON, err := config.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert Config to ConfigJSON: %v", err)
	}

	newConfig, err := configJSON.ToConfig()
	if err != nil {
		t.Fatalf("Failed to convert ConfigJSON to Config: %v", err)
	}

	// Verify that the values are preserved
	if newConfig.StakeMine != config.StakeMine {
		t.Errorf("StakeMine doesn't match: got %v, want %v", newConfig.StakeMine, config.StakeMine)
	}

	if newConfig.MiningDifficulty != config.MiningDifficulty {
		t.Errorf("MiningDifficulty doesn't match: got %v, want %v", newConfig.MiningDifficulty, config.MiningDifficulty)
	}

	if newConfig.DbPath != config.DbPath {
		t.Errorf("DbPath doesn't match: got %v, want %v", newConfig.DbPath, config.DbPath)
	}

	if newConfig.RPCPort != config.RPCPort {
		t.Errorf("RPCPort doesn't match: got %v, want %v", newConfig.RPCPort, config.RPCPort)
	}

	if newConfig.P2PListenAddr != config.P2PListenAddr {
		t.Errorf("P2PListenAddr doesn't match: got %v, want %v", newConfig.P2PListenAddr, config.P2PListenAddr)
	}

	if !reflect.DeepEqual(newConfig.BootstrapPeer, config.BootstrapPeer) {
		t.Errorf("BootstrapPeer doesn't match: got %v, want %v", newConfig.BootstrapPeer, config.BootstrapPeer)
	}

	if newConfig.StakeSum != config.StakeSum {
		t.Errorf("StakeSum doesn't match: got %v, want %v", newConfig.StakeSum, config.StakeSum)
	}

	// Check that InitStake and InitBank were correctly converted
	for addr, stake := range config.InitStake {
		if newConfig.InitStake[addr] != stake {
			t.Errorf("InitStake for address %v doesn't match: got %v, want %v", addr, newConfig.InitStake[addr], stake)
		}
	}

	for addr, balance := range config.InitBank {
		if newConfig.InitBank[addr] != balance {
			t.Errorf("InitBank for address %v doesn't match: got %v, want %v", addr, newConfig.InitBank[addr], balance)
		}
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a test config similar to the one in TestConfigConversion
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	var address [32]byte
	copy(address[:], []byte("test-address-12345678901234567890"))

	config := &Config{
		ID: Account{
			PrvKey:  *privateKey,
			PubKey:  privateKey.PublicKey,
			Address: address,
		},
		StakeMine:        2.5,
		MiningDifficulty: 12,
		DbPath:           "/test/db/path",
		RPCPort:          8080,
		P2PListenAddr:    "localhost:9090",
		BootstrapPeer:    []string{"peer3:9003", "peer4:9004"},
		InitStake: map[[32]byte]float64{
			address: 150.0,
		},
		StakeSum: 150.0,
		InitBank: map[[32]byte]float64{
			address: 1500.0,
		},
	}

	// Create a temporary file for testing
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // clean up after the test

	configPath := filepath.Join(tempDir, "config.json")

	// Save the config to file
	if err := config.SaveToFile(configPath); err != nil {
		t.Fatalf("Failed to save config to file: %v", err)
	}

	// Load the config from file
	loadedConfig, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	// Verify that the values are preserved
	if loadedConfig.StakeMine != config.StakeMine {
		t.Errorf("StakeMine doesn't match: got %v, want %v", loadedConfig.StakeMine, config.StakeMine)
	}

	if loadedConfig.MiningDifficulty != config.MiningDifficulty {
		t.Errorf("MiningDifficulty doesn't match: got %v, want %v", loadedConfig.MiningDifficulty, config.MiningDifficulty)
	}

	if loadedConfig.DbPath != config.DbPath {
		t.Errorf("DbPath doesn't match: got %v, want %v", loadedConfig.DbPath, config.DbPath)
	}

	if loadedConfig.RPCPort != config.RPCPort {
		t.Errorf("RPCPort doesn't match: got %v, want %v", loadedConfig.RPCPort, config.RPCPort)
	}

	if loadedConfig.P2PListenAddr != config.P2PListenAddr {
		t.Errorf("P2PListenAddr doesn't match: got %v, want %v", loadedConfig.P2PListenAddr, config.P2PListenAddr)
	}

	if !reflect.DeepEqual(loadedConfig.BootstrapPeer, config.BootstrapPeer) {
		t.Errorf("BootstrapPeer doesn't match: got %v, want %v", loadedConfig.BootstrapPeer, config.BootstrapPeer)
	}

	if loadedConfig.StakeSum != config.StakeSum {
		t.Errorf("StakeSum doesn't match: got %v, want %v", loadedConfig.StakeSum, config.StakeSum)
	}

	// Check InitStake and InitBank values
	for addr, stake := range config.InitStake {
		if loadedConfig.InitStake[addr] != stake {
			t.Errorf("InitStake doesn't match: got %v, want %v", loadedConfig.InitStake[addr], stake)
		}
	}

	for addr, balance := range config.InitBank {
		if loadedConfig.InitBank[addr] != balance {
			t.Errorf("InitBank doesn't match: got %v, want %v", loadedConfig.InitBank[addr], balance)
		}
	}
}
