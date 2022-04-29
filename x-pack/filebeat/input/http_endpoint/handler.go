// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const headerContentEncoding = "Content-Encoding"

var (
	errBodyEmpty       = errors.New("body cannot be empty")
	errUnsupportedType = errors.New("only JSON objects are accepted")
)

type httpHandler struct {
	log       *logp.Logger
	publisher stateless.Publisher

	messageField          string
	responseCode          int
	responseBody          string
	includeHeaders        []string
	preserveOriginalEvent bool
}

// Triggers if middleware validation returns successful
func (h *httpHandler) apiResponse(w http.ResponseWriter, r *http.Request) {
	body, status, err := getBodyReader(r)
	if err != nil {
		sendAPIErrorResponse(w, r, h.log, status, err)
		return
	}
	defer body.Close()

	objs, _, status, err := httpReadJSON(body)
	if err != nil {
		sendAPIErrorResponse(w, r, h.log, status, err)
		return
	}

	var headers map[string]interface{}
	if len(h.includeHeaders) > 0 {
		headers = getIncludedHeaders(r, h.includeHeaders)
	}

	for _, obj := range objs {
		if err = h.publishEvent(obj, headers); err != nil {
			sendAPIErrorResponse(w, r, h.log, http.StatusInternalServerError, err)
			return
		}
	}

	h.sendResponse(w, h.responseCode, h.responseBody)
}

func (h *httpHandler) sendResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, message); err != nil {
		h.log.Debugw("Failed writing response to client.", "error", err)
	}
}

func (h *httpHandler) publishEvent(obj, headers mapstr.M) error {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields:    mapstr.M{},
	}
	if h.preserveOriginalEvent {
		event.Fields["event"] = mapstr.M{
			"original": obj.String(),
		}
	}
	if len(headers) > 0 {
		event.Fields["headers"] = headers
	}

	if _, err := event.PutValue(h.messageField, obj); err != nil {
		return fmt.Errorf("failed to put data into event key %q: %w", h.messageField, err)
	}

	h.publisher.Publish(event)
	return nil
}

func httpReadJSON(body io.Reader) (objs []mapstr.M, rawMessages []json.RawMessage, status int, err error) {
	if body == http.NoBody {
		return nil, nil, http.StatusNotAcceptable, errBodyEmpty
	}
	obj, rawMessage, err := decodeJSON(body)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}
	return obj, rawMessage, http.StatusOK, err
}

func decodeJSON(body io.Reader) (objs []mapstr.M, rawMessages []json.RawMessage, err error) {
	decoder := json.NewDecoder(body)
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
				break
			}
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", decoder.InputOffset(), err)
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", decoder.InputOffset(), err)
		}
		switch v := obj.(type) {
		case map[string]interface{}:
			objs = append(objs, v)
			rawMessages = append(rawMessages, raw)
		case []interface{}:
			nobjs, nrawMessages, err := decodeJSONArray(bytes.NewReader(raw))
			if err != nil {
				return nil, nil, fmt.Errorf("recursive error %d: %w", decoder.InputOffset(), err)
			}
			objs = append(objs, nobjs...)
			rawMessages = append(rawMessages, nrawMessages...)
		default:
			return nil, nil, errUnsupportedType
		}
	}
	for i := range objs {
		jsontransform.TransformNumbers(objs[i])
	}
	return objs, rawMessages, nil
}

func decodeJSONArray(raw *bytes.Reader) (objs []mapstr.M, rawMessages []json.RawMessage, err error) {
	dec := newJSONDecoder(raw)
	token, err := dec.Token()
	if err != nil {
		if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed reading JSON array: %w", err)
	}
	if token != json.Delim('[') {
		return nil, nil, fmt.Errorf("malformed JSON array, not starting with delimiter [ at position: %d", dec.InputOffset())
	}

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
				break
			}
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", dec.InputOffset(), err)
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", dec.InputOffset(), err)
		}

		m, ok := obj.(map[string]interface{})
		if ok {
			rawMessages = append(rawMessages, raw)
			objs = append(objs, m)
		}
	}
	return objs, rawMessages, nil
}

func getIncludedHeaders(r *http.Request, headerConf []string) (includedHeaders mapstr.M) {
	includedHeaders = mapstr.M{}
	for _, header := range headerConf {
		if value, found := r.Header[header]; found {
			includedHeaders[common.DeDot(header)] = value
		}
	}
	return includedHeaders
}

func newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}

// getBodyReader returns a reader that decodes the specified Content-Encoding.
func getBodyReader(r *http.Request) (body io.ReadCloser, status int, err error) {
	switch enc := r.Header.Get(headerContentEncoding); enc {
	case "gzip", "x-gzip":
		gzipReader, err := newPooledGzipReader(r.Body)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzipReader, 0, nil
	case "":
		// No encoding.
		return r.Body, 0, nil
	default:
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("unsupported Content-Encoding type %q", enc)
	}
}
