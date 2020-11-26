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

package dot_expand

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestDotExpand(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	input := common.MapStr{
		"@timestamp":          "2019-08-06T12:09:12.375Z",
		"log.level":           "INFO",
		"message":             "Tomcat started on port(s): 8080 (http) with context path ''",
		"service.name":        "spring-petclinic",
		"process.thread.name": "restartedMain",
		"log.logger":          "org.springframework.boot.web.embedded.tomcat.TomcatWebServer",
	}

	actual, err := getActualValue(t, testConfig, input)
	require.NoError(t, err)

	expected := common.MapStr{
		"@timestamp": "2019-08-06T12:09:12.375Z",
		"log": common.MapStr{
			"level":  "INFO",
			"logger": "org.springframework.boot.web.embedded.tomcat.TomcatWebServer",
		},
		"message": "Tomcat started on port(s): 8080 (http) with context path ''",
		"service": common.MapStr{"name": "spring-petclinic"},
		"process": common.MapStr{"thread": common.MapStr{"name": "restartedMain"}},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestDotExpandFailOnError(t *testing.T) {
	input := common.MapStr{
		"a":           "not_an_object",
		"a.b":         "not_an_object_either",
		"expanded.in": "reverse",
	}

	// By default, dot_expand will fail on error.
	actual, err := getActualValue(t, common.NewConfig(), input)
	assert.Error(t, err)
	assert.EqualError(t, err, `incompatible types expanded for "a": old: common.MapStr new: string`)
	assert.Exactly(t, input, actual)

	// Even if we don't fail on error, the original input is kept intact.
	testConfig := common.MustNewConfigFrom(map[string]interface{}{"fail_on_error": false})
	actual, err = getActualValue(t, testConfig, input)
	require.NoError(t, err)
	assert.Exactly(t, input, actual)
}

func getActualValue(t *testing.T, config *common.Config, input common.MapStr) (common.MapStr, error) {
	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: input})
	return actual.Fields, err
}
