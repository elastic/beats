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
	"fmt"
	"io"
	"os"
	"regexp"
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
// provide content in the same format as /proc/kallsyms
func getSymbolInfoFromReader(reader io.Reader, symbolName string) (runtimeSymbolInfo, error) {
	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	symReg, err := regexp.Compile(fmt.Sprintf("(?m)^([a-fA-F0-9]+).*?(%s(|.*?)?)(\\s+.*$|$)", symbolName))
	if err != nil {
		return runtimeSymbolInfo{}, err
	}

	// optimised symbols start with the unoptimised symbol name
	// followed by ".{optimisation_type}..."
	optimisedSymbolName := symbolName + "."

	for fileScanner.Scan() {
		matches := symReg.FindAllSubmatch(fileScanner.Bytes(), -1)
		if len(matches) == 0 {
			continue
		}

		for _, match := range matches {
			matchSymbolName := string(match[2])
			switch {
			case strings.HasPrefix(matchSymbolName, optimisedSymbolName):
				return runtimeSymbolInfo{
					symbolName:          symbolName,
					isOptimised:         true,
					optimisedSymbolName: matchSymbolName,
				}, nil
			case strings.EqualFold(matchSymbolName, symbolName):
				return runtimeSymbolInfo{
					symbolName:          symbolName,
					isOptimised:         false,
					optimisedSymbolName: "",
				}, nil
			}
		}
	}

	if fileScanner.Err() != nil {
		return runtimeSymbolInfo{}, err
	}

	return runtimeSymbolInfo{}, ErrSymbolNotFound
}
