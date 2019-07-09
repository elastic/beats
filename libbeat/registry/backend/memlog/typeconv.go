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

type timeUnfolder struct {
	gotype.BaseUnfoldState
	to   *time.Time
	a, b uint64
	st   timeUnfoldState
}

type timeUnfoldState uint8

const (
	timeUnfoldInit timeUnfoldState = iota
	timeUnfoldDone
	timeUnfoldWaitA
	timeUnfoldWaitB
	timeUnfoldWaitDone
)

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
	unfold, _ := gotype.NewUnfolder(nil, gotype.Unfolders(
		unfoldTimestamp,
	))
	fold, err := gotype.NewIterator(unfold, gotype.Folders(
		foldTimestamp,
	))
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
		(uint64(uint16(off)) << 32) |
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

func unfoldTimestamp(to *time.Time) gotype.UnfoldState {
	return &timeUnfolder{to: to}
}

func (u *timeUnfolder) OnString(ctx gotype.UnfoldCtx, in string) (err error) {
	if u.st != timeUnfoldInit {
		return fmt.Errorf("Unexpected string '%v' when trying to unfold a timestamp", in)
	}

	*u.to, err = time.Parse(time.RFC3339, in)
	u.st = timeUnfoldDone
	ctx.Done()
	return err
}

func (u *timeUnfolder) OnArrayStart(ctx gotype.UnfoldCtx, len int, _ structform.BaseType) error {
	if u.st != timeUnfoldInit {
		return errors.New("unexpected array")
	}

	if len >= 0 && len != 2 {
		return fmt.Errorf("%v is no valid encoded timestamp length", len)
	}

	u.st = timeUnfoldWaitA
	return nil
}

func (u *timeUnfolder) OnUint(ctx gotype.UnfoldCtx, in uint64) error {
	switch u.st {
	case timeUnfoldWaitA:
		u.a = in
		u.st = timeUnfoldWaitB
	case timeUnfoldWaitB:
		u.b = in
		u.st = timeUnfoldWaitDone
	default:
		return fmt.Errorf("unexpected number '%v' in timestamp array", in)
	}
	return nil
}

func (u *timeUnfolder) OnInt(ctx gotype.UnfoldCtx, in int64) error {
	return u.OnUint(ctx, uint64(in))
}

func (u *timeUnfolder) OnArrayFinished(ctx gotype.UnfoldCtx) error {
	if u.st != timeUnfoldWaitDone {
		return errors.New("unexpected timestamp array closed")
	}

	u.st = timeUnfoldDone

	version := (u.a >> 56) & 0xff
	if version != 0 {
		return fmt.Errorf("invalid timestamp [%x, %x]", u.a, u.b)
	}

	sec := u.b
	nsec := uint32(u.a)
	off := int16(u.a >> 32)

	ts := u.to
	*ts = time.Unix(int64(sec), int64(nsec))

	// adjust location by offset. time.Unix creates a timestamp in the local zone
	// by default. Only change this if off does not match the local zone it's offset.
	if off == -1 {
		*ts = ts.UTC()
	} else if off != 0 {
		_, locOff := ts.Zone()
		if off != int16(locOff/60) {
			*ts = ts.In(time.FixedZone("", int(off*60)))
		}
	}

	ctx.Done()
	return nil
}
