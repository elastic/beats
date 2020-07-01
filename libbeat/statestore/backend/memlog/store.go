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
	"os"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
)

// store implements an actual memlog based store.
// It holds all key value pairs in memory in a memstore struct.
// All changes to the memstore are logged to the diskstore.
// The store execute a checkpoint operation if the checkpoint predicate
// triggers the operation, or if some error in the update log file has been
// detected by the diskstore.
//
// The store allows only one writer, but multiple concurrent readers.
type store struct {
	lock sync.RWMutex
	disk *diskstore
	mem  memstore
}

// memstore is the in memory key value store
type memstore struct {
	table map[string]entry
}

type entry struct {
	value map[string]interface{}
}

// openStore opens a store from the home path.
// The directory and intermediate directories will be created if it does not exist.
// The open routine loads the full key-value store into memory by first reading the data file and finally applying all outstanding updates
// from the update log file.
// If an error in in the log file is detected, the store opening routine continues from the last known valid state and will trigger a checkpoint
// operation on subsequent writes, also truncating the log file.
// Old data files are scheduled for deletion later.
func openStore(log *logp.Logger, home string, mode os.FileMode, bufSz uint, checkpoint CheckpointPredicate) (*store, error) {
	panic("TODO: implement me")
}

// Close closes access to the update log file and clears the in memory key
// value store. Access to the store after close can lead to a panic.
func (s *store) Close() error {
	panic("TODO: implement me")
}

// Has checks if the key is known. The in memory store does not report any
// errors.
func (s *store) Has(key string) (bool, error) {
	panic("TODO: implement me")
}

// Get retrieves and decodes the key-value pair into to.
func (s *store) Get(key string, to interface{}) error {
	panic("TODO: implement me")
}

// Set inserts or overwrites a key-value pair.
// If encoding was successful the in-memory state will be updated and a
// set-operation is logged to the diskstore.
func (s *store) Set(key string, value interface{}) error {
	panic("TODO: implement me")
}

// Remove removes a key from the in memory store and logs a remove operation to
// the diskstore. The operation does not check if the key exists.
func (s *store) Remove(key string) error {
	panic("TODO: implement me")
}

// Each iterates over all key-value pairs in the store.
func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	panic("TODO: implement me")
}

func (s *memstore) Set(key string, value interface{}) error {
	panic("TODO: implement me")
}

func (s *memstore) Remove(key string) error {
	panic("TODO: implement me")
}

func (e entry) Decode(to interface{}) error {
	return typeconv.Convert(to, e.value)
}
