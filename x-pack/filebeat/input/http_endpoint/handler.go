// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
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

var (
	errBodyEmpty       = errors.New("body cannot be empty")
	errUnsupportedType = errors.New("only JSON objects are accepted")
)

// Triggers if middleware validation returns successful
func (h *httpHandler) apiResponse(w http.ResponseWriter, r *http.Request) {
	var headers map[string]interface{}
	objs, _, status, err := httpReadJSON(r.Body)
	if err != nil {
		sendErrorResponse(w, status, err)
		return
	}
	if len(h.includeHeaders) > 0 {
		headers = getIncludedHeaders(r, h.includeHeaders)
	}
	for _, obj := range objs {
		h.publishEvent(obj, headers)
	}
	h.sendResponse(w, h.responseCode, h.responseBody)
}

func (h *httpHandler) sendResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, message)
}

func (h *httpHandler) publishEvent(obj common.MapStr, headers common.MapStr) {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			h.messageField: obj,
		},
	}
	if h.preserveOriginalEvent {
		event.PutValue("event.original", obj.String())
	}
	if len(headers) > 0 {
		event.PutValue("headers", headers)
	}

	h.publisher.Publish(event)
}

func withValidator(v validator, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if status, err := v.ValidateHeader(r); status != 0 && err != nil {
			sendErrorResponse(w, status, err)
		} else {
			handler(w, r)
		}
	}
}

func sendErrorResponse(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	e.Encode(common.MapStr{"message": err.Error()})
}

func httpReadJSON(body io.Reader) (objs []common.MapStr, rawMessages []json.RawMessage, status int, err error) {
	if body == http.NoBody {
		return nil, nil, http.StatusNotAcceptable, errBodyEmpty
	}
	obj, rawMessage, err := decodeJSON(body)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}
	return obj, rawMessage, http.StatusOK, err
}

func decodeJSON(body io.Reader) (objs []common.MapStr, rawMessages []json.RawMessage, err error) {
	decoder := json.NewDecoder(body)
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, errors.Wrapf(err, "malformed JSON object at stream position %d", decoder.InputOffset())
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, nil, errors.Wrapf(err, "malformed JSON object at stream position %d", decoder.InputOffset())
		}
		switch v := obj.(type) {
		case map[string]interface{}:
			objs = append(objs, v)
			rawMessages = append(rawMessages, raw)
		case []interface{}:
			nobjs, nrawMessages, err := decodeJSONArray(bytes.NewReader(raw))
			if err != nil {
				return nil, nil, errors.Wrapf(err, "recursive error %d", decoder.InputOffset())
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

func decodeJSONArray(raw *bytes.Reader) (objs []common.MapStr, rawMessages []json.RawMessage, err error) {
	dec := newJSONDecoder(raw)
	token, err := dec.Token()
	if token != json.Delim('[') || err != nil {
		return nil, nil, errors.Wrapf(err, "malformed JSON array, not starting with delimiter [ at position: %d", dec.InputOffset())
	}

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, errors.Wrapf(err, "malformed JSON object at stream position %d", dec.InputOffset())
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, nil, errors.Wrapf(err, "malformed JSON object at stream position %d", dec.InputOffset())
		}

		m, ok := obj.(map[string]interface{})
		if ok {
			rawMessages = append(rawMessages, raw)
			objs = append(objs, m)
		}
	}
	return
}

func getIncludedHeaders(r *http.Request, headerConf []string) (includedHeaders common.MapStr) {
	includedHeaders = common.MapStr{}
	for _, header := range headerConf {
		h, found := r.Header[header]
		if found {
			includedHeaders.Put(header, h)
		}
	}
	return includedHeaders
}

func newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}
