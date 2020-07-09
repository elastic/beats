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
	"io/ioutil"
	"net/http"
	"time"

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

var errBodyEmpty = errors.New("Body cannot be empty")
var errUnsupportedType = errors.New("Only JSON objects are accepted")

// Triggers if middleware validation returns successful
func (h *httpHandler) apiResponse(w http.ResponseWriter, r *http.Request) {
	obj, status, err := httpReadJsonObject(r.Body)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		h.sendErrorResponse(w, status, err)
		return
	}

	h.publishEvent(obj)
	w.Header().Add("Content-Type", "application/json")
	h.sendResponse(w, h.responseCode, h.responseBody)
}

func (h *httpHandler) sendResponse(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	io.WriteString(w, message)
}

func (h *httpHandler) sendErrorResponse(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"message": %q}`, err.Error())
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

func httpReadJsonObject(body io.Reader) (obj common.MapStr, status int, err error) {
	if body == http.NoBody {
		return nil, http.StatusNotAcceptable, errBodyEmpty
	}

	contents, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed reading body: %w", err)
	}

	if !isObject(contents) {
		return nil, http.StatusBadRequest, errUnsupportedType
	}

	obj = common.MapStr{}
	if err := json.Unmarshal(contents, &obj); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Malformed JSON body: %w", err)
	}

	return obj, 0, nil
}

func isObject(b []byte) bool {
	obj := bytes.TrimLeft(b, " \t\r\n")
	if len(obj) > 0 && obj[0] == '{' {
		return true
	}
	return false
}
