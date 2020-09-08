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
	"encoding/binary"
	"hash/crc32"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

// ChecksumType specifies what checksum algorithm the queue should use to
// verify its data frames.
type ChecksumType int

// ChecksumTypeNone: Don't compute or verify checksums.
// ChecksumTypeCRC32: Compute the checksum with the Go standard library's
//   "hash/crc32" package.
const (
	ChecksumTypeNone = iota

	ChecksumTypeCRC32
)

type frameEncoder struct {
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

func newFrameEncoder(checksumType ChecksumType) *frameEncoder {
	e := &frameEncoder{checksumType: checksumType}
	e.reset()
	return e
}

func (e *frameEncoder) reset() {
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

func (e *frameEncoder) encode(event *publisher.Event) ([]byte, error) {
	e.buf.Reset()

	var flags uint8
	// TODO: handle guaranteed send?
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

func computeChecksum(data []byte, checksumType ChecksumType) uint32 {
	switch checksumType {
	case ChecksumTypeNone:
		return 0
	case ChecksumTypeCRC32:
		hash := crc32.NewIEEE()
		frameLength := uint32(len(data) + frameMetadataSize)
		binary.Write(hash, binary.LittleEndian, &frameLength)
		hash.Write(data)
		return hash.Sum32()
	default:
		panic("segmentReader: invalid checksum type")
	}
}
