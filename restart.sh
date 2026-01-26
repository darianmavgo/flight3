#!/bin/bash

# Ports we check for the PocketBase instance
PORTS=(8090 8091 8092)

echo "Stopping any running PocketBase instances..."

for PORT in "${PORTS[@]}"; do
    PID=$(lsof -t -i:$PORT)
    if [ ! -z "$PID" ]; then
        echo "Found process $PID on port $PORT. Killing..."
        kill -9 $PID
    fi
done

# Clean up any stale go run binaries
pkill -f "cmd/flight/main.go" 2>/dev/null

echo "Restarting PocketBase..."
mkdir -p logs
echo "Logs redirected to logs/flight_output.txt"

# Note: Using 'go run' for development. 
# You can change this to use a built binary for faster restarts.
go run ./cmd/flight/main.go serve > logs/flight_output.txt 2>&1
