#!/usr/bin/env bash
set -euo pipefail

# compare.sh — Run benchmarks on two branches across all pipeline scenarios.
#
# Usage:
#   ./bench-env/compare.sh [DURATION] [BASE_REF] [PR_REF]
#
# Defaults: 30s, upstream/main, current branch

DURATION=${1:-30}
BASE_REF=${2:-upstream/main}
PR_REF=${3:-$(git branch --show-current)}
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BENCH_DIR="$REPO_ROOT/bench-env"
PIPELINES=(worst-case mid-case best-case)

echo "=========================================="
echo "  Filebeat Benchmark Comparison"
echo "=========================================="
echo "Base:       $BASE_REF"
echo "PR:         $PR_REF"
echo "Duration:   ${DURATION}s per scenario"
echo "Scenarios:  ${PIPELINES[*]}"
echo "CPU limit:  1.5 cores (Docker)"
echo ""

# Stash if needed
ORIGINAL=$(git branch --show-current 2>/dev/null || git rev-parse HEAD)
STASHED=false
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
  git stash push -m "bench-compare auto-stash"
  STASHED=true
fi
cleanup() {
  git checkout "$ORIGINAL" 2>/dev/null || true
  [ "$STASHED" = true ] && git stash pop 2>/dev/null || true
  docker rm -f filebeat-bench mock-es-bench 2>/dev/null || true
}
trap cleanup EXIT

# --- Run all scenarios on base ---
echo ""
echo "====== BASE: $BASE_REF ======"
git checkout "$BASE_REF" 2>&1 | tail -1
for p in "${PIPELINES[@]}"; do
  echo ""
  echo "--- Scenario: $p ---"
  "$BENCH_DIR/run-bench.sh" "$DURATION" "base" "$p"
done

# --- Run all scenarios on PR ---
echo ""
echo "====== PR: $PR_REF ======"
git checkout "$PR_REF" 2>&1 | tail -1
for p in "${PIPELINES[@]}"; do
  echo ""
  echo "--- Scenario: $p ---"
  "$BENCH_DIR/run-bench.sh" "$DURATION" "pr" "$p"
done

# --- Comparison table ---
echo ""
echo "============================================"
echo "  RESULTS COMPARISON"
echo "============================================"
echo ""
printf "%-14s  %10s  %10s  %8s  |  %12s  %12s  %8s\n" \
  "Scenario" "Base EPS" "PR EPS" "Δ EPS" "Base Allocs" "PR Allocs" "Δ Allocs"
printf "%-14s  %10s  %10s  %8s  |  %12s  %12s  %8s\n" \
  "-----------" "--------" "------" "-----" "-----------" "---------" "--------"

for p in "${PIPELINES[@]}"; do
  BASE_S="$BENCH_DIR/results/base-${p}/summary.txt"
  PR_S="$BENCH_DIR/results/pr-${p}/summary.txt"

  BASE_EPS=$(grep "^Events/sec:" "$BASE_S" 2>/dev/null | awk '{print $2}' || echo 0)
  PR_EPS=$(grep "^Events/sec:" "$PR_S" 2>/dev/null | awk '{print $2}' || echo 0)
  BASE_ALLOCS=$(grep "^Total allocs:" "$BASE_S" 2>/dev/null | awk '{print $3}' || echo "?")
  PR_ALLOCS=$(grep "^Total allocs:" "$PR_S" 2>/dev/null | awk '{print $3}' || echo "?")

  if [ "$BASE_EPS" -gt 0 ] 2>/dev/null; then
    EPS_D=$(python3 -c "print(f'{($PR_EPS - $BASE_EPS) / $BASE_EPS * 100:+.1f}%')" 2>/dev/null || echo "?")
  else
    EPS_D="?"
  fi

  # Try to compute allocs delta from pprof
  ALLOCS_D=$(python3 -c "
import re, subprocess, sys
def get_gb(path):
    out = subprocess.check_output(['go', 'tool', 'pprof', '-text', path], stderr=subprocess.DEVNULL, text=True)
    for line in out.split('\n')[:5]:
        m = re.search(r'([\d.]+)GB', line)
        if m: return float(m.group(1))
    return 0
base = get_gb('$BENCH_DIR/results/base-${p}/allocs.pprof')
pr = get_gb('$BENCH_DIR/results/pr-${p}/allocs.pprof')
if base > 0:
    print(f'{(pr - base) / base * 100:+.1f}%')
else:
    print('?')
" 2>/dev/null || echo "?")

  printf "%-14s  %10s  %10s  %8s  |  %12s  %12s  %8s\n" \
    "$p" "$BASE_EPS" "$PR_EPS" "$EPS_D" "$BASE_ALLOCS" "$PR_ALLOCS" "$ALLOCS_D"
done

echo ""
echo "For detailed profile comparison:"
for p in "${PIPELINES[@]}"; do
  echo "  go tool pprof -diff_base=$BENCH_DIR/results/base-${p}/allocs.pprof $BENCH_DIR/results/pr-${p}/allocs.pprof"
done
echo ""
echo "For flamegraph visualization:"
for p in "${PIPELINES[@]}"; do
  echo "  go tool pprof -http=:8080 $BENCH_DIR/results/pr-${p}/cpu.pprof"
done
