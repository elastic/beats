package file

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestConfig(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"file.paths":         []string{"/usr/bin"},
		"file.hash_types":    []string{"md5", "sha256"},
		"file.max_file_size": "1 GiB",
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := config.Unpack(&c); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"md5", "sha256"}, c.HashTypes)
	assert.EqualValues(t, 1024*1024*1024, c.MaxFileSizeBytes)
}

func TestInvalidConfig(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"file.hash_types":    []string{"crc32", "sha256"},
		"file.max_file_size": "32 Hz",
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := config.Unpack(&c); err != nil {
		assert.Error(t, err)
		return
	}

	t.Fatal("expected error")
}
