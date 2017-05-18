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
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	beatInfo := common.BeatInfo{
		Beat:    "testbeat",
		Version: version.GetDefaultVersion(),
	}

	dir, err := ioutil.TempDir("", "test-template")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "template.json")

	config := newConfigFrom(t, TemplateConfig{
		Enabled: true,
		Fields:  absPath + "/fields.yml",
		OutputToFile: OutputToFile{
			Path: path,
		},
	})

	loader, err := NewLoader(config, nil, beatInfo)
	assert.NoError(t, err)

	err = loader.Generate()
	assert.NoError(t, err)

	// Read it back to check it
	fp, err := os.Open(path)
	assert.NoError(t, err)
	jsonParser := json.NewDecoder(fp)
	var parsed common.MapStr
	err = jsonParser.Decode(&parsed)
	assert.NoError(t, err)

	val, err := parsed.GetValue("mappings._default_._meta.version")
	assert.NoError(t, err)
	assert.Equal(t, val.(string), version.GetDefaultVersion())

}

func TestGenerateTemplateWithVersion(t *testing.T) {

	// Load template
	absPath, err := filepath.Abs("../")
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	beatInfo := common.BeatInfo{
		Beat:    "testbeat",
		Version: version.GetDefaultVersion(),
	}

	dir, err := ioutil.TempDir("", "test-template")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "template.json")

	config := newConfigFrom(t, TemplateConfig{
		Enabled: true,
		Fields:  absPath + "/fields.yml",
		OutputToFile: OutputToFile{
			Path:    path,
			Version: "2.4.0",
		},
	})

	loader, err := NewLoader(config, nil, beatInfo)
	assert.NoError(t, err)

	err = loader.Generate()
	assert.NoError(t, err)

	// Read it back to check it
	fp, err := os.Open(path)
	assert.NoError(t, err)
	jsonParser := json.NewDecoder(fp)
	var parsed common.MapStr
	err = jsonParser.Decode(&parsed)
	assert.NoError(t, err)

	// check a setting specific to that version
	val, err := parsed.GetValue("mappings._default_._all.norms.enabled")
	assert.NoError(t, err)
	assert.Equal(t, val.(bool), false)
}

func newConfigFrom(t *testing.T, from interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(from)
	assert.NoError(t, err)
	return cfg
}
