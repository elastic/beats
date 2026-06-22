// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// This file was contributed to by generative AI

package filestream

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/elastic-agent-libs/logp"
)

// The benchmarks below model the broad-glob / large-exclude scenarios from
// https://github.com/elastic/beats/issues/48686. They build a deep directory
// tree and exercise GetFiles with different match/exclude ratios.
//
// This file is self-contained on purpose: to benchmark a baseline, copy it
// unchanged into the same package on another checkout (e.g. the merge-base of
// this branch) and run the same commands there.

const benchTreeFanout, benchTreeDepth = 5, 5

// benchEnvInt reads an integer knob for the benchmarks from the environment,
// falling back to def when unset.
func benchEnvInt(tb testing.TB, name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	require.NoErrorf(tb, err, "invalid %s=%q", name, v)
	return n
}

// benchTreeFileCount is the number of files the scanner benchmarks build. It
// defaults to a CI-friendly size and can be raised for local runs targeting the
// ~300k-file scenario from the issue via BENCH_TREE_FILES, e.g.
// BENCH_TREE_FILES=300000 go test -bench BenchmarkGetFiles -run '^$' ./...
func benchTreeFileCount(tb testing.TB) int {
	return benchEnvInt(tb, "BENCH_TREE_FILES", 50_000)
}

// buildBenchDirTree creates a directory tree of the given fanout and depth under
// root and returns every directory created (root included). Files are later spread
// across these directories so they sit at a range of depths.
func buildBenchDirTree(tb testing.TB, root string, fanout, depth int) []string {
	tb.Helper()
	require.NoError(tb, os.MkdirAll(root, 0o770))
	dirs := []string{root}
	var rec func(base string, d int)
	rec = func(base string, d int) {
		if d == 0 {
			return
		}
		for i := 0; i < fanout; i++ {
			child := filepath.Join(base, fmt.Sprintf("d%d", i))
			require.NoError(tb, os.MkdirAll(child, 0o770))
			dirs = append(dirs, child)
			rec(child, d-1)
		}
	}
	rec(root, depth)
	return dirs
}

// buildBenchTree creates total non-empty files spread round-robin across a nested
// directory tree under root. name maps a file's global index to its basename, so
// each benchmark controls which files match the include glob and the excludes.
func buildBenchTree(tb testing.TB, root string, total int, name func(i int) string) {
	tb.Helper()
	dirs := buildBenchDirTree(tb, root, benchTreeFanout, benchTreeDepth)
	for i := 0; i < total; i++ {
		p := filepath.Join(dirs[i%len(dirs)], name(i))
		require.NoError(tb, os.WriteFile(p, []byte("x"), 0o660))
	}
}

// benchExcludePatterns returns ~33 exclude_files regexes modelled on a real broad
// configuration (issue #48686): a mix of alternation, character classes, escaped
// literals and date-prefix patterns. None of them match the "doc-*" candidate
// files, so every candidate is tested against the whole list (worst case).
func benchExcludePatterns() []match.Matcher {
	raw := []string{
		`/skip/.*`,
		`/skip/sub/.*`,
		`app-svc-.*`,
		`console-svc-.*`,
		`error-svc-.*`,
		`job_export_batch-.*`,
		`job_gather_batch-.*`,
		`request-service-.*`,
		`/trace/tr\.\d+\..+\.log`,
		`/trace/tr\.\d{4,6}\..+\.log`,
		`/.*/catalina\.out-.*`,
		`/.*/server\.out-.*`,
		`riskmgmt\.TRC\..*`,
		`.*\.tmpobj`,
		`.*\.cfs`,
		`.*\.(gz|zip|dat|tmp|trc|bin|war|obi)`,
		`/raw/sys/.*`,
		`/raw/sys/messages`,
		`/raw/sys/journal/.*`,
		`/raw/sys/audit/.*`,
		`/raw/sys/kubernetes/audit/.*`,
		`/raw/minio/auditlog`,
		`/raw/minio/supplementauditlog`,
		`/raw/miniolog`,
		`/raw/apilog`,
		`/raw/batcatalinaout`,
		`/raw/batapplog`,
		`/svc/server\.log\.2026.*`,
		`/svc/wrapper\.log\.[0-9].*`,
		`/io/server\.log\.2026.*`,
		`/ui/server\.log\.2026.*`,
		`/batch/server\.log\.2026.*`,
		`/bpm/server\.log\.2026.*`,
		`/full/_full_\.log\.2026.*`,
	}
	ms := make([]match.Matcher, len(raw))
	for i, p := range raw {
		ms[i] = match.MustCompile(p)
	}
	return ms
}

