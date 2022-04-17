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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/transform/typeconv"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/statestore/backend"
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
func openStore(log *logp.Logger, home string, mode os.FileMode, bufSz uint, ignoreVersionCheck bool, checkpoint CheckpointPredicate) (*store, error) {
	fi, err := os.Stat(home)
	if os.IsNotExist(err) {
		err = os.MkdirAll(home, os.ModeDir|0770)
		if err != nil {
			return nil, err
		}

		err = writeMetaFile(home, mode)
		if err != nil {
			return nil, err
		}
	} else if !fi.Mode().IsDir() {
		return nil, fmt.Errorf("'%v' is not a directory", home)
	} else {
		if err := pathEnsurePermissions(filepath.Join(home, metaFileName), mode); err != nil {
			return nil, fmt.Errorf("failed to update meta file permissions: %w", err)
		}
	}

	if !ignoreVersionCheck {
		meta, err := readMetaFile(home)
		if err != nil {
			return nil, err
		}
		if err := checkMeta(meta); err != nil {
			return nil, err
		}
	}

	if err := pathEnsurePermissions(filepath.Join(home, activeDataFileName), mode); err != nil {
		return nil, fmt.Errorf("failed to update active file permissions: %w", err)
	}

	dataFiles, err := listDataFiles(home)
	if err != nil {
		return nil, err
	}
	for _, df := range dataFiles {
		if err := pathEnsurePermissions(df.path, mode); err != nil {
			return nil, fmt.Errorf("failed to update data file permissions: %w", err)
		}
	}
	if err := pathEnsurePermissions(filepath.Join(home, logFileName), mode); err != nil {
		return nil, fmt.Errorf("failed to update log file permissions: %w", err)
	}

	tbl := map[string]entry{}
	var txid uint64
	if L := len(dataFiles); L > 0 {
		active := dataFiles[L-1]
		txid = active.txid
		if err := loadDataFile(active.path, tbl); err != nil {
			if errors.Is(err, ErrCorruptStore) {
				corruptFilePath := active.path + ".corrupted"
				err := os.Rename(active.path, corruptFilePath)
				if err != nil {
					logp.Debug("Failed to backup corrupt data file '%s': %+v", active.path, err)
				}
				logp.Warn("Data file is corrupt. It has been renamed to %s. Attempting to restore partial state from log file.", corruptFilePath)
			} else {
				return nil, err
			}
		} else {
			logp.Info("Loading data file of '%v' succeeded. Active transaction id=%v", home, txid)
		}
	}

	var entries uint
	memstore := memstore{tbl}
	txid, entries, err = loadLogFile(&memstore, txid, home)
	logp.Info("Finished loading transaction log file for '%v'. Active transaction id=%v", home, txid)

	if err != nil {
		// Error indicates the log file was incomplete or corrupted.
		// Anyways, we already have the table in a valid state and will
		// continue opening the store from here.
		logp.Warn("Incomplete or corrupted log file in %v. Continue with last known complete and consistent state. Reason: %v", home, err)
	}

	diskstore, err := newDiskStore(log, home, dataFiles, txid, mode, entries, err != nil, bufSz, checkpoint)
	if err != nil {
		return nil, err
	}

	return &store{
		disk: diskstore,
		mem:  memstore,
	}, nil
}

// Close closes access to the update log file and clears the in memory key
// value store. Access to the store after close can lead to a panic.
func (s *store) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.mem = memstore{}
	return s.disk.Close()
}

// Has checks if the key is known. The in memory store does not report any
// errors.
func (s *store) Has(key string) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.mem.Has(key), nil
}

// Get retrieves and decodes the key-value pair into to.
func (s *store) Get(key string, to interface{}) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	dec := s.mem.Get(key)
	if dec == nil {
		return errKeyUnknown
	}
	return dec.Decode(to)
}

// Set inserts or overwrites a key-value pair.
// If encoding was successful the in-memory state will be updated and a
// set-operation is logged to the diskstore.
func (s *store) Set(key string, value interface{}) error {
	var tmp common.MapStr
	if err := typeconv.Convert(&tmp, value); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.mem.Set(key, tmp)
	return s.logOperation(&opSet{K: key, V: tmp})
}

// Remove removes a key from the in memory store and logs a remove operation to
// the diskstore. The operation does not check if the key exists.
func (s *store) Remove(key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.mem.Remove(key)
	return s.logOperation(&opRemove{K: key})
}

// Checkpoint triggers a state checkpoint operation. All state will be written
// to a new transaction data file and fsync'ed. The log file will be reset after
// a successful write.
func (s *store) Checkpoint() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.disk.WriteCheckpoint(s.mem.table)
}

// lopOperation ensures that the diskstore reflects the recent changes to the
// in memory store by either triggering a checkpoint operations or adding the
// operation type to the update log file.
func (s *store) logOperation(op op) error {
	if s.disk.mustCheckpoint() {
		err := s.disk.WriteCheckpoint(s.mem.table)
		if err != nil {
			// if writing the new checkpoint file failed we try to fallback to
			// appending the log operation.
			// idea: make append configurable and retry checkpointing with backoff.
			_ = s.disk.LogOperation(op)
		}

		return err
	}

	return s.disk.LogOperation(op)
}

// Each iterates over all key-value pairs in the store.
func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for k, entry := range s.mem.table {
		cont, err := fn(k, entry)
		if !cont || err != nil {
			return err
		}
	}

	return nil
}

func (m *memstore) Has(key string) bool {
	_, exists := m.table[key]
	return exists
}

func (m *memstore) Get(key string) backend.ValueDecoder {
	entry, exists := m.table[key]
	if !exists {
		return nil
	}
	return entry
}

func (m *memstore) Set(key string, value common.MapStr) {
	m.table[key] = entry{value: value}
}

func (m *memstore) Remove(key string) bool {
	_, exists := m.table[key]
	if !exists {
		return false
	}
	delete(m.table, key)
	return true
}

func (e entry) Decode(to interface{}) error {
	return typeconv.Convert(to, e.value)
}
