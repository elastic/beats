package cassandra

import (
	"errors"
	"time"

	"fmt"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos/applayer"
	. "github.com/elastic/beats/packetbeat/protos/cassandra/internal/gocql"
)

type parser struct {
	buf       streambuf.Buffer
	config    *parserConfig
	message   *message
	onMessage func(m *message) error
}

type parserConfig struct {
	maxBytes   int
	compressor Compressor
	ignoredOps map[string]interface{}
}

type message struct {
	applayer.Message

	// indicator for parsed message being complete or requires more messages
	// (if false) to be merged to generate full message.
	isComplete bool

	failed bool
	data   map[string]interface{}
	header map[string]interface{}
	// list element use by 'transactions' for correlation
	next *message

	transactionTimeout time.Duration

	results transactions
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

	isDebug = logp.IsDebug("cassandra")

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

	if !p.buf.Avail(9) {
		logp.Err("not enough bytes, ignore")
		p.message = nil
		return nil, nil
	}

	framer := NewFramer(&p.buf, p.config.compressor)
	head, err := framer.ReadHeader()
	if err != nil {
		logp.Err("%v", err)
		p.message = nil
		return nil, nil
	}

	//check if the ops already ignored
	if p.config.ignoredOps != nil && len(p.config.ignoredOps) > 0 {

		v := p.config.ignoredOps[head.Op.String()]
		if v != nil {
			logp.Debug("cassandra", fmt.Sprintf("Ops: %s was marked to be ignored, ignoring", head.Op.String()))
			p.message = nil
			return nil, nil
		}
	}

	if !p.buf.Avail(head.Length) {
		logp.Err("not enough bytes for frame body, ignore")
		p.message = nil
		return nil, nil
	}

	data, err := framer.ReadFrame()

	frameLength := p.buf.BufferConsumed()
	if err != nil {
		p.message = nil
		logp.Err("%v", err)
		return nil, nil
	}

	// collect leftover
	leftDataSize := head.Length + 9 - frameLength
	if leftDataSize > 0 {
		p.buf.Collect(leftDataSize)

	}

	dir := applayer.NetOriginalDirection

	isRequest := true
	if head.Version.IsResponse() {
		dir = applayer.NetReverseDirection
		isRequest = false
	}

	msg := p.message
	msg.Size = uint64(p.buf.BufferConsumed())
	msg.IsRequest = isRequest
	msg.Direction = dir

	msg.data = data
	msg.header = head.ToMap()

	if msg.IsRequest {
		p.message.results.requests.append(msg)
	} else {
		p.message.results.responses.append(msg)
	}

	return msg, nil
}
