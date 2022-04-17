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

	"github.com/menderesk/beats/v7/libbeat/common/atomic"
)

const (
	fakeRunnersNum = 3
	fakeRunnerName = "fakeRunner"
)

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

	var runners []Runner
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

func TestString(t *testing.T) {
	var runners []Runner
	for i := 0; i < fakeRunnersNum; i++ {
		runners = append(runners, &fakeRunner{
			id: i,
		})
	}
	runnerGroup := newRunnerGroup(runners)
	assert.Equal(t, "RunnerGroup{fakeRunner-0, fakeRunner-1, fakeRunner-2}", runnerGroup.String())
}
