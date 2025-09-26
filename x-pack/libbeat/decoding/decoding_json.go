// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoding

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// JSONDecoder is a decoder for json data.
type JSONDecoder struct {
	offset      int64
	isRootArray bool
	reader      *io.Reader
	decoder     *json.Decoder
}

// NewJSONDecoder creates a new json decoder.
// It returns an error if the json reader cannot be created.
func NewJSONDecoder(config DecoderConfig, r io.Reader) (Decoder, error) {
	r, isRootArray, err := evaluateJSON(bufio.NewReader(r))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate json with error: %w", err)
	}
	dec := json.NewDecoder(r)
	// If array is present at root then read json token and advance decoder
	if isRootArray {
		_, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read JSON token with error: %w", err)
		}
	}
	dec.UseNumber()

	return &JSONDecoder{
		isRootArray: isRootArray,
		reader:      &r,
		decoder:     dec,
	}, nil
}

// Next advances the json decoder to the next data item and returns true if there is more data to be decoded.
func (jd *JSONDecoder) Next() bool {
	return jd.decoder.More()
}

// Decode reads and decodes a json data stream. After reading the json data it decodes
// the output to JSON and returns it as a byte slice. It returns an error if the data cannot be decoded.
func (jd *JSONDecoder) Decode() ([]byte, error) {
	var data []byte
	if err := jd.decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}
	jd.offset = jd.decoder.InputOffset()
	return data, nil
}

// Type returns the underlying type of the decoder.
func (jd *JSONDecoder) Type() interface{} {
	return jd
}

// Offset returns the current offset of the json data stream.
func (jd *JSONDecoder) Offset() int64 {
	return jd.offset
}

// Seek seeks to the given offset in the json data stream.
func (jd *JSONDecoder) Seek(offset int64) error {
	for jd.decoder.InputOffset() < offset {
		_, err := jd.decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read JSON token with error: %w", err)
		}
	}
	return nil
}

// IsRootArray returns true if the root element of the json data is an array.
func (jd *JSONDecoder) IsRootArray() bool {
	return jd.isRootArray
}

// Close closes the json decoder and releases the resources.
func (jd *JSONDecoder) Close() error {
	jd.decoder = nil
	return nil
}
