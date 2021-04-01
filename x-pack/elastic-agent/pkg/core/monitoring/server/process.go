// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
)

const (
	processIDKey      = "processID"
	monitoringSuffix  = "-monitoring"
	separator         = "-"
	timeout           = 10 * time.Second
	errTypeUnexpected = "UNEXPECTED"

	httpPlusPrefix = "http+"
)

var (
	// ErrProgramNotSupported returned when requesting metrics for not supported program.
	ErrProgramNotSupported = errors.New("specified program is not supported")
)

func processHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		vars := mux.Vars(r)
		id, found := vars[processIDKey]

		if !found {
			writeResponse(
				w,
				unexpectedErrorWithReason("productID not found"),
			)
			return
		}

		metricsBytes, statusCode, metricsErr := processMetrics(r.Context(), id)
		switch metricsErr {
		case ErrProgramNotSupported:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(statusCode)
		}

		if metricsErr != nil {
			writeResponse(w, unexpectedErrorWithReason("failed fetching metrics: %s", metricsErr.Error()))
			return
		}

		fmt.Fprint(w, string(metricsBytes))
	}
}

func processMetrics(ctx context.Context, id string) ([]byte, int, error) {
	detail, err := parseID(id)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	endpoint := beats.MonitoringEndpoint(detail.spec, artifact.DefaultConfig().OS(), detail.output)
	if !strings.HasPrefix(endpoint, httpPlusPrefix) && !strings.HasPrefix(endpoint, "http") {
		// add prefix for npipe and unix
		endpoint = httpPlusPrefix + endpoint
	}

	if detail.isMonitoring {
		endpoint += "_monitor"
	}

	hostData, err := parse.ParseURL(endpoint, "http", "", "", "stats", "")
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	dialer, err := hostData.Transport.Make(timeout)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	client := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}

	req, err := http.NewRequest("GET", hostData.URI, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("fetching metrics failed: %v", err.Error())
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, syscall.ENOENT) {
			statusCode = http.StatusNotFound
		}
		return nil, statusCode, err
	}
	defer resp.Body.Close()

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return rb, resp.StatusCode, nil
}

func writeResponse(w http.ResponseWriter, c interface{}) {
	bytes, err := json.Marshal(c)
	if err != nil {
		// json marshal failed
		fmt.Fprintf(w, "Not valid json: %v", err)
		return
	}

	fmt.Fprint(w, string(bytes))

}

type programDetail struct {
	output       string
	binaryName   string
	isMonitoring bool
	spec         program.Spec
}

func parseID(id string) (programDetail, error) {
	var detail programDetail

	if strings.HasSuffix(id, monitoringSuffix) {
		detail.isMonitoring = true
		id = strings.TrimSuffix(id, monitoringSuffix)
	}

	for p, spec := range program.SupportedMap {
		if !strings.HasPrefix(id, p+separator) {
			continue
		}

		detail.binaryName = p
		detail.spec = spec
		break
	}

	if detail.binaryName == "" {
		return detail, ErrProgramNotSupported
	}

	detail.output = strings.TrimPrefix(id, detail.binaryName+separator)

	return detail, nil
}

func unexpectedErrorWithReason(reason string, args ...interface{}) errResponse {
	return errResponse{
		Type:   errTypeUnexpected,
		Reason: fmt.Sprintf(reason, args...),
	}
}
