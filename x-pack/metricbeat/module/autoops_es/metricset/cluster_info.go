// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metricset

import (
	"errors"
	"fmt"
	"slices"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

const MinimumEsVersion = "7.17.0"

var minVersion = libversion.MustNew(MinimumEsVersion)
var isVersionChecked = false

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
	if !isVersionChecked {
		// for some reason log.Fatal() isn't working properly so we need to handle the error in a goroutine
		errChan := make(chan error)
		go handleErrors(m.Logger(), errChan, CLUSTER_INFO_INITIAL_ERROR)

		if err := checkEsVersion(info.Version.Number, errChan); err != nil {
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
			go handleErrors(m.Logger(), errChan, CLUSTER_INFO_RUNTIME_ERROR)
			customErr := fmt.Errorf("autoops agent can't fetch the metrics due to http error! Code: %d, Status: %s",
				httpResponse.StatusCode, httpResponse.Status)
			errChan <- customErr
		}
	}
}

func checkEsVersion(esVersion *libversion.V, errChan chan error) error {
	if esVersion.LessThan(minVersion) {
		isVersionChecked = true
		err := &utils.VersionMismatchError{
			ExpectedVersion: minVersion.String(),
			ActualVersion:   esVersion.String(),
		}
		errChan <- err
		return err
	}

	return nil
}
