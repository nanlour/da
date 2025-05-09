package consensus

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
)

// ConfigJSON is a JSON-friendly version of Config
type ConfigJSON struct {
	ID struct {
		PrivateKey string `json:"private_key"` // PEM format
		PublicKey  string `json:"public_key"`  // PEM format
		Address    string `json:"address"`     // Hex encoded
	} `json:"id"`
	StakeMine        float64            `json:"stake_mine"`
	MiningDifficulty uint64             `json:"mining_difficulty"`
	DbPath           string             `json:"db_path"`
	RPCPort          int                `json:"rpc_port"`
	P2PListenAddr    string             `json:"p2p_listen_addr"`
	BootstrapPeer    []string           `json:"bootstrap_peer"`
	InitStake        map[string]float64 `json:"init_stake"` // Hex-encoded address -> stake
	StakeSum         float64            `json:"stake_sum"`
	InitBank         map[string]float64 `json:"init_bank"` // Hex-encoded address -> balance
}

// LoadConfigFromFile loads configuration from a JSON file
func LoadConfigFromFile(filePath string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON into ConfigJSON
	var configJSON ConfigJSON
	if err := json.Unmarshal(data, &configJSON); err != nil {
		return nil, err
	}

	// Convert ConfigJSON to Config
	return configJSON.ToConfig()
}

// ToConfig converts a ConfigJSON to Config
func (cj *ConfigJSON) ToConfig() (*Config, error) {
	config := &Config{
		StakeMine:        cj.StakeMine,
		MiningDifficulty: cj.MiningDifficulty,
		DbPath:           cj.DbPath,
		RPCPort:          cj.RPCPort,
		P2PListenAddr:    cj.P2PListenAddr,
		BootstrapPeer:    cj.BootstrapPeer,
		StakeSum:         cj.StakeSum,
	}

	// Parse ID Account
	var err error
	if err = parseAccountFromJSON(cj, &config.ID); err != nil {
		return nil, err
	}

	// Parse InitStake
	config.InitStake = make(map[[32]byte]float64)
	for addrStr, stake := range cj.InitStake {
		var addrBytes [32]byte
		if addrBytes, err = hexTo32Bytes(addrStr); err != nil {
			return nil, err
		}
		config.InitStake[addrBytes] = stake
	}

	// Parse InitBank
	config.InitBank = make(map[[32]byte]float64)
	for addrStr, balance := range cj.InitBank {
		var addrBytes [32]byte
		if addrBytes, err = hexTo32Bytes(addrStr); err != nil {
			return nil, err
		}
		config.InitBank[addrBytes] = balance
	}

	return config, nil
}

// ToJSON converts a Config to ConfigJSON
func (c *Config) ToJSON() (*ConfigJSON, error) {
	configJSON := &ConfigJSON{
		StakeMine:        c.StakeMine,
		MiningDifficulty: c.MiningDifficulty,
		DbPath:           c.DbPath,
		RPCPort:          c.RPCPort,
		P2PListenAddr:    c.P2PListenAddr,
		BootstrapPeer:    c.BootstrapPeer,
		StakeSum:         c.StakeSum,
	}

	// Convert ID Account
	privateKeyBytes, err := x509.MarshalECPrivateKey(&c.ID.PrvKey)
	if err != nil {
		return nil, err
	}
	configJSON.ID.PrivateKey = string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&c.ID.PubKey)
	if err != nil {
		return nil, err
	}
	configJSON.ID.PublicKey = string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	configJSON.ID.Address = hex.EncodeToString(c.ID.Address[:])

	// Convert InitStake
	configJSON.InitStake = make(map[string]float64)
	for addr, stake := range c.InitStake {
		configJSON.InitStake[hex.EncodeToString(addr[:])] = stake
	}

	// Convert InitBank
	configJSON.InitBank = make(map[string]float64)
	for addr, balance := range c.InitBank {
		configJSON.InitBank[hex.EncodeToString(addr[:])] = balance
	}

	return configJSON, nil
}

// SaveConfigToFile saves the configuration to a JSON file
func (c *Config) SaveToFile(filePath string) error {
	configJSON, err := c.ToJSON()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(configJSON, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// Helper functions
func parseAccountFromJSON(cj *ConfigJSON, account *Account) error {
	// Parse private key from PEM
	block, _ := pem.Decode([]byte(cj.ID.PrivateKey))
	if block == nil {
		return errors.New("failed to parse private key PEM data")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	account.PrvKey = *privateKey

	// Public key can be derived from private key
	account.PubKey = privateKey.PublicKey

	// Parse address
	addrBytes, err := hexTo32Bytes(cj.ID.Address)
	if err != nil {
		return err
	}
	account.Address = addrBytes

	return nil
}

func hexTo32Bytes(hexStr string) ([32]byte, error) {
	var result [32]byte
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return result, err
	}

	if len(bytes) != 32 {
		return result, errors.New("hex string must decode to exactly 32 bytes")
	}

	copy(result[:], bytes)
	return result, nil
}
