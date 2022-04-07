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

//go:build integration && windows
// +build integration,windows

package service

import (
	"testing"

	"github.com/elastic/beats/v8/libbeat/common"

	"github.com/StackExchange/wmi"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

type Win32Service struct {
	Name        string
	ProcessId   uint32
	DisplayName string
	State       string
	StartName   string
	PathName    string
}

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"service"},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func TestReadService(t *testing.T) {
	reader, err := NewReader()
	if err != nil {
		t.Fatal(err)
	}

	var wmiSrc []Win32Service

	// Get services from WMI, set NonePtrZero so nil fields are turned to empty strings
	wmi.DefaultClient.NonePtrZero = true
	err = wmi.Query("SELECT * FROM Win32_Service ", &wmiSrc)
	if err != nil {
		t.Fatal(err)
	}

	// Get services from Windows module.
	services, err := reader.Read()
	if err != nil {
		t.Fatal(err)
	}

	var stateChangedServices []common.MapStr

	// Compare our module's data against WMI.
	for _, s := range services {
		// Look if the service is in the WMI data.
		var found bool
		for _, w := range wmiSrc {
			if w.Name == s["name"] {
				if s["pid"] != nil {
					assert.Equal(t, w.ProcessId, s["pid"],
						"PID of service %v does not match", w.DisplayName)
				}
				assert.NotEmpty(t, s["start_type"])
				// For some services DisplayName and Name are the same. It seems to be a bug from the wmi query.
				if w.DisplayName != w.Name {
					assert.Equal(t, w.DisplayName, s["display_name"],
						"Display name of service %v does not match", w.Name)
				}
				// some services come back without PathName or StartName from WMI, just skip them
				if s["path_name"] != nil && w.PathName != "" {
					assert.Equal(t, w.PathName, s["path_name"],
						"Path name of service %v does not match", w.Name)
				}
				if s["start_name"] != nil && w.StartName != "" {
					assert.Equal(t, w.StartName, s["start_name"],
						"Start name of service %v does not match", w.Name)
				}
				// Some services have changed state before the second retrieval.
				if w.State != s["state"] {
					changed := s
					changed["initial_state"] = w.State
					stateChangedServices = append(stateChangedServices, changed)
				}
				found = true
				break
			}
		}

		if !found {
			// Service is not in the wmi query
			t.Errorf("Service %s can not be found by wmi query", s["name"])
		}
	}
	// If more than 90% of the services have the same state then we have enough confidence the state check works while being resilient to race conditions,
	// else it will require further investigation on which services are failing
	if stateChangedServices != nil {
		failing := float64(len(stateChangedServices)) / float64(len(services)) * 100
		if failing > 90 {
			// print entire information on the services failing
			t.Errorf("%.2f%% of the services have a different state than initial one \n : %s", failing, stateChangedServices)
		}
	}

}
