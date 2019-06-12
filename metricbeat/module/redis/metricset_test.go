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

// +build !integration

package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPasswordDatabase(t *testing.T) {
	cases := []struct {
		uri              string
		password         string
		expectedPassword string
		expectedDatabase int
	}{
		{
			"redis://127.0.0.1:6379",
			"testpassword",
			"testpassword",
			0,
		},
		{
			"redis://:testpassword@127.0.0.2:6379",
			"testpassword",
			"testpassword",
			0,
		},
		{
			"redis://127.0.0.1:6379?password=test",
			"",
			"test",
			0,
		},
		{
			"redis://127.0.0.1:6379?password=test&db=1",
			"",
			"test",
			1,
		},
	}

	for _, c := range cases {
		password, database, err := getPasswordDatabase(c.uri, c.password)
		assert.NoError(t, err)
		assert.Equal(t, c.expectedPassword, password)
		assert.Equal(t, c.expectedDatabase, database)
	}
}
