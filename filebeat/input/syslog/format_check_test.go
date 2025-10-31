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

package syslog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRFC5424(t *testing.T) {
	assert.True(t, IsRFC5424Format([]byte(RfcDoc65Example1)))
	assert.True(t, IsRFC5424Format([]byte(RfcDoc65Example2)))
	assert.True(t, IsRFC5424Format([]byte(RfcDoc65Example3)))
	assert.True(t, IsRFC5424Format([]byte(RfcDoc65Example4)))
	assert.False(t, IsRFC5424Format([]byte("<190>2018-06-19T02:13:38.635322-0700 super mon message")))
	assert.False(t, IsRFC5424Format([]byte("<190>589265: Feb 8 18:55:31.306: %SEC-11-IPACCESSLOGP: list 177 denied udp 10.0.0.1(53640) -> 10.100.0.1(15600), 1 packet")))
}
