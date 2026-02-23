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

package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type coverageEntry struct {
	position string
	stmt     int
	count    int
}

// AggregateCoverage merges .cov files from a directory into a single report.
// Replaces dev-tools/aggregate_coverage.py.
// Set COV_DIR env var to the input directory and COV_OUT to the output file.
func AggregateCoverage() error {
	outFile := os.Getenv("COV_OUT")
	inputDir := os.Getenv("COV_DIR")
	if outFile == "" || inputDir == "" {
		return fmt.Errorf("COV_OUT and COV_DIR environment variables are required")
	}

	var covFiles []string
	if err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".cov") {
			covFiles = append(covFiles, path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walking input directory %s: %w", inputDir, err)
	}

	outAbs, _ := filepath.Abs(outFile)
	lines := make(map[string]*coverageEntry)

	for _, cf := range covFiles {
		abs, _ := filepath.Abs(cf)
		if abs == outAbs {
			continue
		}
		if err := processCoverageFile(cf, lines); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error processing %s: %v\n", cf, err)
		}
	}

	out, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	fmt.Fprintln(out, "mode: atomic")

	sorted := make([]string, 0, len(lines))
	for _, e := range lines {
		sorted = append(sorted, fmt.Sprintf("%s %d %d", e.position, e.stmt, e.count))
	}
	sort.Strings(sorted)

	for _, line := range sorted {
		fmt.Fprintln(out, line)
	}
	return nil
}

func processCoverageFile(path string, lines map[string]*coverageEntry) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || strings.Contains(line, "vendor") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		position := parts[0]
		stmt, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		if existing, ok := lines[position]; ok {
			existing.count += count
		} else {
			lines[position] = &coverageEntry{position: position, stmt: stmt, count: count}
		}
	}
	return scanner.Err()
}
