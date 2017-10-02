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

func TestAllArgsSet(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	tests := []struct {
		Index Index
	}{
		{Index: Index{IndexName: "beat-index", BeatDir: beatDir, BeatName: "mybeat."}},
		{Index: Index{Version: "6.0", BeatDir: beatDir, BeatName: "mybeat."}},
		{Index: Index{Version: "6.0", IndexName: "beat-index", BeatName: "mybeat."}},
		{Index: Index{Version: "6.0", IndexName: "beat-index", BeatDir: beatDir}},
	}
	for idx, test := range tests {
		err := test.Index.init()
		msg := fmt.Sprintf("(%v): Should have raised error", idx)
		assert.Error(t, err, msg)
	}
}

func TestInit(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)
	// checks for fields.yml
	idx := Index{Version: "7.0", IndexName: "beat-index", BeatDir: filepath.Join(beatDir, "notexistent"), BeatName: "mybeat."}
	err := idx.init()
	assert.Error(t, err)

	idx = Index{Version: "7.0", IndexName: "beat-index", BeatDir: beatDir, BeatName: "mybeat."}
	err = idx.init()
	assert.NoError(t, err)

	// creates file dir and sets name
	expectedDir := filepath.Join(beatDir, "_meta/kibana/default/index-pattern")
	assert.Equal(t, expectedDir, idx.targetDirDefault)
	_, err = os.Stat(idx.targetDirDefault)
	assert.NoError(t, err)

	expectedDir = filepath.Join(beatDir, "_meta/kibana/5.x/index-pattern")
	assert.Equal(t, expectedDir, idx.targetDir5x)
	_, err = os.Stat(idx.targetDir5x)
	assert.NoError(t, err)

	assert.Equal(t, "mybeat.json", idx.targetFilename)
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

func TestDefault(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	index := Index{Version: "7.0.0-alpha1", IndexName: "metricbeat-*", BeatDir: beatDir, BeatName: "metric beat !"}
	index.Create()

	tests := []map[string]string{
		{"existing": "metricbeat-5x-old.json", "created": "_meta/kibana/5.x/index-pattern/metricbeat.json"},
		{"existing": "metricbeat-default-old.json", "created": "_meta/kibana/default/index-pattern/metricbeat.json"},
	}

	for _, test := range tests {
		// compare default
		existing, err := readJson(filepath.Join(beatDir, test["existing"]))
		assert.NoError(t, err)
		created, err := readJson(filepath.Join(beatDir, test["created"]))
		assert.NoError(t, err)

		var attrExisting, attrCreated common.MapStr

		if strings.Contains(test["existing"], "default") {
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
		assert.Equal(t, len(ffmExisting), len(ffmCreated))
		for _, e := range fieldsExisting {
			idx := find(fieldsCreated, e["name"].(string))
			assert.NotEqual(t, -1, idx)
			assert.Equal(t, fieldsCreated[idx], e)
		}
	}
}

func find(a []map[string]interface{}, k string) int {
	for idx, e := range a {
		if e["name"].(string) == k {
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
