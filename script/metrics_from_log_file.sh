#!/bin/sh
set -e

cat "$@" | \
grep 'Non-zero' | \
sed -e 's/^\([0-9T:\.+\-]*\)[^{]*\(.*\)/{"timestamp": "\1", "data": \2}/' | \
jq '{"timestamp": .timestamp, "monitoring": .data.monitoring}'
