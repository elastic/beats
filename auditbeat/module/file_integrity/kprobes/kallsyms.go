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

//go:build linux

package kprobes

import (
	"bufio"
	"io"
	"os"
	"slices"
	"strings"
)

const kAllSymsPath = "/proc/kallsyms"

type runtimeSymbolInfo struct {
	symbolName          string
	isOptimised         bool
	optimisedSymbolName string
}

// getSymbolInfoRuntime returns the runtime symbol information for the given symbolName
// from the /proc/kallsyms file.
func getSymbolInfoRuntime(symbolName string) (runtimeSymbolInfo, error) {
	kAllSymsFile, err := os.Open(kAllSymsPath)
	if err != nil {
		return runtimeSymbolInfo{}, err
	}

	defer kAllSymsFile.Close()

	return getSymbolInfoFromReader(kAllSymsFile, symbolName)
}

// getSymbolInfoFromReader retrieves symbol information from a reader that is expected to
// provide content in the same format as /proc/kallsyms.
//
// An exact match for symbolName always returns immediately. The kernel may list
// compiler-generated variants of a symbol before the symbol itself but only the
// the plain symbol is a reliable probe target. A symbol is reported as optimised
// when no exact match exists and a clone such as "name.constprop.0" or "name.isra.0"
// does. Cold-path fragments ("name.cold") are not clones: the compiler splits rarely
// executed blocks out of the function, which keeps its symbolName entry point, so these
// symbol names never count as optimised variants.
func getSymbolInfoFromReader(reader io.Reader, symbolName string) (runtimeSymbolInfo, error) {
	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	// optimised symbols start with the unoptimised symbol name
	// followed by ".{optimisation_type}..."
	optimisedPrefix := symbolName + "."

	var optimisedName string
	for fileScanner.Scan() {
		// Lines are "address type name [module]".
		fields := strings.Fields(fileScanner.Text())
		if len(fields) < 3 {
			continue
		}
		name := fields[2]

		switch {
		case name == symbolName:
			// Exact match
			return runtimeSymbolInfo{
				symbolName:  symbolName,
				isOptimised: false,
			}, nil
		case optimisedName == "" && strings.HasPrefix(name, optimisedPrefix) &&
			!isColdFragment(strings.TrimPrefix(name, optimisedPrefix)):
			optimisedName = name
		}
	}

	if err := fileScanner.Err(); err != nil {
		return runtimeSymbolInfo{}, err
	}

	if optimisedName != "" {
		return runtimeSymbolInfo{
			symbolName:          symbolName,
			isOptimised:         true,
			optimisedSymbolName: optimisedName,
		}, nil
	}

	return runtimeSymbolInfo{}, ErrSymbolNotFound
}

// isColdFragment reports whether a symbol suffix (the part after "name.") denotes a
// cold-path fragment as emitted by -freorder-blocks-and-partition: "cold", "cold.1",
// or the fragment of a clone such as "constprop.0.cold".
func isColdFragment(suffix string) bool {
	return slices.Contains(strings.Split(suffix, "."), "cold")
}
