// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
)

// csvDecoder is a decoder for CSV data.
type csvDecoder struct {
	r *csv.Reader

	header  []string
	current []string
	coming  []string

	err error
}

// newCSVDecoder creates a new CSV decoder.
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
	var err error
	d.coming, err = d.r.Read()
	if err != nil {
		return nil, err
	}
	d.current = make([]string, 0, len(d.header))
	return &d, nil
}

func (d *csvDecoder) more() bool { return len(d.coming) == len(d.header) }

// next advances the decoder to the next data item and returns true if
// there is more data to be decoded.
func (d *csvDecoder) next() bool {
	if !d.more() && d.err != nil {
		return false
	}
	d.current = d.current[:len(d.header)]
	copy(d.current, d.coming)
	d.coming, d.err = d.r.Read()
	if d.err == io.EOF {
		d.coming = nil
	}
	return true
}

// decode returns the JSON encoded value of the current CSV line. next must
// have been called before any calls to decode.
func (d *csvDecoder) decode() ([]byte, error) {
	err := d.check()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, n := range d.header {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		buf.WriteString(n)
		buf.WriteString(`":"`)
		buf.WriteString(d.current[i])
		buf.WriteByte('"')
	}
	buf.WriteByte('}')
	d.current = d.current[:0]
	return buf.Bytes(), nil
}

// decodeValue returns the value of the current CSV line interpreted as
// an object with fields based on the header held by the receiver. next must
// have been called before any calls to decode.
func (d *csvDecoder) decodeValue() ([]byte, map[string]any, error) {
	err := d.check()
	if err != nil {
		return nil, nil, err
	}
	m := make(map[string]any, len(d.header))
	for i, n := range d.header {
		m[n] = d.current[i]
	}
	d.current = d.current[:0]
	b, err := d.decode()
	if err != nil {
		return nil, nil, err
	}
	return b, m, nil
}

func (d *csvDecoder) check() error {
	if d.err != nil {
		if d.err == io.EOF && d.coming == nil {
			return nil
		}
		return d.err
	}
	if len(d.current) == 0 {
		return fmt.Errorf("decode called before next")
	}
	// By the time we are here, current must be the same
	// length as header; if it was not read, it would be
	// zero, but if it was, it must match by the contract
	// of the csv.Reader.
	return nil
}

// close closes the csv decoder and releases the resources.
func (d *csvDecoder) close() error {
	if d.err == io.EOF {
		return nil
	}
	return d.err
}
