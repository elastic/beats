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

package memlog

import (
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/libbeat/registry/backend"
)

type Registry struct {
	mu     sync.Mutex
	active bool

	// root     string
	// fileMode os.FileMode
	settings Settings

	wg sync.WaitGroup
}

type Settings struct {
	Root       string
	FileMode   os.FileMode
	Checkpoint func(pairs, logs uint) bool
	BufferSize uint // read/write buffer size when reading/writing store files
}

func New(settings Settings) (*Registry, error) {
	if settings.FileMode == 0 {
		settings.FileMode = 0600
	}
	if settings.Checkpoint == nil {
		settings.Checkpoint = CheckpointRatio(2.0, 10.0)
	}
	if settings.BufferSize == 0 {
		settings.BufferSize = 4096
	}

	return &Registry{
		active:   true,
		settings: settings,
	}, nil
}

func (r *Registry) Access(name string) (backend.Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.active {
		return nil, errRegClosed
	}

	root := r.settings.Root
	fileMode := r.settings.FileMode
	bufSz := int(r.settings.BufferSize)
	store, err := newStore(filepath.Join(root, name), fileMode, bufSz)
	if err != nil {
		return nil, err
	}

	store.predCheckpoint = r.settings.Checkpoint
	return store, nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	r.active = false
	r.mu.Unlock()

	// block until all stores have been closed
	r.wg.Wait()
	return nil
}

func CheckpointRatio(minFactor, maxFactor float64) func(uint, uint) bool {
	if maxFactor <= 0 || math.IsNaN(maxFactor) {
		maxFactor = 10.0
	}
	if minFactor <= 1 {
		minFactor = 1
	}
	if minFactor > maxFactor {
		minFactor = maxFactor
	}

	if minFactor == maxFactor {
		return func(pairs, logs uint) bool {
			limit := float64(pairs) * minFactor
			return float64(logs) > limit
		}
	}

	return func(pairs, logs uint) bool {
		if pairs == 0 {
			return true
		}

		if pairs < logs {
			return false
		}

		// Note: the bigger the registry file, the less we want to run expensive checkpoints
		//
		// x      = number of registry entries
		// limit  = number of transaction log entries until next checkpoint
		// factor = ratio of registry file size and transaction log entries
		//          until checkpointing will be enforced. The bigger the registry,
		//          the less we want to execute a checkpoint -> factor should be bigger.
		//
		//      x      limit    factor
		//      1         1        1       // always checkpoint (totalEntries > 2)
		//     10        27      2.66
		//    100       432      4.32
		//   1000      5983      5.98
		//  10000     76439      7.64
		// 100000    930482      9.3
		limit := func(x uint) float64 {
			v := float64(x)
			factor := math.Min(minFactor, math.Max(maxFactor, (1+0.5*math.Log2(v))))
			return v * factor
		}

		return float64(logs) > limit(pairs)
	}
}
