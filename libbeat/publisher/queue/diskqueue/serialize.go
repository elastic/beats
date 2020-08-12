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

func dataFrameForEvent(
	event publisher.Event, checksumType ChecksumType,
) []byte {
	buf := bytes.Buffer{}
	//checksum := computeChecksum(frameContent, checksumType)
	panic("TODO: not implemented")
}

func eventForData(data []byte) publisher.Event {
	panic("TODO: not implemented")
}

type encoder struct {
	buf          bytes.Buffer
	folder       *gotype.Iterator
	checksumType ChecksumType
}

type decoder struct {
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

func newEncoder(checksumType ChecksumType) (*encoder, error) {
	e := &encoder{checksumType: checksumType}
	e.reset()
	return e, nil
}

func (e *encoder) reset() {
	e.folder = nil

	visitor := json.NewVisitor(&e.buf)
	folder, err := gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeTimestampEncoder(),
			codec.MakeBCTimestampEncoder(),
		),
	)
	if err != nil {
		panic(err)
	}

	e.folder = folder
}

func (e *encoder) encode(event *publisher.Event) ([]byte, error) {
	e.buf.Reset()

	var flags uint8
	/*if (event.Flags & publisher.GuaranteedSend) == publisher.GuaranteedSend {
		flags = flagGuaranteed
	}*/

	err := e.folder.Fold(entry{
		Timestamp: event.Content.Timestamp.UTC().UnixNano(),
		Flags:     flags,
		Meta:      event.Content.Meta,
		Fields:    event.Content.Fields,
	})
	if err != nil {
		e.reset()
		return nil, err
	}

	return e.buf.Bytes(), nil
}

func newDecoder() *decoder {
	d := &decoder{}
	d.reset()
	return d
}

func (d *decoder) reset() {
	unfolder, err := gotype.NewUnfolder(nil)
	if err != nil {
		panic(err) // can not happen
	}

	d.unfolder = unfolder
	d.parser = json.NewParser(unfolder)
}

// Buffer prepares the read buffer to hold the next event of n bytes.
func (d *decoder) Buffer(n int) []byte {
	if cap(d.buf) > n {
		d.buf = d.buf[:n]
	} else {
		d.buf = make([]byte, n)
	}
	return d.buf
}

func (d *decoder) Decode() (publisher.Event, error) {
	var (
		to       entry
		err      error
		contents = d.buf[1:]
	)

	d.unfolder.SetTarget(&to)
	defer d.unfolder.Reset()

	err = d.parser.Parse(contents)

	if err != nil {
		d.reset() // reset parser just in case
		return publisher.Event{}, err
	}

	var flags publisher.EventFlags
	/*if (to.Flags & flagGuaranteed) != 0 {
		flags |= publisher.GuaranteedSend
	}*/

	return publisher.Event{
		Flags: flags,
		Content: beat.Event{
			Timestamp: time.Unix(0, to.Timestamp),
			Fields:    to.Fields,
			Meta:      to.Meta,
		},
	}, nil
}
