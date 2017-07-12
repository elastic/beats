package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

// Defaults for config variables which are not set
const (
	DefaultType = "log"
)

type Config struct {
	Prospectors      []*common.Config `config:"prospectors"`
	RegistryFile     string           `config:"registry_file"`
	ConfigDir        string           `config:"config_dir"`
	ShutdownTimeout  time.Duration    `config:"shutdown_timeout"`
	Modules          []*common.Config `config:"modules"`
	ConfigProspector *common.Config   `config:"config.prospectors"`
	ConfigModules    *common.Config   `config:"config.modules"`
}

var (
	DefaultConfig = Config{
		RegistryFile:    "registry",
		ShutdownTimeout: 0,
	}
)

// getConfigFiles returns list of config files.
// In case path is a file, it will be directly returned.
// In case it is a directory, it will fetch all .yml files inside this directory
func getConfigFiles(path string) (configFiles []string, err error) {

	// Check if path is valid file or dir
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Create empty slice for config file list
	configFiles = make([]string, 0)

	if stat.IsDir() {
		files, err := filepath.Glob(path + "/*.yml")

		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, files...)

	} else {
		// Only 1 config file
		configFiles = append(configFiles, path)
	}

	return configFiles, nil
}

// mergeConfigFiles reads in all config files given by list configFiles and merges them into config
func mergeConfigFiles(configFiles []string, config *Config) error {

	for _, file := range configFiles {
		logp.Info("Additional configs loaded from: %s", file)

		tmpConfig := struct {
			Filebeat Config
		}{}
		err := cfgfile.Read(&tmpConfig, file)
		if err != nil {
			return fmt.Errorf("Failed to read %s: %s", file, err)
		}

		config.Prospectors = append(config.Prospectors, tmpConfig.Filebeat.Prospectors...)
	}

	return nil
}

// Fetches and merges all config files given by configDir. All are put into one config object
func (config *Config) FetchConfigs() error {

	configDir := config.ConfigDir

	// If option not set, do nothing
	if configDir == "" {
		return nil
	}

	// If configDir is relative, consider it relative to the config path
	configDir = paths.Resolve(paths.Config, configDir)

	// Check if optional configDir is set to fetch additional config files
	logp.Info("Additional config files are fetched from: %s", configDir)

	configFiles, err := getConfigFiles(configDir)

	if err != nil {
		log.Fatal("Could not use config_dir of: ", configDir, err)
		return err
	}

	err = mergeConfigFiles(configFiles, config)
	if err != nil {
		log.Fatal("Error merging config files: ", err)
		return err
	}

	return nil
}
