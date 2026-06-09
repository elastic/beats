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

package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type collector struct {
	lines []string
}

func (c *collector) Inspect(line string) {
	c.lines = append(c.lines, line)
}
func (c *collector) String() string {
	return ""
}

func TestOutputInspector(t *testing.T) {

	t.Run("OverallInspector", func(t *testing.T) {
		cases := []struct {
			name  string
			input []string
		}{
			{
				name: "all lines are propagated",
				input: []string{
					"first",
					"second",
					"third",
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				inspectors := []OutputInspector{
					&collector{},
					&collector{},
					&collector{},
				}
				inspector := NewOverallInspector(inspectors)

				for _, l := range tc.input {
					inspector.Inspect(l)
				}
				for _, ins := range inspectors {
					c, ok := ins.(*collector)
					require.True(t, ok, "type must be `collector`")
					require.Equal(t, tc.input, c.lines, "collected lines must match the input")
				}
			})
		}
	})
}
