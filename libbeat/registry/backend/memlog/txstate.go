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
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/registry/backend"
)

type txState struct {
	bins map[uint64]txCacheLine

	ops uint // number of operations in this transaction
}

type txCacheLine []txCacheEntry

// Cached per key transaction state.
// All events read will be cached.
//
// Flags (s=set flag, d=delete flag):
// operation     exists modified
//   Get            s
//  Insert          s      s
//  Update          s      s
//  Remove          d      s
type txCacheEntry struct {
	tx memTxParent

	key     backend.Key
	value   common.MapStr
	updates common.MapStr

	ops []op

	exists   bool
	modified bool
}

type cacheEntryRef struct {
	hash uint64
	line txCacheLine
	idx  int
}

func (st *txState) init() {
	st.bins = map[uint64]txCacheLine{}
}

func (st *txState) find(k keyPair) cacheEntryRef {
	line, idx := st.findHash(k.hash, k.key)
	return cacheEntryRef{line: line, hash: k.hash, idx: idx}
}

func (st *txState) findHash(hash uint64, k backend.Key) (txCacheLine, int) {
	bin := st.bins[hash]
	idx := tblBinFind(len(bin), k, func(i int) backend.Key {
		return bin[i].key
	})
	return bin, idx
}

func (st *txState) setEntry(at cacheEntryRef, entry txCacheEntry) cacheEntryRef {
	if at.IsNil() {
		return st.addEntry(at, entry)
	}
	*at.Access() = entry
	return at
}

func (st *txState) addEntry(at cacheEntryRef, entry txCacheEntry) cacheEntryRef {
	line := at.line
	idx := len(line)
	line = append(line, entry)
	st.bins[at.hash] = line

	pos := at
	pos.line = line
	pos.idx = idx
	return pos
}

func (l *txCacheLine) keyFn(i int) backend.Key {
	return (*l)[i].key
}

func (l *txCacheLine) index(k backend.Key) int {
	if l == nil {
		return -1
	}
	return tblBinFind(len(*l), k, l.keyFn)
}

func (e *txCacheEntry) recordInsert(v common.MapStr) {
	e.value = nil
	e.updates = v
	e.ops = []op{&opInsertWith{K: string(e.key), V: v}}
	e.modified = true
	e.exists = true
}

func (e *txCacheEntry) recordUpdate(v common.MapStr) {
	e.modified = true
	if len(e.updates) == 0 {
		e.updates = v
	} else {
		e.updates.DeepUpdate(v)
	}
	e.recordOp(&opUpdate{K: string(e.key), V: v})
}

func (e *txCacheEntry) recordRemove() {
	e.value = nil
	e.updates = nil
	e.ops = []op{&opRemove{K: string(e.key)}}
	e.modified = true
	e.exists = false
}

func (e *txCacheEntry) recordOp(update op) {
	if L := len(e.ops); L == 0 {
		e.ops = []op{update}
	} else {
		prev := e.ops[L-1]
		first, second := mergeKVOp(prev, update)
		e.ops[L-1] = first
		if second != nil {
			e.ops = append(e.ops, second)
		}
	}
}

func (e *txCacheEntry) Decode(to interface{}) (err error) {
	if err := e.tx.checkRead(); err != nil {
		return err
	}

	if !e.exists {
		return errValueRemoved
	}

	tc := e.tx.getTypeConv()
	if e.value != nil {
		if err := tc.Convert(to, e.value); err != nil {
			return err
		}
	}
	if e.updates != nil {
		if err := tc.Convert(to, e.updates); err != nil {
			return err
		}
	}
	return nil
}

func (r cacheEntryRef) IsNil() bool {
	return r.line == nil || r.idx < 0
}

func (r cacheEntryRef) Access() *txCacheEntry {
	return &r.line[r.idx]
}
