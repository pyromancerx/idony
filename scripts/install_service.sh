#!/bin/bash

# Idony Systemd Service Installer
# This script must be run with sudo or appropriate permissions for /etc/systemd/system

set -e

PROJECT_DIR=$(pwd)
SERVICE_NAME="idony.service"
DEST_PATH="/etc/systemd/system/$SERVICE_NAME"

echo "--- Idony Service Installer ---"

# 1. Build the server binary
echo "Building idony-server..."
go build -o idony-server ./cmd/idony-server/main.go

# 2. Update the service file with absolute paths
echo "Configuring service file..."
sed -i "s|ExecStart=.*|ExecStart=$PROJECT_DIR/idony-server|g" idony.service
sed -i "s|WorkingDirectory=.*|WorkingDirectory=$PROJECT_DIR|g" idony.service
sed -i "s|User=.*|User=$USER|g" idony.service
sed -i "s|Group=.*|Group=$(id -gn)|g" idony.service

# 3. Copy/Link the service file
echo "Installing to $DEST_PATH..."
if [ -f "$DEST_PATH" ] || [ -L "$DEST_PATH" ]; then
    echo "Existing service found, removing..."
    sudo rm "$DEST_PATH"
fi

sudo ln -s "$PROJECT_DIR/idony.service" "$DEST_PATH"

# 4. Reload and Start
echo "Reloading systemd and starting service..."
sudo systemctl daemon-reload
sudo systemctl enable idony
sudo systemctl restart idony

echo "--- Installation Complete ---"
sudo systemctl status idony --no-pager
