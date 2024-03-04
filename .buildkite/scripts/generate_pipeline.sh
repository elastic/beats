#!/usr/bin/env bash
set -euo pipefail

echo "--- Install dependencies"
pip3 install --quiet jinja2
pip3 install --quiet PyYAML

echo "--- Run pipeline generator in dry-run mode"
python3 .buildkite/pipeline.py || true

# Run the python generator - likely this should be called in the
# catalog-info.yaml
echo "--- Upload pipeline"
python3 .buildkite/pipeline.py | buildkite-agent pipeline upload
