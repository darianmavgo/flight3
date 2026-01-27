#!/bin/bash

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
RESOURCES_DIR="$DIR/../Resources"

# Set up data directory in user's Application Support
DATA_DIR="$HOME/Library/Application Support/Flight3"
mkdir -p "$DATA_DIR"/{pb_data,logs}

# Export environment variables
export FLIGHT3_DATA_DIR="$DATA_DIR/pb_data"
export FLIGHT3_LOG_DIR="$DATA_DIR/logs"
export FLIGHT3_PUBLIC_DIR="$RESOURCES_DIR/pb_public"

# Launch Flight3
cd "$DATA_DIR"
exec "$DIR/flight3" serve --http=127.0.0.1:8090
