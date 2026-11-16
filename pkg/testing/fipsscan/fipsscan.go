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

//go:build requirefips

// Package fipsscan provides helpers for auditing FIPS 140-3 compliance of Go
// binaries by scanning their dependency trees for forbidden (non-FIPS) imports.
// Only crypto/* stdlib packages are covered by Go's certified FIPS 140-3 module
// (GOFIPS140=v1.0.0); other crypto implementations are potential violations.
package fipsscan

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// XCrypto is the import path prefix for golang.org/x/crypto, which is not
// covered by Go's certified FIPS 140-3 module. Use as the forbiddenPkgs entry
// for standard FIPS compliance checks.
const XCrypto = "golang.org/x/crypto/"

// Violation is a forbidden import discovered in the dependency tree.
type Violation struct {
	Importer string // package that imports the forbidden package
	Imported string // forbidden package being imported
}

type goListPackage struct {
	ImportPath string
	Imports    []string
}

// Scan runs `go list -json -deps -tags requirefips <pkg>` and returns all
// packages that directly import golang.org/x/crypto or any prefix in
// extraForbiddenPkgs, along with the full import graph. Packages whose own
// path matches a forbidden prefix are skipped (avoids flagging internal refs).
// Calls t.Fatalf on subprocess or parse errors.
func Scan(t testing.TB, pkg string, extraForbiddenPkgs []string) ([]Violation, map[string][]string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "go", "list", "-json", "-deps", "-tags", "requirefips", pkg)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("go list failed: %v\n%s", err, exitErr.Stderr)
		}
		t.Fatalf("go list failed: %v", err)
	}

	forbidden := append([]string{XCrypto}, extraForbiddenPkgs...)

	importGraph := make(map[string][]string)
	var violations []Violation

	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p goListPackage
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("parsing go list output: %v", err)
		}
		importGraph[p.ImportPath] = p.Imports

		if isForbidden(p.ImportPath, forbidden) {
			continue
		}
		for _, imp := range p.Imports {
			if isForbidden(imp, forbidden) {
				violations = append(violations, Violation{
					Importer: p.ImportPath,
					Imported: imp,
				})
			}
		}
	}

	return violations, importGraph
}

func isForbidden(pkg string, forbiddenPkgs []string) bool {
	for _, prefix := range forbiddenPkgs {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}
	return false
}

// ShortestChain returns the shortest path from `from` to `to` using forward
// edges in importGraph (package -> packages it imports). Returns nil if no path.
func ShortestChain(from, to string, importGraph map[string][]string) []string {
	if from == to {
		return []string{from}
	}

	type state struct {
		pkg  string
		path []string
	}

	visited := make(map[string]bool)
	visited[from] = true
	queue := []state{{pkg: from, path: []string{from}}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, next := range importGraph[cur.pkg] {
			if visited[next] {
				continue
			}
			newPath := make([]string, len(cur.path)+1)
			copy(newPath, cur.path)
			newPath[len(cur.path)] = next

			if next == to {
				return newPath
			}

			visited[next] = true
			queue = append(queue, state{pkg: next, path: newPath})
		}
	}

	return nil
}

// FormatChain joins a chain with indented " -> " separators for readable output:
//
//	pkg/a
//	      -> pkg/b
//	      -> pkg/c
func FormatChain(chain []string) string {
	if len(chain) == 0 {
		return ""
	}
	parts := make([]string, len(chain))
	parts[0] = chain[0]
	for i := 1; i < len(chain); i++ {
		parts[i] = "      -> " + chain[i]
	}
	return strings.Join(parts, "\n")
}

// CheckViolations scans binaryPkg for golang.org/x/crypto imports and any
// additional prefixes in extraForbiddenPkgs, attributes each violation to the
// first-hop component from rootPkg, and reports unknown violations or stale
// knownViolations entries via t.Errorf.
func CheckViolations(t testing.TB, binaryPkg, rootPkg string, extraForbiddenPkgs []string, knownViolations map[string]string) {
	t.Helper()

	violations, importGraph := Scan(t, binaryPkg, extraForbiddenPkgs)

	found := make(map[string]bool)

	for _, v := range violations {
		chain := ShortestChain(rootPkg, v.Importer, importGraph)
		if chain == nil {
			chain = []string{v.Importer}
		}
		displayChain := make([]string, len(chain)+1)
		copy(displayChain, chain)
		displayChain[len(chain)] = v.Imported

		var component string
		if len(chain) > 1 {
			component = chain[1]
		} else {
			component = chain[0]
		}

		if reason, ok := knownViolations[component]; !ok {
			t.Errorf("NEW violation via unknown component — add to knownViolations or remove the dependency:\n      -> %s", FormatChain(displayChain))
		} else {
			t.Logf("known violation [%s] via %s:\n      -> %s\n      reason: %s", v.Imported, component, FormatChain(displayChain), reason)
			found[component] = true
		}
	}

	for component := range knownViolations {
		if !found[component] {
			t.Errorf("stale knownViolations entry (component no longer reaches a forbidden package — remove it): %s", component)
		}
	}
}
