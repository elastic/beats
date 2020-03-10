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
		testCase{`/var/lib/drop/abc.sock`, "/var/lib/drop"},
		testCase{`npipe://drop`, ""},
		testCase{`http+npipe://drop`, ""},
		testCase{`\\.\pipe\drop`, ""},
		testCase{`unix:///var/lib/drop/abc.sock`, "/var/lib/drop"},
		testCase{`http+unix:///var/lib/drop/abc.sock`, "/var/lib/drop"},
		testCase{`file:///var/lib/drop/abc.sock`, "/var/lib/drop"},
		testCase{`http://localhost/stats`, ""},
		testCase{`localhost/stats`, ""},
		testCase{`http://localhost:8080/stats`, ""},
		testCase{`localhost:8080/stats`, ""},
		testCase{`http://1.2.3.4/stats`, ""},
		testCase{`http://1.2.3.4:5678/stats`, ""},
		testCase{`1.2.3.4:5678/stats`, ""},
		testCase{`http://hithere.com:5678/stats`, ""},
		testCase{`hithere.com:5678/stats`, ""},
	}

	for _, c := range cases {
		t.Run(c.Endpoint, func(t *testing.T) {
			// if runtime.GOOS == "windows" && c.SkipWindows {
			// 	t.Skip("Skipped under windows")
			// }

			drop := monitoringDrop(c.Endpoint)
			if drop != c.Drop {
				t.Errorf("Case[%s]: Expected '%s', got '%s'", c.Endpoint, c.Drop, drop)
			}
		})
	}
}
