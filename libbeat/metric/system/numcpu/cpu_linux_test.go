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

package numcpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCPUParse(t *testing.T) {

	type cpuInput struct {
		input    string
		platform string
		expected int
	}

	cpuList := []cpuInput{
		{input: "0-23", platform: "basic X86", expected: 24},
		{input: "0-1", platform: "ARMv7", expected: 2},
		{input: "0-63", platform: "POWER7", expected: 64},
		{input: "0", platform: "QEMU", expected: 1},
		{input: "0-1,3", platform: "Kernel docs example 1", expected: 3},
		{input: "2,4-31,32-63", platform: "Kernel docs example 2", expected: 61},
	}

	for _, cpuTest := range cpuList {
		res, err := parseCPUList(cpuTest.input)
		assert.NoError(t, err, cpuTest.platform)
		assert.Equal(t, cpuTest.expected, res, cpuTest.platform)
	}

}
