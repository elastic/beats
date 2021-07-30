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
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
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
	invalidChars           = map[rune]struct{}{
		'"':  {},
		'<':  {},
		'>':  {},
		'|':  {},
		0:    {},
		':':  {},
		'*':  {},
		'?':  {},
		'\\': {},
		'/':  {},
		';':  {},
	}
)

func processHandler(statsHandler func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		vars := mux.Vars(r)
		id, found := vars[processIDKey]

		if !found {
			return errorfWithStatus(http.StatusNotFound, "productID not found")
		}

		if id == paths.BinaryName {
			// proxy stats for elastic agent process
			return statsHandler(w, r)
		}

		metricsBytes, statusCode, metricsErr := processMetrics(r.Context(), id)
		if metricsErr != nil {
			return metricsErr
		}

		if statusCode > 0 {
			w.WriteHeader(statusCode)
		}

		fmt.Fprint(w, string(metricsBytes))
		return nil
	}
}

func processMetrics(ctx context.Context, id string) ([]byte, int, error) {
	detail, err := parseID(id)
	if err != nil {
		return nil, 0, err
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
		return nil, 0, errorWithStatus(http.StatusInternalServerError, err)
	}

	dialer, err := hostData.Transport.Make(timeout)
	if err != nil {
		return nil, 0, errorWithStatus(http.StatusInternalServerError, err)
	}

	client := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}

	req, err := http.NewRequest("GET", hostData.URI, nil)
	if err != nil {
		return nil, 0, errorWithStatus(
			http.StatusInternalServerError,
			fmt.Errorf("fetching metrics failed: %v", err.Error()),
		)
	}

	req.Close = true
	cctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	resp, err := client.Do(req.WithContext(cctx))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, syscall.ENOENT) {
			statusCode = http.StatusNotFound
		}
		return nil, 0, errorWithStatus(statusCode, err)
	}
	defer resp.Body.Close()

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, errorWithStatus(http.StatusInternalServerError, err)
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
	if !isIDValid(id) {
		return detail, errorfWithStatus(http.StatusBadRequest, "provided ID is not valid")
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
		return detail, errorWithStatus(http.StatusNotFound, ErrProgramNotSupported)
	}

	if strings.HasSuffix(id, monitoringSuffix) {
		detail.isMonitoring = true
		id = strings.TrimSuffix(id, monitoringSuffix)
	}

	detail.output = strings.TrimPrefix(id, detail.binaryName+separator)

	return detail, nil
}

func isIDValid(id string) bool {
	for _, c := range id {
		if _, found := invalidChars[c]; found {
			return false
		}
	}

	return true
}
