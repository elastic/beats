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

package status

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockStatusReporter struct {
	s   Status
	msg string
}

func (m *mockStatusReporter) UpdateStatus(s Status, msg string) {
	m.s = s
	m.msg = msg
}

func TestGroupStatus(t *testing.T) {
	m := &mockStatusReporter{}
	reporter := NewGroupStatusReporter(m)

	subReporter1, subReporter2, subReporter3 := reporter.GetReporterForRunner(1), reporter.GetReporterForRunner(2), reporter.GetReporterForRunner(3)

	subReporter1.UpdateStatus(Running, "")
	subReporter2.UpdateStatus(Running, "")
	subReporter3.UpdateStatus(Running, "")

	require.Equal(t, m.s, Running)
	require.Equal(t, m.msg, "")

	subReporter1.UpdateStatus(Degraded, "Degrade Runner1")
	require.Equal(t, m.s, Degraded)
	require.Equal(t, m.msg, "Degrade Runner1")

	subReporter3.UpdateStatus(Degraded, "Failed Runner3")
	subReporter2.UpdateStatus(Failed, "Failed Runner2")

	require.Equal(t, m.s, Failed)
	require.Equal(t, m.msg, "Failed Runner2")
}
