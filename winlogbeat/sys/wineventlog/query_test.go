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

//go:build !integration
// +build !integration

package wineventlog

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleQuery() {
	q, _ := Query{Log: "System", EventID: "10, 200-500, -311", Level: "info"}.Build()
	fmt.Println(q)
	// Output: <QueryList>
	//   <Query Id="0">
	//     <Select Path="System">*[System[(EventID=10 or (EventID &gt;= 200 and EventID &lt;= 500)) and (Level = 0 or Level = 4)]]</Select>
	//     <Suppress Path="System">*[System[(EventID=311)]]</Suppress>
	//   </Query>
	// </QueryList>
}

func TestIgnoreOlderQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[TimeCreated[timediff(@SystemTime) &lt;= 3600000]]]</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", IgnoreOlder: time.Hour}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		t.Log(q)
	}
}

func TestEventIDQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[(EventID=1 or (EventID &gt;= 1 and EventID &lt;= 100))]]</Select>
    <Suppress Path="Application">*[System[(EventID=75)]]</Suppress>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", EventID: "1, 1-100, -75"}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		t.Log(q)
	}
}

func TestLevelQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[(Level = 5)]]</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", Level: "Verbose"}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		t.Log(q)
	}
}

func TestProviderQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[Provider[@Name='mysrc']]]</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application", Provider: []string{"mysrc"}}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		t.Log(q)
	}
}

func TestCombinedQuery(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*[System[TimeCreated[timediff(@SystemTime) &lt;= 3600000] and (EventID=1 or (EventID &gt;= 1 and EventID &lt;= 100)) and (Level = 3)]]</Select>
    <Suppress Path="Application">*[System[(EventID=75)]]</Suppress>
  </Query>
</QueryList>`

	q, err := Query{
		Log:         "Application",
		IgnoreOlder: time.Hour,
		EventID:     "1, 1-100, -75",
		Level:       "Warning",
	}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		t.Log(q)
	}
}

func TestQueryNoParams(t *testing.T) {
	const expected = `<QueryList>
  <Query Id="0">
    <Select Path="Application">*</Select>
  </Query>
</QueryList>`

	q, err := Query{Log: "Application"}.Build()
	if assert.NoError(t, err) {
		assert.Equal(t, expected, q)
		t.Log(q)
	}
}
