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

package memcache

// Generic memcache parser types and helper functions for use by binary and text parser protocol parsers.

import (
	"time"

	"github.com/elastic/beats/v8/libbeat/common/streambuf"
)

type parserConfig struct {
	maxValues        int
	maxBytesPerValue int
	parseUnknown     bool
}

type parser struct {
	state   parserState
	message *message
	config  *parserConfig
}

type parserState uint8

const (
	parseStateCommand parserState = iota
	parseStateTextCommand
	parseStateBinaryCommand
	parseStateData
	parseStateDataBinary
	parseStateIncompleteData
	parseStateIncompleteDataBinary
	parseStateFailing
)

type parserStateFn func(parser *parser, buf *streambuf.Buffer) parseResult

type argParser func(parser *parser, hdr, buf *streambuf.Buffer) error

type parseResult struct {
	err error
	msg *message
}

// module init
func init() {
	// link parseCommand (break compile time initialization loop check)
	parseCommand = doParseCommand
}

func newParser(config *parserConfig) *parser {
	var p parser
	p.init(config)
	return &p
}

func (p *parser) init(config *parserConfig) {
	p.state = parseStateCommand
	p.message = nil
	p.config = config
}

func (p *parser) reset() {
	debug("parser(%p) reset", p)
	p.init(p.config)
}

func (p *parser) parse(buf *streambuf.Buffer, ts time.Time) (*message, error) {
	if p.message == nil {
		p.message = newMessage(ts)
	}

	res := p.dispatch(p.state, buf)
	return res.msg, res.err
}

func (p *parser) dispatch(state parserState, buf *streambuf.Buffer) parseResult {
	var f parserStateFn
	switch state {
	case parseStateCommand:
		f = parseCommand
	case parseStateTextCommand:
		f = parseTextCommand
	case parseStateBinaryCommand:
		f = parseBinaryCommand
	case parseStateData:
		f = parseData
	case parseStateIncompleteData:
		f = parseData
	case parseStateDataBinary:
		f = parseDataBinary
	case parseStateIncompleteDataBinary:
		f = parseDataBinary
	case parseStateFailing:
		f = parseFailing
	}
	return f(p, buf)
}

func (p *parser) needMore() parseResult {
	return parseResult{nil, nil}
}

func (p *parser) yield(nbytes int) parseResult {
	p.state = parseStateCommand
	msg := p.message
	msg.Size = uint64(nbytes - int(msg.bytesLost))
	p.message = nil
	debug("yield(%p) memcache message type %v", p, msg.command.code)
	return parseResult{nil, msg}
}

func (p *parser) yieldNoData(buf *streambuf.Buffer) parseResult {
	return p.yield(buf.BufferConsumed())
}

func (p *parser) failing(err error) parseResult {
	p.state = parseStateFailing
	return parseResult{err, nil}
}

func (p *parser) contWith(buf *streambuf.Buffer, state parserState) parseResult {
	p.state = state
	return p.dispatch(state, buf)
}

func (p *parser) contWithShallow(
	buf *streambuf.Buffer,
	fn parserStateFn,
) parseResult {
	return fn(p, buf)
}

func (p *parser) appendMessageData(data []byte) {
	msg := p.message
	if p.config.maxValues != 0 {
		msg.data = memcacheData{data}
		if len(msg.data.data) > p.config.maxBytesPerValue {
			msg.data.data = msg.data.data[0:p.config.maxBytesPerValue]
		}
		msg.values = append(msg.values, msg.data)
	}
	msg.countValues++
}

func parseFailing(parser *parser, buf *streambuf.Buffer) parseResult {
	return parser.failing(errParserCaughtInError)
}

// required to break initialization loop warning
var parseCommand parserStateFn

func doParseCommand(parser *parser, buf *streambuf.Buffer) parseResult {
	// check if binary + text command and dispatch
	if !buf.Avail(2) {
		return parser.needMore()
	}
	magic := buf.Bytes()[0]
	isBinary := magic == memcacheMagicRequest || magic == memcacheMagicResponse
	if isBinary {
		return parser.contWith(buf, parseStateBinaryCommand)
	}
	return parser.contWith(buf, parseStateTextCommand)
}

func argparseNoop(p *parser, h, b *streambuf.Buffer) error {
	return nil
}
