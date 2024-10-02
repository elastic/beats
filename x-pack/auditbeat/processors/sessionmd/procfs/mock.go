// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package procfs

import (
	"fmt"
)

type MockReader struct {
	entries map[uint32]ProcessInfo
}

func NewMockReader() *MockReader {
	return &MockReader{
		entries: make(map[uint32]ProcessInfo),
	}
}

func (r *MockReader) AddEntry(pid uint32, entry ProcessInfo) {
	r.entries[pid] = entry
}

func (r *MockReader) GetProcess(pid uint32) (ProcessInfo, error) {
	entry, ok := r.entries[pid]
	if !ok {
		return ProcessInfo{}, fmt.Errorf("not found")
	}
	return entry, nil
}

func (r *MockReader) GetAllProcesses() ([]ProcessInfo, error) {
	ret := make([]ProcessInfo, 0, len(r.entries))

	for _, entry := range r.entries {
		ret = append(ret, entry)
	}
	return ret, nil
}
