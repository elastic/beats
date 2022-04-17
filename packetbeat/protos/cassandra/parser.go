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

package cassandra

import (
	"errors"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common/streambuf"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/packetbeat/protos/applayer"

	gocql "github.com/menderesk/beats/v7/packetbeat/protos/cassandra/internal/gocql"
)

type parser struct {
	buf       streambuf.Buffer
	config    *parserConfig
	framer    *gocql.Framer
	message   *message
	onMessage func(m *message) error
}

type parserConfig struct {
	maxBytes   int
	compressor gocql.Compressor
	ignoredOps map[gocql.FrameOp]bool
}

// check whether this ops is enabled or not
func (p *parser) CheckFrameOpsIgnored() bool {
	return p.config.ignoredOps[p.framer.Header.Op]
}

type message struct {
	applayer.Message

	// indicator for parsed message being complete or requires more messages
	// (if false) to be merged to generate full message.
	isComplete bool

	failed  bool
	ignored bool

	data   map[string]interface{}
	header map[string]interface{}
	// list element use by 'transactions' for correlation
	next *message

	results transactions
}

// Error code if stream exceeds max allowed size on append.
var (
	errStreamTooLarge = errors.New("Stream data too large")
	isDebug           = false
)

func (p *parser) init(
	cfg *parserConfig,
	onMessage func(*message) error,
) {
	*p = parser{
		buf:       streambuf.Buffer{},
		config:    cfg,
		onMessage: onMessage,
	}

	isDebug = logp.IsDebug("cassandra")
}

func (p *parser) append(data []byte) error {
	_, err := p.buf.Write(data)
	if err != nil {
		return err
	}

	if p.config.maxBytes > 0 && p.buf.Total() > p.config.maxBytes {
		return errStreamTooLarge
	}
	return nil
}

func (p *parser) feed(ts time.Time, data []byte) error {
	if err := p.append(data); err != nil {
		return err
	}

	for p.buf.Total() > 0 {
		if p.message == nil {
			// allocate new message object to be used by parser with current timestamp
			p.message = p.newMessage(ts)
		}

		msg, err := p.parse()
		if err != nil {
			return err
		}
		if msg == nil {
			break // wait for more data
		}

		// reset buffer and message -> handle next message in buffer
		p.buf.Reset()
		p.message = nil

		// call message handler callback
		if err := p.onMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

func (p *parser) newMessage(ts time.Time) *message {
	return &message{
		Message: applayer.Message{
			Ts: ts,
		},
	}
}

func (p *parser) parserBody() (bool, error) {
	headLen := p.framer.Header.HeadLength
	bdyLen := p.framer.Header.BodyLength
	if bdyLen <= 0 {
		return true, nil
	}

	// let's wait for enough buf
	debugf("bodyLength: %d", bdyLen)
	if !p.buf.Avail(bdyLen) {
		if isDebug {
			debugf("buf not enough for body, waiting for more, return")
		}
		return false, nil
	}

	// check if the ops already ignored
	if p.message.ignored {
		if isDebug {
			debugf("message marked to be ignored, let's do this")
		}
		p.buf.Collect(bdyLen)
	} else {
		// start to parse body
		data, err := p.framer.ReadFrame()
		if err != nil {
			// if the frame parsed failed, should ignore the whole message
			p.framer = nil
			return false, err
		}

		// dealing with un-parsed content
		frameParsedLength := p.buf.BufferConsumed()

		// collect leftover
		unParsedSize := bdyLen + headLen - frameParsedLength
		if unParsedSize > 0 {
			if !p.buf.Avail(unParsedSize) {
				err := errors.New("should be enough bytes for cleanup,but not enough")
				logp.Err("Finishing frame failed with: %v", err)
				return false, err
			}

			p.buf.Collect(unParsedSize)
		}

		p.message.data = data
	}

	finalCollectedFrameLength := p.buf.BufferConsumed()
	if finalCollectedFrameLength-headLen != bdyLen {
		logp.Err("body_length:%d, head_length:%d, all_consumed:%d",
			bdyLen, headLen, finalCollectedFrameLength)
		return false, errors.New("data messed while parse frame body")
	}

	return true, nil
}

func (p *parser) parse() (*message, error) {
	// if p.frame is nil then create a new framer, or continue to process the last message
	if p.framer == nil {
		if isDebug {
			debugf("start new framer")
		}
		p.framer = gocql.NewFramer(&p.buf, p.config.compressor)
	}

	// check if the frame header were parsed or not
	if p.framer.Header == nil {
		if isDebug {
			debugf("start to parse header")
		}
		if !p.buf.Avail(9) {
			debugf("not enough head bytes, ignore")
			return nil, nil
		}

		_, err := p.framer.ReadHeader()
		if err != nil {
			logp.Err("%v", err)
			p.framer = nil
			return nil, err
		}
	}

	// check if the ops need to be ignored
	if p.CheckFrameOpsIgnored() {
		// as we already ignore the content, we now mark the result is ignored
		p.message.ignored = true
		if isDebug {
			debugf("Ops: %s was marked to be ignored, ignoring, request:%v", p.framer.Header.Op.String(), p.framer.Header.Version.IsRequest())
		}
	}

	msg := p.message

	finished, err := p.parserBody()
	if err != nil {
		return nil, err
	}

	// ignore and wait for more data
	if !finished {
		return nil, nil
	}

	dir := applayer.NetOriginalDirection

	isRequest := true
	if p.framer.Header.Version.IsResponse() {
		dir = applayer.NetReverseDirection
		isRequest = false
	}

	msg.Size = uint64(p.buf.BufferConsumed())
	msg.IsRequest = isRequest
	msg.Direction = dir

	msg.header = p.framer.Header.ToMap()

	if msg.IsRequest {
		p.message.results.requests.append(msg)
	} else {
		p.message.results.responses.append(msg)
	}
	p.framer = nil
	return msg, nil
}
