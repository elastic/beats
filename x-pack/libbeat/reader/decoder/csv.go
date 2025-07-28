// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoder

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

// csvDecoder is a decoder for CSV data.
type CsvDecoder struct {
	r *csv.Reader

	header  []string
	offset  int64
	current []string

	err error
}

// NewCSVDecoder creates a new CSV decoder.
func NewCSVDecoder(config CsvCodecConfig, r io.Reader) (Decoder, error) {
	d := CsvDecoder{r: csv.NewReader(r)}
	d.r.ReuseRecord = true
	if config.Comma != nil {
		d.r.Comma = rune(*config.Comma)
	}
	d.r.Comment = rune(config.Comment)
	d.r.LazyQuotes = config.LazyQuotes
	d.r.TrimLeadingSpace = config.TrimLeadingSpace
	if len(config.Fields) != 0 {
		d.r.FieldsPerRecord = len(config.Fields)
		d.header = config.Fields
	} else {
		h, err := d.r.Read()
		if err != nil {
			return nil, err
		}
		d.header = slices.Clone(h)
	}
	return &d, nil
}

func (d *CsvDecoder) More() bool { return d.step() }

// next advances the decoder to the next data item and returns true if
// there is more data to be decoded.
func (d *CsvDecoder) Next() bool {
	return d.step()
}

func (d *CsvDecoder) step() bool {
	if d.err != nil {
		return false
	}
	if len(d.current) > 0 {
		return true
	}
	d.offset = d.r.InputOffset()
	d.current, d.err = d.r.Read()
	return d.err == nil
}

// decode returns the JSON encoded value of the current CSV line. next must
// have been called before any calls to decode.
func (d *CsvDecoder) Decode() ([]byte, error) {
	if err := d.Check(); err != nil {
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
func (d *CsvDecoder) DecodeValue() (int64, []byte, map[string]any, error) {
	if err := d.Check(); err != nil {
		return d.offset, nil, nil, err
	}
	m := make(map[string]any, len(d.header))
	// By the time we are here, current must be the same
	// length as header; if it was not read, it would be
	// zero, but if it was, it must match by the contract
	// of the csv.Reader.
	for i, n := range d.header {
		m[n] = d.current[i]
	}
	d.current = d.current[:0]

	b, err := json.Marshal(m)
	return d.offset, b, m, err
}

func (d *CsvDecoder) Check() error {
	if d.err != nil {
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
func (d *CsvDecoder) Close() error {
	if errors.Is(d.err, io.EOF) {
		return nil
	}
	return d.err
}
