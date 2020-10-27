// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"context"
	"fmt"
	"os/user"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/synthexec"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/common"
)

func init() {
	monitors.RegisterActive("browser", create)
	monitors.RegisterActive("synthetic/browser", create)
}

func create(name string, cfg *common.Config) (js []jobs.Job, endpoints int, err error) {
	curUser, err := user.Current()
	if err != nil {
		return nil, 0, fmt.Errorf("could not determine current user for script monitor %w: ", err)
	}
	if curUser.Uid == "0" {
		return nil, 0, fmt.Errorf("script monitors cannot be run as root! Current UID is %s", curUser.Uid)
	}

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	var j jobs.Job
	if config.Path != "" {
		j, err = synthexec.SuiteJob(context.TODO(), config.Path, config.JourneyName, config.Params)
		if err != nil {
			return nil, 0, err
		}
	} else {
		j = synthexec.JourneyJob(context.TODO(), config.Script, config.Params)
	}
	return []jobs.Job{j}, 1, nil
}
