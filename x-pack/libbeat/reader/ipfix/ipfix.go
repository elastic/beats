// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/convert"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	v9 "github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)

// BufferedReader parses ipfix inputs from io streams.
type BufferedReader struct {
	decoder v9.Decoder
	data    []byte
	offset  int
	cfg     *Config
	logger  *logp.Logger
	session *v9.SessionState
	source  net.Addr
}

// NewBufferedReader creates a new reader that can decode ipfix data from an io.Reader.
// It will return an error if the parquet data stream cannot be read.
// Note: As io.ReadAll is used, the entire data stream would be read into memory, so very large data streams
// may cause memory bottleneck issues.
func NewBufferedReader(r io.Reader, cfg *Config) (*BufferedReader, error) {
	logger := logp.L().Named("reader.ipfix")

	var err error
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Failed to read data from reader: %v", err)
	}

	decoder := NewDecoder(cfg, logger)

	return &BufferedReader{
		decoder: *decoder,
		data:    data,
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

	offset := sr.offset
	// make sure there are at least four bytes left
	if offset+4 > len(sr.data) {
		sr.logger.Debugf("Not enough left for reading: %d bytes left", len(sr.data)-offset)
		return false
	}

	// the IPFIX packet is two bytes of version, two bytes of length
	version := binary.BigEndian.Uint16(sr.data[offset+0 : offset+2])
	length := binary.BigEndian.Uint16(sr.data[offset+2 : offset+4])

	// if the version is wrong, nothing else to read
	if version != 10 {
		sr.logger.Debugf("incorrect version (%v)", version)
		return false
	}

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

	// make sure there are at least four bytes left
	if sr.offset+4 > len(sr.data) {
		return nil, fmt.Errorf("Not enough left for reading: %d bytes left", len(sr.data)-sr.offset)
	}

	// the IPFIX packet is two bytes of version, two bytes of length
	offset := sr.offset
	version := binary.BigEndian.Uint16(sr.data[offset+0 : offset+2])
	length := binary.BigEndian.Uint16(sr.data[offset+2 : offset+4])

	// if the version is wrong, nothing else to read
	if version != 10 {
		return nil, fmt.Errorf("incorrect version (%v)", version)
	}

	// if the length is says so, nothing else to read
	if length < 4 {
		return nil, fmt.Errorf("packet is too small (%v)", length)
	}

	buf := sr.data[offset : offset+int(length)]
	pkt := bytes.NewBuffer(buf)
	sr.offset += int(length)

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
