package kibana

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestNewGenerator(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	v, _ := common.NewVersion("7.0.0")
	// checks for fields.yml
	generator, err := NewGenerator("beat-index", "mybeat.", filepath.Join(beatDir, "notexistent"), "7.0", *v)
	assert.Error(t, err)

	generator, err = NewGenerator("beat-index", "mybeat.", beatDir, "7.0", *v)
	assert.NoError(t, err)
	assert.Equal(t, "7.0", generator.beatVersion)
	assert.Equal(t, "beat-index", generator.indexName)
	assert.Equal(t, filepath.Join(beatDir, "fields.yml"), generator.fieldsYaml)

	// creates file dir and sets name
	expectedDir := filepath.Join(beatDir, "_meta/kibana/6/index-pattern")
	assert.Equal(t, expectedDir, generator.targetDir)
	_, err = os.Stat(generator.targetDir)
	assert.NoError(t, err)

	v, _ = common.NewVersion("5.0.0")
	// checks for fields.yml
	generator, err = NewGenerator("beat-index", "mybeat.", beatDir, "7.0", *v)
	assert.NoError(t, err)

	expectedDir = filepath.Join(beatDir, "_meta/kibana/5/index-pattern")
	assert.Equal(t, expectedDir, generator.targetDir)
	_, err = os.Stat(generator.targetDir)

	assert.NoError(t, err)

	assert.Equal(t, "mybeat.json", generator.targetFilename)
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

func TestGenerateFieldsYaml(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	v, _ := common.NewVersion("6.0.0")
	generator, err := NewGenerator("metricbeat-*", "metric beat ?!", beatDir, "7.0.0-alpha1", *v)

	_, err = generator.Generate()
	assert.NoError(t, err)

	generator.fieldsYaml = ""
	_, err = generator.Generate()
	assert.Error(t, err)
}

func TestDumpToFile5x(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)
	v, _ := common.NewVersion("5.0.0")
	generator, err := NewGenerator("metricbeat-*", "metric beat ?!", beatDir, "7.0.0-alpha1", *v)

	_, err = generator.Generate()
	assert.NoError(t, err)

	generator.targetDir = "./non-existing/something"

	_, err = generator.Generate()
	assert.Error(t, err)
}

func TestDumpToFileDefault(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	v, _ := common.NewVersion("7.0.0")
	generator, err := NewGenerator("metricbeat-*", "metric beat ?!", beatDir, "7.0.0-alpha1", *v)

	_, err = generator.Generate()
	assert.NoError(t, err)

	generator.targetDir = "./non-existing/something"

	_, err = generator.Generate()
	assert.Error(t, err)
}

func TestGenerate(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	v5, _ := common.NewVersion("5.0.0")
	v6, _ := common.NewVersion("6.0.0")
	versions := []*common.Version{v5, v6}
	for _, version := range versions {
		generator, err := NewGenerator("beat-*", "b eat ?!", beatDir, "7.0.0-alpha1", *version)
		assert.NoError(t, err)

		_, err = generator.Generate()
		assert.NoError(t, err)
	}

	tests := []map[string]string{
		{"existing": "beat-5.json", "created": "_meta/kibana/5/index-pattern/beat.json"},
		{"existing": "beat-6.json", "created": "_meta/kibana/6/index-pattern/beat.json"},
	}
	testGenerate(t, beatDir, tests, true)
}

func TestGenerateExtensive(t *testing.T) {
	beatDir, err := filepath.Abs("./testdata/extensive")
	if err != nil {
		panic(err)
	}
	defer teardown(beatDir)

	version5, _ := common.NewVersion("5.0.0")
	version6, _ := common.NewVersion("6.0.0")
	versions := []*common.Version{version5, version6}
	for _, version := range versions {
		generator, err := NewGenerator("metricbeat-*", "metric be at ?!", beatDir, "7.0.0-alpha1", *version)
		assert.NoError(t, err)

		_, err = generator.Generate()
		assert.NoError(t, err)
	}

	tests := []map[string]string{
		{"existing": "metricbeat-5.json", "created": "_meta/kibana/5/index-pattern/metricbeat.json"},
		{"existing": "metricbeat-6.json", "created": "_meta/kibana/6/index-pattern/metricbeat.json"},
	}
	testGenerate(t, beatDir, tests, false)
}

func testGenerate(t *testing.T, beatDir string, tests []map[string]string, sourceFilters bool) {
	for _, test := range tests {
		// compare default
		existing, err := readJson(filepath.Join(beatDir, test["existing"]))
		assert.NoError(t, err)
		created, err := readJson(filepath.Join(beatDir, test["created"]))
		assert.NoError(t, err)

		var attrExisting, attrCreated common.MapStr

		if strings.Contains(test["existing"], "6") {
			assert.Equal(t, existing["version"], created["version"])

			objExisting := existing["objects"].([]interface{})[0].(map[string]interface{})
			objCreated := created["objects"].([]interface{})[0].(map[string]interface{})

			assert.Equal(t, objExisting["version"], objCreated["version"])
			assert.Equal(t, objExisting["id"], objCreated["id"])
			assert.Equal(t, objExisting["type"], objCreated["type"])

			attrExisting = objExisting["attributes"].(map[string]interface{})
			attrCreated = objCreated["attributes"].(map[string]interface{})
		} else {
			attrExisting = existing
			attrCreated = created
		}

		// check fieldFormatMap
		var ffmExisting, ffmCreated map[string]interface{}
		err = json.Unmarshal([]byte(attrExisting["fieldFormatMap"].(string)), &ffmExisting)
		assert.NoError(t, err)
		err = json.Unmarshal([]byte(attrCreated["fieldFormatMap"].(string)), &ffmCreated)
		assert.NoError(t, err)
		assert.Equal(t, ffmExisting, ffmCreated)

		// check fields
		var fieldsExisting, fieldsCreated []map[string]interface{}
		err = json.Unmarshal([]byte(attrExisting["fields"].(string)), &fieldsExisting)
		assert.NoError(t, err)
		err = json.Unmarshal([]byte(attrCreated["fields"].(string)), &fieldsCreated)
		assert.NoError(t, err)
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
			assert.NoError(t, err)
			err = json.Unmarshal([]byte(attrCreated["sourceFilters"].(string)), &sfCreated)
			assert.NoError(t, err)
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

func tmpPath() string {
	beatDir, err := filepath.Abs("./testdata")
	if err != nil {
		panic(err)
	}
	return beatDir
}

func teardown(path string) {
	if path == "" {
		path = tmpPath()
	}
	os.RemoveAll(filepath.Join(path, "_meta"))
}
