package cfgfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Output ElasticsearchConfig
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

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &TestConfig{}

	err = Read(config, absPath+"/config.yml")
	assert.Nil(t, err)

	// Access config
	assert.Equal(t, "localhost", config.Output.Elasticsearch.Host)

	// Chat that it is integer
	assert.Equal(t, 9200, config.Output.Elasticsearch.Port)
}
