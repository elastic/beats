// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metricset

import (
	"errors"
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/pkg/logp"
	libversion "github.com/elastic/beats/v7/pkg/version"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const MinimumEsVersion = "7.17.0"

var minVersion = libversion.MustNew(MinimumEsVersion)

// isVersionChecked is set once an unsupported version has been found and the agent is stopping,
// so that only the first metricset to see it stops the agent.
var isVersionChecked atomic.Bool

// fatalErrorHandler stops the agent for each error it receives. It is a variable so that tests can
// replace it.
var fatalErrorHandler = handleFatalErrors

const (
	CLUSTER_INFO_INITIAL_ERROR = 1
	CLUSTER_INFO_RUNTIME_ERROR = 2
)

// List of HTTP status codes that indicate the agent cannot recover
var terminalHttpErrorStatusCodes = []int{401, 403, 404}

func GetInfo(m *elasticsearch.MetricSet) (*utils.ClusterInfo, error) {
	info, err := utils.FetchAPIData[utils.ClusterInfo](m, "/")

	if err != nil {
		var httpResponse *utils.HTTPResponse
		handleClusterInfoError(m, err, httpResponse)
		return nil, err
	} else if info.ClusterID == "" || info.ClusterID == "_na_" {
		return nil, &utils.ClusterInfoError{Message: "cluster ID is unset, which means the cluster is not ready"}
	}

	// because different metricsets can call this function, we need to check the version only once
	if !isVersionChecked.Load() {
		if err := checkEsVersion(m.Logger(), info.Version.Number); err != nil {
			return nil, err
		}
	}

	return info, nil
}

func handleClusterInfoError(m *elasticsearch.MetricSet, err error, httpResponse *utils.HTTPResponse) {
	if errors.As(err, &httpResponse) {
		if slices.Contains(terminalHttpErrorStatusCodes, httpResponse.StatusCode) {
			// in these error cases Autoops agent can't recover itself, hence stop the agent
			errChan := make(chan error)
			go fatalErrorHandler(m.Logger(), errChan, CLUSTER_INFO_RUNTIME_ERROR)
			customErr := fmt.Errorf("autoops agent can't fetch the metrics due to http error! Code: %d, Status: %s",
				httpResponse.StatusCode, httpResponse.Status)
			errChan <- customErr
		}
	}
}

// checkEsVersion returns an error when the Elasticsearch version is not supported, and stops the agent
// the first time that happens.
//
// This runs on every fetch of every metricset, so it must not leave anything behind when the version
// is supported. The goroutine that stops the agent is only started once there is an error to handle:
// starting it up front leaked one goroutine per fetch, blocked forever on a channel that nothing
// wrote to.
func checkEsVersion(logger *logp.Logger, esVersion *libversion.V) error {
	if !esVersion.LessThan(minVersion) {
		return nil
	}

	err := &utils.VersionMismatchError{
		ExpectedVersion: minVersion.String(),
		ActualVersion:   esVersion.String(),
	}

	// different metricsets can get here at the same time, but only one must stop the agent
	if isVersionChecked.CompareAndSwap(false, true) {
		// for some reason log.Fatal() isn't working properly so we need to handle the error in a goroutine
		errChan := make(chan error)
		go fatalErrorHandler(logger, errChan, CLUSTER_INFO_INITIAL_ERROR)
		errChan <- err
	}

	return err
}
