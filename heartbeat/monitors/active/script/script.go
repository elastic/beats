// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package script

import (
	"fmt"
	"net/url"
	"os/user"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/synthexec"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func init() {
	monitors.RegisterActive("script", create)
	monitors.RegisterActive("synthetic/script", create)
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

	job := monitors.MakeSimpleCont(func(event *beat.Event) error {
		logp.Info("Start script job")

		cmdRes, err := synthexec.RunScript(config.Script)
		if err != nil {
			return fmt.Errorf("error running script: %w", err)
		}

		result := cmdRes.Result
		processResult(event, result)

		eventext.MergeEventFields(event, common.MapStr{
			"script": common.MapStr{
				"stdout": cmdRes.Stdout,
				"stderr": cmdRes.Stderr,
			},
		})

		if result != nil && len(result.Journeys) > 0 {
			eventext.MergeEventFields(event, common.MapStr{
				"script": common.MapStr{
					"journey": result.Journeys[0].Raw,
				},
			})
		}

		return nil
	})

	return []jobs.Job{job}, 1, nil
}

func processResult(event *beat.Event, result *synthexec.Result) {
	if result == nil {
		logp.Warn("no result received!")
		return
	}
	if result.Journeys == nil || len(result.Journeys) == 0 {
		logp.Warn("result received with no journies: %#v", result.Raw)
		return
	}

	journey := result.Journeys[0]
	status := "up"
	if journey.Error != nil {
		status = "down"
	}

	eventext.MergeEventFields(event, common.MapStr{
		"monitor": common.MapStr{
			"status":      status,
			"duration.us": journey.Duration,
		},
	})

	u, err := url.Parse(journey.Url)
	if err != nil {
		logp.Warn("Could not parse journey URL %s", journey.Url)
	}

	eventext.MergeEventFields(event, common.MapStr{
		"url": wrappers.URLFields(u),
	})
}
