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

package module

import (
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
)

type runnerGroup struct {
	runners []cfgfile.Runner

	startOnce sync.Once
	stopOnce  sync.Once
}

var _ cfgfile.Runner = new(runnerGroup)

func newRunnerGroup(runners []cfgfile.Runner) cfgfile.Runner {
	return &runnerGroup{
		runners: runners,
	}
}

func (rg *runnerGroup) Start() {
	rg.startOnce.Do(func() {
		for _, runner := range rg.runners {
			runner.Start()
		}
	})
}

func (rg *runnerGroup) Stop() {
	rg.stopOnce.Do(func() {
		for _, runner := range rg.runners {
			runner.Stop()
		}
	})
}

func (rg *runnerGroup) String() string {
	entries := make([]string, 0, len(rg.runners))
	for _, runner := range rg.runners {
		entries = append(entries, runner.String())
	}
	return "RunnerGroup{" + strings.Join(entries, ", ") + "}"
}

// Diagnostics, like the rest of the runner group methods, merely
// calls all the "client" runners and combines the results
func (rg *runnerGroup) Diagnostics() []diagnostics.DiagnosticSetup {
	results := []diagnostics.DiagnosticSetup{}
	for _, runner := range rg.runners {
		if diagHandler, ok := runner.(diagnostics.DiagnosticReporter); ok {
			diags := diagHandler.Diagnostics()
			results = append(results, diags...)
		}

	}
	return results
}
