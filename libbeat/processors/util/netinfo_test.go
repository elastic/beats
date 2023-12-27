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

package util

import (
	"net"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestUnique(t *testing.T) {
	tests := [][]string{
		{},
		{"a"},
		{"a", "a"},
		{"a", "b"},
		{"b", "a"},
		{"a", "b", "c"},
		{"c", "b", "a"},
		{"c", "a", "a", "b", "c", "a"},
	}

	for i, test := range tests {
		// Allocating naive implementation of unique.
		seen := make(map[string]bool)
		for _, e := range test {
			seen[e] = true
		}
		want := make([]string, 0, len(seen))
		for e := range seen {
			want = append(want, e)
		}
		sort.Strings(want)

		got := unique(test)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected result for test %d: got:%q want:%q", i, got, want)
		}
	}
}

func TestFormatHardwareAddr(t *testing.T) {
	tests := []string{
		"00:00:5e:00:53:01",
		"02:00:5e:10:00:00:00:01",
		"00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01",
		"00-00-5e-00-53-01",
		"02-00-5e-10-00-00-00-01",
		"00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01",
		"0000.5e00.5301",
		"0200.5e10.0000.0001",
		"0000.0000.fe80.0000.0000.0000.0200.5e10.0000.0001",
	}

	spec := regexp.MustCompile(`[0-9A-F]{2}(?:[0-9A-F]{2})*`)
	for _, test := range tests {
		addr, err := net.ParseMAC(test)
		if err != nil {
			t.Errorf("failed to parse test case %q", test)
			continue
		}
		got := formatHardwareAddr(addr)
		want := strings.ToUpper(strings.ReplaceAll(addr.String(), ":", "-"))
		if got != want || !spec.MatchString(got) {
			t.Errorf("unexpected format for %q: got:%q want:%q", test, got, want)
		}
	}
}
