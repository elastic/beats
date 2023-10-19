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

	defer func() {
		_ = kAllSymsFile.Close()
	}()

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

	return runtimeSymbolInfo{}, ErrSymbolNotFound
}
