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

package typeconv

import (
	"errors"
	"fmt"
	"sync"
	"time"

	structform "github.com/menderesk/go-structform"
	"github.com/menderesk/go-structform/gotype"
)

// Converter converts structured data between arbitrary typed (serializable)
// go structures and maps/slices/arrays. It uses go-structform/gotype for input
// and output values each, such that any arbitrary structures can be used.
//
// The converter computes and caches mapping operations for go structures it
// has visited.
type Converter struct {
	fold   *gotype.Iterator
	unfold *gotype.Unfolder
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

var convPool = sync.Pool{
	New: func() interface{} {
		return &Converter{}
	},
}

// NewConverter creates a new converter with local state for tracking known
// type conversations.
func NewConverter() *Converter {
	c := &Converter{}
	return c
}

func (c *Converter) init() {
	unfold, _ := gotype.NewUnfolder(nil, gotype.Unfolders(
		unfoldTimestamp,
	))
	fold, err := gotype.NewIterator(unfold, gotype.Folders(
		foldTimestamp,
	))
	if err != nil {
		panic(err)
	}

	c.unfold = unfold
	c.fold = fold
}

// Convert transforms the value of from into to, by translating the structure
// from into a set of events (go-structform.Visitor) that can applied to the
// value given by to.
// The operation fails if the values are not compatible (for example trying to
// convert an object into an int), or `to` is no pointer.
func (c *Converter) Convert(to, from interface{}) (err error) {
	if c.unfold == nil || c.fold == nil {
		c.init()
	}

	defer func() {
		if err != nil {
			c.fold = nil
			c.unfold = nil
		}
	}()

	if err = c.unfold.SetTarget(to); err != nil {
		return err
	}

	defer c.unfold.Reset()
	return c.fold.Fold(from)
}

// Convert transforms the value of from into to, by translating the structure
// from into a set of events (go-structform.Visitor) that can applied to the
// value given by to.
// The operation fails if the values are not compatible (for example trying to
// convert an object into an int).
// To `to` parameter must be a pointer, otherwise the operation fails.
//
// Go structures can influence the transformation via tags using the `struct` namespace.
// If the tag is missing, the structs field names are used. Additional options are separates by `,`.
// options:
//   `squash`, `inline`: The fields in the child struct/map are assumed to be inlined, without reporting a sub-oject.
//   `omitempty`: The field is not converted if it is "empty". For example an
//                empty string, array or `nil` pointers are assumed to be empty. In either case the original value
//                in `to` will not be overwritten.
//   `omit`, `-`: Do not convert the field.
func Convert(to, from interface{}) (err error) {
	c := convPool.Get().(*Converter)
	defer convPool.Put(c)
	return c.Convert(to, from)
}

func foldTimestamp(in *time.Time, v structform.ExtVisitor) error {
	extra, sec := timestampToBits(*in)

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

func (u *timeUnfolder) OnInt(ctx gotype.UnfoldCtx, in int64) error {
	return u.OnUint(ctx, uint64(in))
}
func (u *timeUnfolder) OnFloat(ctx gotype.UnfoldCtx, f float64) error {
	return u.OnUint(ctx, uint64(f))
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

func (u *timeUnfolder) OnArrayFinished(ctx gotype.UnfoldCtx) error {
	defer ctx.Done()

	if u.st != timeUnfoldWaitDone {
		return errors.New("unexpected timestamp array closed")
	}

	u.st = timeUnfoldDone

	ts, err := bitsToTimestamp(u.a, u.b)
	if err != nil {
		return err
	}
	*u.to = ts

	return nil
}

func timestampToBits(ts time.Time) (uint64, uint64) {
	var (
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

	return extra, sec
}

func bitsToTimestamp(extra, sec uint64) (time.Time, error) {
	var ts time.Time

	version := (extra >> 56) & 0xff
	if version != 0 {
		return ts, fmt.Errorf("invalid timestamp [%x, %x]", extra, sec)
	}

	nsec := uint32(extra)
	off := int16(extra >> 32)
	ts = time.Unix(int64(sec), int64(nsec))

	// adjust location by offset. time.Unix creates a timestamp in the local zone
	// by default. Only change this if off does not match the local zone it's offset.
	if off == -1 {
		ts = ts.UTC()
	} else if off != 0 {
		_, locOff := ts.Zone()
		if off != int16(locOff/60) {
			ts = ts.In(time.FixedZone("", int(off*60)))
		}
	}

	return ts, nil
}
