#!/usr/bin/env bash
set -euo pipefail

# Install dependencies
pip3 install --quiet jinja2
pip3 install --quiet PyYAML

# Run the python generator - likely this should be called in the
# catalog-info.yaml
python3 .buildkite/pipeline.py | buildkite-agent pipeline upload
