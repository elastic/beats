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

// Package applayer provides common definitions with common fields
// for use with application layer protocols among beats.
package applayer

import (
	"errors"
	"time"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/streambuf"

	"github.com/menderesk/beats/v7/packetbeat/pb"
)

// A Message its direction indicator
type NetDirection uint8

const (
	// Message due to a response by server
	NetReverseDirection NetDirection = 0

	// Message was send by client
	NetOriginalDirection NetDirection = 1
)

// Transport type indicator. One of TransportUdp or TransportTcp
type Transport uint8

const (
	TransportUDP Transport = iota
	TransportTCP
)

// String returns the transport type its textual representation.
func (t Transport) String() string {
	switch t {
	case TransportUDP:
		return "udp"
	case TransportTCP:
		return "tcp"
	default:
		return "invalid"
	}
}

// A Stream provides buffering data if stream based protocol is used.
// Use Init to initialize a stream with en empty buffer and buffering limit.
// A Stream its zero value is a valid unlimited stream buffer.
type Stream struct {
	// Buf provides the buffering with parsing support
	Buf streambuf.Buffer

	// MaxDataInStream sets the maximum number of bytes held in buffer.
	// If limit is reached append function will return an error.
	MaxDataInStream int
}

// A Transaction defines common fields for all application layer protocols.
type Transaction struct {
	// Type is the name of the application layer protocol transaction be represented.
	Type string

	// Transaction source and destination IPs and Ports.
	Tuple common.IPPortTuple

	// Transport layer type
	Transport Transport

	// Src describes the transaction source/initiator endpoint
	Src common.Endpoint

	// Dst describes the transaction destination endpoint
	Dst common.Endpoint

	// Ts sets the transaction its initial timestamp
	Ts TransactionTimestamp

	// EndTime is the time the transaction ended.
	EndTime time.Time

	// Status of final transaction
	Status string // see libbeat/common/statuses.go

	// Notes holds a list of interesting events and errors encountered when
	// processing the transaction
	Notes []string

	// BytesIn is the number of bytes returned by destination endpoint
	BytesIn uint64

	// BytesOut is the number of bytes send by source endpoint to destination endpoint
	BytesOut uint64
}

// TransactionTimestamp defines a transaction its initial timestamps as unix
// timestamp in milliseconds and time.Time struct.
type TransactionTimestamp struct {
	Millis int64
	Ts     time.Time
}

// Message defines common application layer message fields. Some of these fields
// are required to initialize a Transaction (see (*Transaction).InitWithMsg).
type Message struct {
	Ts           time.Time
	Tuple        common.IPPortTuple
	Transport    Transport
	CmdlineTuple *common.ProcessTuple
	Direction    NetDirection
	IsRequest    bool
	Size         uint64
	Notes        []string
}

// Error code if stream exceeds max allowed size on Append.
var ErrStreamTooLarge = errors.New("Stream data too large")

// Init initializes a stream with an empty buffer and max size. Calling Init
// twice will fully re-initialize the buffer, such that calling Init before putting
// the stream in some object pool, no memory will be leaked.
func (stream *Stream) Init(maxDataInStream int) {
	stream.MaxDataInStream = maxDataInStream
	stream.Buf = streambuf.Buffer{}
}

// Reset will remove all bytes already read from the buffer.
func (stream *Stream) Reset() {
	stream.Buf.Reset()
}

// Append adds data to the Stream its buffer. If internal buffer is nil, data
// will be retained as is. Use Write if you don't intend to retain the buffer in
// the stream.
func (stream *Stream) Append(data []byte) error {
	err := stream.Buf.Append(data)
	if err != nil {
		return err
	}

	if stream.MaxDataInStream > 0 && stream.Buf.Total() > stream.MaxDataInStream {
		return ErrStreamTooLarge
	}
	return nil
}

// Write copies data to the Stream its buffer. The data slice will not be
// retained by the buffer.
func (stream *Stream) Write(data []byte) (int, error) {
	n, err := stream.Buf.Write(data)
	if err != nil {
		return n, err
	}

	if stream.MaxDataInStream > 0 && stream.Buf.Total() > stream.MaxDataInStream {
		return n, ErrStreamTooLarge
	}
	return n, nil
}

// Init initializes some common fields. ResponseTime, Status, BytesIn and
// BytesOut are initialized to zero and must be filled by application code.
func (t *Transaction) Init(
	typ string,
	tuple common.IPPortTuple,
	transport Transport,
	direction NetDirection,
	time time.Time,
	cmdline *common.ProcessTuple,
	notes []string,
) {
	t.Type = typ
	t.Transport = transport
	t.Tuple = tuple

	// transactions have microseconds resolution
	t.Ts.Ts = time
	t.Ts.Millis = int64(time.UnixNano() / 1000)
	t.Src, t.Dst = common.MakeEndpointPair(tuple.BaseTuple, cmdline)
	t.Notes = notes

	if direction == NetReverseDirection {
		t.Src, t.Dst = t.Dst, t.Src
	}
}

// InitWithMsg initializes some common fields from a Message. ResponseTime,
// Status, BytesIn and BytesOut are initialized to zero and must be filled by
// application code.
func (t *Transaction) InitWithMsg(
	typ string,
	msg *Message,
) {
	t.Init(
		typ,
		msg.Tuple,
		msg.Transport,
		msg.Direction,
		msg.Ts,
		msg.CmdlineTuple,
		nil,
	)
}

// Event fills common event fields.
func (t *Transaction) Event(event *beat.Event) error {
	event.Timestamp = t.Ts.Ts

	pbf := pb.NewFields()
	pbf.SetSource(&t.Src)
	pbf.SetDestination(&t.Dst)
	pbf.Source.Bytes = int64(t.BytesIn)
	pbf.Destination.Bytes = int64(t.BytesOut)
	pbf.Event.Dataset = t.Type
	pbf.Event.Start = t.Ts.Ts
	pbf.Event.End = t.EndTime
	pbf.Network.Transport = t.Transport.String()
	pbf.Network.Protocol = pbf.Event.Dataset
	pbf.Error.Message = t.Notes

	fields := event.Fields
	fields[pb.FieldsKey] = pbf
	fields["type"] = pbf.Event.Dataset
	fields["status"] = t.Status
	return nil
}

// AddNotes appends some notes to a message.
func (m *Message) AddNotes(n ...string) {
	m.Notes = append(m.Notes, n...)
}
