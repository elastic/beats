#!/usr/bin/env bash

# Check that all endpoints are available
curl -f http://localhost:5066 || exit 1
curl -f http://localhost:5066/stats || exit 1
curl -f http://localhost:5066/state || exit 1
