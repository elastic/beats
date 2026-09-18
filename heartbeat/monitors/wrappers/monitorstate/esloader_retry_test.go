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

package monitorstate

import "testing"

func TestShouldRetry(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		want   bool
	}{
		{name: "transport failure", status: 0, want: true},
		{name: "server error", status: 500, want: true},
		{name: "client error", status: 404, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.status); got != tt.want {
				t.Errorf("shouldRetry(%d) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}
