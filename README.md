# Blockchain Network with Web UI

This repository contains a blockchain network implementation written in Go, along with a web-based user interface for interacting with the blockchain. The project is designed to simulate a multi-node blockchain network with features like staking, mining, and transaction handling.

## Features

- **Blockchain Nodes**: Multiple blockchain nodes can be deployed using Docker Compose.
- **Web UI**: A web-based interface to interact with the blockchain, view balances, send transactions, and more.
- **Staking and Mining**: Nodes can stake tokens and participate in mining.
- **P2P Communication**: Nodes communicate with each other using a peer-to-peer network.
- **RPC Server**: Each node exposes an RPC server for programmatic interaction.

## Consensus Mechanism

This blockchain utilizes a hybrid consensus mechanism that combines principles from Proof of Stake (PoS) with a computational puzzle based on Verifiable Delay Functions (VDFs). The goal is to simulate aspects of Bitcoin's mining process, particularly its characteristic block time distribution, while incorporating stake.

### Verifiable Delay Function (VDF) as a Mining Puzzle

Instead of traditional Proof of Work (PoW) hashing like in Bitcoin, this system uses a VDF. A VDF requires a specific amount of sequential computation time to solve, but the solution (proof) can be verified very quickly. This mimics the "work" in PoW by introducing a verifiable time delay.

### Simulating Bitcoin's Block Time Distribution

Bitcoin's block generation times approximate an exponential distribution, with a target average (e.g., 10 minutes). This project aims to achieve a similar probabilistic block production behavior using VDFs. The core idea is to map a pseudo-random seed, influenced by the miner and block data, to a VDF difficulty, such that the resulting VDF computation times follow an exponential-like distribution.
For more detailed insights into Bitcoin's block time variance, you can read this article: [Bitcoin Block Time Variance](https://blog.lopp.net/bitcoin-block-time-variance/).

#### VDF Difficulty Generation and Execution

The process for a miner to generate a new block involves these steps:

1.  **Seed Generation**: A deterministic seed is created using the hash of the current epoch's beginning block (`EpochBeginHash`) and the `Height` of the new block being mined.
    ```go
    // Example from miner.go
    seed := ecdsa_da.DifficultySeed(&newBlock.EpochBeginHash, newBlock.Height)
    ```

2.  **Signature as a Source of Randomness**: The miner signs this `seed` using their private key. This signature, unique to the miner and the specific block attempt, serves as a crucial input for difficulty calculation.
    ```go
    // Example from miner.go
    signature, err := ecdsa_da.Sign(&bc.NodeConfig.ID.PrvKey, seed[:])
    ```

3.  **Difficulty Calculation**: The VDF `difficulty` (which dictates the VDF's computation time) is then calculated. This calculation uses:
    *   The `signature` obtained above.
    *   The miner's current stake (`StakeMine` or `bc.NodeConfig.StakeMine`).
    *   The total stake in the network (`StakeSum` or `bc.NodeConfig.StakeSum`).
    *   A base mining difficulty parameter (`MiningDifficulty` or `bc.NodeConfig.MiningDifficulty`).
    The function `ecdsa_da.Difficulty()` encapsulates this logic. The design intends that variations in the input `signature` (due to different miners attempting to mine or changes in block height) will produce a range of VDF difficulty values.
    ```go
    // Example from miner.go
    difficulty := ecdsa_da.Difficulty(signature, bc.NodeConfig.StakeSum, bc.NodeConfig.StakeMine, bc.NodeConfig.MiningDifficulty)
    ```
    The influence of `StakeMine` relative to `StakeSum` means that miners with a larger proportion of the total stake will, on average, receive a lower VDF difficulty, making it statistically quicker for them to produce a block, aligning with PoS principles.

4.  **VDF Execution**: The miner computes the VDF proof using the calculated `difficulty` and the hash of the block (excluding the proof itself).
    ```go
    // Example from miner.go
    vdf := vdf_go.New(int(difficulty), newBlock.HashwithoutProof())
    vdf.Execute(stopChan) // stopChan allows mining to be cancelled if a new block is received
    ```
    The time taken for `vdf.Execute()` is what this system aims to model with an exponential distribution, analogous to how Bitcoin's hash-finding time is probabilistic.

### Block Verification

When other nodes receive a new block, they can:
1.  Verify the block's signature.
2.  Independently recalculate the expected VDF `difficulty` using the block's public key (to fetch the sender's stake), the block's signature, and network stake parameters.
3.  Quickly verify the provided VDF `Proof` against the recalculated difficulty and the block's hash.
    ```go
    // Example from stake.go (VerifyBlock)
    // diff := ecdsa_da.Difficulty(...)
    // vdf := vdf_go.New(int(diff), block.HashwithoutProof())
    // return vdf.Verify(block.Proof)
    ```

