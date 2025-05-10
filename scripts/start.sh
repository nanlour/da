#!/bin/bash

# Navigate to the directory containing the blockchain node and web UI
cd /app

# Start the blockchain node
./blockchain-node -config ${NODE_CONFIG} > nodelog 2>&1 &

# Wait for blockchain node to initialize
echo "Waiting 5 seconds for blockchain node to initialize..."
sleep 5

# Start the web UI
./web-ui -rpc localhost:9000 -basedir web -port 8080 > weblog 2>&1 &

tail -f /dev/null