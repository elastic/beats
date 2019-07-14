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
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/registry/backend"
)

type hashtable struct {
	bins map[uint64]bin
}

type bin []entry

type entry struct {
	key   backend.Key
	value common.MapStr
}

type keyPair struct {
	hash uint64
	key  backend.Key
}

type valueRef struct {
	bin []entry
	idx int
}

func newHashtable() *hashtable {
	t := &hashtable{}
	t.init()
	return t
}

func (tbl *hashtable) init() {
	tbl.bins = map[uint64]bin{}
}

func (tbl *hashtable) find(k keyPair) valueRef {
	bin, i := tbl.findHash(k.hash, k.key)
	return valueRef{bin: bin, idx: i}
}

func (tbl *hashtable) findHash(hash uint64, k backend.Key) ([]entry, int) {
	bin := tbl.bins[hash]
	return bin, bin.index(k)
}

// copySpline copies the hash table structure, but values/keys pairs are still
// shared with the original hashtable (shallow copy).
func (tbl *hashtable) copySpline() hashtable {
	to := hashtable{
		bins: make(map[uint64]bin, len(tbl.bins)),
	}

	for hash, b := range tbl.bins {
		tmp := make(bin, len(b))
		copy(tmp, b)
		to.bins[hash] = tmp
	}

	return to
}

func (tbl *hashtable) set(hash uint64, k backend.Key, v common.MapStr) {
	bin := tbl.bins[hash]
	idx := bin.index(k)
	if idx < 0 {
		tbl.bins[hash] = append(bin, entry{
			key:   k,
			value: v,
		})
	} else {
		bin[idx].value = v
	}
}

func (tbl *hashtable) Len() int {
	i := 0
	for _, bin := range tbl.bins {
		i += len(bin)
	}
	return i
}

func (b *bin) keyFn(i int) backend.Key {
	return (*b)[i].key
}

func (b *bin) index(k backend.Key) int {
	return tblBinFind(len(*b), k, b.keyFn)
}

func (b *bin) remove(i int) {
	L := len(*b)
	if L <= 1 {
		*b = nil
		return
	}

	(*b)[i] = (*b)[L-1]
	(*b) = (*b)[:L-1]
}

func tblBinFind(L int, k backend.Key, keyFn func(idx int) backend.Key) int {
	if L == 0 {
		return -1
	}

	for i := 0; i < L; i++ {
		if k == keyFn(i) {
			return i
		}
	}

	return -1
}

func (r valueRef) IsNil() bool {
	return r.bin == nil || r.idx < 0
}

func (r valueRef) Access() *entry {
	return &r.bin[r.idx]
}
