#!/bin/bash
set -euo pipefail

echo "Building E4 API..."

./scripts/prepare-dist.sh

echo "Building Go binary..."
go build -ldflags="-s -w" -o e4-api main.go

echo "Build complete: e4-api"
echo "Run with: ./e4-api"
