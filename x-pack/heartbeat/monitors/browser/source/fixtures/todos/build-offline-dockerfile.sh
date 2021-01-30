#!/bin/sh
# This demonstrates building and tagging a custom offline docker heartbeat image with synthetics
# with all dependencies pre-bundled.

# You'll want to run this in an environment with internet access so that NPM deps can be installed,
# then, take the resultant image and transfer that to your air gapped network.
docker build --build-arg STACK_VERSION=7.10.0 -t my-custom-heartbeat .
