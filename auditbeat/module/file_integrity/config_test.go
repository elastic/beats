package file_integrity

import (
	"os"
	"path/filepath"
	"regexp/syntax"
	"testing"

	"github.com/joeshaw/multierror"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-ucfg"
)

func TestConfig(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"paths":             []string{"/usr/bin"},
		"hash_types":        []string{"md5", "sha256"},
		"max_file_size":     "1 GiB",
		"scan_rate_per_sec": "10MiB",
		"exclude_files":     []string{`\.DS_Store$`, `\.swp$`},
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := config.Unpack(&c); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []HashType{MD5, SHA256}, c.HashTypes)
	assert.EqualValues(t, 1024*1024*1024, c.MaxFileSizeBytes)
	assert.EqualValues(t, 1024*1024*10, c.ScanRateBytesPerSec)
	assert.Len(t, c.ExcludeFiles, 2)
	assert.EqualValues(t, `\.DS_Store(?-m:$)`, c.ExcludeFiles[0].String())
	assert.EqualValues(t, `\.swp(?-m:$)`, c.ExcludeFiles[1].String())
}

func TestConfigInvalid(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"paths":             []string{"/usr/bin"},
		"hash_types":        []string{"crc32", "sha256", "hmac"},
		"max_file_size":     "32 Hz",
		"scan_rate_per_sec": "32mb/sec",
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	err = config.Unpack(&c)
	if err == nil {
		t.Fatal("expected error")
	}

	t.Log(err)

	ucfgErr, ok := err.(ucfg.Error)
	if !ok {
		t.Fatal("expected ucfg.Error")
	}

	merr, ok := ucfgErr.Reason().(*multierror.MultiError)
	if !ok {
		t.Fatal("expected MultiError")
	}
	assert.Len(t, merr.Errors, 4)

	config, err = common.NewConfigFrom(map[string]interface{}{
		"paths":         []string{"/usr/bin"},
		"hash_types":    []string{"crc32", "sha256", "hmac"},
		"exclude_files": "unmatched)",
	})
	if err != nil {
		t.Fatal(err)
	}

	c = defaultConfig
	err = config.Unpack(&c)
	if err == nil {
		t.Fatal("expected error")
	}

	t.Log(err)

	ucfgErr, ok = err.(ucfg.Error)
	if !ok {
		t.Fatal("expected ucfg.Error")
	}

	_, ok = ucfgErr.Reason().(*syntax.Error)
	assert.True(t, ok)
}

func TestConfigInvalidMaxFileSize(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"paths":         []string{"/usr/bin"},
		"max_file_size": "0", // Value must be >= 0.
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := config.Unpack(&c); err != nil {
		t.Log(err)
		return
	}

	t.Fatal("expected error")
}

func TestConfigEvalSymlinks(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	config, err := common.NewConfigFrom(map[string]interface{}{
		"paths": []string{filepath.Join(dir, "link_to_subdir")},
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := config.Unpack(&c); err != nil {
		t.Log(err)
		return
	}

	// link_to_subdir was resolved to subdir.
	assert.Equal(t, filepath.Base(c.Paths[0]), "subdir")
}

func TestConfigRemoveDuplicates(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"paths": []string{"/path/a", "/path/a"},
	})
	if err != nil {
		t.Fatal(err)
	}

	c := defaultConfig
	if err := config.Unpack(&c); err != nil {
		t.Log(err)
		return
	}

	assert.Len(t, c.Paths, 1)
}
