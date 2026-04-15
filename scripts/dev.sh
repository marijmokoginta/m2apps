#!/usr/bin/env bash

set -e

APP_NAME="m2apps"
DEV_NAME="m2apps-dev"
BUILD_DIR="bin"
DAEMON_WAS_RUNNING=0

echo "Building $APP_NAME (dev mode)..."

mkdir -p $BUILD_DIR

go build -o $BUILD_DIR/$DEV_NAME

echo "Setting executable permission..."
chmod +x $BUILD_DIR/$DEV_NAME

echo "Installing to /usr/local/bin/$DEV_NAME ..."

if command -v "$DEV_NAME" >/dev/null 2>&1; then
  echo "Checking daemon service status..."
  STATUS_OUTPUT=$(sudo "$DEV_NAME" daemon status 2>&1 || true)
  if echo "$STATUS_OUTPUT" | grep -Ei "service status: (running|active)" >/dev/null 2>&1; then
    echo "Stopping daemon service to avoid binary lock..."
    sudo "$DEV_NAME" daemon stop || true
    DAEMON_WAS_RUNNING=1
  fi
fi

sudo cp $BUILD_DIR/$DEV_NAME /usr/local/bin/$DEV_NAME

if [ "$DAEMON_WAS_RUNNING" -eq 1 ]; then
  echo "Restarting daemon service..."
  sudo "$DEV_NAME" daemon start || true
fi

echo "Installation completed!"

echo ""
echo "You can now run:"
echo "$DEV_NAME"