### Fork Resolution

The blockchain resolves forks by adhering to the longest-chain rule. If a node receives a block that creates a fork, and the new chain (after fetching and verifying its constituent blocks) is longer and valid, the node will switch to this longer chain. This involves rolling back transactions from its old chain segment and applying transactions from the new one.

This consensus model attempts to blend the security aspects of time-based computational work (via VDF) with the incentive structures of Proof of Stake.

## Project Structure

- `main.go`: Entry point for the blockchain node.
- `src/consensus`: Contains the core blockchain logic, including staking, mining, and configuration.
- `src/web`: Implements the web server and static assets for the Web UI.
- `src/cmd/webui/main.go`: Entry point for the Web UI.
- `configs/`: Configuration files for each blockchain node.
- `config_local/`: Local configuration files for testing.
- `scripts/`: Utility scripts for managing the network.
- `Dockerfile`: Dockerfile for building the blockchain node and web UI.
- `docker-compose.yml`: Docker Compose file for deploying the blockchain network.

## Getting Started

### Prerequisites

- Docker
- Docker Compose
- Go (for local development)

### Running the Network

The `docker-compose.yml` file orchestrates the deployment of a three-node blockchain network. Each node runs in a separate Docker container and exposes a web UI.

1.  **Build and Start the Network**:
    To launch the network, execute the following command from the root directory of the project:
    ```bash
    docker-compose up
    ```
    This command will:
    - Build the Docker image for the blockchain nodes (if not already built or if changes are detected).
    - Start three blockchain node containers as defined in `docker-compose.yml`.

2.  **Access the Web UI**:
    Once the containers are up and running, you can interact with each node via its web UI:
    - Node 0: [http://localhost:8080](http://localhost:8080)
    - Node 1: [http://localhost:8081](http://localhost:8081)
    - Node 2: [http://localhost:8082](http://localhost:8082)

### Configuration

Each node's configuration is located in the `configs/` directory. Key parameters include:
- `id`: Contains `private_key`, `public_key`, and `address` for the node.
- `stake_mine`: Amount of stake required for mining.
- `mining_difficulty`: Difficulty target for mining new blocks.
- `db_path`: Path to the node's database (inside the Docker container).
- `rpc_port`: Port for the RPC server.
- `p2p_listen_addr`: Address for P2P communication.
- `bootstrap_peer`: List of peers to connect to at startup.
- `init_stake`: Initial stake distribution among nodes.
- `stake_sum`: Total initial stake in the network.
- `init_bank`: Initial token balances for addresses.

### Scripts

- `scripts/start.sh`: Script executed by Docker containers to start the blockchain node and web UI.
- `connect.sh`: Script to reconnect a specific node (e.g., `da-blockchain-node-1-1`) to the `da_blockchain-net` Docker network.
- `disconnect.sh`: Script to isolate a specific node (e.g., `da-blockchain-node-1-1`) from the `da_blockchain-net` Docker network by connecting it to the default `bridge` network.

### Local Testing

For local testing, use the configurations in the `config_local/` directory. These configurations are set up for a local environment with different database paths (e.g., `/tmp/...`) and P2P listen addresses (e.g., `127.0.0.1`).

## Development

### Building Locally

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Build the blockchain node:
   ```bash
   go build -o blockchain-node ./main.go
   ```

3. Build the web UI:
   ```bash
   go build -o web-ui ./src/cmd/webui/main.go
   ```

### Running Tests

Run the tests using the following command:
```bash
go test ./...
```

## License

This project is licensed under the MIT License. (Assuming MIT, please update if incorrect by adding a LICENSE file).

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any changes or improvements.
