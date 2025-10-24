// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package parsers

import (
	"bytes"
	"fmt"
	"os"

	"github.com/parsiya/golnk"
)

func ParseLnk(lnkPath string) ([][]byte, error) {
	Lnk, err := lnk.File(lnkPath)
	if err != nil {
		return nil, err
	}
    fileBytes, err := os.ReadFile(lnkPath)
	if err != nil {
		return nil, err
	}
	var lnkFiles [][]byte
	lnkSignature := []byte{0x4c, 0x00, 0x00, 0x00}

	// Scan through file looking for LNK signatures
	for i := 0; i < len(fileBytes); i++ {
		// Check if we found a LNK signature
		if i+len(lnkSignature) <= len(fileBytes) && 
			bytes.Equal(fileBytes[i:i+len(lnkSignature)], lnkSignature) {

			start := i
			// Find end - either next signature or EOF
			end := len(fileBytes)
			for j := start + len(lnkSignature); j < len(fileBytes)-len(lnkSignature); j++ {
				if bytes.Equal(fileBytes[j:j+len(lnkSignature)], lnkSignature) {
					end = j
					break
				}
			}

			// Extract the LNK file bytes
			lnkFiles = append(lnkFiles, fileBytes[start:end])
			i = end - 1 // Move cursor to end (minus 1 since loop will increment)
		}
	}
	fmt.Fprintf(os.Stdout, "LNK files length: %+v\n", lnkFiles)

	return lnkFiles, nil
}