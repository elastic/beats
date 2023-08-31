// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	stdxml "encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"unicode"

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
	err := json.Unmarshal(p, &dst.body)
	if err != nil {
		return jsonError{error: err, body: p}
	}
	return nil
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
			return jsonError{error: err, body: p}
		}
		results = append(results, o)
	}
	dst.body = results
	return nil
}

type jsonError struct {
	error
	body []byte
}

func (e jsonError) Error() string {
	switch err := e.error.(type) {
	case nil:
		return "<nil>"
	case *json.SyntaxError:
		return fmt.Sprintf("%v: text context %q", err, textContext(e.body, err.Offset))
	case *json.UnmarshalTypeError:
		return fmt.Sprintf("%v: text context %q", err, textContext(e.body, err.Offset))
	default:
		return err.Error()
	}
}

func (e jsonError) Unwrap() error {
	return e.error
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
			return csvError{error: err, body: p}
		}
	}

	dst.body = results

	return nil
}

type csvError struct {
	error
	body []byte
}

func (e csvError) Error() string {
	switch err := e.error.(type) {
	case nil:
		return "<nil>"
	case *csv.ParseError:
		lines := bytes.Split(e.body, []byte{'\n'})
		l := err.Line - 1 // Lines are 1-based.
		if uint(l) >= uint(len(lines)) {
			return err.Error()
		}
		return fmt.Sprintf("%v: text context %q", err, textContext(lines[l], int64(err.Column)))
	default:
		return err.Error()
	}
}

func (e csvError) Unwrap() error {
	return e.error
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
				return jsonError{error: err, body: p}
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
		return xmlError{error: err, body: p}
	}
	dst.body = body
	dst.header["XML-CDATA"] = []string{cdata}
	return nil
}

type xmlError struct {
	error
	body []byte
}

func (e xmlError) Error() string {
	switch err := e.error.(type) {
	case nil:
		return "<nil>"
	case *stdxml.SyntaxError:
		lines := bytes.Split(e.body, []byte{'\n'})
		l := err.Line - 1 // Lines are 1-based.
		if uint(l) >= uint(len(lines)) {
			return err.Error()
		}
		// The xml package does not provide column-level context,
		// so just point to first non-whitespace character of the
		// line. This doesn't make a great deal of difference
		// except in deeply indented XML documents.
		pos := bytes.IndexFunc(lines[l], func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		if pos < 0 {
			pos = 0
		}
		return fmt.Sprintf("%v: text context %q", err, textContext(lines[l], int64(pos)))
	default:
		return err.Error()
	}
}

func (e xmlError) Unwrap() error {
	return e.error
}

// textContext returns the context of text around the provided position starting
// five bytes before pos and extending ten bytes, dependent on the length of the
// text and the value of pos relative to bounds. If a text truncation is made,
// an ellipsis is added to indicate this. The returned []byte should not be mutated
// as it may be shared with the caller.
func textContext(text []byte, pos int64) []byte {
	left := maxInt64(0, pos-5)
	text = text[left:]
	var pad int64
	if left != 0 {
		pad = 3
		text = append([]byte("..."), text...)
	}
	right := minInt(pos+10+pad, int64(len(text)))
	if right != int64(len(text)) {
		// Ensure we don't clobber the body's bytes.
		text = append(text[:right:right], []byte("...")...)
	} else {
		text = text[:right]
	}
	return text
}

func minInt(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
