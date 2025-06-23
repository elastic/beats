// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/elastic/elastic-agent-libs/logp"

	libbeat_reader "github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/convert"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	v9 "github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)

// Copied from x-pack/filebeat/input/awss3/s3_objects.go
//
// isStreamGzipped determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not. A buffered reader is used so the function can peek into the byte
// stream without consuming it. This makes it convenient for code executed after this function call
// to consume the stream if it wants.
func isStreamGzipped(r *bufio.Reader) (bool, error) {
	buf, err := r.Peek(3)
	if err != nil && err != io.EOF {
		return false, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	return bytes.HasPrefix(buf, []byte{0x1F, 0x8B, 0x08}), nil
}

func (p *netflowInput) addGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)

	gzipped, err := isStreamGzipped(bufReader)
	if err != nil {
		return nil, err
	}
	if !gzipped {
		return bufReader, nil
	}

	return gzip.NewReader(bufReader)
}

type IPFIXReader struct {
	cfg    *Config
	reader libbeat_reader.Reader
	logger *logp.Logger
}

func (r *IPFIXReader) Next() (libbeat_reader.Message, error) {
	// need to handle reading things
	// r.reader
	return libbeat_reader.Message{}, nil
}

// BufferedReader parses ipfix inputs from io streams.
type BufferedReader struct {
	decoder v9.Decoder
	reader_ *bufio.Reader
	offset  int
	cfg     *Config
	logger  *logp.Logger
	session *v9.SessionState
	source  net.Addr
}

// NewBufferedReader creates a new reader that can decode ipfix data from an io.Reader.
// It will return an error if the ipfix data stream cannot be read.
func NewBufferedReader(r io.Reader, cfg *Config) (*BufferedReader, error) {
	logger := logp.L().Named("reader.ipfix")

	decoder := NewDecoder(cfg, logger)

	return &BufferedReader{
		decoder: *decoder,
		reader_: bufio.NewReader(r),
		offset:  0,
		cfg:     cfg,
		logger:  logger,
		session: v9.NewSession(logger),
		source:  DefaultExporterAddr(),
	}, nil
}

// Next advances the pointer to point to the next record and returns true if the next record exists.
// It will return false if there are no more records to read.
func (sr *BufferedReader) Next() bool {
	data, err := sr.reader_.Peek(4)
	if err != nil || len(data) < 4 {
		sr.logger.Debugf("Not enough data to read: %v", err)
		return false
	}

	// the IPFIX packet is two bytes of version, two bytes of length
	version := binary.BigEndian.Uint16(data[0:2])
	length := binary.BigEndian.Uint16(data[2:4])

	// TODO: we need to read the rest of the packet and skip the length
	// if the version is wrong, nothing else to read
	if version != 10 {
		sr.logger.Debugf("incorrect version (%v)", version)
		return false
	}

	// TODO: we should skip this one and try another
	// if the length is says so, nothing else to read
	if length < 4 {
		sr.logger.Debugf("packet is too small (%v)", length)
		return false
	}

	// otherwise, seems good!
	return true
}

// Record reads the current record from the current file and returns it as a JSON marshaled byte slice.
// If no more records are available, the []byte slice will be nil and io.EOF will be returned as an error.
// A JSON marshal error will be returned if the record cannot be marshalled.
func (sr *BufferedReader) Record() ([]beat.Event, error) {
	// call the OnPacket() from v9.go / NetflowV9Protocol
	// create metadata exporter
	// loop over flows and update the exporter, etc
	// return
	// read the next packet

	// read the next four bytes
	peek, err := sr.reader_.Peek(4)
	if err != nil {
		return nil, fmt.Errorf("Error reading data: %v", err)
	}

	// the IPFIX packet is two bytes of version, two bytes of length
	version := binary.BigEndian.Uint16(peek[0:2])
	length := binary.BigEndian.Uint16(peek[2:4])

	// if the version is wrong, nothing else to read
	if version != 10 {
		// TODO: read the rest of the packet and skip it
		sr.reader_.Discard(int(length))
		return nil, fmt.Errorf("incorrect version (%v)", version)
	}

	// if the length is says so, nothing else to read
	if length <= 4 {
		sr.reader_.Discard(int(length))
		return nil, fmt.Errorf("packet is too small (%v)", length)
	}

	data := make([]byte, length)
	n, err := io.ReadFull(sr.reader_, data)
	if err != nil || n != int(length) {
		// Not sure how to recover from this
		return nil, fmt.Errorf("error with reading %d out of %d bytes of data: %w", n, length, err)
	}

	pkt := bytes.NewBuffer(data)

	// read the packet header
	header, payload, numFlowSets, err := sr.decoder.ReadPacketHeader(pkt)
	if err != nil {
		return nil, fmt.Errorf("error reading header: %w", err)
	}

	var flows []record.Record

	for ; numFlowSets > 0; numFlowSets-- {
		set, err := sr.decoder.ReadSetHeader(payload)
		if err != nil || set.IsPadding() {
			break
		}
		if payload.Len() < set.BodyLength() {
			sr.logger.Debugf("FlowSet ID %+v overflows packet", set)
			break
		}
		body := bytes.NewBuffer(payload.Next(set.BodyLength()))
		sr.logger.Debugf("FlowSet ID %d length %d", set.SetID, set.BodyLength())

		f, err := sr.parseSet(set.SetID, body)
		if err != nil {
			sr.logger.Debugf("Error parsing set %d: %v", set.SetID, err)
			return nil, fmt.Errorf("error parsing set: %w", err)
		}
		flows = append(flows, f...)
	}

	metadata := header.ExporterMetadata(sr.source)
	for idx := range flows {
		flows[idx].Exporter = metadata
		flows[idx].Timestamp = header.UnixSecs
	}

	// from here, we get an array of flows
	// we need to convert each to an event
	// we then need to marshal each event to a json blob
	// we return the json blobs

	fLen := len(flows)
	if fLen != 0 {
		sr.logger.Debugf("Captured [%v] flows", fLen)
		evs := make([]beat.Event, fLen)
		for flowIdx, flow := range flows {
			sr.logger.Debugf("Captured flow [%v]", flow)
			evs[flowIdx] = convert.RecordToBeatEvent(flow, sr.cfg.InternalNetworks)
		}
		return evs, nil
	}
	return nil, nil
}

func (sr *BufferedReader) parseSet(
	setID uint16,
	buf *bytes.Buffer) (flows []record.Record, err error,
) {
	if setID >= 256 {
		// Flow of Options record, lookup template and generate flows
		template := sr.session.GetTemplate(setID)

		if template == nil {
			sr.logger.Debugf("No template for ID %d", setID)
			return nil, nil
		}

		return template.Apply(buf, 0)
	}

	// Template sets
	templates, err := sr.decoder.ReadTemplateSet(setID, buf)
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		// if this is an options template, see if it has source/sender address
		sr.session.AddTemplate(template)
	}

	return flows, nil
}

// Close closes the stream reader and releases all resources.
// It will return an error if the fileReader fails to close.
func (sr *BufferedReader) Close() error {
	sr.decoder = nil

	return nil
}
