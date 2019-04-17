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

package kibana

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

const (
	fieldsYml = "testdata/fields.yml"
)

func TestNewGenerator(t *testing.T) {
	tmpDir := tmpPath(t)
	defer os.RemoveAll(tmpDir)

	data, err := ioutil.ReadFile("./testdata/fields.yml")
	if err != nil {
		t.Fatal(err)
	}

	v, _ := common.NewVersion("7.0.0")
	// checks for fields.yml
	generator, err := NewGenerator("beat-index", "mybeat.", data, "7.0", *v, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "7.0", generator.beatVersion)
	assert.Equal(t, "beat-index-*", generator.indexName)

}

func TestCleanName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: " beat index pattern", expected: "beatindexpattern"},
		{input: "Beat@Index.!", expected: "BeatIndex"},
		{input: "beatIndex", expected: "beatIndex"},
	}
	for idx, test := range tests {
		output := clean(test.input)
		msg := fmt.Sprintf("(%v): Expected <%s> Received: <%s>", idx, test.expected, output)
		assert.Equal(t, test.expected, output, msg)
	}
}

func TestGenerate(t *testing.T) {
	tmpDir := tmpPath(t)
	defer os.RemoveAll(tmpDir)

	v7, _ := common.NewVersion("7.0.0-alpha1")
	versions := []*common.Version{v7}
	var d common.MapStr
	for _, version := range versions {
		data, err := ioutil.ReadFile("./testdata/fields.yml")
		if err != nil {
			t.Fatal(err)
		}
		generator, err := NewGenerator("beat", "b eat ?!", data, version.String(), *version, true)
		if err != nil {
			t.Fatal(err)
		}

		d, err = generator.Generate()
		if err != nil {
			t.Fatal(err)
		}
	}

	tests := []compare{
		{
			existing: "testdata/beat-6.json",
			created:  d,
		},
	}

	testGenerate(t, tests, true)
}

type compare struct {
	existing string
	created  common.MapStr
}

func TestGenerateExtensive(t *testing.T) {
	tmpDir := tmpPath(t)
	defer os.RemoveAll(tmpDir)

	version7, _ := common.NewVersion("7.0.0-alpha1")
	versions := []*common.Version{version7}

	var d common.MapStr
	for _, version := range versions {
		data, err := ioutil.ReadFile("testdata/extensive/fields.yml")
		if err != nil {
			t.Fatal(err)
		}
		generator, err := NewGenerator("metricbeat", "metric be at ?!", data, version.String(), *version, true)
		if err != nil {
			t.Fatal(err)
		}

		d, err = generator.Generate()
		if err != nil {
			t.Fatal(err)
		}
	}

	tests := []compare{
		{
			existing: "testdata/extensive/metricbeat-6.json",
			created:  d,
		},
	}
	testGenerate(t, tests, false)
}

func testGenerate(t *testing.T, tests []compare, sourceFilters bool) {
	for _, test := range tests {
		// compare default
		existing, err := readJson(test.existing)
		if err != nil {
			t.Fatal(err)
		}

		var attrExisting, attrCreated common.MapStr

		if strings.Contains(test.existing, "6") {
			assert.Equal(t, existing["version"], test.created["version"])

			objExisting := existing["objects"].([]interface{})[0].(map[string]interface{})
			objCreated := test.created["objects"].([]common.MapStr)[0]

			assert.Equal(t, objExisting["version"], objCreated["version"])
			assert.Equal(t, objExisting["id"], objCreated["id"])
			assert.Equal(t, objExisting["type"], objCreated["type"])

			attrExisting = objExisting["attributes"].(map[string]interface{})
			attrCreated = objCreated["attributes"].(common.MapStr)
		} else {
			attrExisting = existing
			attrCreated = test.created
		}

		// check fieldFormatMap
		var ffmExisting, ffmCreated map[string]interface{}
		err = json.Unmarshal([]byte(attrExisting["fieldFormatMap"].(string)), &ffmExisting)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal([]byte(attrCreated["fieldFormatMap"].(string)), &ffmCreated)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, ffmExisting, ffmCreated)

		// check fields
		var fieldsExisting, fieldsCreated []map[string]interface{}
		err = json.Unmarshal([]byte(attrExisting["fields"].(string)), &fieldsExisting)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal([]byte(attrCreated["fields"].(string)), &fieldsCreated)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(fieldsExisting), len(fieldsCreated))
		for _, e := range fieldsExisting {
			idx := find(fieldsCreated, "name", e["name"].(string))
			assert.NotEqual(t, -1, idx)
			assert.Equal(t, e, fieldsCreated[idx])
		}

		// check sourceFilters
		if sourceFilters {
			var sfExisting, sfCreated []map[string]interface{}
			err = json.Unmarshal([]byte(attrExisting["sourceFilters"].(string)), &sfExisting)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal([]byte(attrCreated["sourceFilters"].(string)), &sfCreated)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, len(sfExisting), len(sfCreated))
			for _, e := range sfExisting {
				idx := find(sfCreated, "value", e["value"].(string))
				assert.NotEqual(t, -1, idx)
				assert.Equal(t, e, sfCreated[idx])
			}
		}
	}
}

func find(a []map[string]interface{}, key, val string) int {
	for idx, e := range a {
		if e[key].(string) == val {
			return idx
		}
	}
	return -1
}

func readJson(path string) (map[string]interface{}, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(f, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func tmpPath(t testing.TB) string {
	tmpDir, err := ioutil.TempDir("", "kibana-tests")
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}
