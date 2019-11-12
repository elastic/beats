// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipereader

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/docker/engine/api/types/plugins/logdriver"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fifo"
)

// PipeReader reads from the FIFO pipe we get from the docker container
type PipeReader struct {
	fifoPipe    io.ReadCloser
	byteOrder   binary.ByteOrder
	lenFrameBuf []byte
	bodyBuf     []byte
	maxSize     int
}

// NewReaderFromPath creates a new FIFO pipe reader from a docker log pipe location
func NewReaderFromPath(file string) (*PipeReader, error) {
	inputFile, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening logger fifo: %q", file)
	}

	return &PipeReader{fifoPipe: inputFile, byteOrder: binary.BigEndian, lenFrameBuf: make([]byte, 4), bodyBuf: make([]byte, 4*1024), maxSize: 2e6}, nil
}

// ReadMessage reads a log message from the pipe
// The message stream consists of a 4-byte length frame and a message body
// There's three logical paths for this code to take:
// 1) If length <0, we have bad data, and we cycle through the frames until we get a valid length.
// 2) If length is valid but larger than the max buffer size, we disregard length bytes and continue
// 3) If length is valid and we can consume everything into the buffer, continue.
func (reader *PipeReader) ReadMessage(log *logdriver.LogEntry) error {
	// loop until we're at a valid state and ready to read a message body
	var lenFrame int
	var err error
	for {
		lenFrame, err = reader.getValidLengthFrame()
		if err != nil {
			return errors.Wrap(err, "error getting length frame")
		}
		fmt.Fprintf(os.Stderr, "Got header specifying length %d\n", lenFrame)
		if lenFrame <= reader.maxSize {
			break
		}

		// 2) we have a too-large message. Disregard length bytes
		_, err = io.CopyBuffer(nil, io.LimitReader(reader.fifoPipe, int64(lenFrame)), reader.bodyBuf)
		if err != nil {
			return errors.Wrap(err, "error emptying buffer")
		}
	}

	//proceed with 3)
	reader.setBuffer(lenFrame)
	_, err = io.ReadFull(reader.fifoPipe, reader.bodyBuf[:lenFrame])
	if err != nil {
		return errors.Wrap(err, "error reading buffer")
	}
	fmt.Fprintf(os.Stderr, "Buffer now contains %#v\n", reader.bodyBuf[:lenFrame])
	return proto.Unmarshal(reader.bodyBuf[:lenFrame], log)

}

// setBuffer checks the size and only reallocates if we need to
func (reader *PipeReader) setBuffer(sz int) {
	const minSz = 4 * 1024 // min 4KB read buffer

	if len(reader.bodyBuf) >= sz {
		return
	}

	if sz < minSz {
		sz = minSz
	}

	reader.bodyBuf = make([]byte, sz)
}

// getValidLengthFrame scrolls through the buffer until we get a valid length
func (reader *PipeReader) getValidLengthFrame() (int, error) {
	for {
		if _, err := io.ReadFull(reader.fifoPipe, reader.lenFrameBuf); err != nil {
			return 0, err
		}
		bodyLen := int(reader.byteOrder.Uint32(reader.lenFrameBuf))
		if bodyLen > 0 {
			return bodyLen, nil
		}
	}
}

// Close closes the reader and underlying pipe
func (reader *PipeReader) Close() error {
	return reader.fifoPipe.Close()
}
