#!/usr/bin/env bash
set -euo pipefail

echo "--- Install dependencies"
pip3 install --quiet PyYAML

echo "--- Run pipeline generator in dry-run mode"
python3 .buildkite/pipeline.py | yq .

# Run the python generator - likely this should be called in the
# catalog-info.yaml
# echo "--- Upload pipeline"
## Remove when is ready
# python3 .buildkite/pipeline.py | buildkite-agent pipeline upload
