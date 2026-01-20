#!/usr/bin/env bash

set -e

# Build the Go project
export CGO_CFLAGS="-Wno-deprecated-declarations"

go build -o camel-pad ./cmd/claude-pad

echo "Build complete: camel-pad"