// BenchmarkGetFilesSelective: a large tree where the include glob matches only a
// handful of files (most files have a non-matching extension). Stresses directory
// traversal — the scanner must walk everything but collects few.
func BenchmarkGetFilesSelective(b *testing.B) {
	const matched = 100
	base := filepath.Join(b.TempDir(), "a")
	total := benchTreeFileCount(b)
	buildBenchTree(b, base, total, func(i int) string {
		if i < matched {
			return fmt.Sprintf("match-%d.json", i)
		}
		return fmt.Sprintf("file-%d.log", i)
	})

	cfg := fileScannerConfig{
		RecursiveGlob: true,
		Fingerprint:   fingerprintConfig{Enabled: false},
	}
	s, err := newFileScanner(logp.NewNopLogger(), []string{filepath.Join(base, "**", "*.json")}, cfg, CompressionNone)
	require.NoError(b, err)
	got, _, _ := s.GetFiles(loginp.FileScanOptions{})
	require.Len(b, got, matched)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetFiles(loginp.FileScanOptions{})
	}
}

// BenchmarkGetFilesExcludeMost: the issue's exact case — the include glob matches
// every file, but exclude_files drops almost all of them. The previous
// implementation still materialised the full match list before excluding.
func BenchmarkGetFilesExcludeMost(b *testing.B) {
	const kept = 100
	base := filepath.Join(b.TempDir(), "a")
	total := benchTreeFileCount(b)
	buildBenchTree(b, base, total, func(i int) string {
		if i < kept {
			return fmt.Sprintf("keep-%d.json", i)
		}
		return fmt.Sprintf("excl-%d.json", i)
	})

	cfg := fileScannerConfig{
		ExcludedFiles: []match.Matcher{match.MustCompile("excl-")},
		RecursiveGlob: true,
		Fingerprint:   fingerprintConfig{Enabled: false},
	}
	s, err := newFileScanner(logp.NewNopLogger(), []string{filepath.Join(base, "**", "*.json")}, cfg, CompressionNone)
	require.NoError(b, err)
	got, _, _ := s.GetFiles(loginp.FileScanOptions{})
	require.Len(b, got, kept)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetFiles(loginp.FileScanOptions{})
	}
}

// BenchmarkGetFilesManyPatterns: every file matches the include globs and none are
// excluded, but each candidate is tested against ~33 exclude regexes. The two
// include patterns share the same base dir. Verifies the regex-list cost is not
// regressed by the optimisation.
func BenchmarkGetFilesManyPatterns(b *testing.B) {
	base := filepath.Join(b.TempDir(), "a")
	total := benchTreeFileCount(b)
	buildBenchTree(b, base, total, func(i int) string {
		if i%10 == 0 {
			return fmt.Sprintf("doc-%d.ndjson", i)
		}
		return fmt.Sprintf("doc-%d.json", i)
	})

	cfg := fileScannerConfig{
		ExcludedFiles: benchExcludePatterns(),
		RecursiveGlob: true,
		Fingerprint:   fingerprintConfig{Enabled: false},
	}
	paths := []string{
		filepath.Join(base, "**", "*.json"),
		filepath.Join(base, "**", "*.ndjson"),
	}
	s, err := newFileScanner(logp.NewNopLogger(), paths, cfg, CompressionNone)
	require.NoError(b, err)
	got, _, _ := s.GetFiles(loginp.FileScanOptions{})
	require.Len(b, got, total)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetFiles(loginp.FileScanOptions{})
	}
}

