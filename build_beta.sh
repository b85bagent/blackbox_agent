#!/bin/bash

# Fetch the current tag and commit hash
Tag=$(git describe --tags 2>/dev/null || echo "v0.0.0")
CommitHash=$(git rev-parse --short HEAD)
CommitDate=$(git log -1 --format=%ai $(git describe --tags 2>/dev/null || echo HEAD))

# Build the Docker image
docker build -f Dockerfile_beta -t tonysystex/blackbox-agent:beta . \
    --build-arg TAG=$Tag \
    --build-arg COMMITHASH=$CommitHash \
    --build-arg COMMITDATE="$CommitDate"
