// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	errBodyEmpty       = errors.New("body cannot be empty")
	errUnsupportedType = errors.New("only JSON objects are accepted")
)

// decodeJSON accepts json file data in the form of an io.Reader, decodes the json data and returns the decoded
// data in the form of an object represended by a map[string]interface{}.
func decodeJSON(body io.Reader) ([]mapstr.M, error) {
	if body == http.NoBody {
		return nil, errBodyEmpty
	}
	var objs []mapstr.M
	decoder := json.NewDecoder(body)
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
				break
			}
			return nil, fmt.Errorf("malformed JSON object at stream position %d: %w", decoder.InputOffset(), err)
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, fmt.Errorf("malformed JSON object at stream position %d: %w", decoder.InputOffset(), err)
		}
		switch v := obj.(type) {
		case map[string]interface{}:
			objs = append(objs, v)
		case []interface{}:
			nobjs, err := decodeJSONArray(bytes.NewReader(raw), decoder.InputOffset())
			if err != nil {
				return nil, fmt.Errorf("recursive error %d: %w", decoder.InputOffset(), err)
			}
			objs = append(objs, nobjs...)
		default:
			return nil, errUnsupportedType
		}
	}
	for i := range objs {
		jsontransform.TransformNumbers(objs[i])
	}
	return objs, nil
}

func decodeJSONArray(raw *bytes.Reader, parentOffset int64) ([]mapstr.M, error) {
	var objs []mapstr.M
	dec := newJSONDecoder(raw)
	token, err := dec.Token()
	if err != nil {
		if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
			return nil, nil
		}
		return nil, fmt.Errorf("failed reading JSON array: %w", err)
	}
	if token != json.Delim('[') {
		return nil, fmt.Errorf("malformed JSON array, not starting with delimiter [ at position: %d", parentOffset+dec.InputOffset())
	}

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
				break
			}
			return nil, fmt.Errorf("malformed JSON object at stream position %d: %w", parentOffset+dec.InputOffset(), err)
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, fmt.Errorf("malformed JSON object at stream position %d: %w", parentOffset+dec.InputOffset(), err)
		}

		m, ok := obj.(map[string]interface{})
		if ok {
			objs = append(objs, m)
		}
	}
	return objs, nil
}

func newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}
