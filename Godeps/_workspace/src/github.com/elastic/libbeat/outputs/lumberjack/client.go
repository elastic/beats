package lumberjack

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

// lumberjackClient implements the ProtocolClient interface to be used
// with different mode. The client implements slow start with low window sizes +
// window size backoff in case of long running transactions.
//
// it is suggested to use lumberjack in conjunction with roundRobinConnectionMode
// if logstash becomes unresponsive
type lumberjackClient struct {
	TransportClient
	windowSize      int
	maxOkWindowSize int // max window size sending was successful for
	timeout         time.Duration
	countTimeoutErr int
}

// TODO: make max window size configurable
const (
	minWindowSize             int = 1
	maxWindowSize             int = 1024
	defaultStartMaxWindowSize int = 10

	maxAllowedTimeoutErr int = 3
)

// errors
var (
	// ErrProtocolError is returned if an protocol error was detected in the
	// conversation with lumberjack server.
	ErrProtocolError = errors.New("lumberjack protocol error")
)

var (
	codeWindowSize    = []byte("1W")
	codeJSONDataFrame = []byte("1J")
	codeCompressed    = []byte("1C")
)

func newLumberjackClient(conn TransportClient, timeout time.Duration) *lumberjackClient {
	return &lumberjackClient{
		TransportClient: conn,
		windowSize:      defaultStartMaxWindowSize,
		timeout:         timeout,
	}
}

func (l *lumberjackClient) PublishEvents(events []common.MapStr) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	if len(events) > l.windowSize {
		events = events[:l.windowSize]
	}

	// Abort if sending request takes longer than the configured
	// network timeout.
	conn := l.TransportClient
	if err := conn.SetDeadline(time.Now().Add(l.timeout)); err != nil {
		return l.onFail(0, err)
	}

	count, payload, err := l.compressEvents(events)
	if err != nil {
		return 0, err
	}

	if count == 0 {
		// encoding of all events failed. Let's stop here and report all events
		// as exported so no one tries to send/encode the same events once again
		// The compress/encode function already prints critical per failed encoding
		// failure.
		return len(events), nil
	}

	// send window size:
	if err = l.sendWindowSize(count); err != nil {
		return l.onFail(0, err)
	}

	// send payload
	if err = l.sendCompressed(payload); err != nil {
		return l.onFail(0, err)
	}

	// wait for ACK (accept partial ACK to reset timeout)
	// reset timeout timer for every ACK received.
	var ackSeq uint32
	for ackSeq < count {
		// read until all acks
		if err := conn.SetDeadline(time.Now().Add(l.timeout)); err != nil {
			return l.onFail(ackSeq, err)
		}

		ackSeq, err = l.readACK()
		if err != nil {
			return l.onFail(ackSeq, err)
		}
	}

	// success: increase window size by factor 1.5 until max window size
	// (window size grows exponentially)
	// TODO: use duration until ACK to estimate an ok max window size value
	if l.maxOkWindowSize < l.windowSize {
		l.maxOkWindowSize = l.windowSize

		if l.windowSize < maxWindowSize {
			l.windowSize = l.windowSize + l.windowSize/2
			if l.windowSize > maxWindowSize {
				l.windowSize = maxWindowSize
			}
		}
	} else if l.windowSize < l.maxOkWindowSize {
		l.windowSize = l.windowSize + l.windowSize/2
		if l.windowSize > l.maxOkWindowSize {
			l.windowSize = l.maxOkWindowSize
		}
	}

	return len(events), nil
}

func (l *lumberjackClient) onFail(ackSeq uint32, err error) (int, error) {
	// if timeout error, back off and ignore error
	nerr, ok := err.(net.Error)
	if !ok || !nerr.Timeout() {
		// no timeout error, close connection and return error
		_ = l.Close()
		return int(ackSeq), err
	}

	// if we've seen 3 consecutive timeout errors, close connection
	l.countTimeoutErr++
	if l.countTimeoutErr == maxAllowedTimeoutErr {
		_ = l.Close()
		return int(ackSeq), err
	}

	// timeout error. reduce window size and return 0 published events. Send
	// mode might try to publish again with reduce window size or ask another
	// client to send events (round robin load balancer)
	l.windowSize = l.windowSize / 2
	if l.windowSize < minWindowSize {
		l.windowSize = minWindowSize
	}
	return int(ackSeq), nil
}

func (l *lumberjackClient) compressEvents(
	events []common.MapStr,
) (uint32, []byte, error) {
	buf := bytes.NewBuffer(nil)

	// compress events
	compressor, _ := zlib.NewWriterLevel(buf, 3) // todo make compression level configurable?
	var sequence uint32
	for _, event := range events {
		sequence++
		err := l.writeDataFrame(event, sequence, compressor)
		if err != nil {
			logp.Critical("failed to encode event: %v", err)
			sequence-- //forget this last broken event and continue
		}
	}
	if err := compressor.Close(); err != nil {
		debug("Finalizing zlib compression failed with: %s", err)
		return 0, nil, err
	}
	payload := buf.Bytes()

	return sequence, payload, nil
}

func (l *lumberjackClient) readACK() (uint32, error) {
	response := make([]byte, 6)
	ackbytes := 0
	for ackbytes < 6 {
		n, err := l.Read(response[ackbytes:])
		if err != nil {
			return 0, err
		}
		ackbytes += n
	}

	isACK := response[0] == '1' && response[1] == 'A'
	if !isACK {
		return 0, ErrProtocolError
	}
	seq := binary.BigEndian.Uint32(response[2:])
	return seq, nil
}

func (l *lumberjackClient) sendWindowSize(window uint32) error {
	if _, err := l.Write(codeWindowSize); err != nil {
		return err
	}
	return writeUint32(l, window)
}

func (l *lumberjackClient) sendCompressed(payload []byte) error {
	if _, err := l.Write(codeCompressed); err != nil {
		return err
	}
	if err := writeUint32(l, uint32(len(payload))); err != nil {
		return err
	}

	_, err := l.Write(payload)
	return err
}

func (l *lumberjackClient) writeDataFrame(
	event common.MapStr,
	seq uint32,
	out io.Writer,
) error {
	// Write JSON Data Frame:
	// version: uint8 = '1'
	// code: uint8 = 'J'
	// seq: uint32
	// payloadLen (bytes): uint32
	// payload: JSON document

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		debug("Fail to convert the event to JSON: %s", err)
		return err
	}

	if _, err := out.Write(codeJSONDataFrame); err != nil { // version + code
		return err
	}
	if err := writeUint32(out, seq); err != nil {
		return err
	}
	if err := writeUint32(out, uint32(len(jsonEvent))); err != nil {
		return err
	}
	if _, err := out.Write(jsonEvent); err != nil {
		return err
	}

	return nil
}

func writeUint32(out io.Writer, v uint32) error {
	return binary.Write(out, binary.BigEndian, v)
}
