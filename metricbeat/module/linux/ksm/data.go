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

package ksm

import (
	"fmt"
	"path/filepath"

	"github.com/elastic/beats/v7/metricbeat/module/linux"
)

type ksmData struct {
	PagesShared      int64 `json:"pages_shared"`
	PagesSharing     int64 `json:"pages_sharing"`
	PagesUnshared    int64 `json:"pages_unshared"`
	PagesVolatile    int64 `json:"pages_volatile"`
	FullScans        int64 `json:"full_scans"`
	StableNodeChains int64 `json:"stable_node_chains"`
	StableNodeDups   int64 `json:"stable_node_dups"`
}

// fetchKSMStats reads the KSM stat counters and returns a struct
func fetchKSMStats(ksmPath string) (ksmData, error) {
	// ReadIntFromFile returns pretty verbose error strings, so omit errors.Wrap here
	pshared, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "pages_shared"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from pages_shared: %w", err)
	}

	pSharing, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "pages_sharing"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from pages_sharing: %w", err)
	}

	pUnshared, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "pages_unshared"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from pages_unshared: %w", err)
	}

	pVolatile, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "pages_volatile"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from pages_volatile: %w", err)
	}

	fScans, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "full_scans"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from full_scans: %w", err)
	}

	stableChains, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "stable_node_chains"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from stable_node_chains: %w", err)
	}

	stableDups, err := linux.ReadIntFromFile(filepath.Join(ksmPath, "stable_node_dups"), 10)
	if err != nil {
		return ksmData{}, fmt.Errorf("error reading from stable_node_dups : %w", err)
	}

	return ksmData{PagesShared: pshared, PagesSharing: pSharing, PagesUnshared: pUnshared,
		PagesVolatile: pVolatile, FullScans: fScans, StableNodeChains: stableChains, StableNodeDups: stableDups}, nil

}
