# Isolate node-1-1 from other nodes
docker network disconnect da_blockchain-net da-blockchain-node-1-1 
docker network connect bridge da-blockchain-node-1-1