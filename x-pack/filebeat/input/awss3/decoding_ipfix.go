// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/ipfix"
)

// ipfixDecoder is a decoder for ipfix data.
type ipfixDecoder struct {
	reader *ipfix.BufferedReader
}

// newipfixDecoder creates a new ipfix decoder. It uses the libbeat ipfix reader under the hood.
// It returns an error if the ipfix reader cannot be created.
func newIpfixDecoder(config decoderConfig, r io.Reader) (decoder, error) {
	reader, err := ipfix.NewBufferedReader(r, &ipfix.Config{
		InternalNetworks:  config.Codec.IPFIX.InternalNetworks,
		CustomDefinitions: config.Codec.IPFIX.CustomDefinitions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfix decoder: %w", err)
	}
	return &ipfixDecoder{
		reader: reader,
	}, nil
}

// next advances the ipfix decoder to the next data item and returns true if there is more data to be decoded.
func (pd *ipfixDecoder) next() bool {
	return pd.reader.Next()
}

// decode reads and decodes an IPFIX data stream. After reading the IPFIX data it decodes
// the output to JSON and returns it as a byte slice. It returns an error if the data cannot be decoded.
func (pd *ipfixDecoder) decode() ([]byte, error) {
	v, err := pd.decodeValue()
	if err != nil {
		return nil, err
	}
	output, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return output, err
}

// close closes the ipfix decoder and releases the resources.
func (pd *ipfixDecoder) close() error {
	return pd.reader.Close()
}

// return a json blob, to turn this decoder into a valueDecoder
func (pd *ipfixDecoder) decodeValue() (any, error) {
	flows, err := pd.reader.Record()
	if err != nil {
		return nil, err
	}
	return flows, nil
}
