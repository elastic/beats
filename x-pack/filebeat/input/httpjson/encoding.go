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

	"github.com/elastic/elastic-agent-libs/mapstr"
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
	body, err := trReq.GetValue("body")
	if err == mapstr.ErrKeyNotFound {
		return nil, nil
	}
	header := trReq.header()
	header.Set("Content-Type", "application/json")
	trReq.setHeader(header)
	return json.Marshal(body)
}

// decodeAsJSON decodes the JSON message in p into dst.
func decodeAsJSON(p []byte, dst *response) error {
	err := json.Unmarshal(p, &dst.body)
	if err != nil {
		return textContextError{error: err, body: p}
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
			return textContextError{error: err, body: p}
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
			return textContextError{error: err, body: p}
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
				return textContextError{error: err, body: p}
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
		return textContextError{error: err, body: p}
	}
	dst.body = body
	dst.header["XML-CDATA"] = []string{cdata}
	return nil
}

// textContextError is an error that can provide the text context for
// a decoding error from the csv, json and xml packages.
type textContextError struct {
	error
	body []byte
}

func (e textContextError) Error() string {
	var ctx []byte
	switch err := e.error.(type) {
	case nil:
		return "<nil>"
	case *json.SyntaxError:
		ctx = textContext(e.body, err.Offset)
	case *json.UnmarshalTypeError:
		ctx = textContext(e.body, err.Offset)
	case *csv.ParseError:
		lines := bytes.Split(e.body, []byte{'\n'})
		l := err.Line - 1 // Lines are 1-based.
		if uint(l) >= uint(len(lines)) {
			return err.Error()
		}
		ctx = textContext(lines[l], int64(err.Column))
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
		ctx = textContext(lines[l], int64(pos))
	default:
		return err.Error()
	}
	return fmt.Sprintf("%v: text context %q", e.error, ctx)
}

func (e textContextError) Unwrap() error {
	return e.error
}

// textContext returns the context of text around the provided position starting
// ten bytes before pos and ten bytes after, dependent on the length of the
// text and the value of pos relative to bounds. If a text truncation is made,
// an ellipsis is added to indicate this.
func textContext(text []byte, pos int64) []byte {
	if len(text) == 0 {
		return text
	}
	const (
		dots = "..."
		span = 10
	)
	left := maxInt64(0, pos-span)
	right := minInt(pos+span+1, int64(len(text)))
	ctx := make([]byte, right-left+2*int64(len(dots)))
	copy(ctx[3:], text[left:right])
	if left != 0 {
		copy(ctx, dots)
		left = 0
	} else {
		left = int64(len(dots))
	}
	if right != int64(len(text)) {
		copy(ctx[len(ctx)-len(dots):], dots)
		right = int64(len(ctx))
	} else {
		right = int64(len(ctx) - len(dots))
	}
	return ctx[left:right]
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
