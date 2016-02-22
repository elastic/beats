package cfgfile

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
	"gopkg.in/yaml.v2"
)

// Command line flags
var configfile *string
var testConfig *bool

func init() {
	// The default config cannot include the beat name as it is not initialised when this
	// function is called, but see ChangeDefaultCfgfileFlag
	configfile = flag.String("c", "beat.yml", "Configuration file")
	testConfig = flag.Bool("configtest", false, "Test configuration and exit.")
}

// ChangeDefaultCfgfileFlag replaces the value and default value for the `-c` flag so that
// it reflects the beat name.
func ChangeDefaultCfgfileFlag(beatName string) error {
	cliflag := flag.Lookup("c")
	if cliflag == nil {
		return fmt.Errorf("Flag -c not found")
	}

	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("Failed to set default config file location because the absolute path to %s could not be obtained. %v", os.Args[0], err)
	}

	cliflag.DefValue = filepath.Join(path, beatName+".yml")

	return cliflag.Value.Set(cliflag.DefValue)
}

// Read reads the configuration from a yaml file into the given interface structure.
// In case path is not set this method reads from the default configuration file for the beat.
func Read(out interface{}, path string) error {

	if path == "" {
		path = *configfile
	}

	filecontent, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %s: %v. Exiting.", path, err)
	}

	filecontent = expandEnv(filecontent)

	if err = yaml.Unmarshal(filecontent, out); err != nil {
		return fmt.Errorf("YAML config parsing failed on %s: %v. Exiting", path, err)
	}

	return nil
}

// IsTestConfig returns whether or not this is configuration used for testing
func IsTestConfig() bool {
	return *testConfig
}

// expandEnv replaces ${var} or $var in config according to the values of the
// current environment variables. The replacement is case-sensitive. References
// to undefined variables are replaced by the empty string. A default value
// can be given by using the form ${var:default value}.
func expandEnv(config []byte) []byte {
	return []byte(os.Expand(string(config), func(key string) string {
		keyAndDefault := strings.SplitN(key, ":", 2)
		key = keyAndDefault[0]

		v := os.Getenv(key)
		if v == "" && len(keyAndDefault) == 2 {
			// Set value to the default.
			v = keyAndDefault[1]
			logp.Info("Replacing config environment variable '${%s}' with "+
				"default '%s'", key, keyAndDefault[1])
		} else {
			logp.Info("Replacing config environment variable '${%s}' with '%s'",
				key, v)
		}

		return v
	}))
}
