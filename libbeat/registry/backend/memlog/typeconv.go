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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-structform/gotype"
)

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
	tc.init()
	return tc
}

func (t *typeConv) release() {
	typeConvPool.Put(t)
}

func (t *typeConv) init() {
	unfold, _ := gotype.NewUnfolder(nil)
	fold, err := gotype.NewIterator(unfold)
	if err != nil {
		panic(err)
	}

	t.unfold = unfold
	t.fold = fold
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
