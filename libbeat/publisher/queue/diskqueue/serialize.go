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

// Encoding / decoding routines adapted from
// libbeat/publisher/queue/spool/codec.go.

package diskqueue

import (
	"bytes"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

type eventEncoder struct {
	buf    bytes.Buffer
	folder *gotype.Iterator
}

type eventDecoder struct {
	buf []byte

	parser   *json.Parser
	unfolder *gotype.Unfolder
}

type entry struct {
	Timestamp int64
	Flags     uint8
	Meta      common.MapStr
	Fields    common.MapStr
}

func newEventEncoder() *eventEncoder {
	e := &eventEncoder{}
	e.reset()
	return e
}

func (e *eventEncoder) reset() {
	e.folder = nil

	visitor := json.NewVisitor(&e.buf)
	// This can't return an error: NewIterator is deterministic based on its
	// input, and doesn't return an error when called with valid options. In
	// this case the options are hard-coded to fixed values, so they are
	// guaranteed to be valid and we can safely proceed.
	folder, _ := gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeTimestampEncoder(),
			codec.MakeBCTimestampEncoder(),
		),
	)

	e.folder = folder
}

func (e *eventEncoder) encode(event *publisher.Event) ([]byte, error) {
	e.buf.Reset()

	err := e.folder.Fold(entry{
		Timestamp: event.Content.Timestamp.UTC().UnixNano(),
		Flags:     uint8(event.Flags),
		Meta:      event.Content.Meta,
		Fields:    event.Content.Fields,
	})
	if err != nil {
		e.reset()
		return nil, err
	}

	// Copy the encoded bytes to a new array owned by the caller.
	bytes := e.buf.Bytes()
	result := make([]byte, len(bytes))
	copy(result, bytes)

	return result, nil
}

func newEventDecoder() *eventDecoder {
	d := &eventDecoder{}
	d.reset()
	return d
}

func (d *eventDecoder) reset() {
	// When called on nil, NewUnfolder deterministically returns a nil error,
	// so it's safe to ignore the error result.
	unfolder, _ := gotype.NewUnfolder(nil)

	d.unfolder = unfolder
	d.parser = json.NewParser(unfolder)
}

// Buffer prepares the read buffer to hold the next event of n bytes.
func (d *eventDecoder) Buffer(n int) []byte {
	if cap(d.buf) > n {
		d.buf = d.buf[:n]
	} else {
		d.buf = make([]byte, n)
	}
	return d.buf
}

func (d *eventDecoder) Decode() (publisher.Event, error) {
	var (
		to  entry
		err error
	)

	d.unfolder.SetTarget(&to)
	defer d.unfolder.Reset()

	err = d.parser.Parse(d.buf)

	if err != nil {
		d.reset() // reset parser just in case
		return publisher.Event{}, err
	}

	return publisher.Event{
		Flags: publisher.EventFlags(to.Flags),
		Content: beat.Event{
			Timestamp: time.Unix(0, to.Timestamp),
			Fields:    to.Fields,
			Meta:      to.Meta,
		},
	}, nil
}
