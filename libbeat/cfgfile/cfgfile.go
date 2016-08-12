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
	configfiles = flagArgList("c", "beat.yml", "Configuration file `path`")
	overwrites  = common.NewFlagConfig(nil, nil, "E", "Configuration overwrite")
	testConfig  = flag.Bool("configtest", false, "Test configuration and exit.")

	// Additional default settings, that must be available for variable expansion
	defaults = mustNewConfigFrom(map[string]interface{}{
		"path": map[string]interface{}{
			"home":   ".", // to be initialized by beat
			"config": "${path.home}",
			"data":   fmt.Sprint("${path.home}", string(os.PathSeparator), "data"),
			"logs":   fmt.Sprint("${path.home}", string(os.PathSeparator), "logs"),
		},
	})

	// home-path CLI flag (initialized in init)
	homePath *string
)

func init() {
	// add '-path.x' options overwriting paths in 'overwrites' config
	makePathFlag := func(name, usage string) *string {
		return common.NewFlagOverwrite(nil, overwrites, name, name, "", usage)
	}

	homePath = makePathFlag("path.home", "Home path")
	makePathFlag("path.config", "Configuration path")
	makePathFlag("path.data", "Data path")
	makePathFlag("path.logs", "Logs path")
}

func mustNewConfigFrom(from interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(from)
	if err != nil {
		panic(err)
	}
	return cfg
}

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

// HandleFlags adapts default config settings based on command line flags.
func HandleFlags() error {

	// default for the home path is the binary location
	home, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		if *homePath == "" {
			return fmt.Errorf("The absolute path to %s could not be obtained. %v",
				os.Args[0], err)
		}
		home = *homePath
	}

	defaults.SetString("path.home", -1, home)
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
	var config *common.Config
	var err error

	if path == "" {
		config, err = common.LoadFiles(configfiles.list...)
	} else {
		config, err = common.LoadFile(path)
	}
	if err != nil {
		return nil, err
	}

	return common.MergeConfigs(
		defaults,
		config,
		overwrites,
	)
}

// IsTestConfig returns whether or not this is configuration used for testing
func IsTestConfig() bool {
	return *testConfig
}
