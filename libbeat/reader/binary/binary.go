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

package binary

import (
	"bufio"
	"context"
	"fmt"
	"io"

	g_binary "encoding/binary"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
)

type Config struct {
	offset uint `config: "offset" validate: "required"`
	length uint `config: "length" validate: "required"`
	byteOrder string `config: "byteOrder"`
}

func DefaultConfig() Config {
	return Config{
		offset: 0,
		length: 4,
		byteOrder: "BigEndian",
	}
}

type Encoding struct {
	offset uint
	numBytes uint
	decoder g_binary.ByteOrder
}

// FilterParser accepts a list of matchers to determine if a line
// should be kept or not. If one of the patterns matches the
// contents of the message, it is returned to the next reader.
// If not, the message is dropped.
type Parser struct {
	ctx      ctxtool.CancelContext
	logger   *logp.Logger
	r        *bufio.Reader
	coding   Encoding
}

func NewParser(r io.Reader, c *Config, logger *logp.Logger) *Parser {
	return &Parser{
		ctx:      ctxtool.WithCancelContext(context.Background()),
		logger:   logger.Named("binary_parser"),
		r:        bufio.NewReader(r),
		coding:   Encoding {
			offset:   2,
			numBytes:   2,
			decoder:  g_binary.BigEndian,
		},
	}
}

func (p *Parser) get_length(buf []byte) (length uint, err error) {
	if uint(len(buf)) < p.coding.numBytes {
		return 0, fmt.Errorf("Not enough bytes (%d) -- expected %d", len(buf), p.coding.numBytes)
	}

	buffy := buf[:p.coding.numBytes]
	n, err := g_binary.Decode(buffy, p.coding.decoder, length)
	if err != nil {
		return length, err
	}
	if uint(n) != p.coding.numBytes {
		return length, fmt.Errorf("Wrong number of bytes consumed: %d (should be %d)",
			n, p.coding.numBytes)
	}

	return length, nil
}

func (p *Parser) Next() (message reader.Message, err error) {
	// discardedOffset accounts for the bytes of discarded messages. The inputs
	// need to correctly track the file offset, therefore if only the matching
	// message size is returned, the offset cannot be correctly updated.
	var discardedOffset int
	defer func() {
		message.Offset = discardedOffset
	}()

	for p.ctx.Err() == nil {
		// peek to the end of the length field
		b, err := p.r.Peek(int(p.coding.offset + p.coding.numBytes))
		if err != nil {
			return message, err
		}
		msgLength,err := p.get_length(b[p.coding.offset:p.coding.numBytes])
		p.logger.Debug("this message has %d bytes in it (or error %v)", msgLength, err)
	}
	return reader.Message{}, io.EOF
}

func (p *Parser) Close() error {
	p.ctx.Cancel()
	return nil
}
