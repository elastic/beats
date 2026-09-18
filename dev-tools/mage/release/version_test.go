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

package release

import (
	"testing"
)

func TestSelectLatestReleaseBefore(t *testing.T) {
	tests := []struct {
		name    string
		current string
		tags    []string
		want    string
		wantErr bool
	}{
		{
			name:    "picks highest same-major below current",
			current: "9.5.0",
			tags:    []string{"v8.19.18", "v9.4.3", "v9.3.7", "v9.4.2"},
			want:    "9.4.3",
		},
		{
			name:    "ignores same or newer versions",
			current: "9.4.3",
			tags:    []string{"v9.4.3", "v9.4.2", "v9.5.0"},
			want:    "9.4.2",
		},
		{
			name:    "patch release predecessor",
			current: "9.5.1",
			tags:    []string{"v9.5.0", "v9.4.3"},
			want:    "9.5.0",
		},
		{
			name:    "no candidate",
			current: "9.0.0",
			tags:    []string{"v8.19.18", "v9.0.0"},
			wantErr: true,
		},
		{
			name:    "skips malformed tags",
			current: "9.5.0",
			tags:    []string{"not-a-version", "v9.4.1"},
			want:    "9.4.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectLatestReleaseBefore(tt.tags, tt.current)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
