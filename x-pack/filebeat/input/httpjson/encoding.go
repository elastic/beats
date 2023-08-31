// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/elastic/mito/lib/xml"
)

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

var (
	// registeredEncoders is the set of available encoders.
	registeredEncoders = map[string]encoderFunc{
		"application/json":                  encodeAsJSON,
		"application/x-www-form-urlencoded": encodeAsForm,
	}
	// defaultEncoder is the decoder used when no registers
	// encoder is available.
	defaultEncoder = encodeAsJSON

	// registeredDecoders is the set of available decoders.
	registeredDecoders = map[string]decoderFunc{
		"application/json":        decodeAsJSON,
		"application/x-ndjson":    decodeAsNdjson,
		"text/csv":                decodeAsCSV,
		"application/zip":         decodeAsZip,
		"application/xml":         decodeAsXML,
		"text/xml; charset=utf-8": decodeAsXML,
	}
	// defaultDecoder is the decoder used when no registers
	// decoder is available.
	defaultDecoder = decodeAsJSON
)

type encoderFunc func(trReq transformable) ([]byte, error)
type decoderFunc func(p []byte, dst *response) error

// encodeAsJSON encodes trReq as a JSON message.
func encodeAsJSON(trReq transformable) ([]byte, error) {
	if len(trReq.body()) == 0 {
		return nil, nil
	}
	header := trReq.header()
	header.Set("Content-Type", "application/json")
	trReq.setHeader(header)
	return json.Marshal(trReq.body())
}

// decodeAsJSON decodes the JSON message in p into dst.
func decodeAsJSON(p []byte, dst *response) error {
	return json.Unmarshal(p, &dst.body)
}

// encodeAsForm encodes trReq as a URL encoded form.
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

// decodeAsNdjson decodes the message in p as a JSON object stream
// It is more relaxed than NDJSON.
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

// decodeAsCSV decodes p as a headed CSV document into dst.
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

// decodeAsZip decodes p as a ZIP archive into dst.
func decodeAsZip(p []byte, dst *response) error {
	var results []interface{}
	r, err := zip.NewReader(bytes.NewReader(p), int64(len(p)))
	if err != nil {
		return err
	}

	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		names = append(names, f.Name)

		dec := json.NewDecoder(rc)
		for dec.More() {
			var o interface{}
			if err := dec.Decode(&o); err != nil {
				rc.Close()
				return err
			}
			results = append(results, o)
		}
		rc.Close()
	}

	dst.body = results
	if dst.header == nil {
		dst.header = http.Header{}
	}
	dst.header["X-Zip-Files"] = names

	return nil
}

// decodeAsXML decodes p as an XML document into dst.
func decodeAsXML(p []byte, dst *response) error {
	cdata, body, err := xml.Unmarshal(bytes.NewReader(p), dst.xmlDetails)
	if err != nil {
		return err
	}
	dst.body = body
	dst.header["XML-CDATA"] = []string{cdata}
	return nil
}
