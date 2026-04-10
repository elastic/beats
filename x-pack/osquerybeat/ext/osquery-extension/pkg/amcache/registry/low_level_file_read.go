// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"www.velocidex.com/golang/go-ntfs/parser"
)

// Portions of this function are based on code from the fslib library:
// https://github.com/forensicanalysis/fslib
//
// MIT License
// Copyright (c) 2019-2020 Siemens AG
// Copyright (c) 2019-2021 Jonas Plum
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
func readFileViaNTFS(filePath string) ([]byte, error) {
	if len(filePath) < 3 || filePath[1] != ':' {
		return nil, fmt.Errorf("unsupported path format: %s", filePath)
	}

	driveLetter := filePath[0]
	ntfsPath := "/" + filepath.ToSlash(filePath[3:]) // C:\Windows\foo.txt → /Windows/foo.txt

	volume, err := os.Open(fmt.Sprintf(`\\.\%c:`, driveLetter))
	if err != nil {
		return nil, fmt.Errorf("failed to open volume: %w", err)
	}
	defer volume.Close()

	reader, err := parser.NewPagedReader(volume, 1024*1024, 100*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to create paged reader: %w", err)
	}

	ntfsCtx, err := parser.GetNTFSContext(reader, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NTFS: %w", err)
	}

	root, err := ntfsCtx.GetMFT(5)
	if err != nil {
		return nil, fmt.Errorf("failed to get MFT root: %w", err)
	}

	entry, err := root.Open(ntfsCtx, ntfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s via NTFS: %w", ntfsPath, err)
	}

	attr, err := entry.GetAttribute(ntfsCtx, 128, -1, "") // 128 = $DATA
	if err != nil {
		return nil, fmt.Errorf("failed to get data attribute: %w", err)
	}

	infos, err := parser.ModelMFTEntry(ntfsCtx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	data := make([]byte, infos.Size)
	_, err = attr.Data(ntfsCtx).ReadAt(data, 0)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	return data, nil
}
