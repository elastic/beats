#!/bin/sh
set -e

usage="$(basename "$0") [-h] path_to_beat_log_file

Extracts the monitoring metrics from a Beat log files as JSON, regardless
of whether the Beat log file is in structured JSON format already.

where:
    -h  show this help text

Examples using reference log files:
  $(basename "$0") ./testdata/log_with_metrics.json

  # Extract all metrics, using jq to show changes in the active event count over time.
  # The active event count is the number events in the pipeline, including the the queue.
  $(basename "$0") ./testdata/log_with_metrics.log | jq \"{timestamp: .timestamp, active: .monitoring.metrics.libbeat.pipeline.events.active}\"
"

while getopts ':h' option; do
  case "$option" in
    h) echo "$usage"
       exit 0;;
    *) echo "Unknown option"
       exit 1;;
  esac
done
shift $((OPTIND - 1))

if [ $# -eq 0 ]; then
    echo "No log file to parse."
    exit 0
fi

cat "$@" | \
grep 'Non-zero' | \
sed -e 's/^\([0-9T:\.+\-]*\)[^{]*\(.*\)/{"timestamp": "\1", "data": \2}/' | \
jq '{"timestamp": .timestamp, "monitoring": .data.monitoring}'
