package cfgfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobManagerInit(t *testing.T) {
	// Wrong settings return error
	manager, err := NewGlobManager("dir/*.yml", ".noyml", ".disabled")
	assert.Error(t, err)
	assert.Nil(t, manager)
}

func TestGlobManager(t *testing.T) {
	// Create random temp directory
	dir, err := ioutil.TempDir("", "glob_manager")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Prepare scenario:
	content := []byte("test\n")
	err = ioutil.WriteFile(dir+"/config1.yml", content, 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile(dir+"/config2.yml", content, 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile(dir+"/config3.yml.disabled", content, 0644)
	assert.NoError(t, err)

	// Init Glob Manager
	glob := dir + "/*.yml"
	manager, err := NewGlobManager(glob, ".yml", ".disabled")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, manager.Exists("config1"))
	assert.True(t, manager.Exists("config2"))
	assert.True(t, manager.Exists("config3"))
	assert.False(t, manager.Exists("config4"))

	assert.True(t, manager.Enabled("config1"))
	assert.True(t, manager.Enabled("config2"))
	assert.False(t, manager.Enabled("config3"))

	assert.Equal(t, len(manager.ListEnabled()), 2)
	assert.Equal(t, len(manager.ListDisabled()), 1)

	// Test disable
	if err = manager.Disable("config2"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(manager.ListEnabled()), 1)
	assert.Equal(t, len(manager.ListDisabled()), 2)

	assert.Equal(t, manager.ListEnabled(), []string{"config1"})

	// Test enable
	if err = manager.Enable("config3"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(manager.ListEnabled()), 2)
	assert.Equal(t, len(manager.ListDisabled()), 1)

	assert.Equal(t, manager.ListDisabled(), []string{"config2"})

	// Check correct files layout:
	files, err := filepath.Glob(dir + "/*")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, files, []string{
		dir + "/config1.yml",
		dir + "/config2.yml.disabled",
		dir + "/config3.yml",
	})
}
