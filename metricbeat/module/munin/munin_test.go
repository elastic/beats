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

package munin

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func dummyNode(response string) *Node {
	return &Node{
		writer: &bytes.Buffer{},
		reader: bufio.NewReader(bytes.NewBuffer([]byte(response))),
	}
}

func TestList(t *testing.T) {
	n := dummyNode("cpu df uptime\n")

	list, err := n.List()

	assert.NoError(t, err)

	expected := []string{"cpu", "df", "uptime"}
	assert.ElementsMatch(t, expected, list)
}

const (
	responseCPU = `user.value 4679836
nice.value 59278
system.value 1979168
idle.value 59957502
iowait.value 705373
irq.value 76
softirq.value 36404
steal.value 0
guest.value 0
.
`
	responseUnknown = `some.value U
other.value 42
.
`
	responseWithWrongFields = `user.value 4679836
nice.value 59278
system.value 1979168
idle.value 59957502
user.1000.value 23456
user.0.value 38284
.
`
)

func TestFetch(t *testing.T) {
	cases := []struct {
		title    string
		response string
		expected mapstr.M
	}{
		{
			"normal case",
			responseCPU,
			mapstr.M{
				"user":    float64(4679836),
				"nice":    float64(59278),
				"system":  float64(1979168),
				"idle":    float64(59957502),
				"iowait":  float64(705373),
				"irq":     float64(76),
				"softirq": float64(36404),
				"steal":   float64(0),
				"guest":   float64(0),
			},
		},
		{
			"unknown values",
			responseUnknown,
			mapstr.M{
				"other": float64(42),
			},
		},
		{
			"wrong field names",
			responseWithWrongFields,
			mapstr.M{
				"user":   float64(4679836),
				"nice":   float64(59278),
				"system": float64(1979168),
				"idle":   float64(59957502),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			n := dummyNode(c.response)
			event, err := n.Fetch("cpu", true)
			assert.Equal(t, c.expected, event)
			assert.NoError(t, err)
		})
	}
}

func TestSanitizeName(t *testing.T) {
	cases := []struct {
		name     string
		expected string
	}{
		{
			"if_eth0",
			"if_eth0",
		},
		{
			"/dev/sda1",
			"_dev_sda1",
		},
		{
			"eth0:100",
			"eth0_100",
		},
		{
			"user@host",
			"user_host",
		},
		{
			"404",
			"_04",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, sanitizeName(c.name))
		})
	}
}
