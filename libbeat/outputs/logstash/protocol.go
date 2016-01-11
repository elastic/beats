package logstash

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type protocol struct {
	conn          net.Conn
	timeout       time.Duration
	compressLevel int
}

var (
	// ErrProtocolError is returned if an protocol error was detected in the
	// conversation with lumberjack server.
	ErrProtocolError = errors.New("lumberjack protocol error")

	errAllEventsEncoding = errors.New("failed to encode all events")
)

var (
	codeVersion byte = '2'

	codeWindowSize    = []byte{codeVersion, 'W'}
	codeJSONDataFrame = []byte{codeVersion, 'J'}
	codeCompressed    = []byte{codeVersion, 'C'}
)

func newClientProcol(
	conn net.Conn,
	timeout time.Duration,
	compressLevel int,
) (*protocol, error) {

	// validate by creating and discarding zlib writer with configured level
	if compressLevel > 0 {
		tmp := bytes.NewBuffer(nil)
		w, err := zlib.NewWriterLevel(tmp, compressLevel)
		if err != nil {
			return nil, err
		}
		w.Close()
	}

	return &protocol{
		conn:          conn,
		timeout:       timeout,
		compressLevel: compressLevel,
	}, nil
}

func (p *protocol) sendEvents(events []common.MapStr) (uint32, error) {
	conn := p.conn
	out := bufio.NewWriterSize(conn, 4096)

	if len(events) == 0 {
		return 0, nil
	}

	// serialize all raw events into output buffer, removing all events encoding failed for
	count, payload, err := p.serializeEvents(events)
	if count == 0 {
		// encoding of all events failed. Let's stop here and report all events
		// as exported so no one tries to send/encode the same events once again
		// The compress/encode function already prints critical per failed encoding
		// failure.
		return 0, errAllEventsEncoding
	}

	if err := conn.SetWriteDeadline(time.Now().Add(p.timeout)); err != nil {
		return 0, err
	}

	// send window size:
	if err := p.sendWindowSize(out, count); err != nil {
		return 0, err
	}

	if p.compressLevel > 0 {
		err = p.sendCompressed(out, payload)
	} else {
		err = write(out, payload)
	}
	if err != nil {
		return 0, err
	}

	if err := conn.SetWriteDeadline(time.Now().Add(p.timeout)); err != nil {
		return 0, err
	}
	if err := out.Flush(); err != nil {
		return 0, err
	}

	return count, nil
}

func (p *protocol) recvACK() (uint32, error) {
	conn := p.conn

	if err := conn.SetReadDeadline(time.Now().Add(p.timeout)); err != nil {
		return 0, err
	}

	response := make([]byte, 6)
	ackbytes := 0
	for ackbytes < 6 {
		n, err := conn.Read(response[ackbytes:])
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

// wait for ACK (accept partial ACK to reset timeout)
// reset timeout timer for every ACK received.
func (p *protocol) awaitACK(count uint32) (uint32, error) {
	var ackSeq uint32
	var err error

	// read until all acks
	for ackSeq < count {
		ackSeq, err = p.recvACK()
		if err != nil {
			return ackSeq, err
		}

		debug("received ack: %v", ackSeq)
	}
	if ackSeq > count {
		logp.Warn("invalid ack sequence received(%v): %v ", count, ackSeq)
		ackSeq = count
	}
	return ackSeq, nil
}

func (p *protocol) sendWindowSize(out io.Writer, window uint32) error {
	if err := write(out, codeWindowSize); err != nil {
		return err
	}
	if err := writeUint32(out, window); err != nil {
		return err
	}
	return nil
}

func (p *protocol) sendCompressed(out io.Writer, payload []byte) error {
	if err := write(out, codeCompressed); err != nil {
		return err
	}
	if err := writeUint32(out, uint32(len(payload))); err != nil {
		return err
	}
	return write(out, payload)
}

func (p *protocol) serializeEvents(events []common.MapStr) (uint32, []byte, error) {
	buf := bytes.NewBuffer(nil)
	if p.compressLevel > 0 {
		w, _ := zlib.NewWriterLevel(buf, p.compressLevel)
		count, err := p.doSerializeEvents(w, events)
		if err != nil {
			return 0, nil, err
		}
		if err := w.Close(); err != nil {
			debug("Finalizing zlib compression failed with: %s", err)
			return 0, nil, err
		}
		return count, buf.Bytes(), nil
	}

	count, err := p.doSerializeEvents(buf, events)
	return count, buf.Bytes(), err
}

func (p *protocol) doSerializeEvents(out io.Writer, events []common.MapStr) (uint32, error) {
	var sequence uint32
	for _, event := range events {
		sequence++
		err := p.serializeDataFrame(out, event, sequence)
		if err != nil {
			logp.Critical("failed to encode event: %v", err)
			sequence-- //forget this last broken event and continue
		}
	}
	return sequence, nil
}

func (p *protocol) serializeDataFrame(
	out io.Writer,
	event common.MapStr,
	seq uint32,
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

	if err := write(out, codeJSONDataFrame); err != nil { // version + code
		return err
	}
	if err := writeUint32(out, seq); err != nil {
		return err
	}
	if err := writeUint32(out, uint32(len(jsonEvent))); err != nil {
		return err
	}
	if err := write(out, jsonEvent); err != nil {
		return err
	}

	return nil
}

func write(out io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := out.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func writeUint32(out io.Writer, v uint32) error {
	return binary.Write(out, binary.BigEndian, v)
}
