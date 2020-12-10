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

package registered_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestProcessorRun(t *testing.T) {
	var testCases = []struct {
		Error            bool
		Domain           string
		RegisteredDomain string
		Subdomain        string
	}{
		{false, "www.google.com", "google.com", "www"},
		{false, "www.google.co.uk", "google.co.uk", "www"},
		{false, "www.mail.google.co.uk", "google.co.uk", "www.mail"},
		{false, "google.com", "google.com", ""},
		{false, "www.ak.local", "ak.local", "www"},
		{false, "www.navy.mil", "navy.mil", "www"},

		{true, "com", "", ""},
		{true, ".", ".", ""},
		{true, "", "", ""},
		{true, "localhost", "", ""},
	}

	c := defaultConfig()
	c.Field = "domain"
	c.TargetField = "registered_domain"
	c.TargetSubdomainField = "subdomain"
	p, err := newRegisteredDomain(c)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testCases {
		evt := &beat.Event{
			Fields: common.MapStr{
				"domain": tc.Domain,
			},
		}

		evt, err := p.Run(evt)
		if tc.Error {
			t.Logf("Received expected error on domain [%v]: %v", tc.Domain, err)
			assert.Error(t, err)
			continue
		}
		if err != nil {
			t.Fatalf("Failed on domain [%v]: %v", tc.Domain, err)
		}

		rd, _ := evt.GetValue("registered_domain")
		assert.Equal(t, tc.RegisteredDomain, rd)

		if tc.Subdomain != "" {
			subdomain, _ := evt.GetValue("subdomain")
			assert.Equal(t, tc.Subdomain, subdomain)
		}
	}
}
