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

package beat

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestFQDNAwareHostname(t *testing.T) {
	info := Info{
		Hostname: "foo",
		FQDN:     "foo.bar.internal",
	}
	cases := map[string]struct {
		useFQDN bool
		want    string
	}{
		"fqdn_flag_enabled": {
			useFQDN: true,
			want:    "foo.bar.internal",
		},
		"fqdn_flag_disabled": {
			useFQDN: false,
			want:    "foo",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := info.FQDNAwareHostname(tc.useFQDN)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestInputHTTPMetrics_RegisterMetrics(t *testing.T) {
	type args struct {
		id    string
		input string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Valid Input",
			args: args{
				id:    "testID",
				input: "validInput",
			},
			wantErr: false,
		},
		{
			name: "Invalid Input - Missing ID",
			args: args{
				id:    "",
				input: "validInput",
			},
			wantErr:    true,
			wantErrMsg: "invalid metrics registry: 'id' empty or absent",
		},
		{
			name: "Invalid Input - Missing Input",
			args: args{
				id:    "testID",
				input: "",
			},
			wantErr:    true,
			wantErrMsg: "invalid metrics registry: 'input' empty or absent",
		},
		{
			name: "Invalid Input - Both Missing",
			args: args{
				id:    "",
				input: "",
			},
			wantErr:    true,
			wantErrMsg: "invalid metrics registry: 'id' empty or absent, 'input' empty or absent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up the registry
			reg := monitoring.NewRegistry()
			if tt.args.input != "" {
				monitoring.NewString(reg, "input").Set(tt.args.input)
			}
			if tt.args.id != "" {
				monitoring.NewString(reg, "id").Set(tt.args.id)
			}

			m := NewInputHTTPMetrics()

			inputID := tt.args.id
			err := m.RegisterMetrics(reg)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				got := m.registries[inputID]
				require.NotNil(t, got, "metrics registry was not registered")
				assert.Equal(t, reg, got)
			}
		})
	}
}

func TestInputHTTPMetrics_UnregisterMetrics(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()

	reg := monitoring.NewRegistry()
	monitoring.NewString(reg, "id").Set(id)
	monitoring.NewString(reg, "input").Set("some-input-type")

	m := NewInputHTTPMetrics()
	err := m.RegisterMetrics(reg)
	require.NoError(t, err, "could not register metrics")

	m.UnregisterMetrics(id)
	got := m.registries[id]
	assert.Nil(t, got, "metrics registry was not unregistered")
}

func TestInputHTTPMetrics_CollectStructSnapshot(t *testing.T) {
	tcs := []struct {
		name string
		want map[string]map[string]any
	}{
		{name: "empty", want: map[string]map[string]any{}},
		{name: "valid inputmetrics", want: map[string]map[string]any{
			"id1": {
				"id":    "id1",
				"input": "input-1",
				"foo":   int64(42),
				"boo":   1.618,
			},
			"id2": {
				"id":    "id2",
				"input": "input-2",
				"foo":   int64(10),
				"boo":   3.14,
			},
		}},
	}

	for _, tc := range tcs {
		im := NewInputHTTPMetrics()

		for _, want := range tc.want {
			reg := monitoring.NewRegistry()
			monitoring.NewString(reg, "id").Set(want["id"].(string))
			monitoring.NewString(reg, "input").Set(want["input"].(string))
			monitoring.NewInt(reg, "foo").Set(want["foo"].(int64))
			monitoring.NewFloat(reg, "boo").Set(want["boo"].(float64))

			err := im.RegisterMetrics(reg)
			require.NoError(t, err, "could not register metrics")
		}

		got := im.CollectStructSnapshot()
		assert.Equal(t, tc.want, got)
	}

}
