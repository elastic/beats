// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"testing"
)

type testCase struct {
	Endpoint string
	Drop     string
}

func TestMonitoringDrops(t *testing.T) {
	cases := []testCase{
		{`/var/lib/drop/abc.sock`, "/var/lib/drop"},
		{`npipe://drop`, ""},
		{`http+npipe://drop`, ""},
		{`\\.\pipe\drop`, ""},
		{`unix:///var/lib/drop/abc.sock`, "/var/lib/drop"},
		{`http+unix:///var/lib/drop/abc.sock`, "/var/lib/drop"},
		{`file:///var/lib/drop/abc.sock`, "/var/lib/drop"},
		{`http://localhost/stats`, ""},
		{`localhost/stats`, ""},
		{`http://localhost:8080/stats`, ""},
		{`localhost:8080/stats`, ""},
		{`http://1.2.3.4/stats`, ""},
		{`http://1.2.3.4:5678/stats`, ""},
		{`1.2.3.4:5678/stats`, ""},
		{`http://hithere.com:5678/stats`, ""},
		{`hithere.com:5678/stats`, ""},
	}

	for _, c := range cases {
		t.Run(c.Endpoint, func(t *testing.T) {
			drop := monitoringDrop(c.Endpoint)
			if drop != c.Drop {
				t.Errorf("Case[%s]: Expected '%s', got '%s'", c.Endpoint, c.Drop, drop)
			}
		})
	}
}
