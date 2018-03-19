package cfgfile

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Command line flags.
var (
	// The default config cannot include the beat name as it is not initialized
	// when this variable is created. See ChangeDefaultCfgfileFlag which should
	// be called prior to flags.Parse().
	configfiles = common.StringArrFlag(nil, "c", "beat.yml", "Configuration file, relative to path.config")
	overwrites  = common.SettingFlag(nil, "E", "Configuration overwrite")
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
	homePath   *string
	configPath *string
)

func init() {
	// add '-path.x' options overwriting paths in 'overwrites' config
	makePathFlag := func(name, usage string) *string {
		return common.ConfigOverwriteFlag(nil, overwrites, name, name, "", usage)
	}

	homePath = makePathFlag("path.home", "Home path")
	configPath = makePathFlag("path.config", "Configuration path")
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
	configfiles.SetDefault(beatName + ".yml")
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

	if len(overwrites.GetFields()) > 0 {
		overwrites.PrintDebugf("CLI setting overwrites (-E flag):")
	}

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
		return err
	}

	return config.Unpack(out)
}

// Load reads the configuration from a YAML file structure. If path is empty
// this method reads from the configuration file specified by the '-c' command
// line flag.
func Load(path string) (*common.Config, error) {
	var config *common.Config
	var err error

	cfgpath := GetPathConfig()

	if path == "" {
		list := []string{}
		for _, cfg := range configfiles.List() {
			if !filepath.IsAbs(cfg) {
				list = append(list, filepath.Join(cfgpath, cfg))
			} else {
				list = append(list, cfg)
			}
		}
		config, err = common.LoadFiles(list...)
	} else {
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfgpath, path)
		}
		config, err = common.LoadFile(path)
	}
	if err != nil {
		return nil, err
	}

	config, err = common.MergeConfigs(
		defaults,
		config,
		overwrites,
	)
	if err != nil {
		return nil, err
	}

	config.PrintDebugf("Complete configuration loaded:")
	return config, nil
}

// LoadList loads a list of configs data from the given file.
func LoadList(file string) ([]*common.Config, error) {
	logp.Debug("cfgfile", "Load config from file: %s", file)
	rawConfig, err := common.LoadFile(file)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	var c []*common.Config
	err = rawConfig.Unpack(&c)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration from file %s: %s", file, err)
	}

	return c, nil
}

// GetPathConfig returns ${path.config}. If ${path.config} is not set, ${path.home} is returned.
func GetPathConfig() string {
	if *configPath != "" {
		return *configPath
	} else if *homePath != "" {
		return *homePath
	}
	// TODO: Do we need this or should we always return *homePath?
	return ""
}

// IsTestConfig returns whether or not this is configuration used for testing
func IsTestConfig() bool {
	return *testConfig
}
