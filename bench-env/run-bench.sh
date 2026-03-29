#!/usr/bin/env bash
set -euo pipefail

# run-bench.sh — Build filebeat, start mock-es, run a timed benchmark with
# profiling inside a CPU-limited Docker container.
#
# Usage:
#   ./bench-env/run-bench.sh [DURATION] [LABEL] [PIPELINE]
#
# Arguments:
#   DURATION   Seconds to run (default: 30)
#   LABEL      Results label (default: git short SHA)
#   PIPELINE   Pipeline config: worst-case, mid-case, best-case (default: worst-case)
#
# Examples:
#   ./bench-env/run-bench.sh 30 main worst-case
#   ./bench-env/run-bench.sh 30 pr-49761 best-case
#
# Outputs saved to bench-env/results/<label>-<pipeline>/

DURATION=${1:-30}
LABEL=${2:-$(git rev-parse --short HEAD)}
PIPELINE=${3:-worst-case}
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BENCH_DIR="$REPO_ROOT/bench-env"
RESULTS="$BENCH_DIR/results/${LABEL}-${PIPELINE}"
PIPELINE_CFG="$BENCH_DIR/pipelines/${PIPELINE}.yml"
FILEBEAT_BIN="$BENCH_DIR/build/filebeat"
MOCK_ES_BIN="$BENCH_DIR/build/mock-es"
CPUS="1.5"  # Docker --cpus limit

if [ ! -f "$PIPELINE_CFG" ]; then
  echo "ERROR: Pipeline config not found: $PIPELINE_CFG"
  echo "Available: $(ls "$BENCH_DIR/pipelines/"*.yml 2>/dev/null | xargs -I{} basename {} .yml | tr '\n' ' ')"
  exit 1
fi

mkdir -p "$RESULTS" "$BENCH_DIR/build"

echo "=== Filebeat Benchmark ==="
echo "Label:      $LABEL"
echo "Pipeline:   $PIPELINE"
echo "Duration:   ${DURATION}s"
echo "CPU limit:  ${CPUS} cores (Docker)"
echo ""

# --- Build filebeat (linux/arm64 for Docker) ---
echo "Building filebeat for linux..."
cd "$REPO_ROOT/x-pack/filebeat"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOFLAGS=-mod=mod go build -o "$FILEBEAT_BIN" . 2>&1
echo "Building mock-es for linux..."
cd "$BENCH_DIR/mock-es"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$MOCK_ES_BIN" . 2>&1
echo "Build complete."

# --- Clean up stale containers ---
docker rm -f filebeat-bench mock-es-bench 2>/dev/null || true

# --- Create a shared Docker network ---
docker network create bench-net 2>/dev/null || true

# --- Start mock-es ---
echo ""
echo "Starting mock-es..."
docker run -d --rm \
  --name mock-es-bench \
  --network bench-net \
  -v "$MOCK_ES_BIN:/mock-es:ro" \
  --platform linux/arm64 \
  alpine:3.20 /mock-es :9200

# Wait for mock-es
for i in $(seq 1 10); do
  if docker exec mock-es-bench wget -qO- http://localhost:9200/ >/dev/null 2>&1; then
    echo "mock-es ready."
    break
  fi
  sleep 0.5
done

# --- Prepare filebeat config (rewrite ES host to Docker network name) ---
BENCH_CFG="/tmp/filebeat-bench-cfg.yml"
sed 's|localhost:9200|mock-es-bench:9200|g' "$PIPELINE_CFG" > "$BENCH_CFG"
# Also rewrite pprof to bind 0.0.0.0 so we can reach it from host
sed -i.bak 's|http.host: "localhost"|http.host: "0.0.0.0"|g' "$BENCH_CFG"
rm -f "$BENCH_CFG.bak"

