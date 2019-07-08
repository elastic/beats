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
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	structform "github.com/elastic/go-structform"
	"github.com/elastic/go-structform/gotype"
)

// typeConv can convert structured data between arbitrary typed (serializable)
// go structures and maps/slices/arrays. It uses go-structform/gotype for input
// and output values each, such that any arbitrary structures can be used.
//
// Internally typeConv is used by the ValueDecoder for (de-)serializing an
// users value into common.MapStr, which is used to store objects in the memory
// store.
type typeConv struct {
	fold   *gotype.Iterator
	unfold *gotype.Unfolder
}

type valueDecoder struct {
	tx    memTxParent
	value common.MapStr
}

var typeConvPool = sync.Pool{
	New: func() interface{} {
		t := &typeConv{}
		t.init()
		return t
	},
}

func newTypeConv() *typeConv {
	tc := typeConvPool.Get().(*typeConv)
	return tc
}

func (t *typeConv) release() {
	if t != nil {
		typeConvPool.Put(t)
	}
}

func (t *typeConv) init() {
	unfold, _ := gotype.NewUnfolder(nil)
	fold, err := gotype.NewIterator(unfold, gotype.Folders(
		foldTimestamp,
	))
	if err != nil {
		panic(err)
	}

	t.unfold = unfold
	t.fold = fold
}

func foldTimestamp(in *time.Time, v structform.ExtVisitor) error {
	var (
		ts  = *in
		off int16
		loc = ts.Location()
	)

	const encodingVersion = 0

	if loc == time.UTC {
		off = -1
	} else {
		_, offset := ts.Zone()
		offset /= 60 // Note: best effort. If the zone offset has a factional minute, then we will ignore it here
		if offset < -32768 || offset == -1 || offset > 32767 {
			offset = 0 // Note: best effort. Ignore offset if it becomes an unexpected value
		}
		off = int16(offset)
	}

	sec := uint64(ts.Unix())
	extra := (uint64(encodingVersion) << 56) |
		(uint64(off) << 32) |
		uint64(ts.Nanosecond())

	if err := v.OnArrayStart(2, structform.Uint64Type); err != nil {
		return err
	}
	if err := v.OnUint64(extra); err != nil {
		return err
	}
	if err := v.OnUint64(sec); err != nil {
		return err
	}
	if err := v.OnArrayFinished(); err != nil {
		return err
	}

	return nil
}

func (t *typeConv) Convert(to, from interface{}) error {
	err := t.unfold.SetTarget(to)
	if err != nil {
		return err
	}

	defer t.unfold.Reset()
	return t.fold.Fold(from)
}

func newValueDecoder(tx memTxParent, value common.MapStr) *valueDecoder {
	return &valueDecoder{
		tx:    tx,
		value: value,
	}
}

func (d *valueDecoder) Decode(to interface{}) (err error) {
	if err = d.tx.checkRead(); err == nil {
		err = d.tx.getTypeConv().Convert(to, d.value)
	}
	return
}
