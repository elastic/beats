package cfgfile

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
)

// Command line flags.
var (
	// The default config cannot include the beat name as it is not initialized
	// when this variable is created. See ChangeDefaultCfgfileFlag which should
	// be called prior to flags.Parse().
	configfile = flag.String("c", "beat.yml", "Configuration file")
	testConfig = flag.Bool("configtest", false, "Test configuration and exit.")
)

// ChangeDefaultCfgfileFlag replaces the value and default value for the `-c`
// flag so that it reflects the beat name.
func ChangeDefaultCfgfileFlag(beatName string) error {
	cliflag := flag.Lookup("c")
	if cliflag == nil {
		return fmt.Errorf("Flag -c not found")
	}

	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("Failed to set default config file location because "+
			"the absolute path to %s could not be obtained. %v",
			os.Args[0], err)
	}

	cliflag.DefValue = filepath.Join(path, beatName+".yml")

	return cliflag.Value.Set(cliflag.DefValue)
}

// Deprecated: Please use Load().
//
// Read reads the configuration from a YAML file into the given interface
// structure. If path is empty this method reads from the configuration
// file specified by the '-c' command line flag.
func Read(out interface{}, path string) error {
	config, err := Load(path)
	if err != nil {
		return nil
	}

	return config.Unpack(out)
}

// Load reads the configuration from a YAML file structure. If path is empty
// this method reads from the configuration file specified by the '-c' command
// line flag.
func Load(path string) (*common.Config, error) {
	if path == "" {
		path = *configfile
	}

	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", path, err)
	}
	fileContent, err = expandEnv(filepath.Base(path), fileContent)
	if err != nil {
		return nil, err
	}

	config, err := common.NewConfigWithYAML(fileContent, path)
	if err != nil {
		return nil, fmt.Errorf("YAML config parsing failed on %s: %v", path, err)
	}

	return config, nil
}

// IsTestConfig returns whether or not this is configuration used for testing
func IsTestConfig() bool {
	return *testConfig
}
