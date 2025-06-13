#!/bin/bash

# Set the version to download
FILEBEAT_VERSION="8.17.4"  # Change this to the version you need

# Detect OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture to Elastic format
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="x86_64"
elif [[ "$ARCH" == "arm64" ]]; then
    ARCH="aarch64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi


# # Construct the download URL
# FILEBEAT_URL="https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-${FILEBEAT_VERSION}-${OS}-${ARCH}.tar.gz"

# echo "Downloading Filebeat from: $FILEBEAT_URL"
# curl -L -o filebeat.tar.gz "$FILEBEAT_URL"

# Extract the package
echo "Extracting Filebeat..."
# tar -xzf filebeat.tar.gz

# Run Filebeat with default config (modify as needed)
echo "Starting Filebeat... with version $FILEBEAT_VERSION"
# Start Filebeat in separate terminals
# Start first Filebeat instance in a new terminal window
cd "filebeat-${FILEBEAT_VERSION}-${OS}-${ARCH}/" || exit 1
FILEBEAT_PATH="$(pwd)"

osascript -e "tell application \"Terminal\" to do script \"cd '$FILEBEAT_PATH' && ./filebeat &;  echo \$! > /tmp/filebeat1.pid\""

cd "../../"
echo "Starting local Filebeat"
FILEBEAT_PATH="$(pwd)"
osascript -e "tell application \"Terminal\" to do script \"cd '$FILEBEAT_PATH' && ./filebeat &; echo \$! > /tmp/filebeat2.pid\""

sleep 60


if [ -s /tmp/filebeat1.pid ] && [ -s /tmp/filebeat2.pid ]; then
    PID1=$(cat /tmp/filebeat1.pid)
    PID2=$(cat /tmp/filebeat2.pid)

    echo "Terminating Filebeat instances with PIDs: $PID1, $PID2"
    
    # Check if the PIDs exist before killing
    if ps -p "$PID1" > /dev/null; then kill "$PID1"; else echo "PID $PID1 not found"; fi
    if ps -p "$PID2" > /dev/null; then kill "$PID2"; else echo "PID $PID2 not found"; fi

    # Clean up PID files
    rm -f /tmp/filebeat1.pid /tmp/filebeat2.pid
else
    echo "One or both PID files are empty. Filebeat may not have started correctly."
fi