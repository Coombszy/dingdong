#!/bin/bash

# Build script for cross-platform compilation
# Builds dingdong for Windows, macOS, and Linux

set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="./dist"

echo "Building dingdong version: $VERSION"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o "$OUTPUT_DIR/dingdong-windows-amd64.exe" -ldflags="-s -w" .

# macOS Intel (amd64)
echo "Building for macOS Intel (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o "$OUTPUT_DIR/dingdong-darwin-amd64" -ldflags="-s -w" .

# macOS Apple Silicon (arm64)
echo "Building for macOS Apple Silicon (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o "$OUTPUT_DIR/dingdong-darwin-arm64" -ldflags="-s -w" .

# Linux (amd64)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/dingdong-linux-amd64" -ldflags="-s -w" .

# Linux (arm64) - bonus for ARM servers
echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o "$OUTPUT_DIR/dingdong-linux-arm64" -ldflags="-s -w" .

echo ""
echo "Build complete! Binaries are in $OUTPUT_DIR:"
ls -lh "$OUTPUT_DIR"
