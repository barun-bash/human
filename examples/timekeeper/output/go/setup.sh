#!/bin/bash
# Setup script for the generated Go backend
# Run this once after code generation to download dependencies

set -e

echo "Downloading Go dependencies..."
go mod tidy
echo "Building..."
go build ./...
echo "Setup complete!"
