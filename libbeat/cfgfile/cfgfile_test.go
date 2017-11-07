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
