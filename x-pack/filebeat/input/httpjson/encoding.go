// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type encoderFunc func(trReq transformable) ([]byte, error)

type decoderFunc func(p []byte, dst *response) error

var (
	registeredEncoders             = map[string]encoderFunc{}
	registeredDecoders             = map[string]decoderFunc{}
	defaultEncoder     encoderFunc = encodeAsJSON
	defaultDecoder     decoderFunc = decodeAsJSON
)

func registerEncoder(contentType string, enc encoderFunc) error {
	if contentType == "" {
		return errors.New("content-type can't be empty")
	}

	if enc == nil {
		return errors.New("encoder can't be nil")
	}

	if _, found := registeredEncoders[contentType]; found {
		return errors.New("already registered")
	}

	registeredEncoders[contentType] = enc

	return nil
}

func registerDecoder(contentType string, dec decoderFunc) error {
	if contentType == "" {
		return errors.New("content-type can't be empty")
	}

	if dec == nil {
		return errors.New("decoder can't be nil")
	}

	if _, found := registeredDecoders[contentType]; found {
		return errors.New("already registered")
	}

	registeredDecoders[contentType] = dec

	return nil
}

func encode(contentType string, trReq transformable) ([]byte, error) {
	enc, found := registeredEncoders[contentType]
	if !found {
		return defaultEncoder(trReq)
	}
	return enc(trReq)
}

func decode(contentType string, p []byte, dst *response) error {
	dec, found := registeredDecoders[contentType]
	if !found {
		return defaultDecoder(p, dst)
	}
	return dec(p, dst)
}

func registerEncoders() {
	log := logp.L().Named(logName)
	log.Debugf("registering encoder 'application/json': returned error: %#v",
		registerEncoder("application/json", encodeAsJSON))

	log.Debugf("registering encoder 'application/x-www-form-urlencoded': returned error: %#v",
		registerEncoder("application/x-www-form-urlencoded", encodeAsForm))
}

func registerDecoders() {
	log := logp.L().Named(logName)
	log.Debugf("registering decoder 'application/json': returned error: %#v",
		registerDecoder("application/json", decodeAsJSON))

	log.Debugf("registering decoder 'application/x-ndjson': returned error: %#v",
		registerDecoder("application/x-ndjson", decodeAsNdjson))

	log.Debugf("registering decoder 'text/csv': returned error: %#v",
		registerDecoder("text/csv", decodeAsCSV))
}

func encodeAsJSON(trReq transformable) ([]byte, error) {
	if len(trReq.body()) == 0 {
		return nil, nil
	}
	header := trReq.header()
	header.Set("Content-Type", "application/json")
	trReq.setHeader(header)
	return json.Marshal(trReq.body())
}

func decodeAsJSON(p []byte, dst *response) error {
	return json.Unmarshal(p, &dst.body)
}

func encodeAsForm(trReq transformable) ([]byte, error) {
	url := trReq.url()
	body := []byte(url.RawQuery)
	url.RawQuery = ""
	trReq.setURL(url)
	header := trReq.header()
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	trReq.setHeader(header)
	return body, nil
}

func decodeAsNdjson(p []byte, dst *response) error {
	var results []interface{}
	dec := json.NewDecoder(bytes.NewReader(p))
	for dec.More() {
		var o interface{}
		if err := dec.Decode(&o); err != nil {
			return err
		}
		results = append(results, o)
	}
	dst.body = results
	return nil
}

func decodeAsCSV(p []byte, dst *response) error {
	var results []interface{}

	r := csv.NewReader(bytes.NewReader(p))

	// a header is always expected, otherwise we can't map
	// values to keys in the event
	header, err := r.Read()
	if err != nil {
		if err == io.EOF { //nolint:errorlint // csv.Reader never wraps io.EOF.
			return nil
		}
		return err
	}

	event, err := r.Read()
	for ; err == nil; event, err = r.Read() {
		o := make(map[string]interface{}, len(header))
		if len(header) != len(event) {
			// sanity check, csv.Reader should fail on this scenario
			// and this code path should be unreachable
			return errors.New("malformed CSV, record does not match header length")
		}
		for i, h := range header {
			o[h] = event[i]
		}
		results = append(results, o)
	}

	if err != nil {
		if err != io.EOF { //nolint:errorlint // csv.Reader never wraps io.EOF.
			return err
		}
	}

	dst.body = results

	return nil
}
