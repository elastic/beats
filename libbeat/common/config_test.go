package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigPrintDebug(t *testing.T) {
	tests := []struct {
		name      string
		selectors string
		config    map[string]interface{}
		expected  string
	}{
		{
			"No selector -> no output",
			"",
			map[string]interface{}{"name": "test"},
			"",
		},
		{
			"config selector redacts password in nested config",
			"config",
			map[string]interface{}{
				"config": map[string]interface{}{
					"password": "secret",
				},
			},
			`test:
{
  "config": {
    "password": "xxxxx"
  }
}
`,
		},
		{
			"config selector redacts password in nested array",
			"config",
			map[string]interface{}{
				"arr": []interface{}{
					map[string]interface{}{
						"password": "secret",
					},
				},
			},
			`test:
{
  "arr": [
    {
      "password": "xxxxx"
    }
  ]
}
`,
		},
		{
			"config-with-passwords does not redact",
			"config-with-passwords",
			map[string]interface{}{
				"config": map[string]interface{}{
					"password": "secret",
				},
			},
			`test:
{
  "config": {
    "password": "secret"
  }
}
`,
		},
	}

	origSelector := hasSelector
	origDebugf := configDebugf
	defer func() {
		hasSelector = origSelector
		configDebugf = origDebugf
	}()

	var buf string
	configDebugf = func(selector, msg string, extra ...interface{}) {
		if hasSelector(selector) {
			buf = buf + fmt.Sprintf(msg, extra...) + "\n"
		}
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.name)

		// reset selector and output buffer
		selectors := MakeStringSet(strings.Split(test.selectors, ",")...)
		buf = ""
		hasSelector = selectors.Has

		// create config
		cfg, err := NewConfigFrom(test.config)
		if err != nil {
			t.Fatal(err)
		}

		// create debug output
		cfg.PrintDebugf("test:")

		// validate debug output
		assert.Equal(t, test.expected, buf)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	if !IsStrictPerms() {
		t.Skip("Skipping test because strict.perms is disabled")
	}

	f, err := ioutil.TempFile("", "writableConfig.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	f.WriteString(`test.data: [1, 2, 3, 4]`)
	f.Sync()

	if _, err = LoadFile(f.Name()); err != nil {
		t.Fatal(err)
	}

	// Permissions checking isn't implemented for Windows DACLs.
	if runtime.GOOS == "windows" {
		return
	}

	if err = os.Chmod(f.Name(), 0460); err != nil {
		t.Fatal(err)
	}

	// Read will fail because config is group writable.
	_, err = LoadFile(f.Name())
	if assert.Error(t, err, "expected writable error") {
		assert.Contains(t, err.Error(), "writable")
	}
}
