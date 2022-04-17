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
	"net"
	"os/exec"
	"testing"
	"time"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	sockPath := "/var/run/libvirt/libvirt-sock"
	checkLibvirt(t, sockPath)

	f := mbtest.NewFetcher(t, getConfig("unix://"+sockPath))
	f.WriteEvents(t, "")
}

func checkLibvirt(t *testing.T, sockPath string) {
	if exec.Command("kvm-ok").Run() != nil {
		t.Skip("kvm not available")
	}

	c, err := net.DialTimeout("unix", sockPath, 5*time.Second)
	if err != nil {
		t.Skipf("cannot connect to %s: %v", sockPath, err)
	}
	c.Close()
}
