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
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
	"github.com/elastic/go-structform/cborl"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

type SerializationFormat int

const (
	SerializationJSON     SerializationFormat = iota // 0
	SerializationCBOR                                // 1
	SerializationProtobuf                            // 2
)

type eventEncoder struct {
	buf                 bytes.Buffer
	folder              *gotype.Iterator
	serializationFormat SerializationFormat
}

type eventDecoder struct {
	buf                 []byte
	jsonParser          *json.Parser
	cborlParser         *cborl.Parser
	unfolder            *gotype.Unfolder
	serializationFormat SerializationFormat
}

type entry struct {
	Timestamp int64
	Flags     uint32
	Meta      mapstr.M
	Fields    mapstr.M
}

func newEventEncoder(format SerializationFormat) *eventEncoder {
	e := &eventEncoder{}
	e.serializationFormat = format
	e.reset()
	return e
}

func (e *eventEncoder) reset() {
	e.folder = nil

	visitor := cborl.NewVisitor(&e.buf)
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

func (e *eventEncoder) encode(evt interface{}) ([]byte, error) {
	switch v := evt.(type) {
	case publisher.Event:
		if e.serializationFormat != SerializationCBOR {
			return nil, fmt.Errorf("incompatible serialization for type %T. Only CBOR is supported", v)
		}
		return e.encode_publisher_event(v)
	case *messages.Event:
		if e.serializationFormat != SerializationProtobuf {
			return nil, fmt.Errorf("incompatible serialization for type %T. Only Protobuf is supported", v)
		}
		return proto.Marshal(v)
	default:
		return nil, fmt.Errorf("no known serialization format for type %T", v)
	}
}

func (e *eventEncoder) encode_publisher_event(event publisher.Event) ([]byte, error) {
	e.buf.Reset()

	err := e.folder.Fold(entry{
		Timestamp: event.Content.Timestamp.UTC().UnixNano(),
		Flags:     uint32(event.Flags),
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
	d.jsonParser = json.NewParser(unfolder)
	d.cborlParser = cborl.NewParser(unfolder)
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

func (d *eventDecoder) Decode() (interface{}, error) {
	switch d.serializationFormat {
	case SerializationJSON, SerializationCBOR:
		return d.decodeJSONAndCBOR()
	case SerializationProtobuf:
		return d.decodeProtobuf()
	default:
		return nil, fmt.Errorf("unknown serialization format: %d", d.serializationFormat)
	}
}

func (d *eventDecoder) decodeJSONAndCBOR() (publisher.Event, error) {

	var to entry

	err := d.unfolder.SetTarget(&to)
	if err != nil {
		return publisher.Event{}, err
	}
	defer d.unfolder.Reset()

	switch d.serializationFormat {
	case SerializationJSON:
		err = d.jsonParser.Parse(d.buf)
	case SerializationCBOR:
		err = d.cborlParser.Parse(d.buf)
	default:
		err = fmt.Errorf("unknown serialization format: %d", d.serializationFormat)
	}

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

func (d *eventDecoder) decodeProtobuf() (*messages.Event, error) {
	e := messages.Event{}
	err := proto.Unmarshal(d.buf, &e)
	return &e, err
}
