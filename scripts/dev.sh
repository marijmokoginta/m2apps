#!/usr/bin/env bash

set -e

APP_NAME="m2apps"
DEV_NAME="m2apps-dev"
BUILD_DIR="bin"

echo "Building $APP_NAME (dev mode)..."

mkdir -p $BUILD_DIR

go build -o $BUILD_DIR/$DEV_NAME

echo "Setting executable permission..."
chmod +x $BUILD_DIR/$DEV_NAME

echo "Installing to /usr/local/bin/$DEV_NAME ..."

sudo cp $BUILD_DIR/$DEV_NAME /usr/local/bin/$DEV_NAME

echo "Installation completed!"

echo ""
echo "You can now run:"
echo "$DEV_NAME"