# --- Start filebeat in Docker with CPU limit ---
echo ""
echo "Starting filebeat (${DURATION}s, --cpus=$CPUS)..."
docker run -d --rm \
  --name filebeat-bench \
  --network bench-net \
  --cpus="$CPUS" \
  -p 5066:5066 \
  -v "$FILEBEAT_BIN:/filebeat:ro" \
  -v "$BENCH_CFG:/filebeat.yml:ro" \
  --platform linux/arm64 \
  alpine:3.20 /filebeat run -c /filebeat.yml --path.data /tmp/fb-data --path.home /tmp/fb-home

PPROF="http://localhost:5066"

# Wait for pprof
echo "Waiting for pprof endpoint..."
for i in $(seq 1 30); do
  if curl -s "$PPROF/debug/pprof/" >/dev/null 2>&1; then
    echo "pprof ready."
    break
  fi
  if ! docker ps --format '{{.Names}}' | grep -q filebeat-bench; then
    echo "ERROR: filebeat exited early."
    docker logs filebeat-bench 2>&1 | tail -20
    docker rm -f mock-es-bench 2>/dev/null
    exit 1
  fi
  sleep 1
done

# --- Grab initial event count ---
EVENTS_START=$(curl -s "$PPROF/stats" 2>/dev/null | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get('libbeat',{}).get('pipeline',{}).get('events',{}).get('total',0))
except:
    print(0)
" 2>/dev/null || echo 0)

# --- Collect CPU profile for full duration ---
echo "Collecting ${DURATION}s CPU profile..."
curl -s "$PPROF/debug/pprof/profile?seconds=$DURATION" -o "$RESULTS/cpu.pprof" &
CURL_PID=$!

sleep "$DURATION"
wait "$CURL_PID" 2>/dev/null || true

# --- Grab heap + allocs profiles ---
echo "Collecting heap and allocs profiles..."
curl -s "$PPROF/debug/pprof/heap" -o "$RESULTS/heap.pprof"
curl -s "$PPROF/debug/pprof/allocs" -o "$RESULTS/allocs.pprof"

# --- Grab final metrics ---
curl -s "$PPROF/stats" > "$RESULTS/metrics.json" 2>/dev/null || true
EVENTS_END=$(python3 -c "
import json
d = json.load(open('$RESULTS/metrics.json'))
print(d.get('libbeat',{}).get('pipeline',{}).get('events',{}).get('total',0))
" 2>/dev/null || echo 0)

# --- Mock-es doc count ---
MOCK_DOCS=$(docker exec mock-es-bench wget -qO- http://localhost:9200/_mock/stats 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('docs_ingested',0))" 2>/dev/null || echo "N/A")

# --- Stop containers ---
echo "Stopping containers..."
docker rm -f filebeat-bench mock-es-bench 2>/dev/null || true

# --- Calculate results ---
EVENTS=$((EVENTS_END - EVENTS_START))
if [ "$EVENTS" -le 0 ]; then EVENTS=0; fi
EPS=$((EVENTS / DURATION))

# --- Total allocs from pprof ---
TOTAL_ALLOCS=$(go tool pprof -text "$RESULTS/allocs.pprof" 2>/dev/null | head -5 | grep 'of.*total' | grep -oE '[0-9.]+[A-Z]+' | head -1 || echo "N/A")

# --- Summary ---
cat > "$RESULTS/summary.txt" <<SUMMARY
Filebeat Benchmark Results
==========================
Label:            $LABEL
Pipeline:         $PIPELINE
Duration:         ${DURATION}s
CPU limit:        ${CPUS} cores (Docker)
GOMAXPROCS:       default

Events published: $EVENTS
Events/sec:       $EPS
Mock-ES docs:     $MOCK_DOCS
Total allocs:     $TOTAL_ALLOCS

Profiles:
  CPU:    $RESULTS/cpu.pprof
  Heap:   $RESULTS/heap.pprof
  Allocs: $RESULTS/allocs.pprof
SUMMARY

echo ""
echo "=== Results ==="
cat "$RESULTS/summary.txt"
echo ""

echo "=== Top Allocations ==="
go tool pprof -top -cum "$RESULTS/allocs.pprof" 2>/dev/null | head -25
echo ""
echo "Results saved to: $RESULTS/"
