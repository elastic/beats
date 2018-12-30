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

package cfgfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Output     ElasticsearchConfig
	Env        string `config:"env.test_key"`
	EnvDefault string `config:"env.default"`
}

type ElasticsearchConfig struct {
	Elasticsearch Connection
}

type Connection struct {
	Port int
	Host string
}

func TestRead(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")
	os.Setenv("TEST_KEY", "test_value")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &TestConfig{}

	if err = Read(config, absPath+"/config.yml"); err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, "localhost", config.Output.Elasticsearch.Host)
	assert.Equal(t, 9200, config.Output.Elasticsearch.Port)
	assert.Equal(t, "test_value", config.Env)
	assert.Equal(t, "default", config.EnvDefault)
}
