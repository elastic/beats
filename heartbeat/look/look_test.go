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

package look

import (
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"

	reason2 "github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// helper
func testRTT(t *testing.T, expected time.Duration, provided time.Duration) {
	actual, err := RTT(provided).GetValue("us")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestPositiveRTTIsKept(t *testing.T) {
	testRTT(t, 5, time.Duration(5*time.Microsecond))
}

func TestNegativeRTTIsZero(t *testing.T) {
	testRTT(t, time.Duration(0), time.Duration(-1))
}

func TestReason(t *testing.T) {
	reason := reason2.ValidateFailed(fmt.Errorf("an error"))
	res := Reason(reason)
	assert.Equal(t,
		mapstr.M{
			"type":    reason.Type(),
			"message": reason.Error(),
		}, res)
}

func TestReasonGenericError(t *testing.T) {
	msg := "An error"
	res := Reason(fmt.Errorf(msg))
	assert.Equal(t, mapstr.M{
		"type":    "io",
		"message": msg,
	}, res)
}

func TestTimestamp(t *testing.T) {
	now := time.Now()
	assert.Equal(t, common.Time(now), Timestamp(now))
}

func TestStatusNil(t *testing.T) {
	assert.Equal(t, "up", Status(nil))
}

func TestStatusErr(t *testing.T) {
	assert.Equal(t, "down", Status(fmt.Errorf("something")))
}
