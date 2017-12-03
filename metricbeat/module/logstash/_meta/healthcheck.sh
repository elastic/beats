#!/usr/bin/env bash

# Check that both endpoints are available
curl -f http://localhost:9600/_node || exit 1
curl -f http://localhost:9600/_node/stats || exit 1
curl -f http://localhost:9600/_node/stats | grep '"in":' || exit 1
