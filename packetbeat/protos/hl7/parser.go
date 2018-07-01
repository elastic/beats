package hl7

import (
	"errors"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type parser struct {
	buf       streambuf.Buffer
	config    *parserConfig
	message   *message
	onMessage func(m *message) error
}

type parserConfig struct {
	maxBytes     int
	NewLineChars string
}

type message struct {
	applayer.Message

	// indicator for parsed message being complete or requires more messages
	// (if false) to be merged to generate full message.
	isComplete bool

	// list element use by 'transactions' for correlation
	next *message

	failed  bool
	content common.NetString
}

// Error code if stream exceeds max allowed size on append.
var (
	ErrStreamTooLarge = errors.New("Stream data too large")
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

func (p *parser) parse() (*message, error) {

	// wait for message being complete
	// HL7 messages end with <fs><cr> (0x1c0x0d)

	buf, err := p.buf.CollectUntil([]byte{0x1c, 0x0d})
	if err == streambuf.ErrNoMoreBytes {
		return nil, nil
	}

	msg := p.message
	msg.Size = uint64(p.buf.BufferConsumed())

	isRequest := true
	hl7Type := ""

	dir := applayer.NetOriginalDirection

	if len(buf) > 0 {

		// First char in an HL7 should be <vt> (0x0b)
		if !(buf[0] == '\v') {
			// Not a well formed hl7 messages
			return nil, nil
		} else {
			buf = buf[1:]
		}

		// Remove the ending <fs><cr> (0x1c0x0d)
		buf = buf[:len(buf)-2]

		// v2 or v3 message
		if string(buf[0]) == "M" {
			hl7Type = "v2"
		} else if string(buf[0]) == "<" {
			hl7Type = "v3"
		} else {
			return nil, nil
		}

		if hl7Type == "v2" {
			// Split into segments
			segments := strings.Split(string(buf[:]), p.config.NewLineChars)

			// Split MSH segment into fields
			msh := strings.Split(segments[0], string(segments[0][3]))

			// If the 8th value in MSH segment contains ACK then it's a response
			isRequest = !strings.Contains(msh[8], "ACK")
			if !isRequest {
				dir = applayer.NetReverseDirection
			}
		} else {
			// If contains <acknowledgement> then it's a response
			isRequest = !strings.Contains(string(buf[:]), "<acknowledgement>")
			if !isRequest {
				dir = applayer.NetReverseDirection
			}
		}
	}

	msg.content = common.NetString(buf)
	msg.IsRequest = isRequest
	msg.Direction = dir

	return msg, nil
}
