// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type httpBodyDecoder func(body io.Reader) (objs []common.MapStr, status int, err error)

type httpHandler struct {
	log       *logp.Logger
	publisher stateless.Publisher

	messageField string
	responseCode int
	responseBody string
	bodyDecoder  httpBodyDecoder
}

var (
	errBodyEmpty       = errors.New("body cannot be empty")
	errUnsupportedType = errors.New("only JSON objects are accepted")
)

// Triggers if middleware validation returns successful
func (h *httpHandler) apiResponse(w http.ResponseWriter, r *http.Request) {
	objs, status, err := h.bodyDecoder(r.Body)
	if err != nil {
		sendErrorResponse(w, status, err)
		return
	}

	for _, obj := range objs {
		h.publishEvent(obj)
	}
	h.sendResponse(w, h.responseCode, h.responseBody)
}

func (h *httpHandler) sendResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, message)
}

func (h *httpHandler) publishEvent(obj common.MapStr) {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			h.messageField: obj,
		},
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

func httpReadJSON(body io.Reader) (objs []common.MapStr, status int, err error) {
	if body == http.NoBody {
		return nil, http.StatusNotAcceptable, errBodyEmpty
	}

	contents, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed reading body: %w", err)
	}

	var jsBody interface{}
	if err := json.Unmarshal(contents, &jsBody); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("malformed JSON body: %w", err)
	}

	switch v := jsBody.(type) {
	case map[string]interface{}:
		objs = append(objs, v)
	case []interface{}:
		for idx, obj := range v {
			asMap, ok := obj.(map[string]interface{})
			if !ok {
				return nil, http.StatusBadRequest, fmt.Errorf("%v at index %d", errUnsupportedType, idx)
			}
			objs = append(objs, asMap)
		}
	default:
		return nil, http.StatusBadRequest, errUnsupportedType
	}
	return objs, 0, nil
}

func httpReadNDJSON(body io.Reader) (objs []common.MapStr, status int, err error) {
	if body == http.NoBody {
		return nil, http.StatusNotAcceptable, errBodyEmpty
	}

	decoder := json.NewDecoder(body)
	for idx := 0; ; idx++ {
		var obj interface{}
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, http.StatusBadRequest, errors.Wrapf(err, "malformed JSON object at stream position %d", idx)
		}
		asMap, ok := obj.(map[string]interface{})
		if !ok {
			return nil, http.StatusBadRequest, fmt.Errorf("%v at index %d", errUnsupportedType, idx)
		}
		objs = append(objs, asMap)
	}
	return objs, 0, nil
}
