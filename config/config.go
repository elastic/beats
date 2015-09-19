package config

import (
	"github.com/elastic/libbeat/cfgfile"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Defaults for config variables which are not set
const (
	DefaultRegistryFile                      = ".filebeat"
	DefaultIgnoreOlderDuration time.Duration = math.MaxInt64
	DefaultScanFrequency       time.Duration = 10 * time.Second // 10 seconds
	DefaultSpoolSize           uint64        = 1024
	DefaultIdleTimeout         time.Duration = time.Second * 5
	DefaultHarvesterBufferSize int           = 16 << 10 // 16384
	DefaultTailOnRotate                      = false
)

type Config struct {
	Filebeat FilebeatConfig
}

type FilebeatConfig struct {
	Files               []FileConfig
	SpoolSize           uint64 `yaml:"spoolSize"`
	CpuProfileFile      string // TODO: Still needed?
	IdleTimeout         string `yaml:"idleTimeout"`
	IdleTimeoutDuration time.Duration
	RegistryFile        string
}

type FileConfig struct {
	Paths                 []string
	Fields                map[string]string
	Input                 string
	IgnoreOlder           string `yaml:"ignoreOlder"`
	IgnoreOlderDuration   time.Duration
	ScanFrequency         string `yaml:"scanFrequency"`
	ScanFrequencyDuration time.Duration
	HarvesterBufferSize   int  `yaml:"harvesterBufferSize"`
	TailOnRotate          bool `yaml:"tailOnRotate"`
}

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
		tmpConfig := &Config{}
		cfgfile.Read(tmpConfig, file)

		config.Filebeat.Files = append(config.Filebeat.Files, tmpConfig.Filebeat.Files...)
	}

	return nil
}

// Fetches and merges all config files given by Options.configArgs. All are put into one config object
func (config *Config) FetchConfigs(path string) {

	configFiles, err := getConfigFiles(path)

	if err != nil {
		log.Fatal("Could not use -configDir of ", path, err)
	}

	err = mergeConfigFiles(configFiles, config)

	if err != nil {
		log.Fatal("Error merging config files: ", err)
	}

	if len(config.Filebeat.Files) == 0 {
		log.Fatalf("No paths given. What files do you want me to watch?")
	}

}
