#!/bin/bash

# Idony Systemd User Service Uninstaller
# This script stops, disables, and removes the idony user-level service.

SERVICE_NAME="idony.service"
USER_DEST_PATH="$HOME/.config/systemd/user/$SERVICE_NAME"
SYSTEM_DEST_PATH="/etc/systemd/system/$SERVICE_NAME"

echo "--- Idony Service Uninstaller ---"

# 1. Handle User Service
if [ -f "$USER_DEST_PATH" ] || [ -L "$USER_DEST_PATH" ]; then
    echo "Stopping idony user service..."
    systemctl --user stop idony || true
    echo "Disabling idony user service..."
    systemctl --user disable idony || true
    echo "Removing $USER_DEST_PATH..."
    rm "$USER_DEST_PATH"
fi

# 2. Handle leftover System Service (if any)
if [ -f "$SYSTEM_DEST_PATH" ] || [ -L "$SYSTEM_DEST_PATH" ]; then
    echo "Cleaning up legacy system service (requires sudo)..."
    sudo systemctl stop idony || true
    sudo systemctl disable idony || true
    sudo rm "$SYSTEM_DEST_PATH"
fi

echo "Reloading systemd daemons..."
systemctl --user daemon-reload
sudo systemctl daemon-reload || true

echo "--- Uninstallation Complete ---"
