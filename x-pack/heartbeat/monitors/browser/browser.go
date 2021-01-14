// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"sync"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/synthexec"
)

func init() {
	monitors.RegisterActive("browser", create, "synthetic", "synthetics/synthetic")
}

var showExperimentalOnce = sync.Once{}

var NotSyntheticsCapableError = fmt.Errorf("synthetic monitors cannot be created outside the official elastic docker image")

func create(name string, cfg *common.Config) (js []jobs.Job, endpoints int, err error) {
	// We don't want users running synthetics in environments that don't have the required GUI libraries etc, so we check
	// this flag. When we're ready to support the many possible configurations of systems outside the docker environment
	// we can remove this check.
	if os.Getenv("ELASTIC_SYNTHETICS_CAPABLE") != "true" {
		return nil, 0, NotSyntheticsCapableError
	}

	showExperimentalOnce.Do(func() {
		logp.Info("Synthetic monitor detected! Please note synthetic monitors are an experimental unsupported feature!")
	})

	curUser, err := user.Current()
	if err != nil {
		return nil, 0, fmt.Errorf("could not determine current user for script monitor %w: ", err)
	}
	if curUser.Uid == "0" {
		return nil, 0, fmt.Errorf("script monitors cannot be run as root! Current UID is %s", curUser.Uid)
	}

	ss, err := NewSuite(cfg)
	if err != nil {
		return nil, 0, err
	}

	// TODO: Run this before each suite job invocation, not just at startup
	ss.Fetch()

	j, err := synthexec.SuiteJob(context.TODO(), ss.Workdir(), ss.Params())
	if err != nil {
			return nil, 0, err
		}
	return []jobs.Job{j}, 1, nil
}
