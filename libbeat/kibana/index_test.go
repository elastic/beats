package kibana

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	beatDir := tmpPath()
	defer teardown(beatDir)

	//requires all args
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

	//checks for fields.yml
	idx := Index{Version: "7.0", IndexName: "beat-index", BeatDir: filepath.Join(beatDir, "notexistent"), BeatName: "mybeat."}
	err := idx.init()
	assert.Error(t, err)

	idx = Index{Version: "7.0", IndexName: "beat-index", BeatDir: beatDir, BeatName: "mybeat."}
	err = idx.init()
	assert.NoError(t, err)
	//creates file dir and sets name
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

	index := Index{Version: "7.0", IndexName: "libbeat-*", BeatDir: beatDir, BeatName: "lib beat !"}
	index.Create()

	tests := []map[string]string{
		{"existing": "5.x-libbeat.json", "new": "_meta/kibana/5.x/index-pattern/libbeat.json"},
		{"existing": "default-libbeat.json", "new": "_meta/kibana/default/index-pattern/libbeat.json"},
	}

	for _, test := range tests {
		//compare default
		existing, err := readJson(filepath.Join(beatDir, test["existing"]))
		assert.NoError(t, err)
		new, err := readJson(filepath.Join(beatDir, test["new"]))
		assert.NoError(t, err)
		assert.Equal(t, existing, new)
	}
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
