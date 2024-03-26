#!/usr/bin/env bash
set -euo pipefail

echo "~~~ Install dependencies"
python3 -mpip install --quiet "ruamel.yaml<0.18.0"
# temporary solution until we have this into a base container
curl -fsSL --retry-max-time 60 --retry 3 --retry-delay 5 -o /usr/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
chmod a+x /usr/bin/yq

.buildkite/scripts/run_dynamic_pipeline_tests.sh

echo "+++ Run pipeline generator in dry-run mode"
python3 .buildkite/pipeline.py | yq .

echo "~~~ Upload pipeline"
python3 .buildkite/pipeline.py | buildkite-agent pipeline upload