// BenchmarkGetFilesLiteralMidComponent: a glob with a literal component after the
// first wildcard (base/*/app/*.log) over a tree where most second-level subtrees
// cannot match. filepath.Glob only opened the literal "app" child; the walker must
// prune the non-matching siblings instead of reading them to the pattern depth.
//
// Like the other benchmarks, the file count scales with BENCH_TREE_FILES. The
// breadth is fixed (topDirs hosts, each with one matching "app" dir and
// siblingDirs prunable "other-*" dirs), and the files are spread evenly across
// every leaf dir, so 1/(1+siblingDirs) of them match and the rest live in
// subtrees the walker must prune.
func BenchmarkGetFilesLiteralMidComponent(b *testing.B) {
	base := filepath.Join(b.TempDir(), "logs")
	const topDirs, siblingDirs = 20, 5

	perDir := benchTreeFileCount(b) / (topDirs * (1 + siblingDirs))
	if perDir < 1 {
		perDir = 1
	}
	writeN := func(tb testing.TB, dir, prefix string) {
		require.NoError(tb, os.MkdirAll(dir, 0o770))
		for k := 0; k < perDir; k++ {
			require.NoError(tb, os.WriteFile(
				filepath.Join(dir, fmt.Sprintf("%s-%d.log", prefix, k)), []byte("x"), 0o660))
		}
	}
	for i := 0; i < topDirs; i++ {
		host := filepath.Join(base, fmt.Sprintf("host-%d", i))
		writeN(b, filepath.Join(host, "app"), "f")
		for j := 0; j < siblingDirs; j++ {
			writeN(b, filepath.Join(host, fmt.Sprintf("other-%d", j)), "g")
		}
	}

	cfg := fileScannerConfig{Fingerprint: fingerprintConfig{Enabled: false}}
	s, err := newFileScanner(logp.NewNopLogger(),
		[]string{filepath.Join(base, "*", "app", "*.log")}, cfg, CompressionNone)
	require.NoError(b, err)
	got, _, _ := s.GetFiles(loginp.FileScanOptions{})
	require.Len(b, got, topDirs*perDir)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetFiles(loginp.FileScanOptions{})
	}
}

// BenchmarkGetFilesMixed models a realistic broad configuration: several include
// globs — two recursive ("**") plus one with a literal component after a wildcard
// ("hosts/*/app/*.log") — combined with the ~33 exclude_files regexes. It exercises
// the whole pipeline at once: multiple walk groups, component pruning of the
// non-"app" subtrees, the "**" descent, exclusion actually dropping some matches,
// and the per-candidate regex-list cost.
func BenchmarkGetFilesMixed(b *testing.B) {
	root := b.TempDir()
	total := benchTreeFileCount(b)

	// ~half the budget under a **-glob subtree; every file matches *.json/*.ndjson.
	structured := filepath.Join(root, "structured")
	structuredFiles := total / 2
	buildBenchTree(b, structured, structuredFiles, func(i int) string {
		if i%10 == 0 {
			return fmt.Sprintf("doc-%d.ndjson", i)
		}
		return fmt.Sprintf("doc-%d.json", i)
	})

	// A handful of matched-but-excluded files under a /skip/ directory (dropped by
	// the "/skip/.*" exclude), so exclusion actually removes candidates rather than
	// only paying the regex-list cost.
	const excluded = 100
	skipDir := filepath.Join(structured, "skip")
	require.NoError(b, os.MkdirAll(skipDir, 0o770))
	for i := 0; i < excluded; i++ {
		require.NoError(b, os.WriteFile(
			filepath.Join(skipDir, fmt.Sprintf("drop-%d.json", i)), []byte("x"), 0o660))
	}

	// The rest under host/{app,other-*} dirs; only the "app" files match
	// hosts/*/app/*.log, the sibling dirs must be pruned.
	hosts := filepath.Join(root, "hosts")
	const hostCount, siblingDirs = 20, 5
	perDir := (total - structuredFiles) / (hostCount * (1 + siblingDirs))
	if perDir < 1 {
		perDir = 1
	}
	writeN := func(dir, prefix string) {
		require.NoError(b, os.MkdirAll(dir, 0o770))
		for k := 0; k < perDir; k++ {
			require.NoError(b, os.WriteFile(
				filepath.Join(dir, fmt.Sprintf("%s-%d.log", prefix, k)), []byte("x"), 0o660))
		}
	}
	for i := 0; i < hostCount; i++ {
		host := filepath.Join(hosts, fmt.Sprintf("host-%d", i))
		writeN(filepath.Join(host, "app"), "f")
		for j := 0; j < siblingDirs; j++ {
			writeN(filepath.Join(host, fmt.Sprintf("other-%d", j)), "g")
		}
	}

	cfg := fileScannerConfig{
		ExcludedFiles: benchExcludePatterns(),
		RecursiveGlob: true,
		Fingerprint:   fingerprintConfig{Enabled: false},
	}
	paths := []string{
		filepath.Join(structured, "**", "*.json"),
		filepath.Join(structured, "**", "*.ndjson"),
		filepath.Join(hosts, "*", "app", "*.log"),
	}
	s, err := newFileScanner(logp.NewNopLogger(), paths, cfg, CompressionNone)
	require.NoError(b, err)
	matched := structuredFiles + hostCount*perDir // skip/ files excluded, sibling dirs pruned
	got, _, _ := s.GetFiles(loginp.FileScanOptions{})
	require.Len(b, got, matched)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetFiles(loginp.FileScanOptions{})
	}
}
