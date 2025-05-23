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

package redis

import (
	"testing"

	"github.com/elastic/beats/v7/metricbeat/mb"

	"github.com/stretchr/testify/assert"
)

func TestGetPasswordDBNumber(t *testing.T) {
	cases := []struct {
		title            string
		hostData         mb.HostData
		expectedUser     string
		expectedPassword string
		expectedDatabase int
	}{
		{
			"test redis://127.0.0.1:6379 without password",
			mb.HostData{URI: "redis://127.0.0.1:6379", Password: ""},
			"",
			"",
			0,
		},
		{
			"test redis URI with password in userinfo",
			mb.HostData{URI: "redis://:password@127.0.0.2:6379", Password: "password"},
			"",
			"password",
			0,
		},
		{
			"test redis URI with password in query parameter",
			mb.HostData{URI: "redis://127.0.0.1:6379?password=test", Password: ""},
			"",
			"test",
			0,
		},
		{
			"test redis URI with password and db in query parameter",
			mb.HostData{URI: "redis://127.0.0.1:6379?password=test&db=1", Password: ""},
			"",
			"test",
			1,
		},
		{
			"test redis URI with password in userinfo and URI's query parameter",
			mb.HostData{URI: "redis://:password1@127.0.0.2:6379?password=password2", Password: "password1"},
			"",
			"password2",
			0,
		},
		{
			"test redis URI with db number in URI's query parameter and password in userinfo",
			mb.HostData{URI: "redis://:password1@127.0.0.2:6379/1", Password: "password1"},
			"",
			"password1",
			1,
		},
		{
			"test redis URI with db number and password in URI's query parameter and password in userinfo",
			mb.HostData{URI: "redis://:password1@127.0.0.2:6379/1?password=password2&db=2", Password: "password1"},
			"",
			"password2",
			2,
		},
		{
			"test redis URI with db number, user and password in URI's query parameter and password in userinfo",
			mb.HostData{URI: "redis://antirez:password1@127.0.0.2:6379/1?password=password2&db=2", User: "antirez", Password: "password1"},
			"antirez",
			"password2",
			2,
		},
		{
			"test redis URI with db number, user & password in URI's query parameter and user & password in userinfo",
			mb.HostData{URI: "redis://antirez:password1@127.0.0.2:6379/1?username=salvatore&password=password2&db=2", User: "antirez", Password: "password1"},
			"salvatore",
			"password2",
			2,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			username, password, database, err := getUsernamePasswordDBNumber(c.hostData)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedUser, username)
			assert.Equal(t, c.expectedPassword, password)
			assert.Equal(t, c.expectedDatabase, database)
		})
	}
}
