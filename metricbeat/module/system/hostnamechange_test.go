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

package system

import (
	"bytes"
	"io/ioutil"
	"testing"
)

// Checks that the Host Overview dashboard contains the CHANGEME_HOSTNAME variable
// that the dashboard loader code magically changes to the hostname on which the Beat
// is running.
func TestHostDashboardHasChangeableHost(t *testing.T) {
	dashPath := "_meta/kibana/7/dashboard/79ffd6e0-faa0-11e6-947f-177f697178b8-ecs.json"
	contents, err := ioutil.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("Error reading file %s: %v", dashPath, err)
	}
	if !bytes.Contains(contents, []byte("CHANGEME_HOSTNAME")) {
		t.Errorf("Dashboard '%s' doesn't contain string 'CHANGEME_HOSTNAME'. See elastic/beats#5340", dashPath)
	}
}
