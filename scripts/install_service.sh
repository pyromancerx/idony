#!/bin/bash

# Idony Systemd User Service Installer
# This script installs Idony as a user-level service (no sudo required for management).

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"

SERVICE_NAME="idony.service"
SERVICE_FILE="$PROJECT_DIR/$SERVICE_NAME"
USER_SYSTEMD_DIR="$HOME/.config/systemd/user"
DEST_PATH="$USER_SYSTEMD_DIR/$SERVICE_NAME"

echo "--- Idony User Service Installer ---"
echo "Project Directory: $PROJECT_DIR"

mkdir -p "$USER_SYSTEMD_DIR"

if [ ! -f "$SERVICE_FILE" ]; then
    echo "Error: $SERVICE_NAME not found in $PROJECT_DIR"
    exit 1
fi

# 1. Build the server binary
echo "Building idony-server..."
cd "$PROJECT_DIR"
go build -o idony-server ./cmd/idony-server/main.go

# 2. Update the service file with absolute paths
# For User services, we remove User/Group lines as they are implied.
echo "Configuring service file for User Mode..."
TEMP_SERVICE=$(mktemp)
cp "$SERVICE_FILE" "$TEMP_SERVICE"

sed -i "s|ExecStart=.*|ExecStart=$PROJECT_DIR/idony-server|g" "$TEMP_SERVICE"
sed -i "s|WorkingDirectory=.*|WorkingDirectory=$PROJECT_DIR|g" "$TEMP_SERVICE"
# Remove lines starting with User= or Group=
sed -i "/^User=/d" "$TEMP_SERVICE"
sed -i "/^Group=/d" "$TEMP_SERVICE"

# Copy the configured content back to the project service file
cat "$TEMP_SERVICE" > "$SERVICE_FILE"
rm "$TEMP_SERVICE"

# 3. Link the service file to user systemd directory
echo "Installing to $DEST_PATH..."
if [ -f "$DEST_PATH" ] || [ -L "$DEST_PATH" ]; then
    rm "$DEST_PATH"
fi

ln -s "$SERVICE_FILE" "$DEST_PATH"

# 4. Reload and Start (User Mode)
echo "Reloading user systemd and starting service..."
systemctl --user daemon-reload
systemctl --user enable idony
systemctl --user restart idony

# 5. Optional: Enable lingering so it starts at boot without login
echo "Enabling lingering for $USER (ensures service runs at boot)..."
sudo loginctl enable-linger "$USER"

echo "--- Installation Complete ---"
echo "You can now manage Idony with: systemctl --user [start|stop|restart|status] idony"
systemctl --user status idony --no-pager
