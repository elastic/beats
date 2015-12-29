package config

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Defaults for config variables which are not set
const (
	DefaultRegistryFile                      = ".filebeat"
	DefaultIgnoreOlderDuration time.Duration = 24 * time.Hour
	DefaultScanFrequency       time.Duration = 10 * time.Second
	DefaultSpoolSize           uint64        = 1024
	DefaultIdleTimeout         time.Duration = 5 * time.Second
	DefaultHarvesterBufferSize int           = 16 << 10 // 16384
	DefaultInputType                         = "log"
	DefaultDocumentType                      = "log"
	DefaultTailFiles                         = false
	DefaultBackoff                           = 1 * time.Second
	DefaultBackoffFactor                     = 2
	DefaultMaxBackoff                        = 10 * time.Second
	DefaultForceCloseFiles                   = false
)

type Config struct {
	Filebeat FilebeatConfig
}

type FilebeatConfig struct {
	Prospectors         []ProspectorConfig
	SpoolSize           uint64 `yaml:"spool_size"`
	IdleTimeout         string `yaml:"idle_timeout"`
	IdleTimeoutDuration time.Duration
	RegistryFile        string `yaml:"registry_file"`
	ConfigDir           string `yaml:"config_dir"`
}

type ProspectorConfig struct {
	Paths                 []string
	Input                 string
	IgnoreOlder           string `yaml:"ignore_older"`
	IgnoreOlderDuration   time.Duration
	ScanFrequency         string `yaml:"scan_frequency"`
	ScanFrequencyDuration time.Duration
	Harvester             HarvesterConfig `yaml:",inline"`
	ExcludeFiles          []string        `yaml:"exclude_files"`
	ExcludeFilesRegexp    []*regexp.Regexp
}

type HarvesterConfig struct {
	InputType          string `yaml:"input_type"`
	Fields             common.MapStr
	FieldsUnderRoot    bool   `yaml:"fields_under_root"`
	BufferSize         int    `yaml:"harvester_buffer_size"`
	TailFiles          bool   `yaml:"tail_files"`
	Encoding           string `yaml:"encoding"`
	DocumentType       string `yaml:"document_type"`
	Backoff            string `yaml:"backoff"`
	BackoffDuration    time.Duration
	BackoffFactor      int    `yaml:"backoff_factor"`
	MaxBackoff         string `yaml:"max_backoff"`
	MaxBackoffDuration time.Duration
	ForceCloseFiles    bool             `yaml:"force_close_files"`
	ExcludeLines       []string         `yaml:"exclude_lines"`
	IncludeLines       []string         `yaml:"include_lines"`
	MaxBytes           *int             `yaml:"max_bytes"`
	Multiline          *MultilineConfig `yaml:"multiline"`
}

type MultilineConfig struct {
	Pattern  string `yaml:"pattern"`
	Negate   bool   `yaml:"negate"`
	Match    string `yaml:"match"`
	MaxLines *int   `yaml:"max_lines"`
	Timeout  string `yaml:"timeout"`
}

const (
	LogInputType   = "log"
	StdinInputType = "stdin"
)

// List of valid input types
var ValidInputType = map[string]struct{}{
	StdinInputType: {},
	LogInputType:   {},
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
		logp.Info("Additional configs loaded from: %s", file)

		tmpConfig := &Config{}
		cfgfile.Read(tmpConfig, file)

		config.Filebeat.Prospectors = append(config.Filebeat.Prospectors, tmpConfig.Filebeat.Prospectors...)
	}

	return nil
}

// Fetches and merges all config files given by configDir. All are put into one config object
func (config *Config) FetchConfigs() {

	configDir := config.Filebeat.ConfigDir

	// If option not set, do nothing
	if configDir == "" {
		return
	}

	// Check if optional configDir is set to fetch additional config files
	logp.Info("Additional config files are fetched from: %s", configDir)

	configFiles, err := getConfigFiles(configDir)

	if err != nil {
		log.Fatal("Could not use config_dir of: ", configDir, err)
	}

	err = mergeConfigFiles(configFiles, config)

	if err != nil {
		log.Fatal("Error merging config files: ", err)
	}

	if len(config.Filebeat.Prospectors) == 0 {
		log.Fatalf("No paths given. What files do you want me to watch?")
	}
}
