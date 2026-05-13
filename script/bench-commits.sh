#!/usr/bin/env bash
# Benchmark each commit between <base> and HEAD, then run benchstat across
# adjacent commits and base-vs-tip.
#
# With no arguments, defaults are tuned for the harvester-logger-clones PR:
# benchmark all commits between upstream/main..HEAD using BenchmarkFilestream
# in ./filebeat/input/filestream/ with count=10. Because the first commit on
# this branch only changes the benchmark harness (no production change), it
# is a faithful proxy for upstream/main and benchstat against it measures
# the impact of the production commits that follow.
#
# Usage:
#   ./script/bench-commits.sh [base] [pattern] [package] [count] [benchtime]
set -euo pipefail

base="${1:-upstream/main}"
pattern="${2:-BenchmarkFilestream}"
pkg="${3:-./filebeat/input/filestream/}"
count="${4:-10}"
benchtime="${5:-1s}"

outdir="bench-results"
mkdir -p "$outdir"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "error: working tree has uncommitted changes; commit or stash first" >&2
  exit 1
fi

if ! command -v benchstat >/dev/null; then
  echo "error: benchstat not found in PATH (install: go install golang.org/x/perf/cmd/benchstat@latest)" >&2
  exit 1
fi

original_ref=$(git symbolic-ref --quiet --short HEAD || git rev-parse HEAD)
trap 'git checkout --quiet "$original_ref"' EXIT INT TERM

merge_base=$(git merge-base "$base" HEAD)
commits=$(git rev-list --reverse "${merge_base}..HEAD")
if [[ -z "$commits" ]]; then
  echo "no commits between $base and HEAD" >&2
  exit 1
fi

n=$(echo "$commits" | wc -l)
echo "benchmarking $n commits between $base ($merge_base) and HEAD"
echo "  pattern=$pattern package=$pkg count=$count benchtime=$benchtime"
echo "  results -> $outdir/"

# Pin GOMAXPROCS so the benchmark name suffix (-N) is stable across commits.
export GOMAXPROCS="${GOMAXPROCS:-$(nproc)}"

for commit in $commits; do
  short=$(git rev-parse --short "$commit")
  subject=$(git log -1 --format='%s' "$commit")
  outfile="${outdir}/${short}.txt"
  metafile="${outdir}/${short}.meta"

  if [[ -f "$outfile" && "${FORCE:-0}" != "1" ]]; then
    echo "==> skip $short  ($subject) -- exists, FORCE=1 to redo"
    continue
  fi

  echo "==> $short  $subject"
  git checkout --quiet --detach "$commit"

  # Metadata in a sibling file so it doesn't confuse benchstat's config parser.
  {
    echo "commit: $commit"
    echo "subject: $subject"
  } > "$metafile"

  # Warm caches: a no-bench run pulls deps and compiles the binary so
  # `go: downloading` doesn't pollute the output file; a single discarded
  # bench iteration warms the filesystem cache and CPU caches.
  go test -run='^$' -bench='^$' "$pkg" >/dev/null 2>&1 || true
  go test -run='^$' -bench="$pattern" -benchtime=1x -count=1 \
      -timeout=10m "$pkg" >/dev/null 2>&1 || true

  if go test -run='^$' -bench="$pattern" -benchmem \
      -benchtime="$benchtime" -count="$count" \
      -timeout=30m "$pkg" > "$outfile" 2>&1; then
    tail -5 "$outfile"
  else
    echo "    FAILED (see $outfile)" >&2
  fi
done

echo
echo "=== benchstat: adjacent commits ==="
prev=""
for commit in $commits; do
  short=$(git rev-parse --short "$commit")
  if [[ -n "$prev" ]]; then
    echo
    echo "--- $prev -> $short ---"
    benchstat "$outdir/$prev.txt" "$outdir/$short.txt" || true
  fi
  prev="$short"
done

first=$(git rev-parse --short "$(echo "$commits" | head -1)")
last=$(git rev-parse --short "$(echo "$commits" | tail -1)")
if [[ "$first" != "$last" ]]; then
  echo
  echo "=== benchstat: $first (base) -> $last (tip) ==="
  benchstat "$outdir/$first.txt" "$outdir/$last.txt" || true
fi
