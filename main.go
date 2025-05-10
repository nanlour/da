package da

import (
	"log"

	"github.com/nanlour/da/src/consensus"
)

func main() {
	bc := consensus.BlockChain{}
	config, err := consensus.LoadConfigFromFile("./config1.json")
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	bc.SetConfig(config)
	bc.Init()
}
