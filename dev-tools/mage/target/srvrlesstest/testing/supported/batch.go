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

package supported

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"strings"
	"unicode/utf8"
)

// CreateBatches creates the OSBatch set based on the defined supported OS's.
func CreateBatches(batches []define.Batch, platforms []define.OS, groups []string, matrix bool, singleTest string) ([]common.OSBatch, error) {
	var err error
	var osBatches []common.OSBatch
	for _, b := range batches {
		lbs, err := createBatchesFromBatch(b, platforms, groups, matrix)
		if err != nil {
			return nil, err
		}
		if lbs != nil {
			osBatches = append(osBatches, lbs...)
		}
	}
	if singleTest != "" {
		osBatches, err = filterSingleTest(osBatches, singleTest)
		if err != nil {
			return nil, err
		}
	}

	return osBatches, nil
}

func createBatchesFromBatch(batch define.Batch, platforms []define.OS, groups []string, matrix bool) ([]common.OSBatch, error) {
	var batches []common.OSBatch
	if len(groups) > 0 && !batchInGroups(batch, groups) {
		return nil, nil
	}
	specifics, err := getSupported(batch.OS, platforms)
	if errors.Is(err, ErrOSNotSupported) {
		var s common.SupportedOS
		s.OS.Type = batch.OS.Type
		s.OS.Arch = batch.OS.Arch
		s.OS.Distro = batch.OS.Distro
		if s.OS.Distro == "" {
			s.OS.Distro = "unknown"
		}
		if s.OS.Version == "" {
			s.OS.Version = "unknown"
		}
		b := common.OSBatch{
			OS:    s,
			Batch: batch,
			Skip:  true,
		}
		b.ID = createBatchID(b)
		batches = append(batches, b)
		return batches, nil
	} else if err != nil {
		return nil, err
	}
	if matrix {
		for _, s := range specifics {
			b := common.OSBatch{
				OS:    s,
				Batch: batch,
				Skip:  false,
			}
			b.ID = createBatchID(b)
			batches = append(batches, b)
		}
	} else {
		b := common.OSBatch{
			OS:    specifics[0],
			Batch: batch,
			Skip:  false,
		}
		b.ID = createBatchID(b)
		batches = append(batches, b)
	}
	return batches, nil
}

func batchInGroups(batch define.Batch, groups []string) bool {
	for _, g := range groups {
		if batch.Group == g {
			return true
		}
	}
	return false
}

func filterSingleTest(batches []common.OSBatch, singleTest string) ([]common.OSBatch, error) {
	var filtered []common.OSBatch
	for _, batch := range batches {
		batch, ok := filterSingleTestBatch(batch, singleTest)
		if ok {
			filtered = append(filtered, batch)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("test not found: %s", singleTest)
	}
	return filtered, nil
}

func filterSingleTestBatch(batch common.OSBatch, testName string) (common.OSBatch, bool) {
	for _, pt := range batch.Batch.Tests {
		for _, t := range pt.Tests {
			if t.Name == testName {
				// filter batch to only run one test
				batch.Batch.Tests = []define.BatchPackageTests{
					{
						Name:  pt.Name,
						Tests: []define.BatchPackageTest{t},
					},
				}
				batch.Batch.SudoTests = nil
				// remove stack requirement when the test doesn't need a stack
				if !t.Stack {
					batch.Batch.Stack = nil
				}
				return batch, true
			}
		}
	}
	for _, pt := range batch.Batch.SudoTests {
		for _, t := range pt.Tests {
			if t.Name == testName {
				// filter batch to only run one test
				batch.Batch.SudoTests = []define.BatchPackageTests{
					{
						Name:  pt.Name,
						Tests: []define.BatchPackageTest{t},
					},
				}
				batch.Batch.Tests = nil
				// remove stack requirement when the test doesn't need a stack
				if !t.Stack {
					batch.Batch.Stack = nil
				}
				return batch, true
			}
		}
	}
	return batch, false
}

// createBatchID creates a consistent/unique ID for the batch
//
// ID needs to be consistent so each execution of the runner always
// selects the same ID for each batch.
func createBatchID(batch common.OSBatch) string {
	id := batch.OS.Type + "-" + batch.OS.Arch
	if batch.OS.Type == define.Linux {
		id += "-" + batch.OS.Distro
	}
	if batch.OS.Version != "" {
		id += "-" + strings.Replace(batch.OS.Version, ".", "", -1)
	}
	if batch.OS.Type == define.Kubernetes && batch.OS.DockerVariant != "" {
		id += "-" + batch.OS.DockerVariant
	}
	id += "-" + strings.Replace(batch.Batch.Group, ".", "", -1)

	// The batchID needs to be at most 63 characters long otherwise
	// OGC will fail to instantiate the VM.
	maxIDLen := 63
	if len(id) > maxIDLen {
		hash := fmt.Sprintf("%x", sha512.Sum384([]byte(id)))
		hashLen := utf8.RuneCountInString(hash)
		id = id[:maxIDLen-hashLen-1] + "-" + hash
	}

	return strings.ToLower(id)
}
