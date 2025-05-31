// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metricset

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/logp"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

const MinimumEsVersion = "7.17.0"

var minVersion = libversion.MustNew(MinimumEsVersion)
var isVersionChecked = false

func GetInfo(m *elasticsearch.MetricSet) (*utils.ClusterInfo, error) {
	info, err := utils.FetchAPIData[utils.ClusterInfo](m, "/")

	if err != nil {
		return nil, err
	} else if info.ClusterID == "" || info.ClusterID == "_na_" {
		return nil, errors.New("cluster ID is unset, which means the cluster is not ready")
	}

	// because different metricsets can call this function, we need to check the version only once
	if !isVersionChecked {
		// for some reason log.Fatal() isn't working properly so we need to handle the error in a goroutine
		errChan := make(chan error)
		go handleErrors(errChan)

		if err := checkEsVersion(info.Version.Number, errChan); err != nil {
			return nil, err
		}
	}

	return info, nil
}

func checkEsVersion(esVersion *libversion.V, errChan chan error) error {
	if esVersion.LessThan(minVersion) {
		isVersionChecked = true
		err := fmt.Errorf("version %s is less than the minimum required version %s", esVersion.String(), minVersion)
		errChan <- err
		return err
	}

	return nil
}

func handleErrors(errChan chan error) {
	for err := range errChan {
		logp.Error(err)
		// sleep is needed to make sure the error is logged and error event is sent before exiting
		time.Sleep(time.Second * 5)
		os.Exit(1)
	}
}
