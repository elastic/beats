// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"fmt"
	"net/http"
)

type apiError interface {
	Status() int
}

func createHandler(fn func(w http.ResponseWriter, r *http.Request) error) *apiHandler {
	return &apiHandler{
		innerFn: fn,
	}
}

type apiHandler struct {
	innerFn func(w http.ResponseWriter, r *http.Request) error
}

// ServeHTTP sets status code based on err returned
func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.innerFn(w, r)
	if err != nil {
		switch e := err.(type) {
		case apiError:
			w.WriteHeader(e.Status())
		default:
			w.WriteHeader(http.StatusInternalServerError)

		}

		writeResponse(w, unexpectedErrorWithReason(err.Error()))
	}
}

func unexpectedErrorWithReason(reason string, args ...interface{}) errResponse {
	return errResponse{
		Type:   errTypeUnexpected,
		Reason: fmt.Sprintf(reason, args...),
	}
}
