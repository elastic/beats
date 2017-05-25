// +build !integration

package template

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/version"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTemplate(t *testing.T) {
	// Load template
	absPath, err := filepath.Abs("../")
	if err != nil {
		t.Fatal(err)
	}

	beatInfo := common.BeatInfo{
		Beat:    "testbeat",
		Version: version.GetDefaultVersion(),
	}

	dir, err := ioutil.TempDir("", "test-template")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	outputFile := filepath.Join(dir, "template.json")

	config := newConfigFrom(t, TemplateConfig{
		Enabled: true,
		Fields:  filepath.Join(absPath, "fields.yml"),
		OutputToFile: OutputToFile{
			Path: outputFile,
		},
	})

	loader, err := NewLoader(config, nil, beatInfo)
	if err != nil {
		t.Fatal(err)
	}

	if err = loader.Generate(); err != nil {
		t.Fatal("generate failed", err)
	}

	// Read it back to check it
	fp, err := os.Open(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	jsonParser := json.NewDecoder(fp)
	var parsed common.MapStr
	if err = jsonParser.Decode(&parsed); err != nil {
		t.Fatal("decoding failed", err)
	}

	val, err := parsed.GetValue("mappings._default_._meta.version")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, val.(string), version.GetDefaultVersion())
}

func TestGenerateTemplateWithVersion(t *testing.T) {
	// Load template
	absPath, err := filepath.Abs("../")
	if err != nil {
		t.Fatal(err)
	}

	beatInfo := common.BeatInfo{
		Beat:    "testbeat",
		Version: version.GetDefaultVersion(),
	}

	dir, err := ioutil.TempDir("", "test-template")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	outputFile := filepath.Join(dir, "template.json")

	config := newConfigFrom(t, TemplateConfig{
		Enabled: true,
		Fields:  filepath.Join(absPath, "fields.yml"),
		OutputToFile: OutputToFile{
			Path:    outputFile,
			Version: "2.4.0",
		},
	})

	loader, err := NewLoader(config, nil, beatInfo)
	if err != nil {
		t.Fatal(err)
	}

	if err = loader.Generate(); err != nil {
		t.Fatal("generate failed", err)
	}

	// Read it back to check it
	fp, err := os.Open(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	jsonParser := json.NewDecoder(fp)
	var parsed common.MapStr
	if err = jsonParser.Decode(&parsed); err != nil {
		t.Fatal("decoding failed", err)
	}

	// Check a setting specific to that version.
	val, err := parsed.GetValue("mappings._default_._all.norms.enabled")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, val.(bool), false)
}

func newConfigFrom(t *testing.T, from interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(from)
	assert.NoError(t, err)
	return cfg
}
