// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
)

// csvDecoder is a decoder for CSV data.
type csvDecoder struct {
	r *csv.Reader

	header  []string
	offset  int64
	current []string

	err error
}

// newParquetDecoder creates a new CSV decoder.
func newCSVDecoder(config decoderConfig, r io.Reader) (decoder, error) {
	d := csvDecoder{r: csv.NewReader(r)}
	d.r.ReuseRecord = true
	if config.Codec.CSV.Comma != nil {
		d.r.Comma = rune(*config.Codec.CSV.Comma)
	}
	d.r.Comment = rune(config.Codec.CSV.Comment)
	d.r.LazyQuotes = config.Codec.CSV.LazyQuotes
	d.r.TrimLeadingSpace = config.Codec.CSV.TrimLeadingSpace
	if len(config.Codec.CSV.Fields) != 0 {
		d.r.FieldsPerRecord = len(config.Codec.CSV.Fields)
		d.header = config.Codec.CSV.Fields
	} else {
		h, err := d.r.Read()
		if err != nil {
			return nil, err
		}
		d.header = slices.Clone(h)
	}
	return &d, nil
}

// next advances the decoder to the next data item and returns true if
// there is more data to be decoded.
func (d *csvDecoder) next() bool {
	if d.err != nil {
		return false
	}
	d.offset = d.r.InputOffset()
	d.current, d.err = d.r.Read()
	return d.err == nil
}

// decode returns the JSON encoded value of the current CSV line. next must
// have been called before any calls to decode.
func (d *csvDecoder) decode() ([]byte, error) {
	_, v, err := d.decodeValue()
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// decodeValue returns the value of the current CSV line interpreted as
// an object with fields based on the header held by the receiver. next must
// have been called before any calls to decode.
func (d *csvDecoder) decodeValue() (offset int64, val any, _ error) {
	if d.err != nil {
		return d.offset, nil, d.err
	}
	if len(d.current) == 0 {
		return d.offset, nil, fmt.Errorf("decode called before next")
	}
	m := make(map[string]string, len(d.header))
	// By the time we are here, current must be the same
	// length as header; if it was not read, it would be
	// zero, but if it was, it must match by the contract
	// of the csv.Reader.
	for i, n := range d.header {
		m[n] = d.current[i]
	}
	return d.offset, m, nil
}

// close closes the parquet decoder and releases the resources.
func (d *csvDecoder) close() error {
	if d.err == io.EOF {
		return nil
	}
	return d.err
}
