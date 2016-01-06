package logstash

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"expvar"
	"io"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Metrics that can retrieved through the expvar web interface.
var (
	ackedEvents            = expvar.NewInt("libbeatLogstashPublishedAndAckedEvents")
	eventsNotAcked         = expvar.NewInt("libbeatLogstashPublishedButNotAckedEvents")
	publishEventsCallCount = expvar.NewInt("libbeatLogstashPublishEventsCallCount")
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
	maxWindowSize   int
	timeout         time.Duration
	countTimeoutErr int
	compressLevel   int
}

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
	maxAllowedTimeoutErr      int = 3
)

// errors
var (
	// ErrProtocolError is returned if an protocol error was detected in the
	// conversation with lumberjack server.
	ErrProtocolError = errors.New("lumberjack protocol error")
)

var (
	codeVersion byte = '2'

	codeWindowSize    = []byte{codeVersion, 'W'}
	codeJSONDataFrame = []byte{codeVersion, 'J'}
	codeCompressed    = []byte{codeVersion, 'C'}
)

func newLumberjackClient(
	conn TransportClient,
	compressLevel int,
	maxWindowSize int,
	timeout time.Duration,
) (*lumberjackClient, error) {

	// validate by creating and discarding zlib writer with configured level
	if compressLevel > 0 {
		tmp := bytes.NewBuffer(nil)
		w, err := zlib.NewWriterLevel(tmp, compressLevel)
		if err != nil {
			return nil, err
		}
		w.Close()
	}

	return &lumberjackClient{
		TransportClient: conn,
		windowSize:      defaultStartMaxWindowSize,
		timeout:         timeout,
		maxWindowSize:   maxWindowSize,
		compressLevel:   compressLevel,
	}, nil
}

func (l *lumberjackClient) Connect(timeout time.Duration) error {
	logp.Debug("logstash", "connect")
	return l.TransportClient.Connect(timeout)
}

func (l *lumberjackClient) Close() error {
	logp.Debug("logstash", "close connection")
	return l.TransportClient.Close()
}

func (l *lumberjackClient) PublishEvent(event common.MapStr) error {
	_, err := l.PublishEvents([]common.MapStr{event})
	return err
}

// PublishEvents sends all events to logstash. On error a slice with all events
// not published or confirmed to be processed by logstash will be returned.
func (l *lumberjackClient) PublishEvents(
	events []common.MapStr,
) ([]common.MapStr, error) {
	publishEventsCallCount.Add(1)
	totalNumberOfEvents := len(events)
	for len(events) > 0 {
		n, err := l.publishWindowed(events)

		logp.Debug("logstash", "%v events out of %v events sent to logstash. Continue sending ...", n, len(events))
		events = events[n:]
		if err != nil {
			eventsNotAcked.Add(int64(len(events)))
			ackedEvents.Add(int64(totalNumberOfEvents - len(events)))
			return events, err
		}
	}
	ackedEvents.Add(int64(totalNumberOfEvents))
	return nil, nil
}

// publishWindowed published events with current maximum window size to logstash
// returning the total number of events sent (due to window size, or acks until
// failure).
func (l *lumberjackClient) publishWindowed(events []common.MapStr) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	batchSize := len(events)
	debug("Try to publish %v events to logstash with window size %v",
		batchSize, l.windowSize)

	// prepare message payload
	if len(events) > l.windowSize {
		events = events[:l.windowSize]
	}

	// serialize all raw events into output buffer, removing all events encoding failed for
	count, payload, err := l.serializeEvents(events)
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

	if l.compressLevel > 0 {
		err = l.sendCompressed(payload)
	} else {
		_, err = l.Write(payload)
	}
	if err != nil {
		return l.onFail(0, err)
	}

	// wait for ACK (accept partial ACK to reset timeout)
	// reset timeout timer for every ACK received.
	var ackSeq uint32
	for ackSeq < count {
		// read until all acks
		ackSeq, err = l.readACK()
		if err != nil {
			return l.onFail(int(ackSeq), err)
		}
	}

	// success: increase window size by factor 1.5 until max window size
	// (window size grows exponentially)
	// TODO: use duration until ACK to estimate an ok max window size value
	if l.maxOkWindowSize < l.windowSize {
		l.maxOkWindowSize = l.windowSize

		if l.windowSize < l.maxWindowSize {
			l.windowSize = l.windowSize + l.windowSize/2
			if l.windowSize > l.maxWindowSize {
				l.windowSize = l.maxWindowSize
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

func (l *lumberjackClient) onFail(n int, err error) (int, error) {
	// if timeout error, back off and ignore error
	nerr, ok := err.(net.Error)
	if !ok || !nerr.Timeout() {
		// no timeout error, close connection and return error
		_ = l.Close()
		return n, err
	}

	// if we've seen 3 consecutive timeout errors, close connection
	l.countTimeoutErr++
	if l.countTimeoutErr == maxAllowedTimeoutErr {
		_ = l.Close()
		return n, err
	}

	// timeout error. reduce window size and return 0 published events. Send
	// mode might try to publish again with reduce window size or ask another
	// client to send events
	l.windowSize = l.windowSize / 2
	if l.windowSize < minWindowSize {
		l.windowSize = minWindowSize
	}
	return n, nil
}

func (l *lumberjackClient) serializeEvents(
	events []common.MapStr,
) (uint32, []byte, error) {
	buf := bytes.NewBuffer(nil)

	if l.compressLevel > 0 {
		w, _ := zlib.NewWriterLevel(buf, l.compressLevel)
		count, err := l.doSerializeEvents(w, events)
		if err != nil {
			return 0, nil, err
		}
		if err := w.Close(); err != nil {
			debug("Finalizing zlib compression failed with: %s", err)
			return 0, nil, err
		}
		return count, buf.Bytes(), nil
	}

	count, err := l.doSerializeEvents(buf, events)
	return count, buf.Bytes(), err
}

func (l *lumberjackClient) doSerializeEvents(out io.Writer, events []common.MapStr) (uint32, error) {
	var sequence uint32
	for _, event := range events {
		sequence++
		err := l.writeDataFrame(event, sequence, out)
		if err != nil {
			logp.Critical("failed to encode event: %v", err)
			sequence-- //forget this last broken event and continue
		}
	}
	return sequence, nil
}

func (l *lumberjackClient) readACK() (uint32, error) {
	if err := l.SetDeadline(time.Now().Add(l.timeout)); err != nil {
		return 0, err
	}

	response := make([]byte, 6)
	ackbytes := 0
	for ackbytes < 6 {
		n, err := l.Read(response[ackbytes:])
		if err != nil {
			return 0, err
		}
		ackbytes += n
	}

	isACK := response[0] == codeVersion && response[1] == 'A'
	if !isACK {
		return 0, ErrProtocolError
	}
	seq := binary.BigEndian.Uint32(response[2:])
	return seq, nil
}

func (l *lumberjackClient) sendWindowSize(window uint32) error {
	if err := l.SetDeadline(time.Now().Add(l.timeout)); err != nil {
		return err
	}
	if _, err := l.Write(codeWindowSize); err != nil {
		return err
	}
	return writeUint32(l, window)
}

func (l *lumberjackClient) sendCompressed(payload []byte) error {
	if err := l.SetDeadline(time.Now().Add(l.timeout)); err != nil {
		return err
	}

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
	// version: uint8 = '2'
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
