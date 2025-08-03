#!/bin/bash

# Build script for kubegraph with Git version information

set -e

# Get Git information
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

echo "Building kubegraph..."
echo "Git commit: $GIT_COMMIT"
echo "Git branch: $GIT_BRANCH"

# Build with version information
go build -ldflags "-X 'kubegraph/pkg/version.GitCommit=$GIT_COMMIT' -X 'kubegraph/pkg/version.GitBranch=$GIT_BRANCH'" -o kubegraph .

echo "Build complete!"
echo "Run './kubegraph --help' for usage information" 
