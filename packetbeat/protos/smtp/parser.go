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

package smtp

import (
	"bytes"
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type parser struct {
	buf     streambuf.Buffer
	config  *parserConfig
	pub     *transPub
	message *message
	state   parseState
	conn    *connection

	onMessage func(m *message) error
}

type parserConfig struct {
	maxBytes int
}

type message struct {
	applayer.Message

	// Request
	command common.NetString
	param   common.NetString

	// Response
	statusCode    uint
	statusPhrases []common.NetString

	raw streambuf.Buffer

	// list element use by 'transactions' for correlation
	next *message
}

const (
	// Responses are at least len("XXX\r\n") in size
	constMinRespSize  = 5
	constRespCodeSize = 3
	constPhraseOffset = 4
	constStatusSize   = 3
	constCRLFSize     = 2
)

var (
	constCRLF = []byte("\r\n")
	constEOD  = []byte(".\r\n")
	constMAIL = []byte("MAIL")
	constRCPT = []byte("RCPT")
	constDATA = []byte("DATA")

	// ErrStreamTooLarge is returned if stream exceeds max allowed size on append.
	ErrStreamTooLarge = errors.New("Stream data too large")
)

func (p *parser) init(
	cfg *parserConfig,
	pub *transPub,
	conn *connection,
	onMessage func(*message) error,
) {
	*p = parser{
		buf:       streambuf.Buffer{},
		config:    cfg,
		pub:       pub,
		conn:      conn,
		onMessage: onMessage,
	}
}

func (p *parser) append(data []byte) error {
	_, err := p.buf.Write(data)
	if err != nil {
		return err
	}

	if p.config.maxBytes > 0 && p.buf.Total() > p.config.maxBytes {
		return ErrStreamTooLarge
	}
	return nil
}

// process parses message(s) from the stream.
// If yield is set, parsing is driven by the syncer,
// process will return after parsing a message.
func (p *parser) process(ts time.Time, dir uint8, yield bool) error {
	if p.state == stateUnsynced {
		// First message of this stream is a response, syncer just got
		// done before getting to it
		p.state = stateCommand
	}

	for p.buf.Total() > 0 {
		if p.message == nil {
			// allocate new message object to be used by parser with current timestamp
			p.message = p.newMessage(ts, dir)
		}

		msg, err := p.parse()
		if err != nil {
			return err
		}
		if msg == nil {
			return nil // wait for more data
		}

		// remove processed bytes from buffer
		p.buf.Reset()
		p.message = nil

		// call message handler callback
		if err := p.onMessage(msg); err != nil {
			return err
		}

		if yield {
			break
		}
	}

	return nil
}

func (p *parser) newMessage(ts time.Time, dir uint8) *message {
	return &message{
		Message: applayer.Message{
			Ts:        ts,
			Direction: applayer.NetDirection(dir),
		},
	}
}

// SMTP server's reply message (if any) is separated from the
// 3-digit status code with either a space or a hyphen. The latter
// indicates that more lines are to follow.
func (*parser) isResponseComplete(raw []byte) bool {
	switch raw[constStatusSize] {
	case ' ', '\r':
		return true
	case '-':
		return false
	default:
		debugf("Failed to parse SMTP status code: %s",
			string(raw[:len(raw)-constCRLFSize]))
	}

	return true
}

func isResponseCode(raw []byte) bool {
	tail := constRespCodeSize

	for i := 0; i < tail; i++ {
		if raw[i] < '0' || raw[0] > '9' {
			return false
		}
	}
	if raw[tail] != ' ' && raw[tail] != '-' && raw[tail] != '\r' {
		return false
	}

	return true
}

func btoi(raw []byte) uint {
	var res uint
	for _, b := range raw {
		res = res*10 + uint(b-'0')
	}
	return res
}

// Extract path (email address) from a MAIL/RCPT request param
func getPath(param []byte) []byte {
	l := bytes.IndexByte(param, byte('<'))
	r := bytes.IndexByte(param, byte('>'))
	if r > l {
		return param[l+1 : r]
	}
	return nil
}

// Make sure client's stateData is not invalidated by server's
// response
func (p *parser) verifyStateData() {
	m := p.message

	// Only on error response
	if m.statusCode < 400 {
		return
	}

	st := p.conn.streams[m.Direction^1]

	if st != nil && st.parser.state == stateData {
		debugf("Resetting request parser to stateCommand")
		st.parser.state = stateCommand
	}
}

func (p *parser) parse() (*message, error) {
	m := p.message

	for {
		var err error

		raw, err := p.buf.CollectUntil(constCRLF)
		if err != nil {
			// Get more data
			return nil, nil
		}

		nbytes := len(raw)
		var words [][]byte
		isCode := nbytes >= constMinRespSize &&
			isResponseCode(raw[:constMinRespSize])

		if nbytes < constMinRespSize || !isCode || p.state == stateData {
			// Request
			m.IsRequest = true
			if p.state != stateData {
				words = bytes.SplitN(raw[:nbytes-constCRLFSize], []byte(" "), 2)
				m.command = bytes.ToUpper(words[0])
				if len(words) == 2 {
					m.param = words[1]
				}
				if p.pub.sendRequest {
					if err = m.raw.Append(raw); err != nil {
						return nil, err
					}
				}
			}
		} else {
			// Response
			m.statusCode = btoi(raw[:3])
			if nbytes > constMinRespSize+1 {
				m.statusPhrases = append(m.statusPhrases,
					raw[constPhraseOffset:nbytes-constCRLFSize])
			}
			if p.pub.sendResponse {
				if err = m.raw.Append(raw); err != nil {
					return nil, err
				}
			}
		}

		m.Size += uint64(nbytes)

		switch p.state {

		case stateCommand:
			if m.IsRequest {
				if bytes.Equal(words[0], constDATA) {
					p.state = stateData
				}
			} else {
				if !p.isResponseComplete(raw) {
					continue
				}
				p.verifyStateData()
			}

		case stateData: // request only state
			if !bytes.Equal(raw, constEOD) {
				if p.pub.sendDataHeaders || p.pub.sendDataBody {
					if err = m.raw.Append(raw); err != nil {
						return nil, err
					}
				}
				continue
			} else {
				m.command = constEOD
			}

			p.state = stateCommand
		}

		break
	}

	return m, nil
}
