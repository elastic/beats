// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type httpHandler struct {
	log       *logp.Logger
	publisher stateless.Publisher

	messageField string
	responseCode int
	responseBody string
}

var (
	errBodyEmpty       = errors.New("body cannot be empty")
	errUnsupportedType = errors.New("only JSON objects are accepted")
)

// Triggers if middleware validation returns successful
func (h *httpHandler) apiResponse(w http.ResponseWriter, r *http.Request) {
	objs, status, err := httpReadJSON(r.Body)
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

	decoder := json.NewDecoder(body)
	for idx := 0; ; idx++ {
		var obj interface{}
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, http.StatusBadRequest, errors.Wrapf(err, "malformed JSON object at stream position %d", idx)
		}
		switch v := obj.(type) {
		case map[string]interface{}:
			objs = append(objs, v)
		case []interface{}:
			for listIdx, listObj := range v {
				asMap, ok := listObj.(map[string]interface{})
				if !ok {
					return nil, http.StatusBadRequest, fmt.Errorf("%v at stream %d index %d", errUnsupportedType, idx, listIdx)
				}
				objs = append(objs, asMap)
			}
		default:
			return nil, http.StatusBadRequest, errUnsupportedType
		}
	}
	return objs, 0, nil
}
