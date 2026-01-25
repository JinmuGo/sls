#!/bin/bash
set -e

# Build artifacts if not already present
if [ ! -d "dist" ]; then
    echo "Building artifacts..."
    goreleaser release --snapshot --clean
fi

echo "Running package publication test in Docker..."

docker run --rm \
    -v $(pwd):/workspace \
    -v $(pwd)/dist:/artifacts \
    -w /workspace \
    ubuntu:latest \
    /workspace/scripts/publish-logic.sh