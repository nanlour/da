package main

import (
	"flag"
	"log"

	"github.com/nanlour/da/src/consensus"
)

func main() {
	// Define command-line flag for config path
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	bc := consensus.BlockChain{}
	config, err := consensus.LoadConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	bc.SetConfig(config)
	bc.Init()
}
