// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/convert"
	nf_decoder "github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
)

// BufferedReader parses ipfix inputs from io streams.
type BufferedReader struct {
	decoder *nf_decoder.Decoder
	data    []byte
	offset  int
	cfg     *Config
	logger  *logp.Logger
}

// NewBufferedReader creates a new reader that can decode parquet data from an io.Reader.
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

	decoder, err := nf_decoder.NewDecoder(nf_decoder.NewConfig(logger).
		WithProtocols("ipfix").
		WithCustomFields(cfg.Fields()).
		WithSequenceResetEnabled(true).
		WithSharedTemplates(true).
		WithCache(false).
		WithFileSupport(true))

	logger.Infof("Got decoder: %v", decoder)
	if err != nil {
		logger.Errorf("Failed to initialize IPFIX decoder: %v", err)
		return nil, err
	}

	logger.Info("Starting netflow decoder")
	if err := decoder.Start(); err != nil {
		logger.Errorf("Failed to start netflow decoder: %v", err)
		return nil, err
	}

	return &BufferedReader{
		decoder: decoder,
		data:    data,
		offset:  0,
		cfg:     cfg,
		logger:  logger,
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

type DummyAddr struct {
	NetworkValue string
	StringValue  string
}

func (d DummyAddr) Network() string {
	return d.NetworkValue
}
func (d DummyAddr) String() string {
	return d.StringValue
}

func NewDummyAddr() DummyAddr {
	return DummyAddr{NetworkValue: "tcp", StringValue: "100::"}
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

	source := NewDummyAddr()

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

	sr.logger.Infof("Read a [%v / %#02x] record of length [%v / %#02x]; new offset is [%v / %#02x]", version, version, length, length, sr.offset, sr.offset)

	flows, err := sr.decoder.Read(pkt, source)
	if err != nil {
		sr.logger.Warnf("Error parsing Netflow packet of length %d: %v", pkt.Len(), err)
		return nil, err
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

// Close closes the stream reader and releases all resources.
// It will return an error if the fileReader fails to close.
func (sr *BufferedReader) Close() error {
	if err := sr.decoder.Stop(); err != nil {
		sr.logger.Errorw("Error stopping decoder", "error", err)
	}
	sr.decoder = nil

	return nil
}

/*
// BufferedReader parses ipfix inputs from io streams.
type BufferedReader struct {
	protocol *ip.IPFixProtocol
	data     []byte
	offset   int
	cfg      *Config
	logger   *logp.Logger
}

func NewConfig() *nf_config.Config {
	cfg := nf_config.Defaults()
	return &cfg
}

func NewProtocol(cfg nf_config.Config) *ip.IPFixProtocol {
	decoder := ip.DecoderIPFIX{
		DecoderV9: v9.DecoderV9{Logger: cfg.LogOutput(), Fields: cfg.Fields()},
		FileBased: true,
	}
	proto := &ip.IPFixProtocol{
		NetflowV9Protocol: *v9.NewProtocolWithDecoder(decoder, cfg, cfg.LogOutput()),
	}
	return proto
}

// NewBufferedReader creates a new reader that can decode parquet data from an io.Reader.
// It will return an error if the parquet data stream cannot be read.
// Note: As io.ReadAll is used, the entire data stream would be read into memory, so very large data streams
// may cause memory bottleneck issues.
func NewBufferedReader(r io.Reader, cfg *Config) (*BufferedReader, error) {
	logger := logp.L().Named("reader.ipfix")

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Failed to read data from reader: %v", err)
	}

	// make a new netflow config object to pass into the protocol creator
	nfConfig := NewConfig().
		WithProtocols("ipfix").
		WithLogOutput(logger).
		WithCache(false).
		WithCustomFields(cfg.Fields()).
		WithSequenceResetEnabled(false).
		WithSharedTemplates(true)

	ipfixProtocol := NewProtocol(*nfConfig)

	return &BufferedReader{
		protocol: ipfixProtocol,
		data:     data,
		offset:   0,
		cfg:      cfg,
		logger:   logger,
	}, nil
}

// Next advances the pointer to point to the next record and returns true if the next record exists.
// It will return false if there are no more records to read.
func (sr *BufferedReader) Next() bool {

	offset := sr.offset
	// make sure there are at least four bytes left
	if offset + 4 > len(sr.data) {
		sr.logger.Debugf("Not enough left for reading: %d bytes left", len(sr.data) - offset)
		return false
	}

	// the IPFIX packet is two bytes of version, two bytes of length
	version := binary.BigEndian.Uint16(sr.data[offset+0:offset+2])
	length := binary.BigEndian.Uint16(sr.data[offset+2:offset+4])

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

type DummyAddr struct {
	NetworkValue string
	StringValue  string
}

func (d DummyAddr) Network() string {
	return d.NetworkValue
}
func (d DummyAddr) String() string {
	return d.StringValue
}

func NewDummyAddr() DummyAddr {
	return DummyAddr{NetworkValue: "tcp", StringValue: "100::"}
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

	source := NewDummyAddr()

	// make sure there are at least four bytes left
	if sr.offset + 4 > len(sr.data) {
		return nil, fmt.Errorf("Not enough left for reading: %d bytes left", len(sr.data) - sr.offset)
	}

	// the IPFIX packet is two bytes of version, two bytes of length
	offset := sr.offset
	version := binary.BigEndian.Uint16(sr.data[offset+0:offset+2])
	length := binary.BigEndian.Uint16(sr.data[offset+2:offset+4])

	// if the version is wrong, nothing else to read
	if version != 10 {
		return nil, fmt.Errorf("incorrect version (%v)", version)
	}

	// if the length is says so, nothing else to read
	if length < 4 {
		return nil, fmt.Errorf("packet is too small (%v)", length)
	}

	buf := sr.data[offset:offset+int(length)]
	pkt := bytes.NewBuffer(buf)
	sr.offset += int(length)

	sr.logger.Infof("Read a [%v / %#02x] record of length [%v / %#02x]; new offset is [%v / %#02x]", version, version, length, length, sr.offset, sr.offset)

	flows, err := sr.protocol.OnPacket(pkt, source)
	if err != nil {
		return nil, err
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

// Close closes the stream reader and releases all resources.
// It will return an error if the fileReader fails to close.
func (sr *BufferedReader) Close() error {
	return nil
}
*/
