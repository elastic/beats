package v1

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/klauspost/compress/zlib"

	"github.com/elastic/go-lumber/lj"
	"github.com/elastic/go-lumber/log"
	protocol "github.com/elastic/go-lumber/protocol/v1"
)

type reader struct {
	in      *bufio.Reader
	conn    net.Conn
	timeout time.Duration
	buf     []byte
}

func newReader(c net.Conn, to time.Duration) *reader {
	r := &reader{
		in:      bufio.NewReader(c),
		conn:    c,
		timeout: to,
		buf:     make([]byte, 0, 64),
	}
	return r
}

func (r *reader) ReadBatch() (*lj.Batch, error) {
	// 1. read window size
	var win [6]byte
	_ = r.conn.SetReadDeadline(time.Time{}) // wait for next batch without timeout
	if err := readFull(r.in, win[:]); err != nil {
		return nil, err
	}

	if win[0] != protocol.CodeVersion && win[1] != protocol.CodeWindowSize {
		log.Printf("Expected window from. Received %v", win[0:1])
		return nil, ErrProtocolError
	}

	count := int(binary.BigEndian.Uint32(win[2:]))
	if count == 0 {
		return nil, nil
	}

	if err := r.conn.SetReadDeadline(time.Now().Add(r.timeout)); err != nil {
		return nil, err
	}

	events, err := r.readEvents(r.in, make([]interface{}, 0, count))
	if events == nil || err != nil {
		log.Printf("readEvents failed with: %v", err)
		return nil, err
	}

	return lj.NewBatch(events), nil
}

func (r *reader) readEvents(in io.Reader, events []interface{}) ([]interface{}, error) {
	for len(events) < cap(events) {
		var hdr [2]byte
		if err := readFull(in, hdr[:]); err != nil {
			return nil, err
		}

		if hdr[0] != protocol.CodeVersion {
			log.Println("Event protocol version error")
			return nil, ErrProtocolError
		}

		switch hdr[1] {
		case protocol.CodeDataFrame:
			event, err := r.readEvent(in)
			if err != nil {
				log.Printf("failed to read json event with: %v\n", err)
				return nil, err
			}
			events = append(events, event)
		case protocol.CodeCompressed:
			readEvents, err := r.readCompressed(in, events)
			if err != nil {
				return nil, err
			}
			events = readEvents
		default:
			log.Printf("Unknown frame type: %v", hdr[1])
			return nil, ErrProtocolError
		}
	}
	return events, nil
}

func (r *reader) readCompressed(in io.Reader, events []interface{}) ([]interface{}, error) {
	var hdr [4]byte
	if err := readFull(in, hdr[:]); err != nil {
		return nil, err
	}

	payloadSz := binary.BigEndian.Uint32(hdr[:])
	limit := io.LimitReader(in, int64(payloadSz))
	reader, err := zlib.NewReader(limit)
	if err != nil {
		log.Printf("Failed to initialized zlib reader %v\n", err)
		return nil, err
	}

	events, err = r.readEvents(reader, events)
	if err != nil {
		_ = reader.Close()
		return nil, err
	}
	if err := reader.Close(); err != nil {
		return nil, err
	}

	// consume final bytes from limit reader
	for {
		var tmp [16]byte
		if _, err := limit.Read(tmp[:]); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
	}
	return events, nil
}

func (r *reader) readEvent(in io.Reader) (interface{}, error) {
	var hdr [8]byte
	if err := readFull(in, hdr[:]); err != nil {
		return nil, err
	}

	readString := func() (string, error) {
		var bufBytes [4]byte
		if err := readFull(in, bufBytes[:]); err != nil {
			return "", err
		}

		bytes := int(binary.BigEndian.Uint32(bufBytes[:]))
		if bytes > len(r.buf) {
			r.buf = make([]byte, bytes)
		}

		buf := r.buf[:bytes]
		if err := readFull(in, buf); err != nil {
			return "", err
		}

		return string(buf[:]), nil
	}

	event := map[string]string{}
	pairs := int(binary.BigEndian.Uint32(hdr[4:]))
	for i := 0; i < pairs; i++ {
		k, err := readString()
		if err != nil {
			return nil, err
		}

		v, err := readString()
		if err != nil {
			return nil, err
		}

		event[k] = v
	}
	return event, nil
}

func readFull(in io.Reader, buf []byte) error {
	_, err := io.ReadFull(in, buf)
	return err
}
