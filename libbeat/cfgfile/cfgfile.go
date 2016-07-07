package cfgfile

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
)

// Command line flags.
var (
	// The default config cannot include the beat name as it is not initialized
	// when this variable is created. See ChangeDefaultCfgfileFlag which should
	// be called prior to flags.Parse().
	configfiles = flagArgList("c", "beat.yml", "Configuration file")
	testConfig  = flag.Bool("configtest", false, "Test configuration and exit.")
)

// ChangeDefaultCfgfileFlag replaces the value and default value for the `-c`
// flag so that it reflects the beat name.
func ChangeDefaultCfgfileFlag(beatName string) error {
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("Failed to set default config file location because "+
			"the absolute path to %s could not be obtained. %v",
			os.Args[0], err)
	}

	path = filepath.Join(path, beatName+".yml")
	configfiles.SetDefault(path)
	return nil
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
		return common.LoadFiles(configfiles.list...)
	}
	return common.LoadFile(path)
}

// IsTestConfig returns whether or not this is configuration used for testing
func IsTestConfig() bool {
	return *testConfig
}
