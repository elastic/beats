// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"runtime"
	"testing"
)

type testCase struct {
	Endpoint    string
	Drop        string
	SkipWindows bool
}

func TestMonitoringDrops(t *testing.T) {
	cases := []testCase{
		testCase{`/var/lib/drop/abc.sock`, "/var/lib/drop", false},
		testCase{`npipe://drop`, "", false},
		testCase{`http+npipe://drop`, "", false},
		testCase{`\\.\pipe\drop`, "", false},
		testCase{`unix:///var/lib/drop/abc.sock`, "/var/lib/drop", true},
		testCase{`http+unix:///var/lib/drop/abc.sock`, "/var/lib/drop", true},
		testCase{`file:///var/lib/drop/abc.sock`, "/var/lib/drop", true},
		testCase{`http://localhost/stats`, "", false},
		testCase{`localhost/stats`, "", false},
		testCase{`http://localhost:8080/stats`, "", false},
		testCase{`localhost:8080/stats`, "", false},
		testCase{`http://1.2.3.4/stats`, "", false},
		testCase{`http://1.2.3.4:5678/stats`, "", false},
		testCase{`1.2.3.4:5678/stats`, "", false},
		testCase{`http://hithere.com:5678/stats`, "", false},
		testCase{`hithere.com:5678/stats`, "", false},
	}

	for _, c := range cases {
		t.Run(c.Endpoint, func(t *testing.T) {
			if runtime.GOOS == "windows" && c.SkipWindows {
				t.Skip("Skipped under windows")
			}

			drop := monitoringDrop(c.Endpoint)
			if drop != c.Drop {
				t.Errorf("Case[%s]: Expected '%s', got '%s'", c.Endpoint, c.Drop, drop)
			}
		})
	}
}
