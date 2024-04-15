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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
)

const (
	fakeRunnersNum = 3
	fakeRunnerName = "fakeRunner"
)

type fakeRunnerDiag struct {
	id int
}

func (fr *fakeRunnerDiag) Start() {}
func (fr *fakeRunnerDiag) Stop()  {}
func (fr *fakeRunnerDiag) String() string {
	return fmt.Sprintf("%s-%d", fakeRunnerName, fr.id)
}
func (fr *fakeRunnerDiag) Diagnostics() []diagnostics.DiagnosticSetup {
	return []diagnostics.DiagnosticSetup{
		{
			Name:     "test-diagnostic",
			Callback: func() []byte { return []byte("test result") },
		},
	}
}

type fakeRunner struct {
	id int

	startCounter *atomic.Int
	stopCounter  *atomic.Int
}

func (fr *fakeRunner) Start() {
	if fr.startCounter != nil {
		fr.startCounter.Inc()
	}
}

func (fr *fakeRunner) Stop() {
	if fr.stopCounter != nil {
		fr.stopCounter.Inc()
	}
}

func (fr *fakeRunner) String() string {
	return fmt.Sprintf("%s-%d", fakeRunnerName, fr.id)
}

func TestStartStop(t *testing.T) {
	startCounter := atomic.NewInt(0)
	stopCounter := atomic.NewInt(0)

	runners := make([]cfgfile.Runner, 0, fakeRunnersNum)
	for i := 0; i < fakeRunnersNum; i++ {
		runners = append(runners, &fakeRunner{
			id:           i,
			startCounter: startCounter,
			stopCounter:  stopCounter,
		})
	}

	runnerGroup := newRunnerGroup(runners)
	runnerGroup.Start()

	runnerGroup.Stop()

	assert.Equal(t, fakeRunnersNum, startCounter.Load())
	assert.Equal(t, fakeRunnersNum, stopCounter.Load())
}

func TestDiagnosticsUnsupported(t *testing.T) {
	runners := make([]cfgfile.Runner, 0, fakeRunnersNum)
	for i := 0; i < fakeRunnersNum; i++ {
		runners = append(runners, &fakeRunner{
			id:           i,
			startCounter: atomic.NewInt(0),
			stopCounter:  atomic.NewInt(0),
		})
	}

	runnerGroup := newRunnerGroup(runners)
	runnerGroup.Start()

	// fakeRunner doesn't support diagnostics, make sure nothing panics/returns invalid values
	diags, ok := runnerGroup.(diagnostics.DiagnosticReporter)
	// the runner group does implement the interface, but should return nothing
	require.True(t, ok)
	res := diags.Diagnostics()
	require.Empty(t, res)
}

func TestDiagosticsSupported(t *testing.T) {
	runners := make([]cfgfile.Runner, 0, fakeRunnersNum)
	for i := 0; i < fakeRunnersNum; i++ {
		runners = append(runners, &fakeRunnerDiag{
			id: i,
		})
	}
	runnerGroup := newRunnerGroup(runners)
	runnerGroup.Start()
	diags, ok := runnerGroup.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	res := diags.Diagnostics()
	require.NotEmpty(t, res)
}

func TestString(t *testing.T) {
	runners := make([]cfgfile.Runner, 0, fakeRunnersNum)
	for i := 0; i < fakeRunnersNum; i++ {
		runners = append(runners, &fakeRunner{
			id: i,
		})
	}
	runnerGroup := newRunnerGroup(runners)
	assert.Equal(t, "RunnerGroup{fakeRunner-0, fakeRunner-1, fakeRunner-2}", runnerGroup.String())
}
