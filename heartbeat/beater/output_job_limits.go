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

package beater

import (
	"github.com/elastic/beats/v7/heartbeat/config"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// applyOutputJobLimits applies the Heartbeat-specific control-plane fields
// carried by the Elastic Agent output unit. Input units only carry monitors,
// so per-type limits must be read here rather than from every stream.
func (bt *Heartbeat) applyOutputJobLimits(outputConfig *conf.C) {
	if outputConfig == nil {
		bt.logger.Warn("received empty output expected config; retaining current job limits")
		return
	}

	heartbeatConfig, err := outputConfig.Child("heartbeat", -1)
	if err != nil {
		bt.logger.Debug("output expected config has no heartbeat.jobs; retaining current job limits")
		return
	}
	jobsConfig, err := heartbeatConfig.Child("jobs", -1)
	if err != nil {
		bt.logger.Warn("output expected config has malformed heartbeat.jobs; retaining current job limits")
		return
	}

	limits := make(map[string]*config.JobLimit)
	if err := jobsConfig.Unpack(&limits); err != nil {
		bt.logger.Warnf("could not unpack output expected heartbeat.jobs: %v; retaining current job limits", err)
		return
	}
	if err := bt.scheduler.ApplyJobLimits(limits); err != nil {
		bt.logger.Warnf("could not apply output expected heartbeat.jobs: %v; retaining current job limits", err)
	}
}
