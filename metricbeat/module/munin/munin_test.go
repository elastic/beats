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

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
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

	assert.Nil(t, err)

	expected := []string{"cpu", "df", "uptime"}
	assert.ElementsMatch(t, expected, list)
}

func TestFetch(t *testing.T) {
	response := `user.value 4679836
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
	n := dummyNode(response)

	event, err := n.Fetch("cpu", "swap")

	assert.Nil(t, err)

	expected := common.MapStr{
		"cpu": common.MapStr{
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
	}
	assert.Equal(t, expected, event)
}

func TestFetchUnknown(t *testing.T) {
	response := `some.value U
other.value 42
.
`
	n := dummyNode(response)

	event, err := n.Fetch("test")

	assert.NotNil(t, err)

	expected := common.MapStr{
		"test": common.MapStr{
			"other": float64(42),
		},
	}
	assert.Equal(t, expected, event)
}
