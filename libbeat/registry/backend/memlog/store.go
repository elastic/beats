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
	"fmt"
	"os"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/registry/backend"
)

type store struct {
	lock lock

	active         bool
	predCheckpoint func(pairs, logs uint) bool
	mem            *memStore
	disk           *diskStore
}

type memStore struct {
	tbl hashtable
}

func newStore(home string, mode os.FileMode, bufSz int) (*store, error) {
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
		return nil, fmt.Errorf("'%v' is no directory", home)
	}

	meta, err := readMetaFile(home)
	if err != nil {
		return nil, err
	}

	if err := checkMeta(meta); err != nil {
		return nil, err
	}

	dataFiles, err := listDataFiles(home)
	if err != nil {
		return nil, err
	}

	active, txid := activeDataFile(dataFiles)
	tbl, err := loadDataFile(active)
	if err != nil {
		return nil, err
	}

	logp.Info("Loading data file of '%v' succeeded. Active transaction id=%v", home, txid)

	mem := &memStore{tbl: tbl}
	txid, entries, complete, err := loadLogFile(mem, txid, home)
	logp.Info("Finished loading transaction log file for '%v'. Active transaction id=%v", home, txid)

	if err != nil || !complete {
		// Error indicates the log file was incomplete or corrupted.
		// Anyways, we already have the table in a valid state and will
		// continue opening the store from here.
		logp.Warn("Incomplete or corrupted log file in %v. Continue with last known complete and consistend state. Reason: %v", home, err)
	}

	store := &store{
		active: true,
		mem:    mem,
		disk:   newDiskStore(home, dataFiles, txid, mode, entries, !complete, bufSz),
	}
	store.lock.init()
	return store, nil
}

func (s *store) Close() error {
	if !s.active {
		return errStoreClosed
	}

	err := s.disk.Close()

	// Registry frontend keeps ref-count of active data files.
	// -> Clear all state final close.
	*s = store{active: false}
	return err
}

func (s *store) Begin(readonly bool) (backend.Tx, error) {
	if !s.active {
		return nil, errStoreClosed
	}

	lock := chooseTxLock(&s.lock, readonly)
	lock.Lock()
	return newTransaction(s, readonly), nil
}

func (s *memStore) has(key keyPair) bool {
	return !s.find(key).IsNil()
}

func (s *memStore) find(key keyPair) valueRef {
	return s.tbl.find(key)
}

func chooseTxLock(lock *lock, readonly bool) sync.Locker {
	if readonly {
		return lock.Shared()
	}
	return lock.Reserved()
}
