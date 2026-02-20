#!/bin/bash

# Idony Systemd Service Installer
# This script should be run from the project root: ./scripts/install_service.sh

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Project root is the parent of the script directory
PROJECT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"

SERVICE_NAME="idony.service"
SERVICE_FILE="$PROJECT_DIR/$SERVICE_NAME"
DEST_PATH="/etc/systemd/system/$SERVICE_NAME"

echo "--- Idony Service Installer ---"
echo "Project Directory: $PROJECT_DIR"

if [ ! -f "$SERVICE_FILE" ]; then
    echo "Error: $SERVICE_NAME not found in $PROJECT_DIR"
    exit 1
fi

# 1. Build the server binary
echo "Building idony-server..."
cd "$PROJECT_DIR"
go build -o idony-server ./cmd/idony-server/main.go

# 2. Update the service file with absolute paths
# We use a temporary file to avoid permission issues with sed -i on a potentially linked file
echo "Configuring service file..."
TEMP_SERVICE=$(mktemp)
cp "$SERVICE_FILE" "$TEMP_SERVICE"

sed -i "s|ExecStart=.*|ExecStart=$PROJECT_DIR/idony-server|g" "$TEMP_SERVICE"
sed -i "s|WorkingDirectory=.*|WorkingDirectory=$PROJECT_DIR|g" "$TEMP_SERVICE"
sed -i "s|User=.*|User=$USER|g" "$TEMP_SERVICE"
sed -i "s|Group=.*|Group=$(id -gn)|g" "$TEMP_SERVICE"

# Copy the configured content back to the project service file
cat "$TEMP_SERVICE" > "$SERVICE_FILE"
rm "$TEMP_SERVICE"

# 3. Copy/Link the service file to systemd
echo "Installing to $DEST_PATH..."
if [ -f "$DEST_PATH" ] || [ -L "$DEST_PATH" ]; then
    echo "Existing service found, removing..."
    sudo rm "$DEST_PATH"
fi

sudo ln -s "$SERVICE_FILE" "$DEST_PATH"

# 4. Reload and Start
echo "Reloading systemd and starting service..."
sudo systemctl daemon-reload
sudo systemctl enable idony
sudo systemctl restart idony

echo "--- Installation Complete ---"
sudo systemctl status idony --no-pager
