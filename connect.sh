# Connect node-1-1 from other nodes
docker network disconnect bridge da-blockchain-node-1-1
docker network connect da_blockchain-net da-blockchain-node-1-1 