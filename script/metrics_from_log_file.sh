#!/bin/sh
set -e

usage="$(basename "$0") [-h] path_to_beat_log_file

Extracts monitoring metrics from beat log files that are not in structured JSON format.
For log files using structured JSON format, extract metrics with 'grep 'Non-zero' | jq'

where:
    -h  show this help text

Example:
# Extract all metrics, using jq to show changes in the active event count over time.
# The active event count is the number events in the pipeline, including the the queue.
$(basename "$0") ./sample_logs/log_with_metrics.log | jq \"{timestamp: .timestamp, active: .monitoring.metrics.libbeat.pipeline.events.active}\"
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

# The command below uses a sed expression to convert the start of a log line like:
#  2022-06-15T14:20:46.928+0200	INFO	[monitoring]	log/log.go:184	Non-zero metrics in the last 30s	{"monitoring":
# to a JSON line like:
#  {"timestamp": "2022-06-15T14:20:46.928+0200", "data": {"monitoring":
# and then uses jq to restore the desired JSON structure of:
#  {"timestamp": "2022-06-15T14:20:46.928+0200", "monitoring": {...}
cat "$@" | \
grep 'Non-zero' | \
sed -e 's/^\([0-9T:\.+\-]*\)[^{]*\(.*\)/{"timestamp": "\1", "data": \2}/' | \
jq '{"timestamp": .timestamp, "monitoring": .data.monitoring}'
