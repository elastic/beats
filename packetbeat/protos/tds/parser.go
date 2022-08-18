package tds

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
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
	// (if false) to be merage to generate full message
	isComplete bool

	// list element use by 'transactions' for correlation
	next *message

	// todo: not sure if this needs to live somewhere else?
	requestType string
}

// Error code if stream exceeds max allowed size on opened.
var (
	ErrStreamTooLarge = errors.New("Stream data too large")
)

func (p *parser) init(cfg *parserConfig, onMessage func(*message) error) {
	logp.Info("parser.init")
	*p = parser{
		buf:       streambuf.Buffer{},
		config:    cfg,
		onMessage: onMessage,
	}
}

func (p *parser) append(data []byte) error {
	logp.Info("parser.append()")
	logp.Info("- data: %s", data)
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
	logp.Info("parser.feed()")
	if err := p.append(data); err != nil {
		return err
	}

	for p.buf.Total() > 0 {
		if p.message == nil {
			// allocate new message object to be used by parser with current timestamp
			p.message = p.newMessage(ts)
			logp.Info("* New message allocated: %v", p.message)
		}

		msg, err := p.parse()
		logp.Info("* Parsed message: %v", msg)
		if err != nil {
			logp.Info("* parse returned error: %s", err)
			return err
		}
		if msg == nil {
			logp.Info("* parse return nil msg")
			break // wait for more data
		}

		// reset buffer and message -> handle next message in buffer
		// If we aren't at the end of the message does the actually rest
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
	logp.Info("parser.newMessage()")
	return &message{
		Message: applayer.Message{
			Ts: ts,
		},
	}
}

func (p *parser) parse() (*message, error) {
	/* 2.2.3.1 Packet Handler
	To implement messages to top of existing, arbitrary transport layers, a packet hander is include as part of the packet.
	The packet header precedes all data within the packet. It is always 8 bytes in length.
	Most importantly, the packet header states the Type and length of the entire packet
	*/

	// Split out a function to process the packet header?
	msg := p.message
	// Empty buffer - do nothing (should we wait for 8 bytes)
	if p.buf.Len() < 2 {
		logp.Info("* Empty(ish) buffer")
		return nil, errors.New("Empty buffer")
	}

	// Second byte dictates whether this is the end of the message
	status, err := p.buf.PeekByteFrom(1)
	if err != nil {
		return nil, err
	}

	if status&0x01 != 0x01 {
		// Not end of message so crack on - no error to return though
		logp.Info("* Not end of message")
		return nil, nil
	}

	// 2nd byte is a bit field - see if this is the end of the message
	/* Spec:
	0x00 "Normal" message.
	0x01 End of message (EOM). The packet is the last packet in the whole request
	0x02 (From client to server) Ignore this evnet (0x01 MUST also be set).
	0x03 RESETCONNECTION
		(Introduced in TDS 7.1)
		(From client to server) Reset this connection before processing event, Only set for event Batch, RPC, or Transaction Manager request. If clients want to set this bit, it MUST be part of the first packet of the message. This signals the server to clean up the environment state of the connection back to the default environment setting, effectively simulating a logout and a subsequent login, and provides server support for connection pooling. This bit SHOULD be ingored if it is set in a packet that is not the first packet of the message
		This status bit MUST NOT be set in conjunction with the RESETCONNECTIONSKIPTRAN bit. Distributed transaction and isolation levels with not be reset
	0x10 RESETCONNECTIONSKIPTRAN
		(Introduced in TDS 7.3)
		(From client to server) Reset thie connection before processing event do not modify the transaction state (the state will remain the same before and after the reset). The transaction in session can be a local transaction that is started from the session or it can be a distributed transaction in which the sesssion is enlisted. This status bit MUST NOT be set in conjunction with the RESETCONNECTION bit.
		Otherwise identical to RESETCONNECTION.
	*/
	logp.Info("* Processing end of message")

	batchType, err := p.buf.PeekByteFrom(0)
	switch batchType {
	case 0x01:
		msg.requestType = "SQL Batch"
		msg.IsRequest = true
	case 0x02:
		msg.requestType = "Pre-TDS7 Login"
		msg.IsRequest = true
	case 0x03:
		msg.requestType = "RPC"
		msg.IsRequest = true
	case 0x04:
		msg.requestType = "Tabular result"
		msg.IsRequest = true
	// case 0x05:
	// 	logp.Info("* Type: Unused")
	case 0x06:
		msg.requestType = "Attention Signal"
		msg.IsRequest = true
	case 0x07:
		msg.requestType = "Bulk load data"
		msg.IsRequest = true
	case 0x08:
		msg.requestType = "Federated Authentication Token"
		msg.IsRequest = true
	// case 0x09, 0x0A, 0x0B, 0x0C, 0x0D:
	// 	logp.Info("* Type: Unused")
	case 0x0E:
		msg.requestType = "Transaction Manager Request"
		msg.IsRequest = true
	// case 0x0F:
	// 	logp.Info("* Type: Unused")
	case 0x10:
		msg.requestType = "TDS7 Login"
		msg.IsRequest = true
	case 0x20:
		msg.requestType = "SSPI"
		msg.IsRequest = true
	case 0x30:
		msg.requestType = "Pre-Login"
		msg.IsRequest = true
	default:
		return nil, fmt.Errorf("Unrecognised TDS Type")
	}

	// As we have only peeked at the buffer we need to advance so that a reset clears the buffer
	p.buf.Advance(p.buf.Len())
	return msg, nil
}
