#!/usr/bin/env bash
set -euo pipefail

echo "~~~ Install dependencies"
pip3 install --quiet "ruamel.yaml<0.18.0"

echo "+++ Run pipeline generator in dry-run mode"
python3 .buildkite/pipeline.py | yq .

echo "~~~ Upload pipeline"
python3 .buildkite/pipeline.py | buildkite-agent pipeline upload
