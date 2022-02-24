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

package safemapstr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestPut(t *testing.T) {
	m := mapstr.M{
		"subMap": mapstr.M{
			"a": 1,
		},
	}

	// Add new value to the top-level.
	err := Put(m, "a", "ok")
	assert.NoError(t, err)
	assert.Equal(t, mapstr.M{"a": "ok", "subMap": mapstr.M{"a": 1}}, m)

	// Add new value to subMap.
	err = Put(m, "subMap.b", 2)
	assert.NoError(t, err)
	assert.Equal(t, mapstr.M{"a": "ok", "subMap": mapstr.M{"a": 1, "b": 2}}, m)

	// Overwrite a value in subMap.
	err = Put(m, "subMap.a", 2)
	assert.NoError(t, err)
	assert.Equal(t, mapstr.M{"a": "ok", "subMap": mapstr.M{"a": 2, "b": 2}}, m)

	// Add value to map that does not exist.
	m = mapstr.M{}
	err = Put(m, "subMap.newMap.a", 1)
	assert.NoError(t, err)
	assert.Equal(t, mapstr.M{"subMap": mapstr.M{"newMap": mapstr.M{"a": 1}}}, m)
}

func TestPutRenames(t *testing.T) {
	assert := assert.New(t)

	a := mapstr.M{}
	err := Put(a, "com.docker.swarm.task", "x")
	require.Nil(t, err)
	err = Put(a, "com.docker.swarm.task.id", 1)
	require.Nil(t, err)
	err = Put(a, "com.docker.swarm.task.name", "foobar")
	require.Nil(t, err)
	assert.Equal(mapstr.M{"com": mapstr.M{"docker": mapstr.M{"swarm": mapstr.M{
		"task": mapstr.M{
			"id":    1,
			"name":  "foobar",
			"value": "x",
		}}}}}, a)

	// order is not important:
	b := mapstr.M{}
	err = Put(b, "com.docker.swarm.task.id", 1)
	require.Nil(t, err)
	err = Put(b, "com.docker.swarm.task.name", "foobar")
	require.Nil(t, err)
	err = Put(b, "com.docker.swarm.task", "x")
	require.Nil(t, err)
	assert.Equal(mapstr.M{"com": mapstr.M{"docker": mapstr.M{"swarm": mapstr.M{
		"task": mapstr.M{
			"id":    1,
			"name":  "foobar",
			"value": "x",
		}}}}}, b)
}
