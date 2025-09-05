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
	"context"
	"errors"
	"fmt"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
	"encoding/hex"
	"strings"


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

type ConversionFunction func(inBytes []byte) (string, error)

var DefaultConverter ConversionFunction = convertToPrintableAsciiDropping

func convertToHex(inBytes []byte) (string, error) {
	return hex.EncodeToString(inBytes), nil
}
func convertToPrintableAscii(inBytes []byte) (string, error) {
	replacementChar := '.' // Character to replace non-ASCII bytes with

	var result string
	for _, b := range inBytes {
		if b >= 32 && b <= 126 { // Check if the byte is a printable ASCII character
			result += string(b)
		} else {
			result += string(replacementChar)
		}
	}
	return result, nil
}

func convertToPrintableAsciiDropping(inBytes []byte) (string, error) {
	var result string
	for _, b := range inBytes {
		if b >= 32 && b <= 126 { // Check if the byte is a printable ASCII character
			result += string(b)
		}
	}
	return result, nil
}

type Encoding struct {
	Enabled bool `config:"enabled"`
	// default is convert-bytes-to-hex
	Convert *string `config:"convert-by"`
	convertFunc ConversionFunction
}


// FilterParser accepts a list of matchers to determine if a line
// should be kept or not. If one of the patterns matches the
// contents of the message, it is returned to the next reader.
// If not, the message is dropped.
type Parser struct {
	ctx      ctxtool.CancelContext
	logger   *logp.Logger
	r        reader.Reader
	coding   Encoding
}

func NewParser(r reader.Reader, c *Config, logger *logp.Logger) *Parser {
	// parse the config
	return &Parser{
		ctx:      ctxtool.WithCancelContext(context.Background()),
		logger:   logger.Named("binary_parser"),
		r:        r,
		coding:   Encoding {
			Enabled: true,
			convertFunc: DefaultConverter,
		},
	}
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
		message, err := p.r.Next()
		if err != nil {
			return message, err
		}

		content := message.Content
		p.logger.Debug("this message has %d bytes in it (or error %v)", len(content), err)
		return message, nil
	}
	return reader.Message{}, p.ctx.Err()
}

func (p *Parser) Close() error {
	p.ctx.Cancel()
	return nil
}
var ErrNotEnoughBytes = errors.New("not enough data in slice")

type BinaryDecodeError struct {
	msg string
}

func (d BinaryDecodeError) Error() string {
	return d.msg
}

func NewBinaryDecodeError(msg string) BinaryDecodeError {
	return BinaryDecodeError{
		msg: msg,
	}
}

type BinaryConfigError struct {
	msg string
}

func (d BinaryConfigError) Error() string {
	return d.msg
}

func NewBinaryConfigError(msg string) BinaryConfigError {
	return BinaryConfigError{
		msg: msg,
	}
}

func (b *Encoding) Validate() error {

	if b.Convert != nil {
		if strings.Contains(*b.Convert, "ascii") {
			if strings.Contains(*b.Convert, "drop") {
				b.convertFunc = convertToPrintableAsciiDropping
			} else {
				b.convertFunc = convertToPrintableAscii
			}
		} else if strings.Contains(*b.Convert, "hex") {
			b.convertFunc = convertToHex
		}
	}

	if b.convertFunc == nil {
		b.convertFunc = DefaultConverter
	}

	return nil
}

func DefaultEncoding() Encoding {
	def := "ascii-dropping"
	return Encoding {
		Enabled: false,
		Convert: &def,
		convertFunc: DefaultConverter,
	}
}

type binenc struct {}

// func (b binary) Reset() {}
// func (b binary) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) { }

func (binenc) NewDecoder() *encoding.Decoder {
	fmt.Println("Creating new Binary Decoder")
	return &encoding.Decoder{
		Transformer: transform.Nop,
	}
}

func (binenc) NewEncoder() *encoding.Encoder {
	fmt.Println("Creating new Binary Encoder")
	return &encoding.Encoder{
		Transformer: transform.Nop,
	}
}

var binaryEncoder = binenc{}

func NewDecoder() *encoding.Decoder {
	return binaryEncoder.NewDecoder()
}
func NewEncoder() *encoding.Encoder {
	return binaryEncoder.NewEncoder()
}
