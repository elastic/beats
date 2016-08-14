package lumberjack

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type parser struct {
	buf     streambuf.Buffer
	config  *parserConfig
	message *message

	onMessage func(m *message) error
}

type parserConfig struct {
	maxBytes int
}

type message struct {
	applayer.Message

	// indicator for parsed message being complete or requires more messages
	// (if false) to be merged to generate full message.
	isComplete bool
	ignore     bool

	op    opcode
	seq   uint32
	count uint32
	size  uint32

	// list element use by 'transactions' for correlation
	next *message
}

// Error code if stream exceeds max allowed size on append.
var (
	ErrStreamTooLarge = errors.New("Stream data too large")

	errInvalidVersion = errors.New("unsupported protocol version")
)

type opcode byte

const (
	opUnknown    opcode = 0
	opACK        opcode = 'A'
	opCompressed opcode = 'C'
	opData       opcode = 'D'
	opJSON       opcode = 'J'
	opWindow     opcode = 'W'
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
		msg.Size = uint64(p.buf.BufferConsumed())

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
	if !p.buf.Avail(2) {
		return nil, nil
	}

	version, _ := p.buf.ReadNetUint8At(0)
	frameType, _ := p.buf.ReadNetUint8At(1)

	// only support protocol version 2 with json data frame only
	if version != '2' {
		return nil, errInvalidVersion
	}

	op := opcode(frameType)
	p.message.op = op
	p.message.IsRequest = op != opACK

	switch op {
	case opACK:
		return p.parseACKFrame()
	case opCompressed:
		return p.parseCompressedFrame()
	case opData:
		return p.parseDataFrame()
	case opJSON:
		return p.parseJSONDataFrame()
	case opWindow:
		return p.parseWindowFrame()
	default:
		return nil, fmt.Errorf("unknown opcode: %v", frameType)
	}
}

func (p *parser) parseDataFrame() (*message, error) {
	return nil, nil
}

func (p *parser) parseJSONDataFrame() (*message, error) {
	if !p.buf.Avail(10) {
		return nil, nil
	}

	seq, _ := p.buf.ReadNetUint32At(2)
	sz, _ := p.buf.ReadNetUint32At(6)
	total := int(sz) + 6
	if !p.buf.Avail(total) {
		return nil, nil
	}

	_, err := p.buf.Collect(total)
	if err != nil {
		return nil, err
	}

	p.message.size = sz
	p.message.seq = seq
	return nil, nil
}

func (p *parser) parseACKFrame() (*message, error) {
	if !p.buf.Avail(6) {
		return nil, nil
	}

	seq, _ := p.buf.ReadNetUint32At(2)
	p.message.seq = seq

	return p.message, nil
}

func (p *parser) parseWindowFrame() (*message, error) {
	if !p.buf.Avail(6) {
		return nil, nil
	}

	seq, _ := p.buf.ReadNetUint32At(2)
	p.message.seq = seq

	return p.message, nil
}

func (p *parser) parseCompressedFrame() (*message, error) {
	if !p.buf.Avail(6) {
		return nil, nil
	}

	sz, _ := p.buf.ReadNetUint32At(2)
	total := int(sz) + 6
	if !p.buf.Avail(total) {
		return nil, nil
	}

	_, err := p.buf.Collect(total)
	if err != nil {
		return nil, err
	}

	p.message.size = sz
	return nil, nil
}
