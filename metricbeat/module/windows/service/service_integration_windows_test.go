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

// +build integration windows

package service

import (
	"testing"

	"github.com/StackExchange/wmi"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

type Win32Service struct {
	Name        string
	ProcessId   uint32
	DisplayName string
	State       string
}

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"service"},
	}

	f := mbtest.NewEventsFetcher(t, config)
	f.Fetch()
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestReadService(t *testing.T) {
	reader, err := NewServiceReader()
	if err != nil {
		t.Fatal(err)
	}

	var wmiSrc []Win32Service

	// Get services per WMI
	err = wmi.Query("SELECT * FROM Win32_Service ", &wmiSrc)
	if err != nil {
		t.Fatal(err)
	}

	// Get services per windows module
	services, err := reader.Read()
	if err != nil {
		t.Fatal(err)
	}

	//Compare them
	for _, s := range services {
		// Look if the service is in the wmi src
		var found bool
		for _, w := range wmiSrc {
			if w.Name == s["name"] {
				if s["pid"] != nil {
					assert.Equal(t, w.ProcessId, s["pid"])
				}
				assert.Equal(t, w.State, s["state"])

				// For some services DisplayName and Name are the same. It seems to be a bug from the wmi query.
				if w.DisplayName != w.Name {
					assert.Equal(t, w.DisplayName, s["display_name"])
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
}
